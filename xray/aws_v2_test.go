package xray

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type S3ListBucketsAPI interface {
	ListBuckets(ctx context.Context,
		params *s3.ListBucketsInput,
		optFns ...func(*s3.Options)) (*s3.ListBucketsOutput, error)
}

func GetAllBuckets(c context.Context, api S3ListBucketsAPI, input *s3.ListBucketsInput) (*s3.ListBucketsOutput, error) {
	return api.ListBuckets(c, input)
}

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
			responseStatus: 500,
			responseBody: "Internal Server Error",
			expectedRegion: "us-east-1",
			expectedError: "Error",
			expectedRequestID: "b25f48e8-84fd-11e6-80d9-574e0c4664cb",
			expectedStatusCode: 500,
		},

		"error response": {
			responseStatus: 404,
			responseBody: "Page Not Found",
			expectedRegion: "us-west-1",
			expectedError: "Error",
			expectedRequestID: "1234567890A",
			expectedStatusCode: 404,
		},

		"success response": {
			responseStatus: 200,
			responseBody: "Ok",
			expectedRegion: "us-west-2",
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

func TestAWSV2_S3(t *testing.T) {
	ctx, root := BeginSegment(context.TODO(), "AWSSDKV2_S3")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-west-2"))
	if err != nil {
		panic("configuration error, " + err.Error())
	}

	// Instrumenting AWS SDK v2
	AppendMiddlewares(&cfg.APIOptions)

	client := s3.NewFromConfig(cfg)
	input := &s3.ListBucketsInput{}

	_, err = GetAllBuckets(ctx, client, input)
	if err != nil {
		fmt.Println("Got an error retrieving buckets:")
		fmt.Println(err)
		return
	}

	if e, a := "S3", root.rawSubsegments[0].Name; !strings.EqualFold(e, a) {
		t.Errorf("expected segment name to be %s, got %s", e, a)
	}

	if e, a := "us-west-2", fmt.Sprintf("%v", root.rawSubsegments[0].GetAWS()["region"]); !strings.EqualFold(e, a) {
		t.Errorf("expected subsegment name to be %s, got %s", e, a)
	}

	if e, a := "ListBuckets", fmt.Sprintf("%v", root.rawSubsegments[0].GetAWS()["operation"]); !strings.EqualFold(e, a) {
		t.Errorf("expected operation to be %s, got %s", e, a)
	}

	if e, a := fmt.Sprint(200), fmt.Sprintf("%v", root.rawSubsegments[0].GetHTTP().GetResponse().Status); !strings.EqualFold(e, a) {
		t.Errorf("expected status code to be %s, got %s", e, a)
	}

	if e, a := "aws", root.rawSubsegments[0].Namespace; !strings.EqualFold(e, a) {
		t.Errorf("expected namespace to be %s, got %s", e, a)
	}
}

func TestAWSV2_Dynamo(t *testing.T) {
	ctx, root := BeginSegment(context.TODO(), "AWSSDKV2_Dynamodb")
	defer root.Close(nil)

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-west-2"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	// Instrumenting AWS SDK v2
	AppendMiddlewares(&cfg.APIOptions)

	// Using the Config value, create the DynamoDB client
	svc := dynamodb.NewFromConfig(cfg)

	// Build the request with its input parameters
	_, err = svc.ListTables(ctx, &dynamodb.ListTablesInput{
		Limit: aws.Int32(5),
	})
	if err != nil {
		log.Fatalf("failed to list tables, %v", err)
	}

	if e, a := "DynamoDB", root.rawSubsegments[0].Name; !strings.EqualFold(e, a) {
		t.Errorf("expected segment name to be %s, got %s", e, a)
	}

	if e, a := "us-west-2", fmt.Sprintf("%v", root.rawSubsegments[0].GetAWS()["region"]); !strings.EqualFold(e, a) {
		t.Errorf("expected subsegment name to be %s, got %s", e, a)
	}

	if e, a := "ListTables", fmt.Sprintf("%v", root.rawSubsegments[0].GetAWS()["operation"]); !strings.EqualFold(e, a) {
		t.Errorf("expected operation to be %s, got %s", e, a)
	}

	if e, a := fmt.Sprint(200), fmt.Sprintf("%v", root.rawSubsegments[0].GetHTTP().GetResponse().Status); !strings.EqualFold(e, a) {
		t.Errorf("expected status code to be %s, got %s", e, a)
	}

	if e, a := "aws", root.rawSubsegments[0].Namespace; !strings.EqualFold(e, a) {
		t.Errorf("expected namespace to be %s, got %s", e, a)
	}
}