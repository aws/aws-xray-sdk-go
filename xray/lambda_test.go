package xray

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLambdaSegmentEmit(t *testing.T) {
	ctx := context.WithValue(context.Background(), LambdaTraceHeaderKey, "Root=fakeid; Parent=reqid; Sampled=1")
	_, subseg := BeginSubsegment(ctx, "test-lambda")
	subseg.Close(nil)

	seg, e := TestDaemon.Recv()
	assert.NoError(t, e)
	assert.Equal(t, "fakeid", seg.TraceID)
	assert.Equal(t, "reqid", seg.ParentID)
	assert.Equal(t, true, seg.Sampled)
	assert.Equal(t, "subsegment", seg.Type)
}
