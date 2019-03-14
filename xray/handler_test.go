// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package xray

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"context"
	"os"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewFixedSegmentName(t *testing.T) {
	n := NewFixedSegmentNamer("test")
	assert.Equal(t, "test", n.FixedName)
}

func TestNewFixedSegmentNameFromEnv(t *testing.T) {
	os.Setenv("AWS_XRAY_TRACING_NAME", "test_env")
	n := NewFixedSegmentNamer("test")
	assert.Equal(t, "test_env", n.FixedName)
	os.Unsetenv("AWS_XRAY_TRACING_NAME")
}

func TestNewDynamicSegmentName(t *testing.T) {
	n := NewDynamicSegmentNamer("test", "a/b/c")
	assert.Equal(t, "test", n.FallbackName)
	assert.Equal(t, "a/b/c", n.RecognizedHosts)
}

func TestNewDynamicSegmentNameFromEnv(t *testing.T) {
	os.Setenv("AWS_XRAY_TRACING_NAME", "test_env")
	n := NewDynamicSegmentNamer("test", "a/b/c")
	assert.Equal(t, "test_env", n.FallbackName)
	assert.Equal(t, "a/b/c", n.RecognizedHosts)
	os.Unsetenv("AWS_XRAY_TRACING_NAME")
}

func TestHandlerWithContextForRootHandler(t *testing.T) {
	ctx, _ := ContextWithConfig(context.Background(), Config{
		ServiceVersion: "1.0.0",
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		b := []byte(`200 - OK`)
		w.Write(b)
	})

	ts := httptest.NewServer(HandlerWithContext(ctx, NewFixedSegmentNamer("test"), handler))
	defer ts.Close()

	req := httptest.NewRequest("POST", ts.URL, strings.NewReader(""))
	req.Header.Set("User-Agent", "UnitTest")
	req.Header.Set("X-Forwarded-For", "127.0.0.1")

	_, err := http.DefaultTransport.RoundTrip(req)
	assert.NoError(t, err)

	s, e := TestDaemon.Recv()
	assert.NoError(t, e)

	assert.Equal(t, http.StatusOK, s.HTTP.Response.Status)
	assert.Equal(t, "POST", s.HTTP.Request.Method)
	assert.Equal(t, ts.URL+"/", s.HTTP.Request.URL)
	assert.Equal(t, "127.0.0.1", s.HTTP.Request.ClientIP)
	assert.Equal(t, "UnitTest", s.HTTP.Request.UserAgent)
	assert.Equal(t, "1.0.0", s.Service.Version)
}

func TestHandlerWithContextForNonRootHandler(t *testing.T) {
	ctx, _ := ContextWithConfig(context.Background(), Config{
		ServiceVersion: "1.0.0",
	})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ts := httptest.NewServer(HandlerWithContext(ctx, NewFixedSegmentNamer("test"), handler))
	defer ts.Close()

	req := httptest.NewRequest("DELETE", ts.URL, strings.NewReader(""))
	req.Header.Set(TraceIDHeaderKey, "Root=fakeid; Parent=reqid; Sampled=1")

	_, err := http.DefaultTransport.RoundTrip(req)
	assert.NoError(t, err)

	s, e := TestDaemon.Recv()
	assert.NoError(t, e)

	assert.Equal(t, "fakeid", s.TraceID)
	assert.Equal(t, "reqid", s.ParentID)
	assert.Equal(t, true, s.Sampled)
	assert.Equal(t, "1.0.0", s.Service.Version)
}

func TestRootHandler(t *testing.T) {
	// keep a sleep here because Reservoir allows a specified amount of `Take()`s per second.
	time.Sleep(1 * time.Second)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		b := []byte(`200 - OK`)
		w.Write(b)
	})

	ts := httptest.NewServer(Handler(NewFixedSegmentNamer("test"), handler))
	defer ts.Close()

	req := httptest.NewRequest("POST", ts.URL, strings.NewReader(""))
	req.Header.Set("User-Agent", "UnitTest")
	req.Header.Set("X-Forwarded-For", "127.0.0.1")

	_, err := http.DefaultTransport.RoundTrip(req)
	assert.NoError(t, err)

	s, e := TestDaemon.Recv()
	assert.NoError(t, e)

	assert.Equal(t, http.StatusOK, s.HTTP.Response.Status)
	assert.Equal(t, "POST", s.HTTP.Request.Method)
	assert.Equal(t, ts.URL+"/", s.HTTP.Request.URL)
	assert.Equal(t, "127.0.0.1", s.HTTP.Request.ClientIP)
	assert.Equal(t, "UnitTest", s.HTTP.Request.UserAgent)
}

func TestNonRootHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ts := httptest.NewServer(Handler(NewFixedSegmentNamer("test"), handler))
	defer ts.Close()

	req := httptest.NewRequest("DELETE", ts.URL, strings.NewReader(""))
	req.Header.Set(TraceIDHeaderKey, "Root=fakeid; Parent=reqid; Sampled=1")

	_, err := http.DefaultTransport.RoundTrip(req)
	assert.NoError(t, err)

	s, e := TestDaemon.Recv()
	assert.NoError(t, e)

	assert.Equal(t, "fakeid", s.TraceID)
	assert.Equal(t, "reqid", s.ParentID)
	assert.Equal(t, true, s.Sampled)
}

func TestXRayHandlerPreservesOptionalInterfaces(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, isCloseNotifier := w.(http.CloseNotifier)
		_, isFlusher := w.(http.Flusher)
		_, isHijacker := w.(http.Hijacker)
		_, isPusher := w.(http.Pusher)
		_, isReaderFrom := w.(io.ReaderFrom)

		assert.True(t, isCloseNotifier)
		assert.True(t, isFlusher)
		assert.True(t, isHijacker)
		assert.True(t, isReaderFrom)
		// Pusher is only available when using http/2, so should not be present
		assert.False(t, isPusher)

		w.WriteHeader(202)
	})

	ts := httptest.NewServer(Handler(NewFixedSegmentNamer("test"), handler))
	defer ts.Close()

	req := httptest.NewRequest("GET", ts.URL, strings.NewReader(""))

	_, err := http.DefaultTransport.RoundTrip(req)
	assert.NoError(t, err)
}
