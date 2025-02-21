package xray

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-xray-sdk-go/v2/header"
	"github.com/stretchr/testify/assert"
)

const ExampleTraceHeader string = "Root=1-57ff426a-80c11c39b0c928905eb0828d;Parent=1234abcd1234abcd;Sampled=1"

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

func TestLambdaMix(t *testing.T) {
	// Setup
	ctx, td := NewTestDaemon()
	defer td.Close()
	ctx = context.WithValue(ctx, LambdaTraceHeaderKey, ExampleTraceHeader)

	// First
	ctx1, _ := BeginSubsegment(ctx, "test-lambda-1")
	testHelper(ctx1, t, td, true)

	// Second
	ctx2, _ := BeginSubsegmentWithoutSampling(ctx, "test-lambda-2")
	testHelper(ctx2, t, td, false)

	// Third
	ctx3, _ := BeginSubsegment(ctx, "test-lambda-3")
	testHelper(ctx3, t, td, true)

	// Forth
	ctx4, _ := BeginSubsegmentWithoutSampling(ctx, "test-lambda-4")
	testHelper(ctx4, t, td, false)
}

/*
This helper function creates a request and validates the response using the context provided.
*/
func testHelper(ctx context.Context, t *testing.T, td *TestDaemon, sampled bool) {
	var subseg = GetSegment(ctx)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`200 - OK`)); err != nil {
			panic(err)
		}
	})

	// Create Test Server
	ts := httptest.NewServer(HandlerWithContext(context.Background(), NewFixedSegmentNamer("RequestSegment"), handler))
	defer ts.Close()

	// Perform Request
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL, strings.NewReader(""))
	req.Header.Add(TraceIDHeaderKey, generateHeader(subseg).String())
	resp, _ := http.DefaultClient.Do(req)

	// Close the test server down
	ts.Close()

	// Ensure the return value is valid
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	assert.Equal(t, subseg.TraceID, header.FromString(resp.Header.Get("x-amzn-trace-id")).TraceID)
	assert.Equal(t, subseg.ID, header.FromString(resp.Header.Get("x-amzn-trace-id")).ParentID)

	subseg.Close(nil)
	emittedSeg, e := td.Recv()

	if sampled {
		assert.Equal(t, header.Sampled, header.FromString(resp.Header.Get("x-amzn-trace-id")).SamplingDecision)
		assert.NoError(t, e)
		assert.Equal(t, true, emittedSeg.Sampled)
		assert.Equal(t, subseg.Name, emittedSeg.Name)
	} else {
		assert.Equal(t, header.NotSampled, header.FromString(resp.Header.Get("x-amzn-trace-id")).SamplingDecision)
		assert.Equal(t, (*Segment)(nil), emittedSeg)
	}
}

func generateHeader(seg *Segment) header.Header {
	var samplingDecision = header.Sampled
	if !seg.Sampled {
		samplingDecision = header.NotSampled
	}

	return header.Header{
		TraceID:          seg.TraceID,
		ParentID:         seg.ID,
		SamplingDecision: samplingDecision,

		AdditionalData: make(map[string]string),
	}
}
