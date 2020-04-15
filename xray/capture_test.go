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
	"errors"
	"sync"
	"testing"

	"github.com/aws/aws-xray-sdk-go/strategy/exception"
	"github.com/stretchr/testify/assert"
)

func TestSimpleCapture(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	ctx, root := BeginSegment(ctx, "Test")
	err := Capture(ctx, "TestService", func(context.Context) error {
		root.Close(nil)
		return nil
	})
	assert.NoError(t, err)

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, "Test", seg.Name)
	assert.Equal(t, root.TraceID, seg.TraceID)
	assert.Equal(t, root.ID, seg.ID)
	assert.Equal(t, root.StartTime, seg.StartTime)
	assert.Equal(t, root.EndTime, seg.EndTime)
	assert.NotNil(t, seg.Subsegments)
	var subseg *Segment
	if assert.NoError(t, json.Unmarshal(seg.Subsegments[0], &subseg)) {
		assert.Equal(t, "TestService", subseg.Name)
	}
}

func TestErrorCapture(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	ctx, root := BeginSegment(ctx, "Test")
	defaultStrategy, err := exception.NewDefaultFormattingStrategy()
	if !assert.NoError(t, err) {
		return
	}
	captureErr := Capture(ctx, "ErrorService", func(context.Context) error {
		defer root.Close(nil)
		return defaultStrategy.Error("MyError")
	})
	if !assert.Error(t, captureErr) {
		return
	}

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}
	var subseg *Segment
	if !assert.NoError(t, json.Unmarshal(seg.Subsegments[0], &subseg)) {
		return
	}
	assert.Equal(t, captureErr.Error(), subseg.Cause.Exceptions[0].Message)
	assert.Equal(t, true, subseg.Fault)
	assert.Equal(t, "error", subseg.Cause.Exceptions[0].Type)
	assert.Equal(t, "TestErrorCapture.func1", subseg.Cause.Exceptions[0].Stack[0].Label)
	assert.Equal(t, "Capture", subseg.Cause.Exceptions[0].Stack[1].Label)
}

func TestPanicCapture(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	ctx, root := BeginSegment(ctx, "Test")
	var captureErr error
	func() {
		defer func() {
			if p := recover(); p != nil {
				captureErr = errors.New(p.(string))
			}
			root.Close(captureErr)
		}()
		_ = Capture(ctx, "PanicService", func(context.Context) error {
			panic("MyPanic")
		})
	}()

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}
	var subseg *Segment
	if !assert.NoError(t, json.Unmarshal(seg.Subsegments[0], &subseg)) {
		return
	}
	assert.Equal(t, captureErr.Error(), subseg.Cause.Exceptions[0].Message)
	assert.Equal(t, "panic", subseg.Cause.Exceptions[0].Type)
	assert.Equal(t, "TestPanicCapture.func1.2", subseg.Cause.Exceptions[0].Stack[0].Label)
	assert.Equal(t, "Capture", subseg.Cause.Exceptions[0].Stack[1].Label)
	assert.Equal(t, "TestPanicCapture.func1", subseg.Cause.Exceptions[0].Stack[2].Label)
	assert.Equal(t, "TestPanicCapture", subseg.Cause.Exceptions[0].Stack[3].Label)
}

func TestNoSegmentCapture(t *testing.T) {
	var err error
	func() {
		defer func() {
			if p := recover(); p != nil {
				err = errors.New(p.(string))
			}
		}()
		_ = Capture(context.Background(), "PanicService", func(context.Context) error {
			panic("MyPanic")
		})
	}()

	assert.NotNil(t, err)
	assert.Equal(t, "failed to begin subsegment named 'PanicService': segment cannot be found.", err.Error())
}

func TestCaptureAsync(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	ctx, root := BeginSegment(ctx, "Test")
	CaptureAsync(ctx, "TestService", func(context.Context) error {
		defer wg.Done()
		root.Close(nil)
		return nil
	})

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}

	wg.Wait()
	assert.Equal(t, "Test", seg.Name)
	assert.Equal(t, root.TraceID, seg.TraceID)
	assert.Equal(t, root.ID, seg.ID)
	assert.Equal(t, root.StartTime, seg.StartTime)
	assert.Equal(t, root.EndTime, seg.EndTime)
	assert.NotNil(t, seg.Subsegments)
	var subseg *Segment
	if assert.NoError(t, json.Unmarshal(seg.Subsegments[0], &subseg)) {
		assert.Equal(t, "TestService", subseg.Name)
	}
}

// Benchmarks
func BenchmarkCapture(b *testing.B) {
	ctx, seg:= BeginSegment(context.Background(), "TestCaptureSeg")
	for i:=0; i<b.N; i++ {
		Capture(ctx, "TestCaptureSubSeg", func(ctx context.Context) error {
			return nil
		})
	}
	seg.Close(nil)
}
