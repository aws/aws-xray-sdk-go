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
	"net/http/httptrace"
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	log "github.com/cihub/seelog"
)

func beginSubsegmentV2(r *aws.Request, name string) {
	ctx, _ := BeginSubsegment(r.HTTPRequest.Context(), name)
	r.HTTPRequest = r.HTTPRequest.WithContext(ctx)
}

func endSubsegmentV2(r *aws.Request) {
	seg := GetSegment(r.HTTPRequest.Context())
	if seg == nil {
		return
	}
	seg.Close(r.Error)
	r.HTTPRequest = r.HTTPRequest.WithContext(context.WithValue(r.HTTPRequest.Context(), ContextKey, seg.parent))
}

var xRayBeforeValidateHandlerV2 = aws.NamedHandler{
	Name: "XRayBeforeValidateHandlerV2",
	Fn: func(r *aws.Request) {
		ctx, opseg := BeginSubsegment(r.HTTPRequest.Context(), r.Metadata.ServiceName)
		if opseg == nil {
			return
		}
		opseg.Namespace = "aws"
		marshalctx, _ := BeginSubsegment(ctx, "marshal")

		r.HTTPRequest = r.HTTPRequest.WithContext(marshalctx)
		r.HTTPRequest.Header.Set("x-amzn-trace-id", opseg.DownstreamHeader().String())
	},
}

var xRayAfterBuildHandlerV2 = aws.NamedHandler{
	Name: "XRayAfterBuildHandlerV2",
	Fn: func(r *aws.Request) {
		endSubsegmentV2(r)
	},
}

var xRayBeforeSignHandlerV2 = aws.NamedHandler{
	Name: "XRayBeforeSignHandlerV2",
	Fn: func(r *aws.Request) {
		ctx, seg := BeginSubsegment(r.HTTPRequest.Context(), "attempt")
		if seg == nil {
			return
		}
		ct, _ := NewClientTrace(ctx)
		r.HTTPRequest = r.HTTPRequest.WithContext(httptrace.WithClientTrace(ctx, ct.httpTrace))
	},
}

var xRayAfterSignHandlerV2 = aws.NamedHandler{
	Name: "XRayAfterSignHandlerV2",
	Fn: func(r *aws.Request) {
		endSubsegmentV2(r)
	},
}

var xRayBeforeSendHandlerV2 = aws.NamedHandler{
	Name: "XRayBeforeSendHandlerV2",
	Fn: func(r *aws.Request) {
	},
}

var xRayAfterSendHandlerV2 = aws.NamedHandler{
	Name: "XRayAfterSendHandlerV2",
	Fn: func(r *aws.Request) {
		endSubsegmentV2(r)
	},
}

var xRayBeforeUnmarshalHandlerV2 = aws.NamedHandler{
	Name: "XRayBeforeUnmarshalHandlerV2",
	Fn: func(r *aws.Request) {
		endSubsegmentV2(r) // end attempt subsegment
		beginSubsegmentV2(r, "unmarshal")
	},
}

var xRayAfterUnmarshalHandlerV2 = aws.NamedHandler{
	Name: "XRayAfterUnmarshalHandlerV2",
	Fn: func(r *aws.Request) {
		endSubsegmentV2(r)
	},
}

var xRayBeforeRetryHandlerV2 = aws.NamedHandler{
	Name: "XRayBeforeRetryHandlerV2",
	Fn: func(r *aws.Request) {
		endSubsegmentV2(r) // end attempt subsegment
		ctx, _ := BeginSubsegment(r.HTTPRequest.Context(), "wait")

		r.HTTPRequest = r.HTTPRequest.WithContext(ctx)
	},
}

var xRayAfterRetryHandlerV2 = aws.NamedHandler{
	Name: "XRayAfterRetryHandlerV2",
	Fn: func(r *aws.Request) {
		endSubsegmentV2(r)
	},
}

func pushHandlersV2(c *aws.Client) {
	c.Handlers.Validate.PushFrontNamed(xRayBeforeValidateHandlerV2)
	c.Handlers.Build.PushBackNamed(xRayAfterBuildHandlerV2)
	c.Handlers.Sign.PushFrontNamed(xRayBeforeSignHandlerV2)
	c.Handlers.Unmarshal.PushFrontNamed(xRayBeforeUnmarshalHandlerV2)
	c.Handlers.Unmarshal.PushBackNamed(xRayAfterUnmarshalHandlerV2)
	c.Handlers.Retry.PushFrontNamed(xRayBeforeRetryHandlerV2)
	c.Handlers.AfterRetry.PushBackNamed(xRayAfterRetryHandlerV2)
}

// AWSv2 adds X-Ray tracing to an AWS V2 client.
func AWSv2(c *aws.Client) {
	if c == nil {
		panic("Please initialize the provided AWS client before passing to the AWS() method.")
	}
	pushHandlersV2(c)
	c.Handlers.Complete.PushFrontNamed(xrayCompleteHandlerV2(""))
}

// AWSWithWhitelistV2 allows a custom parameter whitelist JSON file to be defined.
func AWSWithWhitelistV2(c *aws.Client, filename string) {
	if c == nil {
		panic("Please initialize the provided AWS client before passing to the AWSWithWhitelist() method.")
	}
	pushHandlersV2(c)
	c.Handlers.Complete.PushFrontNamed(xrayCompleteHandlerV2(filename))
}

func xrayCompleteHandlerV2(filename string) aws.NamedHandler {
	whitelistJSON := parseWhitelistJSON(filename)
	whitelist := &jsonMap{}
	err := json.Unmarshal(whitelistJSON, &whitelist.object)
	if err != nil {
		panic(err)
	}

	return aws.NamedHandler{
		Name: "XRayCompleteHandler",
		Fn: func(r *aws.Request) {
			curseg := GetSegment(r.HTTPRequest.Context())

			for curseg != nil && curseg.Namespace != "aws" {
				curseg.Close(nil)
				curseg = curseg.parent
			}
			if curseg == nil {
				return
			}

			opseg := curseg

			opseg.Lock()
			for k, v := range extractRequestParametersV2(r, whitelist) {
				opseg.GetAWS()[strings.ToLower(addUnderScoreBetweenWords(k))] = v
			}
			for k, v := range extractResponseParametersV2(r, whitelist) {
				opseg.GetAWS()[strings.ToLower(addUnderScoreBetweenWords(k))] = v
			}

			opseg.GetAWS()["region"] = r.Metadata.SigningRegion
			opseg.GetAWS()["operation"] = r.Operation.Name
			opseg.GetAWS()["retries"] = r.RetryCount
			opseg.GetAWS()[RequestIDKey] = r.RequestID

			if r.HTTPResponse != nil {
				opseg.GetHTTP().GetResponse().Status = r.HTTPResponse.StatusCode
				opseg.GetHTTP().GetResponse().ContentLength = int(r.HTTPResponse.ContentLength)

				if extendedRequestID := r.HTTPResponse.Header.Get(S3ExtendedRequestIDHeaderKey); extendedRequestID != "" {
					opseg.GetAWS()[ExtendedRequestIDKey] = extendedRequestID
				}
			}

			if request.IsErrorThrottle(r.Error) {
				opseg.Throttle = true
			}

			opseg.Unlock()
			opseg.Close(r.Error)
		},
	}
}

func extractRequestParametersV2(r *aws.Request, whitelist *jsonMap) map[string]interface{} {
	valueMap := make(map[string]interface{})

	extractParametersV2("request_parameters", requestKeyword, r, whitelist, valueMap)
	extractDescriptorsV2("request_descriptors", requestKeyword, r, whitelist, valueMap)

	return valueMap
}

func extractResponseParametersV2(r *aws.Request, whitelist *jsonMap) map[string]interface{} {
	valueMap := make(map[string]interface{})

	extractParametersV2("response_parameters", responseKeyword, r, whitelist, valueMap)
	extractDescriptorsV2("response_descriptors", responseKeyword, r, whitelist, valueMap)

	return valueMap
}

func extractParametersV2(whitelistKey string, rType int, r *aws.Request, whitelist *jsonMap, valueMap map[string]interface{}) {
	params := whitelist.search("services", r.Metadata.ServiceName, "operations", r.Operation.Name, whitelistKey)
	if params != nil {
		children, err := params.children()
		if err != nil {
			log.Errorf("failed to get values for aws attribute: %v", err)
			return
		}
		for _, child := range children {
			if child != nil {
				var value interface{}
				if rType == requestKeyword {
					value = keyValue(r.Params, child.(string))
				} else if rType == responseKeyword {
					value = keyValue(r.Data, child.(string))
				}
				if (value != reflect.Value{}) {
					valueMap[child.(string)] = value
				}
			}
		}
	}
}

func extractDescriptorsV2(whitelistKey string, rType int, r *aws.Request, whitelist *jsonMap, valueMap map[string]interface{}) {
	responseDtr := whitelist.search("services", r.Metadata.ServiceName, "operations", r.Operation.Name, whitelistKey)
	if responseDtr != nil {
		items, err := responseDtr.childrenMap()
		if err != nil {
			log.Errorf("failed to get values for aws attribute: %v", err)
			return
		}
		for k := range items {
			descriptorMap, _ := whitelist.search("services", r.Metadata.ServiceName, "operations", r.Operation.Name, whitelistKey, k).childrenMap()
			if rType == requestKeyword {
				insertDescriptorValuesIntoMap(k, r.Params, descriptorMap, valueMap)
			} else if rType == responseKeyword {
				insertDescriptorValuesIntoMap(k, r.Data, descriptorMap, valueMap)
			}
		}
	}
}
