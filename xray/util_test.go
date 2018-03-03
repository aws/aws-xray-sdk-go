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
	"net"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newTestDaemon(t *testing.T, ctx ...context.Context) *Testdaemon {
	t.Helper()

	if len(ctx) > 1 {
		t.Fatal("newTestDaemon expect at most 1 context")
	}
	if len(ctx) == 0 {
		ctx = append(ctx, newCtx())
	}

	// Start a listener on a random port.
	addr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	require.NoError(t, err)
	ln, err := net.ListenUDP("udp", addr)
	require.NoError(t, err)

	ctx1, cancel := context.WithCancel(ctx[0])

	td := &Testdaemon{
		Addr:       ln.LocalAddr().String(),
		connection: ln,
		channel:    make(chan *result, 200),
		Ctx:        ctx1,
		cancel:     cancel,
	}
	// TODO: We should have a way to have a scopped emitter
	//       that respects segment.Configuration.DaemonAddr.
	require.NoError(t, Configure(Config{DaemonAddr: td.Addr}))

	td.wg.Add(1)
	go td.Run()

	// Wait for the daemon to be up.
	ctx1, cancel1 := context.WithTimeout(td.Ctx, 1*time.Second)
	defer cancel1()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	conn, err := net.Dial("udp", td.Addr)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }() //  Best effort.
	for {
		fmt.Fprintf(conn, "%s\n", Header)
		select {
		case <-ticker.C:
		case <-td.channel:
			return td
		case <-ctx1.Done():
			t.Fatal("Timeout waiting for test daemon to start.")
		}
	}
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
	defer td.wg.Done()

	ch := make(chan []byte, 200)

	ctx := td.Ctx

	go func() {
		defer close(ch)

		for {
			buffer := make([]byte, 64000) // Realloc each time so the unmarshal can take it's time.
			n, err := io.ReadAtLeast(td.connection, buffer, len(Header))
			if err != nil {
				return
			}
			select {
			case ch <- buffer[:n]:
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case buf, ok := <-ch:
			if !ok {
				return
			}
			go func(buf []byte) {
				seg := &Segment{Sampled: true}
				err := json.Unmarshal(buf[len(Header):], seg)
				select {
				case td.channel <- &result{seg, err}:
				case <-ctx.Done():
				}
			}(buf)
		}
	}
}

func (td *Testdaemon) Recv() (*Segment, error) {
	td.wg.Add(1)
	defer td.wg.Done()

	// NOTE: The race detector can make thing slow.
	ctx, cancel := context.WithTimeout(td.Ctx, 10*time.Second)
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
