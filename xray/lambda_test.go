package xray

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-xray-sdk-go/header"
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
	newCtx1, subseg1 := BeginSubsegment(ctx, "test-lambda-1")
	var resp1 = testHelper(t, newCtx1)
	assert.Equal(t, header.Sampled, header.FromString(resp1.Header.Get("x-amzn-trace-id")).SamplingDecision)
	assert.Equal(t, subseg1.TraceID, header.FromString(resp1.Header.Get("x-amzn-trace-id")).TraceID)
	assert.Equal(t, subseg1.ID, header.FromString(resp1.Header.Get("x-amzn-trace-id")).ParentID)
	td.Recv()
	seg1, e := td.Recv()
	assert.NoError(t, e)
	assert.Equal(t, true, seg1.Sampled)
	assert.Equal(t, "test-lambda-1", seg1.Name)

	// Second
	newCtx2, subseg2 := BeginSubsegmentWithoutSampling(ctx, "test-lambda-2")
	var resp2 = testHelper(t, newCtx2)
	assert.Equal(t, header.NotSampled, header.FromString(resp2.Header.Get("x-amzn-trace-id")).SamplingDecision)
	assert.Equal(t, subseg2.TraceID, header.FromString(resp2.Header.Get("x-amzn-trace-id")).TraceID)
	assert.Equal(t, subseg2.ID, header.FromString(resp2.Header.Get("x-amzn-trace-id")).ParentID)
	td.Recv()
	seg2, _ := td.Recv()
	assert.Equal(t, (*Segment)(nil), seg2)

	// Third
	newCtx3, subseg3 := BeginSubsegment(ctx, "test-lambda-3")
	var resp3 = testHelper(t, newCtx3)
	assert.Equal(t, header.Sampled, header.FromString(resp3.Header.Get("x-amzn-trace-id")).SamplingDecision)
	assert.Equal(t, subseg3.TraceID, header.FromString(resp3.Header.Get("x-amzn-trace-id")).TraceID)
	assert.Equal(t, subseg3.ID, header.FromString(resp3.Header.Get("x-amzn-trace-id")).ParentID)
	td.Recv()
	seg3, e3 := td.Recv()
	assert.NoError(t, e3)
	assert.Equal(t, true, seg3.Sampled)
	assert.Equal(t, "test-lambda-3", seg3.Name)

	// Forth
	newCtx4, subseg4 := BeginSubsegmentWithoutSampling(ctx, "test-lambda-4")
	var resp4 = testHelper(t, newCtx4)
	assert.Equal(t, header.NotSampled, header.FromString(resp4.Header.Get("x-amzn-trace-id")).SamplingDecision)
	assert.Equal(t, subseg4.TraceID, header.FromString(resp4.Header.Get("x-amzn-trace-id")).TraceID)
	assert.Equal(t, subseg4.ID, header.FromString(resp4.Header.Get("x-amzn-trace-id")).ParentID)
	td.Recv()
	seg4, _ := td.Recv()
	assert.Equal(t, (*Segment)(nil), seg4)
}

/*
	This helper function creates a request and reads the resonse using the context provided.
	It returns the response from the local server.
	It also closes down the segment created for the "downstream" call.
*/
func testHelper(t *testing.T, ctx context.Context) *http.Response {

	var subseg = GetSegment(ctx)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(`200 - OK`)); err != nil {
			panic(err)
		}
	})

	ts := httptest.NewServer(HandlerWithContext(ctx, NewFixedSegmentNamer("test"), handler))
	defer ts.Close()

	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, ts.URL, strings.NewReader(""))

	var samplingDecision = header.Sampled
	if !subseg.Sampled {
		samplingDecision = header.NotSampled
	}

	req.Header.Add(TraceIDHeaderKey, header.Header{
		TraceID:          subseg.TraceID,
		ParentID:         subseg.ID,
		SamplingDecision: samplingDecision,

		AdditionalData: make(map[string]string),
	}.String())

	resp, _ := http.DefaultClient.Do(req)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	GetSegment(req.Context()).Close(nil)
	resp.Body.Close()

	ts.Close()

	subseg.Close(nil)

	return resp
}
