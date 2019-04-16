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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/http2"
)

var rt *roundtripper

func init() {
	rt = &roundtripper{
		Base: http.DefaultTransport,
	}
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
	var responseContentLength int
	var headers XRayHeaders
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = ParseHeadersForTest(r.Header)
		b := []byte(`200 - Nothing to see`)
		responseContentLength = len(b)
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}))

	defer ts.Close()

	reader := strings.NewReader("")
	ctx, root := BeginSegment(context.Background(), "Test")
	req, _ := http.NewRequest("GET", ts.URL, reader)
	req = req.WithContext(ctx)
	_, err := rt.RoundTrip(req)
	root.Close(nil)
	assert.NoError(t, err)

	s, e := TestDaemon.Recv()
	assert.NoError(t, e)
	subseg := &Segment{}
	assert.NoError(t, json.Unmarshal(s.Subsegments[0], &subseg))
	assert.Equal(t, "remote", subseg.Namespace)
	assert.Equal(t, "GET", subseg.HTTP.Request.Method)
	assert.Equal(t, ts.URL, subseg.HTTP.Request.URL)
	assert.Equal(t, 200, subseg.HTTP.Response.Status)
	assert.Equal(t, responseContentLength, subseg.HTTP.Response.ContentLength)
	assert.Equal(t, headers.RootTraceID, s.TraceID)

	connectSeg := &Segment{}
	for _, sub := range subseg.Subsegments {
		tempSeg := &Segment{}
		assert.NoError(t, json.Unmarshal(sub, &tempSeg))
		if tempSeg.Name == "connect" {
			connectSeg = tempSeg
			break
		}
	}

	// Ensure that a 'connect' subsegment was created and closed
	assert.Equal(t, "connect", connectSeg.Name)
	assert.False(t, connectSeg.InProgress)
	assert.NotZero(t, connectSeg.EndTime)
	assert.NotEmpty(t, connectSeg.Subsegments)

	// Ensure that the 'connect' subsegments are completed.
	for _, sub := range connectSeg.Subsegments {
		tempSeg := &Segment{}
		assert.NoError(t, json.Unmarshal(sub, &tempSeg))
		assert.False(t, tempSeg.InProgress)
		assert.NotZero(t, tempSeg.EndTime)
	}
}

func TestRoundTripWithError(t *testing.T) {
	var responseContentLength int
	var headers XRayHeaders
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = ParseHeadersForTest(r.Header)
		b := []byte(`403 - Nothing to see`)
		responseContentLength = len(b)
		w.WriteHeader(http.StatusForbidden)
		w.Write(b)
	}))

	defer ts.Close()

	reader := strings.NewReader("")
	ctx, root := BeginSegment(context.Background(), "Test")
	req, _ := http.NewRequest("GET", ts.URL, reader)
	req = req.WithContext(ctx)
	_, err := rt.RoundTrip(req)
	root.Close(nil)
	assert.NoError(t, err)

	s, e := TestDaemon.Recv()
	assert.NoError(t, e)
	subseg := &Segment{}
	assert.NoError(t, json.Unmarshal(s.Subsegments[0], &subseg))
	assert.Equal(t, "remote", subseg.Namespace)
	assert.Equal(t, "GET", subseg.HTTP.Request.Method)
	assert.Equal(t, ts.URL, subseg.HTTP.Request.URL)
	assert.Equal(t, 403, subseg.HTTP.Response.Status)
	assert.Equal(t, responseContentLength, subseg.HTTP.Response.ContentLength)
	assert.Equal(t, headers.RootTraceID, s.TraceID)
}

func TestRoundTripWithThrottle(t *testing.T) {
	var responseContentLength int
	var headers XRayHeaders
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = ParseHeadersForTest(r.Header)

		b := []byte(`429 - Nothing to see`)
		responseContentLength = len(b)
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write(b)
	}))

	defer ts.Close()

	reader := strings.NewReader("")
	ctx, root := BeginSegment(context.Background(), "Test")
	req := httptest.NewRequest("GET", ts.URL, reader)
	req = req.WithContext(ctx)
	_, err := rt.RoundTrip(req)
	root.Close(nil)
	assert.NoError(t, err)

	s, e := TestDaemon.Recv()
	assert.NoError(t, e)
	subseg := &Segment{}
	assert.NoError(t, json.Unmarshal(s.Subsegments[0], &subseg))
	assert.Equal(t, "remote", subseg.Namespace)
	assert.Equal(t, "GET", subseg.HTTP.Request.Method)
	assert.Equal(t, ts.URL, subseg.HTTP.Request.URL)
	assert.Equal(t, 429, subseg.HTTP.Response.Status)
	assert.Equal(t, responseContentLength, subseg.HTTP.Response.ContentLength)
	assert.Equal(t, headers.RootTraceID, s.TraceID)
}

func TestRoundTripFault(t *testing.T) {
	var responseContentLength int
	var headers XRayHeaders
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = ParseHeadersForTest(r.Header)

		b := []byte(`510 - Nothing to see`)
		responseContentLength = len(b)
		w.WriteHeader(http.StatusNotExtended)
		w.Write(b)
	}))

	defer ts.Close()

	reader := strings.NewReader("")
	ctx, root := BeginSegment(context.Background(), "Test")
	req := httptest.NewRequest("GET", ts.URL, reader)
	req = req.WithContext(ctx)
	_, err := rt.RoundTrip(req)
	root.Close(nil)
	assert.NoError(t, err)

	s, e := TestDaemon.Recv()
	assert.NoError(t, e)
	subseg := &Segment{}
	assert.NoError(t, json.Unmarshal(s.Subsegments[0], &subseg))
	assert.Equal(t, "remote", subseg.Namespace)
	assert.Equal(t, "GET", subseg.HTTP.Request.Method)
	assert.Equal(t, ts.URL, subseg.HTTP.Request.URL)
	assert.Equal(t, 510, subseg.HTTP.Response.Status)
	assert.Equal(t, responseContentLength, subseg.HTTP.Response.ContentLength)
	assert.Equal(t, headers.RootTraceID, s.TraceID)
}

func TestBadRoundTrip(t *testing.T) {
	ctx, root := BeginSegment(context.Background(), "Test")
	reader := strings.NewReader("")
	req := httptest.NewRequest("GET", "httpz://localhost:8000", reader)
	req = req.WithContext(ctx)
	_, err := rt.RoundTrip(req)
	root.Close(nil)
	assert.Error(t, err)

	s, e := TestDaemon.Recv()
	assert.NoError(t, e)
	subseg := &Segment{}
	assert.NoError(t, json.Unmarshal(s.Subsegments[0], &subseg))
	assert.Equal(t, fmt.Sprintf("%v", err), subseg.Cause.Exceptions[0].Message)
}

func TestBadRoundTripDial(t *testing.T) {
	ctx, root := BeginSegment(context.Background(), "Test")
	reader := strings.NewReader("")
	// Make a request against an unreachable endpoint.
	req := httptest.NewRequest("GET", "https://0.0.0.0:0", reader)
	req = req.WithContext(ctx)
	_, err := rt.RoundTrip(req)
	root.Close(nil)
	assert.Error(t, err)

	s, e := TestDaemon.Recv()
	assert.NoError(t, e)
	subseg := &Segment{}
	assert.NoError(t, json.Unmarshal(s.Subsegments[0], &subseg))
	assert.Equal(t, fmt.Sprintf("%v", err), subseg.Cause.Exceptions[0].Message)

	// Also ensure that the 'connect' subsegment is closed and showing fault
	connectSeg := &Segment{}
	assert.NoError(t, json.Unmarshal(subseg.Subsegments[0], &connectSeg))
	assert.Equal(t, "connect", connectSeg.Name)
	assert.NotZero(t, connectSeg.EndTime)
	assert.False(t, connectSeg.InProgress)
	assert.True(t, connectSeg.Fault)
	assert.NotEmpty(t, connectSeg.Subsegments)
}

func TestRoundTripReuseDatarace(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := []byte(`200 - Nothing to see`)
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}))

	defer ts.Close()

	wg := sync.WaitGroup{}
	n := 30
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			reader := strings.NewReader("")
			ctx, root := BeginSegment(context.Background(), "Test")
			req, _ := http.NewRequest("GET", strings.Replace(ts.URL, "127.0.0.1", "localhost", -1), reader)
			req = req.WithContext(ctx)
			res, err := rt.RoundTrip(req)
			ioutil.ReadAll(res.Body)
			res.Body.Close() // make net/http/transport.go connection reuse
			root.Close(nil)
			assert.NoError(t, err)
		}()
	}
	for i := 0; i < n; i++ {
		_, e := TestDaemon.Recv()
		assert.NoError(t, e)
	}
	wg.Wait()
}

func TestRoundTripReuseTLSDatarace(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := []byte(`200 - Nothing to see`)
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}))
	defer ts.Close()

	certpool := x509.NewCertPool()
	certpool.AddCert(ts.Certificate())
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: certpool,
		},
	}
	rt := &roundtripper{
		Base: tr,
	}

	wg := sync.WaitGroup{}
	n := 30
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			reader := strings.NewReader("")
			ctx, root := BeginSegment(context.Background(), "Test")
			req, _ := http.NewRequest("GET", ts.URL, reader)
			req = req.WithContext(ctx)
			res, err := rt.RoundTrip(req)
			assert.NoError(t, err)
			ioutil.ReadAll(res.Body)
			res.Body.Close() // make net/http/transport.go connection reuse
			root.Close(nil)
		}()
	}
	for i := 0; i < n; i++ {
		_, e := TestDaemon.Recv()
		assert.NoError(t, e)
	}
	wg.Wait()
}

func TestRoundTripHttp2Datarace(t *testing.T) {
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := []byte(`200 - Nothing to see`)
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}))
	err := http2.ConfigureServer(ts.Config, nil)
	assert.NoError(t, err)
	ts.TLS = ts.Config.TLSConfig
	ts.StartTLS()

	defer ts.Close()

	certpool := x509.NewCertPool()
	certpool.AddCert(ts.Certificate())
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: certpool,
		},
	}
	http2.ConfigureTransport(tr)
	rt := &roundtripper{
		Base: tr,
	}

	reader := strings.NewReader("")
	ctx, root := BeginSegment(context.Background(), "Test")
	req, _ := http.NewRequest("GET", ts.URL, reader)
	req = req.WithContext(ctx)
	res, err := rt.RoundTrip(req)
	assert.NoError(t, err)
	ioutil.ReadAll(res.Body)
	res.Body.Close()
	root.Close(nil)

	_, e := TestDaemon.Recv()
	assert.NoError(t, e)
}
