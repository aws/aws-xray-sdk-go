// Copyright 2017-2021 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package xray

import (
	"context"
	"github.com/aws/smithy-go/middleware"
)

func initializeMiddlewareBefore(stack *middleware.Stack) error {
	return stack.Initialize.Add(middleware.InitializeMiddlewareFunc("XRayInitializeMiddlewareBefore", func(
		ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler) (
		out middleware.InitializeOutput, metadata middleware.Metadata, err error) {

		//ctx = context.WithValue(ctx, spanTimestampKey{}, time.Now())
		return next.HandleInitialize(ctx, in)
	}),
		middleware.Before)
}

func initializeMiddlewareAfter(stack *middleware.Stack) error {
	return stack.Initialize.Add(middleware.InitializeMiddlewareFunc("XRayInitializeMiddlewareAfter", func(
		ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler) (
		out middleware.InitializeOutput, metadata middleware.Metadata, err error) {

		//serviceID := v2Middleware.GetServiceID(ctx)
		//opts := []trace.SpanOption{
		//	trace.WithTimestamp(ctx.Value(spanTimestampKey{}).(time.Time)),
		//	trace.WithSpanKind(trace.SpanKindClient),
		//	trace.WithAttributes(ServiceAttr(serviceID),
		//		RegionAttr(v2Middleware.GetRegion(ctx)),
		//		OperationAttr(v2Middleware.GetOperationName(ctx))),
		//}


		//ctx, span := m.tracer.Start(ctx, serviceID, opts...)
		//defer span.End() //TODO: what is defer? Why do we start the span and end in the same func?

		out, metadata, err = next.HandleInitialize(ctx, in)
		if err != nil {
			//span.RecordError(err)
			//span.SetStatus(codes.Error, err.Error())
		}

		return out, metadata, err
	}),
		middleware.After)
}

//
//var initializeMiddlewareBefore = middleware.InitializeMiddlewareFunc("XRayInitializeMiddlewareBefore", func(
//	ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler,) (
//	out middleware.InitializeOutput, metadata middleware.Metadata, err error,) {
//
//	// Insert begin subsegment
//
//	return next.HandleInitialize(ctx, in)
//})

func AppendMiddlewares(apiOptions *[]func(*middleware.Stack) error) {
	*apiOptions = append(*apiOptions, initializeMiddlewareBefore, initializeMiddlewareAfter)
}