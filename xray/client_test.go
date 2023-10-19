// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package xray

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/http2"
)

func newRequest(ctx context.Context, method, url string, body io.Reader) (context.Context, *Segment, *http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, nil, nil, err
	}
	ctx, root := BeginSegment(ctx, "Test")
	req = req.WithContext(ctx)
	return ctx, root, req, nil
}

func httpDoTest(ctx context.Context, client *http.Client, method, url string, body io.Reader) error {
	_, root, req, err := newRequest(ctx, method, url, body)
	if err != nil {
		return err
	}
	defer root.Close(nil)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(ioutil.Discard, resp.Body)
	return nil
}

func TestNilClient(t *testing.T) {
	c := Client(nil)
	assert.Equal(t, http.DefaultClient.Jar, c.Jar)
	assert.Equal(t, http.DefaultClient.Timeout, c.Timeout)
	assert.Equal(t, &roundtripper{Base: http.DefaultTransport}, c.Transport)
}

func TestRoundTripper(t *testing.T) {
	ht := http.DefaultTransport
	rt := RoundTripper(ht)
	assert.Equal(t, &roundtripper{Base: http.DefaultTransport}, rt)
}

func TestRoundTrip(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	const content = `200 - Nothing to see`
	const responseContentLength = len(content)

	ch := make(chan XRayHeaders, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch <- ParseHeadersForTest(r.Header)
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(content)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	client := Client(nil)

	err := httpDoTest(ctx, client, http.MethodGet, ts.URL, nil)
	if !assert.NoError(t, err) {
		return
	}

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}
	var subseg *Segment
	if assert.NoError(t, json.Unmarshal(seg.Subsegments[0], &subseg)) {
		assert.Equal(t, "remote", subseg.Namespace)
		assert.Equal(t, http.MethodGet, subseg.HTTP.Request.Method)
		assert.Equal(t, ts.URL, subseg.HTTP.Request.URL)
		assert.Equal(t, http.StatusOK, subseg.HTTP.Response.Status)
		assert.Equal(t, responseContentLength, subseg.HTTP.Response.ContentLength)
		assert.False(t, subseg.Throttle)
		assert.False(t, subseg.Error)
		assert.False(t, subseg.Fault)
	}
	headers := <-ch
	assert.Equal(t, headers.RootTraceID, seg.TraceID)

	var connectSeg *Segment
	for _, sub := range subseg.Subsegments {
		var seg *Segment
		if !assert.NoError(t, json.Unmarshal(sub, &seg)) {
			continue
		}
		if seg.Name == "connect" {
			connectSeg = seg
		}
	}

	// Ensure that a 'connect' subsegment was created and closed
	assert.Equal(t, "connect", connectSeg.Name)
	assert.False(t, connectSeg.InProgress)
	assert.NotZero(t, connectSeg.EndTime)
	assert.NotEmpty(t, connectSeg.Subsegments)

	// Ensure that the 'connect' subsegments are completed.
	for _, sub := range connectSeg.Subsegments {
		var seg *Segment
		if !assert.NoError(t, json.Unmarshal(sub, &seg)) {
			continue
		}
		assert.False(t, seg.InProgress)
		assert.NotZero(t, seg.EndTime)
	}
}

func TestRoundTripWithQueryParameter(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	const content = `200 - Nothing to see`
	const responseContentLength = len(content)
	const queryParam = `?key=value`

	ch := make(chan XRayHeaders, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.String(), queryParam)
		ch <- ParseHeadersForTest(r.Header)
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(content)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	client := Client(nil)

	err := httpDoTest(ctx, client, http.MethodGet, ts.URL+queryParam, nil)
	if !assert.NoError(t, err) {
		return
	}

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}
	var subseg *Segment
	if assert.NoError(t, json.Unmarshal(seg.Subsegments[0], &subseg)) {
		assert.Equal(t, "remote", subseg.Namespace)
		assert.Equal(t, http.MethodGet, subseg.HTTP.Request.Method)
		assert.Equal(t, ts.URL, subseg.HTTP.Request.URL)
		assert.Equal(t, http.StatusOK, subseg.HTTP.Response.Status)
		assert.Equal(t, responseContentLength, subseg.HTTP.Response.ContentLength)
		assert.False(t, subseg.Throttle)
		assert.False(t, subseg.Error)
		assert.False(t, subseg.Fault)
	}
	headers := <-ch
	assert.Equal(t, headers.RootTraceID, seg.TraceID)
}

func TestRoundTripWithBasicAuth(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	const content = `200 - Nothing to see`
	const responseContentLength = len(content)

	var userInfo = url.UserPassword("user", "pass")

	ch := make(chan XRayHeaders, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		pass, _ := userInfo.Password()
		assert.Equal(t, ok, true)
		assert.Equal(t, username, userInfo.Username())
		assert.Equal(t, password, pass)
		ch <- ParseHeadersForTest(r.Header)
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(content)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	client := Client(nil)

	u, err := url.Parse(ts.URL)
	if !assert.NoError(t, err) {
		return
	}
	u.User = userInfo

	err = httpDoTest(ctx, client, http.MethodGet, u.String(), nil)
	if !assert.NoError(t, err) {
		return
	}

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}
	var subseg *Segment
	if assert.NoError(t, json.Unmarshal(seg.Subsegments[0], &subseg)) {
		assert.Equal(t, "remote", subseg.Namespace)
		assert.Equal(t, http.MethodGet, subseg.HTTP.Request.Method)
		assert.Equal(t, stripURL(*u), subseg.HTTP.Request.URL)
		assert.Equal(t, http.StatusOK, subseg.HTTP.Response.Status)
		assert.Equal(t, responseContentLength, subseg.HTTP.Response.ContentLength)
		assert.False(t, subseg.Throttle)
		assert.False(t, subseg.Error)
		assert.False(t, subseg.Fault)
	}
	headers := <-ch
	assert.Equal(t, headers.RootTraceID, seg.TraceID)
}

func TestRoundTripWithError(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	const content = `403 - Nothing to see`
	const responseContentLength = len(content)

	ch := make(chan XRayHeaders, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch <- ParseHeadersForTest(r.Header)
		w.WriteHeader(http.StatusForbidden)
		if _, err := w.Write([]byte(content)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	client := Client(nil)

	err := httpDoTest(ctx, client, http.MethodGet, ts.URL, nil)
	if !assert.NoError(t, err) {
		return
	}

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}
	var subseg *Segment
	if assert.NoError(t, json.Unmarshal(seg.Subsegments[0], &subseg)) {
		assert.Equal(t, "remote", subseg.Namespace)
		assert.Equal(t, http.MethodGet, subseg.HTTP.Request.Method)
		assert.Equal(t, ts.URL, subseg.HTTP.Request.URL)
		assert.Equal(t, http.StatusForbidden, subseg.HTTP.Response.Status)
		assert.Equal(t, responseContentLength, subseg.HTTP.Response.ContentLength)
		assert.False(t, subseg.Throttle)
		assert.True(t, subseg.Error)
		assert.False(t, subseg.Fault)
	}
	headers := <-ch
	assert.Equal(t, headers.RootTraceID, seg.TraceID)
}

func TestRoundTripWithThrottle(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	const content = `429 - Nothing to see`
	const responseContentLength = len(content)

	ch := make(chan XRayHeaders, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch <- ParseHeadersForTest(r.Header)
		w.WriteHeader(http.StatusTooManyRequests)
		if _, err := w.Write([]byte(content)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	client := Client(nil)

	err := httpDoTest(ctx, client, http.MethodGet, ts.URL, nil)
	if !assert.NoError(t, err) {
		return
	}

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}
	var subseg *Segment
	if assert.NoError(t, json.Unmarshal(seg.Subsegments[0], &subseg)) {
		assert.Equal(t, "remote", subseg.Namespace)
		assert.Equal(t, http.MethodGet, subseg.HTTP.Request.Method)
		assert.Equal(t, ts.URL, subseg.HTTP.Request.URL)
		assert.Equal(t, http.StatusTooManyRequests, subseg.HTTP.Response.Status)
		assert.Equal(t, responseContentLength, subseg.HTTP.Response.ContentLength)
		assert.True(t, subseg.Throttle)
		assert.True(t, subseg.Error)
		assert.False(t, subseg.Fault)
	}
	headers := <-ch
	assert.Equal(t, headers.RootTraceID, seg.TraceID)
}

func TestRoundTripFault(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	const content = `503 - Nothing to see`
	const responseContentLength = len(content)

	ch := make(chan XRayHeaders, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ch <- ParseHeadersForTest(r.Header)
		w.WriteHeader(http.StatusServiceUnavailable)
		if _, err := w.Write([]byte(content)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	client := Client(nil)

	err := httpDoTest(ctx, client, http.MethodGet, ts.URL, nil)
	if !assert.NoError(t, err) {
		return
	}

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}
	var subseg *Segment
	if assert.NoError(t, json.Unmarshal(seg.Subsegments[0], &subseg)) {
		assert.Equal(t, "remote", subseg.Namespace)
		assert.Equal(t, http.MethodGet, subseg.HTTP.Request.Method)
		assert.Equal(t, ts.URL, subseg.HTTP.Request.URL)
		assert.Equal(t, http.StatusServiceUnavailable, subseg.HTTP.Response.Status)
		assert.Equal(t, responseContentLength, subseg.HTTP.Response.ContentLength)
		assert.False(t, subseg.Throttle)
		assert.False(t, subseg.Error)
		assert.True(t, subseg.Fault)
	}
	headers := <-ch
	assert.Equal(t, headers.RootTraceID, seg.TraceID)
}

func TestBadRoundTrip(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	client := Client(nil)

	doErr := httpDoTest(ctx, client, http.MethodGet, "unknown-scheme://localhost:8000", nil)
	assert.Error(t, doErr)

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}
	var subseg *Segment
	if assert.NoError(t, json.Unmarshal(seg.Subsegments[0], &subseg)) {
		assert.Contains(t, fmt.Sprintf("%v", doErr), subseg.Cause.Exceptions[0].Message)
	}
}

func TestBadRoundTripDial(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	client := Client(nil)

	doErr := httpDoTest(ctx, client, http.MethodGet, "http://domain.invalid:8000", nil)
	assert.Error(t, doErr)

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}
	var subseg *Segment
	if assert.NoError(t, json.Unmarshal(seg.Subsegments[0], &subseg)) {
		assert.Contains(t, fmt.Sprintf("%v", doErr), subseg.Cause.Exceptions[0].Message)

		// Also ensure that the 'connect' subsegment is closed and showing fault
		var connectSeg *Segment
		if assert.NoError(t, json.Unmarshal(subseg.Subsegments[0], &connectSeg)) {
			assert.Equal(t, "connect", connectSeg.Name)
			assert.NotZero(t, connectSeg.EndTime)
			assert.False(t, connectSeg.InProgress)
			assert.True(t, connectSeg.Fault)
			assert.NotEmpty(t, connectSeg.Subsegments)
		}
	}
}

func TestRoundTripReuseDatarace(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`200 - Nothing to see`)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	client := Client(nil)

	var wg sync.WaitGroup
	n := 100
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			err := httpDoTest(ctx, client, http.MethodGet, ts.URL, nil)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
}

func TestRoundTripReuseTLSDatarace(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`200 - Nothing to see`)); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	client := Client(ts.Client())

	var wg sync.WaitGroup
	n := 100
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			err := httpDoTest(ctx, client, http.MethodGet, ts.URL, nil)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
}

func TestRoundTripReuseHTTP2Datarace(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !r.ProtoAtLeast(2, 0) {
			panic("want http/2, got " + r.Proto)
		}
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`200 - Nothing to see`)); err != nil {
			panic(err)
		}
	}))

	// configure http/2
	if err := http2.ConfigureServer(ts.Config, nil); !assert.NoError(t, err) {
		return
	}
	ts.TLS = ts.Config.TLSConfig
	ts.StartTLS()
	defer ts.Close()
	client := ts.Client()
	if err := http2.ConfigureTransport(client.Transport.(*http.Transport)); !assert.NoError(t, err) {
		return
	}
	client = Client(client)

	var wg sync.WaitGroup
	n := 100
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			err := httpDoTest(ctx, client, http.MethodGet, ts.URL, nil)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
}

// Benchmarks
func BenchmarkClient(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Client(nil)
	}
}
