// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package xray

import (
	"errors"
	"testing"

	"github.com/creack/aws-xray-sdk-go/strategy/exception"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTraceID(t *testing.T) {
	t.Parallel()
	ctx, seg := BeginSegment(newCtx(), "test")
	traceID := TraceID(ctx)
	assert.Equal(t, seg.TraceID, traceID)
}

func TestEmptyTraceID(t *testing.T) {
	t.Parallel()
	traceID := TraceID(newCtx())
	assert.Empty(t, traceID)
}

func TestRequestWasNotTraced(t *testing.T) {
	t.Parallel()
	ctx, seg := BeginSegment(newCtx(), "test")
	assert.Equal(t, seg.RequestWasTraced, RequestWasTraced(ctx))
}

func TestDetachContext(t *testing.T) {
	t.Parallel()
	ctx := newCtx()
	nctx := DetachContext(ctx)
	assert.NotEqual(t, ctx, nctx)
}

func TestValidAnnotations(t *testing.T) {
	td := newTestDaemon(t)
	defer td.Close()

	ctx, root := BeginSegment(td.Ctx, "Test")
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

	s, e := td.Recv()
	require.NoError(t, e)

	assert.Equal(t, "str", s.Annotations["string"])
	assert.Equal(t, 1.0, s.Annotations["int"]) //json encoder turns this into a float64
	assert.Equal(t, 1.1, s.Annotations["float"])
	assert.Equal(t, true, s.Annotations["bool"])
}

func TestInvalidAnnotations(t *testing.T) {
	td := newTestDaemon(t)
	defer td.Close()

	ctx, root := BeginSegment(td.Ctx, "Test")
	type MyObject struct{}

	err := AddAnnotation(ctx, "Object", &MyObject{})
	root.Close(err)
	assert.Error(t, err)

	_, e := td.Recv()
	require.NoError(t, e)
}

func TestSimpleMetadata(t *testing.T) {
	td := newTestDaemon(t)
	defer td.Close()

	ctx, root := BeginSegment(td.Ctx, "Test")
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
	s, e := td.Recv()
	require.NoError(t, e)

	assert.Equal(t, "str", s.Metadata["default"]["string"])
	assert.Equal(t, 1.0, s.Metadata["default"]["int"])
	assert.Equal(t, 1.1, s.Metadata["default"]["float"])
	assert.Equal(t, true, s.Metadata["default"]["bool"])
}

func TestAddError(t *testing.T) {
	td := newTestDaemon(t)
	defer td.Close()

	ctx, root := BeginSegment(td.Ctx, "Test")
	err := AddError(ctx, errors.New("New Error"))
	assert.Nil(t, err)
	root.Close(err)
	s, e := td.Recv()
	require.NoError(t, e)

	assert.Equal(t, "New Error", s.Cause.Exceptions[0].Message)
	assert.Equal(t, "error", s.Cause.Exceptions[0].Type)
}
