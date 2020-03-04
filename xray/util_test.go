// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package xray

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/aws/aws-xray-sdk-go/header"
)

func NewTestDaemon() (context.Context, *TestDaemon) {
	c := make(chan *result, 200)
	conn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		if conn, err = net.ListenPacket("udp6", "[::1]:0"); err != nil {
			panic(fmt.Sprintf("xray: failed to listen: %v", err))
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	d := &TestDaemon{
		ch:     c,
		conn:   conn,
		ctx:    ctx,
		cancel: cancel,
	}
	emitter, err := NewDefaultEmitter(conn.LocalAddr().(*net.UDPAddr))
	if err != nil {
		panic(fmt.Sprintf("xray: failed to created emitter: %v", err))
	}

	ctx, err = ContextWithConfig(ctx, Config{
		Emitter:                emitter,
		DaemonAddr:             conn.LocalAddr().String(),
		ServiceVersion:         "TestVersion",
		SamplingStrategy:       &TestSamplingStrategy{},
		ContextMissingStrategy: &TestContextMissingStrategy{},
		StreamingStrategy:      &TestStreamingStrategy{},
	})
	if err != nil {
		panic(fmt.Sprintf("xray: failed to configure: %v", err))
	}
	go d.run(c)
	return ctx, d
}

type TestDaemon struct {
	ch        <-chan *result
	conn      net.PacketConn
	ctx       context.Context
	cancel    context.CancelFunc
	closeOnce sync.Once
}

type result struct {
	Segment *Segment
	Error   error
}

func (td *TestDaemon) Close() {
	td.closeOnce.Do(func() {
		td.cancel()
		td.conn.Close()
	})
}

func (td *TestDaemon) run(c chan *result) {
	buffer := make([]byte, 64*1024)
	for {
		n, _, err := td.conn.ReadFrom(buffer)
		if err != nil {
			select {
			case c <- &result{nil, err}:
			case <-td.ctx.Done():
				return
			}
			continue
		}

		idx := bytes.IndexByte(buffer, '\n')
		buffered := buffer[idx+1 : n]

		seg := &Segment{}
		err = json.Unmarshal(buffered, &seg)
		if err != nil {
			select {
			case c <- &result{nil, err}:
			case <-td.ctx.Done():
				return
			}
			continue
		}

		seg.Sampled = true
		select {
		case c <- &result{seg, nil}:
		case <-td.ctx.Done():
			return
		}
	}
}

func (td *TestDaemon) Recv() (*Segment, error) {
	ctx, cancel := context.WithTimeout(td.ctx, 500*time.Millisecond)
	defer cancel()
	select {
	case r := <-td.ch:
		return r.Segment, r.Error
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type XRayHeaders struct {
	RootTraceID string
	ParentID    string
	Sampled     bool
}

func ParseHeadersForTest(h http.Header) XRayHeaders {
	traceHeader := header.FromString(h.Get(TraceIDHeaderKey))

	return XRayHeaders{
		RootTraceID: traceHeader.TraceID,
		ParentID:    traceHeader.ParentID,
		Sampled:     traceHeader.SamplingDecision == header.Sampled,
	}
}
