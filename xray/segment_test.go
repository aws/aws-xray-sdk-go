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
