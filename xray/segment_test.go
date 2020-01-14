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
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-xray-sdk-go/header"
	"github.com/stretchr/testify/assert"
)

func TestSegmentDataRace(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for i := 0; i < 10; i++ { // flaky data race test, so we run it multiple times
		_, seg := BeginSegment(ctx, "TestSegment")

		go seg.Close(nil)
		cancel()
	}
}

func TestSubsegmentDataRace(t *testing.T) {
	ctx := context.Background()
	ctx, seg := BeginSegment(ctx, "TestSegment")
	defer seg.Close(nil)

	wg := sync.WaitGroup{}
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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx, seg := BeginSegment(ctx, "TestSegment")

	wg := sync.WaitGroup{}

	for i := 0; i < 4; i++ {
		if i != 3 {
			wg.Add(1)
		}
		go func(i int) {
			if i != 3 {
				time.Sleep(time.Nanosecond)
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
	ctx := context.Background()
	ctx, seg := NewSegmentFromHeader(ctx, "TestSegment", &header.Header{
		TraceID:  "fakeid",
		ParentID: "reqid",
	})
	defer seg.Close(nil)

	wg := sync.WaitGroup{}
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
	ctx := context.Background()

	ctx, seg := BeginSegment(ctx, "test")

	wg := sync.WaitGroup{}
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
