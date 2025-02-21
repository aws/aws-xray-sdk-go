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
	"reflect"
	"strings"
	"testing"

	"github.com/aws/aws-xray-sdk-go/v2/header"
	"github.com/stretchr/testify/assert"
)

func TestNewFixedSegmentName(t *testing.T) {
	n := NewFixedSegmentNamer("test")
	assert.Equal(t, "test", n.FixedName)
}

func TestNewDynamicSegmentName(t *testing.T) {
	n := NewDynamicSegmentNamer("test", "a/b/c")
	assert.Equal(t, "test", n.FallbackName)
	assert.Equal(t, "a/b/c", n.RecognizedHosts)
}

func TestHandlerWithContextForRootHandler(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`200 - OK`)); err != nil {
			panic(err)
		}
	})

	ts := httptest.NewServer(HandlerWithContext(ctx, NewFixedSegmentNamer("test"), handler))
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPost, ts.URL, strings.NewReader(""))
	if !assert.NoError(t, err) {
		return
	}
	req.Header.Set("User-Agent", "UnitTest")

	resp, err := http.DefaultClient.Do(req)
	if !assert.NoError(t, err) {
		return
	}
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// make sure all connections are closed.
	ts.Close()

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, http.StatusOK, seg.HTTP.Response.Status)
	assert.Equal(t, "POST", seg.HTTP.Request.Method)
	assert.Equal(t, ts.URL+"/", seg.HTTP.Request.URL)
	assert.Equal(t, "127.0.0.1", seg.HTTP.Request.ClientIP)
	assert.Equal(t, "UnitTest", seg.HTTP.Request.UserAgent)
	assert.Equal(t, "TestVersion", seg.Service.Version)
}

func TestHandlerWithContextForNonRootHandler(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ts := httptest.NewServer(HandlerWithContext(ctx, NewFixedSegmentNamer("test"), handler))
	defer ts.Close()

	req, err := http.NewRequest(http.MethodDelete, ts.URL, strings.NewReader(""))
	if !assert.NoError(t, err) {
		return
	}
	req.Header.Set(TraceIDHeaderKey, "Root=fakeid; Parent=reqid; Sampled=1")

	resp, err := http.DefaultClient.Do(req)
	if !assert.NoError(t, err) {
		return
	}
	resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// make sure all connections are closed.
	ts.Close()

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "fakeid", seg.TraceID)
	assert.Equal(t, "reqid", seg.ParentID)
	assert.Equal(t, true, seg.Sampled)
	assert.Equal(t, "TestVersion", seg.Service.Version)
}

func TestXRayHandlerPreservesOptionalInterfaces(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

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

	ts := httptest.NewServer(HandlerWithContext(ctx, NewFixedSegmentNamer("test"), handler))
	defer ts.Close()

	req := httptest.NewRequest(http.MethodGet, ts.URL, strings.NewReader(""))

	_, err := http.DefaultTransport.RoundTrip(req)
	assert.NoError(t, err)
}

// Benchmarks
func BenchmarkHandler(b *testing.B) {
	ctx, td := NewTestDaemon()
	defer td.Close()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	})

	for i := 0; i < b.N; i++ {
		ts := httptest.NewServer(HandlerWithContext(ctx, NewFixedSegmentNamer("test"), handler))
		req := httptest.NewRequest(http.MethodGet, ts.URL, strings.NewReader(""))
		http.DefaultTransport.RoundTrip(req)
		ts.Close()
	}
}

func TestGenerateTraceIDHeaderValue(t *testing.T) {
	type args struct {
		seg         *Segment
		traceHeader *header.Header
	}
	tests := []struct {
		name  string
		args  func(t *testing.T) args
		want1 string
	}{
		{
			name: "TraceID with sampling decision",
			args: func(*testing.T) args {
				return args{
					seg: &Segment{
						TraceID: "x-traceid",
						Sampled: true,
					},
					traceHeader: &header.Header{
						SamplingDecision: header.Requested,
					},
				}
			},
			want1: "Root=x-traceid;Sampled=1",
		},
		{
			name: "TraceID without Sampled",
			args: func(*testing.T) args {
				return args{
					seg: &Segment{
						TraceID: "x-traceid",
						Sampled: true,
					},
					traceHeader: &header.Header{},
				}
			},
			want1: "Root=x-traceid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tArgs := tt.args(t)
			got1 := generateTraceIDHeaderValue(tArgs.seg, tArgs.traceHeader)
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Segment.TraceHeaderID got1 = %v, want1: %v", got1, tt.want1)
			}
		})
	}
}

func TestHTTPCaptureResponse(t *testing.T) {
	type args struct {
		seg        *Segment
		statusCode int
	}
	tests := []struct {
		name    string
		inspect func(r *Segment, t *testing.T) //inspects receiver after test run

		args func(t *testing.T) args
	}{
		{
			name: "StatudCode 400 >= 400 and < 500 is a error",
			inspect: func(s *Segment, t *testing.T) {
				if !s.Error {
					t.Errorf("Segment error, got = false, want1: true")
				}
			},
			args: func(*testing.T) args {
				return args{
					seg:        &Segment{},
					statusCode: 401,
				}
			},
		},
		{
			name: "StatudCode 429 set error/throttle",
			inspect: func(s *Segment, t *testing.T) {
				if !s.Error {
					t.Errorf("Segment error, got = false, want1: true")
				}

				if !s.Throttle {
					t.Errorf("Segment.Throttle error, got = false, want1: true")
				}

			},
			args: func(*testing.T) args {
				return args{
					seg:        &Segment{},
					statusCode: 429,
				}
			},
		},
		{
			name: "StatusCode 500 is a fault error",
			inspect: func(s *Segment, t *testing.T) {
				if !s.Fault {
					t.Errorf("Segment.Fault error, got = false, want1: true")
				}

			},
			args: func(*testing.T) args {
				return args{
					seg:        &Segment{},
					statusCode: 500,
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tArgs := tt.args(t)

			HttpCaptureResponse(tArgs.seg, tArgs.statusCode)

			if tt.inspect != nil {
				tt.inspect(tArgs.seg, t)
			}

			if tArgs.seg.GetHTTP().GetResponse().Status != tArgs.statusCode {
				t.Errorf("Status code error, got = %d, want1: %d", tArgs.seg.GetHTTP().GetResponse().Status, tArgs.statusCode)
			}

		})
	}
}
