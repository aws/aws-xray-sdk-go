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
	"net"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newTestDaemon(t *testing.T) *Testdaemon {
	t.Helper()

	// Start a listener on a random port.
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)
	ln, err := net.ListenUDP("udp", addr)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(newCtx())

	td := &Testdaemon{
		Addr:       ln.LocalAddr().String(),
		connection: ln,
		channel:    make(chan *result),
		Ctx:        ctx,
		cancel:     cancel,
	}
	go td.Run()

	// TODO: We should have a way to have a scopped emitter
	//       that respects segment.Configuration.DaemonAddr.
	require.NoError(t, Configure(Config{DaemonAddr: td.Addr}))

	return td
}

type Testdaemon struct {
	Addr string

	wg         sync.WaitGroup
	connection *net.UDPConn
	channel    chan *result
	Ctx        context.Context
	cancel     func()
}

func (td *Testdaemon) Close() {
	if td.cancel != nil {
		td.cancel()
	}
	_ = td.connection.Close() // Best effort.
	td.wg.Wait()
	close(td.channel)
}

type result struct {
	Segment *Segment
	Error   error
}

func (td *Testdaemon) Run() {
	td.wg.Add(1)
	defer td.wg.Done()

	buffer := make([]byte, 64000)
	for {
		select {
		case <-td.Ctx.Done():
		default:
		}
		n, _, err := td.connection.ReadFromUDP(buffer)
		if err != nil {
			select {
			case <-td.Ctx.Done():
				return
			case td.channel <- &result{nil, err}:
			}
			continue
		}

		buffered := buffer[len(Header):n]

		seg := &Segment{}
		if e1 := json.Unmarshal(buffered, seg); e1 != nil {
			select {
			case <-td.Ctx.Done():
				return
			case td.channel <- &result{nil, e1}:
			}
			continue
		}

		seg.Sampled = true

		select {
		case <-td.Ctx.Done():
			return
		case td.channel <- &result{seg, err}:
		}
	}
}

func (td *Testdaemon) Recv() (*Segment, error) {
	td.wg.Add(1)
	defer td.wg.Done()

	ctx, cancel := context.WithTimeout(td.Ctx, 1000*time.Millisecond)
	defer cancel()

	select {
	case r := <-td.channel:
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
	m := parseHeaders(h)
	s, _ := strconv.ParseBool(m["Sampled"])

	return XRayHeaders{
		RootTraceID: m["Root"],
		ParentID:    m["Parent"],
		Sampled:     s,
	}
}

// emptyCtx implements a custom context.
type emptyCtx int

func (*emptyCtx) Deadline() (deadline time.Time, ok bool) { return }
func (*emptyCtx) Done() <-chan struct{}                   { return nil }
func (*emptyCtx) Err() error                              { return nil }
func (*emptyCtx) Value(key interface{}) interface{}       { return nil }

// newCtx returns a new, isolted context.
// Useful to make sure we don't share context accross tests.
func newCtx() context.Context {
	return new(emptyCtx)
}
