// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

// +build go1.13

package xray

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootHandler(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`200 - OK`)); err != nil {
			panic(err)
		}
	})

	ts := httptest.NewUnstartedServer(Handler(NewFixedSegmentNamer("test"), handler))
	ts.Config.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}
	ts.Start()
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
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, http.StatusOK, seg.HTTP.Response.Status)
	assert.Equal(t, http.MethodPost, seg.HTTP.Request.Method)
	assert.Equal(t, ts.URL+"/", seg.HTTP.Request.URL)
	assert.Equal(t, "127.0.0.1", seg.HTTP.Request.ClientIP)
	assert.Equal(t, "UnitTest", seg.HTTP.Request.UserAgent)
}

func TestNonRootHandler(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ts := httptest.NewUnstartedServer(Handler(NewFixedSegmentNamer("test"), handler))
	ts.Config.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}
	ts.Start()
	defer ts.Close()

	req, err := http.NewRequest(http.MethodPost, ts.URL, strings.NewReader(""))
	if !assert.NoError(t, err) {
		return
	}
	req.Header.Set("User-Agent", "UnitTest")
	req.Header.Set(TraceIDHeaderKey, "Root=fakeid; Parent=reqid; Sampled=1")

	resp, err := http.DefaultClient.Do(req)
	if !assert.NoError(t, err) {
		return
	}
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "fakeid", seg.TraceID)
	assert.Equal(t, "reqid", seg.ParentID)
	assert.Equal(t, true, seg.Sampled)
}
