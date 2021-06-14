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
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
)

func TestAWSV2(t *testing.T) {
	cases := map[string]struct {
		responseStatus     int
		responseBody       string
		expectedRegion     string
		expectedError      string
		expectedRequestID  string
		expectedStatusCode int
	}{
		"fault response": {
			responseStatus:     500,
			responseBody:       "Internal Server Error",
			expectedRegion:     "us-east-1",
			expectedError:      "Error",
			expectedRequestID:  "b25f48e8-84fd-11e6-80d9-574e0c4664cb",
			expectedStatusCode: 500,
		},

		"error response": {
			responseStatus:     404,
			responseBody:       "Page Not Found",
			expectedRegion:     "us-west-1",
			expectedError:      "Error",
			expectedRequestID:  "1234567890A",
			expectedStatusCode: 404,
		},

		"success response": {
			responseStatus:     200,
			responseBody:       "Ok",
			expectedRegion:     "us-west-2",
			expectedStatusCode: 200,
		},
	}

	for name, c := range cases {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(c.responseStatus)
				_, err := w.Write([]byte(c.responseBody))
				if err != nil {
					t.Fatal(err)
				}
			}))

		defer server.Close()

		t.Run(name, func(t *testing.T) {
			ctx, root := BeginSegment(context.TODO(), "AWSSDKV2_Route53")

			svc := route53.NewFromConfig(aws.Config{
				Region: c.expectedRegion,
				EndpointResolver: aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
					return aws.Endpoint{
						URL:         server.URL,
						SigningName: "route53",
					}, nil
				}),
				Retryer: func() aws.Retryer {
					return aws.NopRetryer{}
				},
			})

			_, _ = svc.ChangeResourceRecordSets(ctx, &route53.ChangeResourceRecordSetsInput{
				ChangeBatch: &types.ChangeBatch{
					Changes: []types.Change{},
					Comment: aws.String("mock"),
				},
				HostedZoneId: aws.String("zone"),
			}, func(options *route53.Options) {
				AppendMiddlewares(&options.APIOptions)
			})

			if e, a := "Route 53", root.rawSubsegments[0].Name; !strings.EqualFold(e, a) {
				t.Errorf("expected segment name to be %s, got %s", e, a)
			}

			if e, a := c.expectedRegion, fmt.Sprintf("%v", root.rawSubsegments[0].GetAWS()["region"]); !strings.EqualFold(e, a) {
				t.Errorf("expected subsegment name to be %s, got %s", e, a)
			}

			if e, a := "ChangeResourceRecordSets", fmt.Sprintf("%v", root.rawSubsegments[0].GetAWS()["operation"]); !strings.EqualFold(e, a) {
				t.Errorf("expected operation to be %s, got %s", e, a)
			}

			if e, a := fmt.Sprint(c.expectedStatusCode), fmt.Sprintf("%v", root.rawSubsegments[0].GetHTTP().GetResponse().Status); !strings.EqualFold(e, a) {
				t.Errorf("expected status code to be %s, got %s", e, a)
			}

			if e, a := "aws", root.rawSubsegments[0].Namespace; !strings.EqualFold(e, a) {
				t.Errorf("expected namespace to be %s, got %s", e, a)
			}

			root.Close(nil)
		})
	}
}
