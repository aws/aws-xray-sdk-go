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
	"errors"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-xray-sdk-go/header"
	"github.com/stretchr/testify/assert"
)

func TestSegmentDataRace(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	n := 100
	wg.Add(n)
	for i := 0; i < n; i++ { // flaky data race test, so we run it multiple times
		_, seg := BeginSegment(ctx, "TestSegment")

		go func() {
			defer wg.Done()
			seg.Close(nil)
		}()
	}
	cancel()
	wg.Wait()
}

func TestSubsegmentDataRace(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()
	ctx, seg := BeginSegment(ctx, "TestSegment")
	defer seg.Close(nil)

	var wg sync.WaitGroup
	n := 5
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			ctx, seg := BeginSubsegment(ctx, "TestSubsegment1")
			_, seg2 := BeginSubsegment(ctx, "TestSubsegment2")
			seg2.Close(nil)
			seg.Close(nil)
		}()
	}
	wg.Wait()
}

func TestSubsegmentDataRaceWithContextCancel(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx, seg := BeginSegment(ctx, "TestSegment")

	wg := sync.WaitGroup{}

	for i := 0; i < 4; i++ {
		if i != 3 {
			wg.Add(1)
		}
		go func(i int) {
			if i != 3 {
				time.Sleep(1)
				defer wg.Done()
			}
			_, seg := BeginSubsegment(ctx, "TestSubsegment1")
			seg.Close(nil)
			if i == 3 {
				cancel() // Context is cancelled abruptly
			}
		}(i)
	}
	wg.Wait()
	seg.Close(nil)
}

func TestSegmentDownstreamHeader(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	ctx, seg := NewSegmentFromHeader(ctx, "TestSegment", &http.Request{URL: &url.URL{}}, &header.Header{
		TraceID:  "fakeid",
		ParentID: "reqid",
	})
	defer seg.Close(nil)

	var wg sync.WaitGroup
	n := 2
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			_, seg2 := BeginSubsegment(ctx, "TestSubsegment")
			seg2.DownstreamHeader() // simulate roundtripper.RoundTrip sets TraceIDHeaderKey
			wg.Done()
		}()
	}
	wg.Wait()
}

func TestParentSegmentTotalCount(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()
	ctx, seg := BeginSegment(ctx, "test")

	var wg sync.WaitGroup
	n := 2
	wg.Add(2 * n)

	for i := 0; i < n; i++ {
		go func(ctx context.Context) { // add async nested subsegments
			c1, _ := BeginSubsegment(ctx, "TestSubsegment1")
			c2, _ := BeginSubsegment(c1, "TestSubsegment2")

			go func(ctx context.Context) { // add async nested subsegments
				c1, _ := BeginSubsegment(ctx, "TestSubsegment1")
				BeginSubsegment(c1, "TestSubsegment2")
				wg.Done()
			}(c2) // passing context

			wg.Done()
		}(ctx)
	}
	wg.Wait()

	assert.Equal(t, 4*uint32(n), seg.ParentSegment.totalSubSegments, "totalSubSegments count should be correctly registered on the parent segment")
}

func TestSegment_isDummy(t *testing.T) {
	ctx, root := BeginSegment(context.Background(), "Segment")
	ctxSubSeg1, subSeg1 := BeginSubsegment(ctx, "Subegment1")
	_, subSeg2 := BeginSubsegment(ctxSubSeg1, "Subsegment2")
	subSeg2.Close(nil)
	subSeg1.Close(nil)
	root.Close(nil)

	assert.False(t, root.Dummy)
	assert.False(t, subSeg1.Dummy)
	assert.False(t, subSeg2.Dummy)
}

func TestSDKDisable_inOrder(t *testing.T) {
	os.Setenv("AWS_XRAY_SDK_DISABLED", "TRue")
	ctx, td := NewTestDaemon()
	defer td.Close()
	ctx, root := BeginSegment(ctx, "Segment")
	ctxSubSeg1, subSeg1 := BeginSubsegment(ctx, "Subegment1")
	_, subSeg2 := BeginSubsegment(ctxSubSeg1, "Subsegment2")
	subSeg2.Close(nil)
	subSeg1.Close(nil)
	root.Close(nil)

	assert.Equal(t, root, &Segment{})
	assert.Equal(t, subSeg1, &Segment{})
	assert.Equal(t, subSeg2, &Segment{})

	os.Setenv("AWS_XRAY_SDK_DISABLED", "FALSE")
}

func TestSDKDisable_outOrder(t *testing.T) {
	os.Setenv("AWS_XRAY_SDK_DISABLED", "TRUE")
	ctx, td := NewTestDaemon()
	defer td.Close()
	_, subSeg := BeginSubsegment(ctx, "Subegment1")
	_, seg := BeginSegment(context.Background(), "Segment")

	subSeg.Close(nil)
	seg.Close(nil)

	assert.Equal(t, subSeg, &Segment{})
	assert.Equal(t, seg, &Segment{})
	os.Setenv("AWS_XRAY_SDK_DISABLED", "FALSE")
}

func TestSDKDisable_otherMethods(t *testing.T) {
	os.Setenv("AWS_XRAY_SDK_DISABLED", "true")
	ctx, td := NewTestDaemon()
	defer td.Close()
	ctx, seg := BeginSegment(ctx, "Segment")
	_, subSeg := BeginSubsegment(ctx, "Subegment1")

	if err := seg.AddAnnotation("key", "value"); err != nil {
		return
	}
	if err := seg.AddMetadata("key", "value"); err != nil {
		return
	}
	seg.DownstreamHeader()

	subSeg.Close(nil)
	seg.Close(nil)

	assert.Equal(t, seg, &Segment{})
	assert.Equal(t, subSeg, &Segment{})
	os.Setenv("AWS_XRAY_SDK_DISABLED", "FALSE")
}

func TestIDGeneration_noOPTrue(t *testing.T) {
	os.Setenv("AWS_XRAY_NOOP_ID", "true")
	seg := &Segment{parent: nil}
	seg.Sampled = false
	idGeneration(seg)

	assert.Equal(t, seg.Sampled, false)
	assert.Equal(t, seg.TraceID, "1-00000000-000000000000000000000000")
	assert.Equal(t, seg.ID, "0000000000000000")
	os.Unsetenv("AWS_XRAY_NOOP_ID")
}

func TestIDGeneration_noOpFalse(t *testing.T) {
	os.Setenv("AWS_XRAY_NOOP_ID", "FALSE")
	seg := &Segment{parent: nil}
	seg.Sampled = false
	idGeneration(seg)

	assert.Equal(t, seg.Sampled, false)
	assert.NotEqual(t, seg.TraceID, "1-00000000-000000000000000000000000")
	assert.NotEqual(t, seg.ID, "0000000000000000")
	os.Unsetenv("AWS_XRAY_NOOP_ID")
}

func TestIDGeneration_samplingFalse(t *testing.T) {
	seg := &Segment{parent: nil}
	seg.Sampled = false
	idGeneration(seg)

	assert.Equal(t, seg.Sampled, false)
	assert.Equal(t, seg.TraceID, "1-00000000-000000000000000000000000")
	assert.Equal(t, seg.ID, "0000000000000000")
}

func TestIDGeneration_samplingTrue(t *testing.T) {
	seg := &Segment{parent: nil}
	seg.Sampled = true
	idGeneration(seg)

	assert.Equal(t, seg.Sampled, true)
	assert.NotEqual(t, seg.TraceID, "1-00000000-000000000000000000000000")
	assert.NotEqual(t, seg.ID, "0000000000000000")
}

func TestIDGeneration_segSubSeg(t *testing.T) {
	os.Setenv("AWS_XRAY_NOOP_ID", "true")
	ctx, td := NewTestDaemon()
	defer td.Close()
	ctx, seg := BeginSegment(ctx, "Segment")
	_, subSeg := BeginSubsegment(ctx, "Subegment1")

	subSeg.Close(nil)
	seg.Close(nil)

	assert.Equal(t, seg.Sampled, true)
	assert.NotEqual(t, seg.TraceID, "1-00000000-000000000000000000000000")
	assert.NotEqual(t, seg.ID, "0000000000000000")
	assert.NotEqual(t, subSeg.ID, "0000000000000000")
	os.Unsetenv("AWS_XRAY_NOOP_ID")
}

// Benchmarks
func BenchmarkBeginSegment(b *testing.B) {
	ctx, td := NewTestDaemon()
	defer td.Close()
	for i := 0; i < b.N; i++ {
		_, seg := BeginSegment(ctx, "TestBenchSeg")
		seg.Close(nil)
	}
}

func BenchmarkBeginSubsegment(b *testing.B) {
	ctx, td := NewTestDaemon()
	defer td.Close()
	ctx, seg := BeginSegment(ctx, "TestBenchSeg")
	for i := 0; i < b.N; i++ {
		_, subSeg := BeginSubsegment(ctx, "TestBenchSubSeg")
		subSeg.Close(nil)
	}
	seg.Sampled = false
	seg.Close(nil)
}

func BenchmarkAddError(b *testing.B) {
	ctx, td := NewTestDaemon()
	defer td.Close()
	_, seg := BeginSegment(ctx, "TestBenchSeg")
	for i := 0; i < b.N; i++ {
		seg.AddError(errors.New("new error"))
	}
	seg.Sampled = false
	seg.Close(nil)
}

func BenchmarkIdGeneration_noOpTrue(b *testing.B) {
	os.Setenv("AWS_XRAY_NOOP_ID", "true")
	seg := &Segment{parent: nil}
	for i := 0; i < b.N; i++ {
		idGeneration(seg)
	}
	os.Unsetenv("AWS_XRAY_NOOP_ID")
}

func BenchmarkIdGeneration_noOpFalse(b *testing.B) {
	os.Setenv("AWS_XRAY_NOOP_ID", "false")
	seg := &Segment{parent: nil}
	for i := 0; i < b.N; i++ {
		idGeneration(seg)
	}
	os.Unsetenv("AWS_XRAY_NOOP_ID")
}

func TestSegment_TraceHeaderID(t *testing.T) {
	type args struct {
		traceHeader *header.Header
	}
	tests := []struct {
		name    string
		init    func(t *testing.T) *Segment
		inspect func(r *Segment, t *testing.T) //inspects receiver after test run

		args func(t *testing.T) args

		want1 string
	}{
		{
			name: "TraceID with sampling decision",
			init: func(*testing.T) *Segment {
				return &Segment{
					TraceID: "x-traceid",
					Sampled: true,
				}
			},
			inspect: func(*Segment, *testing.T) {},
			args: func(*testing.T) args {
				return args{
					traceHeader: &header.Header{
						SamplingDecision: header.Requested,
					},
				}
			},
			want1: "Root=x-traceid;Sampled=1",
		},
		{
			name: "TraceID without Sampled",
			init: func(*testing.T) *Segment {
				return &Segment{
					TraceID: "x-traceid",
					Sampled: true,
				}
			},
			inspect: func(*Segment, *testing.T) {},
			args: func(*testing.T) args {
				return args{
					traceHeader: &header.Header{},
				}
			},
			want1: "Root=x-traceid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tArgs := tt.args(t)

			receiver := tt.init(t)
			got1 := receiver.TraceHeaderID(tArgs.traceHeader)

			if tt.inspect != nil {
				tt.inspect(receiver, t)
			}

			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Segment.TraceHeaderID got1 = %v, want1: %v", got1, tt.want1)
			}
		})
	}
}

func TestSegment_HTTPCapture(t *testing.T) {
	type args struct {
		statusCode    int
		contentLength int
	}
	tests := []struct {
		name    string
		init    func(t *testing.T) *Segment
		inspect func(r *Segment, t *testing.T) //inspects receiver after test run

		args func(t *testing.T) args
	}{
		{
			name: "StatudCode 400 >= 400 and < 500 is a error",
			init: func(*testing.T) *Segment {
				return &Segment{}
			},
			inspect: func(s *Segment, t *testing.T) {
				if !s.Error {
					t.Errorf("Segment error, got = false, want1: true")
				}
			},
			args: func(*testing.T) args {
				return args{
					statusCode:    401,
					contentLength: 100,
				}
			},
		},
		{
			name: "StatudCode 429 set error/throttle",
			init: func(*testing.T) *Segment {
				return &Segment{}
			},
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
					statusCode:    429,
					contentLength: 100,
				}
			},
		},
		{
			name: "StatusCode 500 is a fault error",
			init: func(*testing.T) *Segment {
				return &Segment{}
			},
			inspect: func(s *Segment, t *testing.T) {
				if !s.Fault {
					t.Errorf("Segment.Fault error, got = false, want1: true")
				}

			},
			args: func(*testing.T) args {
				return args{
					statusCode:    500,
					contentLength: 100,
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tArgs := tt.args(t)

			receiver := tt.init(t)
			receiver.HTTPCapture(tArgs.statusCode, tArgs.contentLength)

			if tt.inspect != nil {
				tt.inspect(receiver, t)
			}

			if receiver.GetHTTP().GetResponse().Status != tArgs.statusCode {
				t.Errorf("Status code error, got = %d, want1: %d", receiver.GetHTTP().GetResponse().Status, tArgs.statusCode)
			}

			if receiver.GetHTTP().GetResponse().ContentLength != tArgs.contentLength {
				t.Errorf("ContentLength error, got = %d, want1: %d", receiver.GetHTTP().GetResponse().ContentLength, tArgs.contentLength)
			}

		})
	}
}
