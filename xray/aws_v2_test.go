package xray

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
	ctx, root := BeginSegment(context.TODO(), "AWSSDKV2_Test")
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic("configuration error, " + err.Error())
	}

	// Instrumenting AWS SDK v2
	AppendMiddlewares(&cfg.APIOptions)

	client := s3.NewFromConfig(cfg)
	input := &s3.ListBucketsInput{}

	result, err := GetAllBuckets(ctx, client, input)
	if err != nil {
		fmt.Println("Got an error retrieving buckets:")
		fmt.Println(err)
		return
	}

	fmt.Println("Buckets:")

	for _, bucket := range result.Buckets {
		fmt.Println(*bucket.Name + ": " + bucket.CreationDate.Format("2006-01-02 15:04:05 Monday"))
	}

	root.Close(nil)

}

func TestAWSV2_ListBucket(t *testing.T) {
	ctx, root := BeginSegment(context.TODO(), "AWSSDKV2_Test_ListBucket")
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		panic("configuration error, " + err.Error())
	}

	// Instrumenting AWS SDK v2
	AppendMiddlewares(&cfg.APIOptions)

	client := s3.NewFromConfig(cfg)
	input := &s3.ListObjectsInput{Bucket: aws.String("srprash_test_bucket")}

	_, err = client.ListObjects(ctx, input)
	if err != nil {
		fmt.Println("Got an error listing objects:")
		fmt.Println(err)
	}


	root.Close(nil)

}

