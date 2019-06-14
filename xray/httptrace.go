// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

// +build go1.8

package xray

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http/httptrace"
	"sync"
)

// HTTPSubsegments is a set of context in different HTTP operation.
// Note: from ClientTrace godoc
//               Functions may be called concurrently from different goroutines
//
// HTTPSubsegments must operate as though all functions on it can be called in
// different goroutines and must protect against races
type HTTPSubsegments struct {
	opCtx       context.Context
	connCtx     context.Context
	dnsCtx      context.Context
	connectCtx  context.Context
	tlsCtx      context.Context
	reqCtx      context.Context
	responseCtx context.Context
	mu          sync.Mutex
}

// NewHTTPSubsegments creates a new HTTPSubsegments to use in
// httptrace.ClientTrace functions
func NewHTTPSubsegments(opCtx context.Context) *HTTPSubsegments {
	return &HTTPSubsegments{opCtx: opCtx}
}

// GetConn begins a connect subsegment if the HTTP operation
// subsegment is still in progress.
func (xt *HTTPSubsegments) GetConn(hostPort string) {
	xt.mu.Lock()
	defer xt.mu.Unlock()
	if GetSegment(xt.opCtx).safeInProgress() {
		xt.connCtx, _ = BeginSubsegment(xt.opCtx, "connect")
	}
}

// DNSStart begins a dns subsegment if the HTTP operation
// subsegment is still in progress.
func (xt *HTTPSubsegments) DNSStart(info httptrace.DNSStartInfo) {
	xt.mu.Lock()
	defer xt.mu.Unlock()
	if GetSegment(xt.opCtx).safeInProgress() && xt.connCtx != nil {
		xt.dnsCtx, _ = BeginSubsegment(xt.connCtx, "dns")
	}
}

// DNSDone closes the dns subsegment if the HTTP operation
// subsegment is still in progress, passing the error value
// (if any). Information about the address values looked up,
// and whether or not the call was coalesced is added as
// metadata to the dns subsegment.
func (xt *HTTPSubsegments) DNSDone(info httptrace.DNSDoneInfo) {
	xt.mu.Lock()
	defer xt.mu.Unlock()
	if xt.dnsCtx != nil && GetSegment(xt.opCtx).safeInProgress() {
		metadata := make(map[string]interface{})
		metadata["addresses"] = info.Addrs
		metadata["coalesced"] = info.Coalesced

		AddMetadataToNamespace(xt.dnsCtx, "http", "dns", metadata)
		GetSegment(xt.dnsCtx).Close(info.Err)
	}
}

// ConnectStart begins a dial subsegment if the HTTP operation
// subsegment is still in progress.
func (xt *HTTPSubsegments) ConnectStart(network, addr string) {
	xt.mu.Lock()
	defer xt.mu.Unlock()
	if GetSegment(xt.opCtx).safeInProgress() && xt.connCtx != nil {
		xt.connectCtx, _ = BeginSubsegment(xt.connCtx, "dial")
	}
}

// ConnectDone closes the dial subsegment if the HTTP operation
// subsegment is still in progress, passing the error value
// (if any). Information about the network over which the dial
// was made is added as metadata to the subsegment.
func (xt *HTTPSubsegments) ConnectDone(network, addr string, err error) {
	xt.mu.Lock()
	defer xt.mu.Unlock()
	if xt.connectCtx != nil && GetSegment(xt.opCtx).safeInProgress() {
		metadata := make(map[string]interface{})
		metadata["network"] = network

		AddMetadataToNamespace(xt.connectCtx, "http", "connect", metadata)
		GetSegment(xt.connectCtx).Close(err)
	}
}

// TLSHandshakeStart begins a tls subsegment if the HTTP operation
// subsegment is still in progress.
func (xt *HTTPSubsegments) TLSHandshakeStart() {
	xt.mu.Lock()
	defer xt.mu.Unlock()
	if GetSegment(xt.opCtx).safeInProgress() && xt.connCtx != nil {
		xt.tlsCtx, _ = BeginSubsegment(xt.connCtx, "tls")
	}
}

// TLSHandshakeDone closes the tls subsegment if the HTTP
// operation subsegment is still in progress, passing the
// error value(if any). Information about the tls connection
// is added as metadata to the subsegment.
func (xt *HTTPSubsegments) TLSHandshakeDone(connState tls.ConnectionState, err error) {
	xt.mu.Lock()
	defer xt.mu.Unlock()
	if xt.tlsCtx != nil && GetSegment(xt.opCtx).safeInProgress() {
		metadata := make(map[string]interface{})
		metadata["did_resume"] = connState.DidResume
		metadata["negotiated_protocol"] = connState.NegotiatedProtocol
		metadata["negotiated_protocol_is_mutual"] = connState.NegotiatedProtocolIsMutual
		metadata["cipher_suite"] = connState.CipherSuite

		AddMetadataToNamespace(xt.tlsCtx, "http", "tls", metadata)
		GetSegment(xt.tlsCtx).Close(err)
	}
}

// GotConn closes the connect subsegment if the HTTP operation
// subsegment is still in progress, passing the error value
// (if any). Information about the connection is added as
// metadata to the subsegment. If the connection is marked as reused,
// the connect subsegment is deleted.
func (xt *HTTPSubsegments) GotConn(info *httptrace.GotConnInfo, err error) {
	xt.mu.Lock()
	defer xt.mu.Unlock()
	if xt.connCtx != nil && GetSegment(xt.opCtx).safeInProgress() { // GetConn may not have been called (client_test.TestBadRoundTrip)
		if info != nil {
			if info.Reused {
				GetSegment(xt.opCtx).RemoveSubsegment(GetSegment(xt.connCtx))
				// Remove the connCtx context since it is no longer needed.
				xt.connCtx = nil
			} else {
				metadata := make(map[string]interface{})
				metadata["reused"] = info.Reused
				metadata["was_idle"] = info.WasIdle
				if info.WasIdle {
					metadata["idle_time"] = info.IdleTime
				}

				AddMetadataToNamespace(xt.connCtx, "http", "connection", metadata)
				GetSegment(xt.connCtx).Close(err)
			}
		} else if xt.connCtx != nil && GetSegment(xt.connCtx).safeInProgress() {
			GetSegment(xt.connCtx).Close(err)
		}

		if err == nil {
			xt.reqCtx, _ = BeginSubsegment(xt.opCtx, "request")
		}

	}
}

// WroteRequest closes the request subsegment if the HTTP operation
// subsegment is still in progress, passing the error value
// (if any). The response subsegment is then begun.
func (xt *HTTPSubsegments) WroteRequest(info httptrace.WroteRequestInfo) {
	xt.mu.Lock()
	defer xt.mu.Unlock()
	if xt.reqCtx != nil && GetSegment(xt.opCtx).safeInProgress() {
		GetSegment(xt.reqCtx).Close(info.Err)
		resCtx, _ := BeginSubsegment(xt.opCtx, "response")
		xt.responseCtx = resCtx
	}

	// In case the GotConn http trace handler wasn't called,
	// we close the connection subsegment since a connection
	// had to have been acquired before attempting to write
	// the request.
	if xt.connCtx != nil && GetSegment(xt.connCtx).safeInProgress() {
		GetSegment(xt.connCtx).Close(nil)
	}
}

// GotFirstResponseByte closes the response subsegment if the HTTP
// operation subsegment is still in progress.
func (xt *HTTPSubsegments) GotFirstResponseByte() {
	xt.mu.Lock()
	defer xt.mu.Unlock()
	resCtx := xt.responseCtx
	if resCtx != nil && GetSegment(xt.opCtx).safeInProgress() {
		GetSegment(resCtx).Close(nil)
	}
}

// ClientTrace is a set of pointers of HTTPSubsegments and ClientTrace.
type ClientTrace struct {
	subsegments *HTTPSubsegments
	httpTrace   *httptrace.ClientTrace
}

// NewClientTrace returns an instance of xray.ClientTrace, a wrapper
// around httptrace.ClientTrace. The ClientTrace implementation will
// generate subsegments for connection time, DNS lookup time, TLS
// handshake time, and provides additional information about the HTTP round trip
func NewClientTrace(opCtx context.Context) (ct *ClientTrace, err error) {
	if opCtx == nil {
		return nil, errors.New("opCtx must be non-nil")
	}

	segs := NewHTTPSubsegments(opCtx)

	return &ClientTrace{
		subsegments: segs,
		httpTrace: &httptrace.ClientTrace{
			GetConn: func(hostPort string) {
				segs.GetConn(hostPort)
			},
			DNSStart: func(info httptrace.DNSStartInfo) {
				segs.DNSStart(info)
			},
			DNSDone: func(info httptrace.DNSDoneInfo) {
				segs.DNSDone(info)
			},
			ConnectStart: func(network, addr string) {
				segs.ConnectStart(network, addr)
			},
			ConnectDone: func(network, addr string, err error) {
				segs.ConnectDone(network, addr, err)
			},
			TLSHandshakeStart: func() {
				segs.TLSHandshakeStart()
			},
			TLSHandshakeDone: func(connState tls.ConnectionState, err error) {
				segs.TLSHandshakeDone(connState, err)
			},
			GotConn: func(info httptrace.GotConnInfo) {
				segs.GotConn(&info, nil)
			},
			WroteRequest: func(info httptrace.WroteRequestInfo) {
				segs.WroteRequest(info)
			},
			GotFirstResponseByte: func() {
				segs.GotFirstResponseByte()
			},
		},
	}, nil
}
