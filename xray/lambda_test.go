package xray

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLambdaSegmentEmit(t *testing.T) {
	ctx, td := NewTestDaemon()
	defer td.Close()

	// go-lint warns "should not use basic type string as key in context.WithValue",
	// but it must be string type because the trace header comes from aws/aws-lambda-go.
	// https://github.com/aws/aws-lambda-go/blob/b5b7267d297de263cc5b61f8c37543daa9c95ffd/lambda/function.go#L65
	ctx = context.WithValue(ctx, LambdaTraceHeaderKey, "Root=fakeid; Parent=reqid; Sampled=1")
	_, subseg := BeginSubsegment(ctx, "test-lambda")
	subseg.Close(nil)

	seg, e := td.Recv()
	assert.NoError(t, e)
	assert.Equal(t, "fakeid", seg.TraceID)
	assert.Equal(t, "reqid", seg.ParentID)
	assert.Equal(t, true, seg.Sampled)
	assert.Equal(t, "subsegment", seg.Type)
}
