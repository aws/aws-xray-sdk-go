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
	"testing"

	"github.com/aws/aws-xray-sdk-go/strategy/exception"
	"github.com/stretchr/testify/assert"
)

func TestTraceID(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	ctx, seg := BeginSegment(ctx, "test")
	defer seg.Close(nil)
	traceID := TraceID(ctx)
	assert.Equal(t, seg.TraceID, traceID)
}

func TestEmptyTraceID(t *testing.T) {
	traceID := TraceID(context.Background())
	assert.Empty(t, traceID)
}

func TestRequestWasNotTraced(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	ctx, seg := BeginSegment(ctx, "test")
	defer seg.Close(nil)
	assert.Equal(t, seg.RequestWasTraced, RequestWasTraced(ctx))
}

func TestDetachContext(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ctx1, seg := BeginSegment(ctx, "test")
	defer seg.Close(nil)
	ctx2 := DetachContext(ctx1)
	cancel()

	assert.Equal(t, seg, GetSegment(ctx2))
	select {
	case <-ctx2.Done():
		assert.Error(t, ctx2.Err())
	default:
		// ctx1 is canceled, but ctx2 is not.
	}
}

func TestValidAnnotations(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	ctx, root := BeginSegment(ctx, "Test")

	var err exception.MultiError
	if e := AddAnnotation(ctx, "string", "str"); e != nil {
		err = append(err, e)
	}
	if e := AddAnnotation(ctx, "int", 1); e != nil {
		err = append(err, e)
	}
	if e := AddAnnotation(ctx, "bool", true); e != nil {
		err = append(err, e)
	}
	if e := AddAnnotation(ctx, "float", 1.1); e != nil {
		err = append(err, e)
	}
	root.Close(err)

	seg, e := td.Recv()
	if !assert.NoError(t, e) {
		return
	}

	assert.Equal(t, "str", seg.Annotations["string"])
	assert.Equal(t, 1.0, seg.Annotations["int"]) //json encoder turns this into a float64
	assert.Equal(t, 1.1, seg.Annotations["float"])
	assert.Equal(t, true, seg.Annotations["bool"])
}

func TestInvalidAnnotations(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	ctx, root := BeginSegment(ctx, "Test")
	type MyObject struct{}

	err := AddAnnotation(ctx, "Object", &MyObject{})
	root.Close(err)
	assert.Error(t, err)

	seg, err := td.Recv()
	if assert.NoError(t, err) {
		assert.NotContains(t, seg.Annotations, "Object")
	}
}

func TestSimpleMetadata(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	ctx, root := BeginSegment(ctx, "Test")
	var err exception.MultiError
	if e := AddMetadata(ctx, "string", "str"); e != nil {
		err = append(err, e)
	}
	if e := AddMetadata(ctx, "int", 1); e != nil {
		err = append(err, e)
	}
	if e := AddMetadata(ctx, "bool", true); e != nil {
		err = append(err, e)
	}
	if e := AddMetadata(ctx, "float", 1.1); e != nil {
		err = append(err, e)
	}
	assert.Nil(t, err)
	root.Close(err)

	seg, e := td.Recv()
	if !assert.NoError(t, e) {
		return
	}
	assert.Equal(t, "str", seg.Metadata["default"]["string"])
	assert.Equal(t, 1.0, seg.Metadata["default"]["int"]) //json encoder turns this into a float64
	assert.Equal(t, 1.1, seg.Metadata["default"]["float"])
	assert.Equal(t, true, seg.Metadata["default"]["bool"])
}

func TestAddError(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	ctx, root := BeginSegment(ctx, "Test")
	err := AddError(ctx, errors.New("New Error"))
	assert.NoError(t, err)
	root.Close(err)

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, "New Error", seg.Cause.Exceptions[0].Message)
	assert.Equal(t, "errors.errorString", seg.Cause.Exceptions[0].Type)
}

// Benchmarks
func BenchmarkGetRecorder(b *testing.B) {
	ctx, td := NewTestDaemon()
	defer td.Close()
	ctx, seg := BeginSegment(ctx, "TestSeg")
	for i := 0; i < b.N; i++ {
		GetRecorder(ctx)
	}
	seg.Close(nil)
}

func BenchmarkGetSegment(b *testing.B) {
	ctx, td := NewTestDaemon()
	defer td.Close()
	ctx, seg := BeginSegment(ctx, "TestSeg")
	for i := 0; i < b.N; i++ {
		GetSegment(ctx)
	}
	seg.Close(nil)
}

func BenchmarkDetachContext(b *testing.B) {
	ctx, td := NewTestDaemon()
	defer td.Close()
	ctx, seg := BeginSegment(ctx, "TestSeg")
	for i := 0; i < b.N; i++ {
		DetachContext(ctx)
	}
	seg.Close(nil)
}

func BenchmarkAddAnnotation(b *testing.B) {
	ctx, td := NewTestDaemon()
	defer td.Close()
	ctx, seg := BeginSegment(ctx, "TestSeg")
	for i := 0; i < b.N; i++ {
		err := AddAnnotation(ctx, "key", "value")
		if err != nil {
			return
		}
	}
	seg.Close(nil)
}

func BenchmarkAddMetadata(b *testing.B) {
	ctx, td := NewTestDaemon()
	defer td.Close()
	ctx, seg := BeginSegment(ctx, "TestSeg")
	for i := 0; i < b.N; i++ {
		err := AddMetadata(ctx, "key", "value")
		if err != nil {
			return
		}
	}
	seg.Close(nil)
}
