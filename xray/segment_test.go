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

	"github.com/aws/aws-xray-sdk-go/header"
)

func TestSegmentDataRace(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

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
			ctx, seg2 := BeginSubsegment(ctx, "TestSubsegment2")
			seg2.Close(nil)
			seg.Close(nil)
		}()
	}
	wg.Wait()
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
