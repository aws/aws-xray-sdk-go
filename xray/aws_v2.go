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

	v2Middleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

type awsV2SubsegmentKey struct{}

func initializeMiddlewareAfter(stack *middleware.Stack) error {
	return stack.Initialize.Add(middleware.InitializeMiddlewareFunc("XRayInitializeMiddlewareAfter", func(
		ctx context.Context, in middleware.InitializeInput, next middleware.InitializeHandler) (
		out middleware.InitializeOutput, metadata middleware.Metadata, err error) {

		serviceName := v2Middleware.GetServiceID(ctx)
		// Start the subsegment
		ctx, subseg := BeginSubsegment(ctx, serviceName)
		if subseg == nil {
			return
		}
		subseg.Namespace = "aws"
		subseg.GetAWS()["region"] = v2Middleware.GetRegion(ctx)
		subseg.GetAWS()["operation"] = v2Middleware.GetOperationName(ctx)

		// set the subsegment in the context
		ctx = context.WithValue(ctx, awsV2SubsegmentKey{}, subseg)

		out, metadata, err = next.HandleInitialize(ctx, in)

		// End the subsegment when the response returns from this middleware
		defer subseg.Close(err)

		return out, metadata, err
	}),
		middleware.After)
}

func deserializeMiddleware(stack *middleware.Stack) error {
	return stack.Deserialize.Add(middleware.DeserializeMiddlewareFunc("XRayDeserializeMiddleware", func(
		ctx context.Context, in middleware.DeserializeInput, next middleware.DeserializeHandler) (
		out middleware.DeserializeOutput, metadata middleware.Metadata, err error) {

		subseg := ctx.Value(awsV2SubsegmentKey{}).(*Segment)
		in.Request.(*smithyhttp.Request).Header.Set(TraceIDHeaderKey, subseg.DownstreamHeader().String())

		out, metadata, err = next.HandleDeserialize(ctx, in)

		resp, ok := out.RawResponse.(*smithyhttp.Response)
		if !ok {
			// No raw response to wrap with.
			return out, metadata, err
		}

		subseg.GetHTTP().GetResponse().Status = resp.StatusCode
		subseg.GetHTTP().GetResponse().ContentLength = int(resp.ContentLength)
		requestID, ok := v2Middleware.GetRequestIDMetadata(metadata)

		if ok {
			subseg.GetAWS()[RequestIDKey] = requestID
		}
		if extendedRequestID := resp.Header.Get(S3ExtendedRequestIDHeaderKey); extendedRequestID != "" {
			subseg.GetAWS()[ExtendedRequestIDKey] = extendedRequestID
		}

		if resp.StatusCode >= 400 && resp.StatusCode <= 499 {
			subseg.Error = true
			if resp.StatusCode == 429 {
				subseg.Throttle = true
			}
		} else if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
			subseg.Fault = true
		}

		return out, metadata, err
	}),
		middleware.Before)
}

func AppendMiddlewares(apiOptions *[]func(*middleware.Stack) error) {
	*apiOptions = append(*apiOptions, initializeMiddlewareAfter, deserializeMiddleware)
}
