package xray

import (
	"context"
	"encoding/json"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAWS(t *testing.T) {
	// Runs a suite of tests against two different methods of registering
	// handlers on an AWS client.

	type test func(*testing.T, *lambda.Client)
	tests := []struct {
		name     string
		test     test
		failConn bool
	}{
		{"failed connection", testClientFailedConnection, true},
		{"successful connection", testClientSuccessfulConnection, false},
		{"without segment", testClientWithoutSegment, false},
	}

	onClient := func(s aws.Config) *lambda.Client {
		svc := lambda.New(s)
		AWS(svc.Client)
		return svc
	}

	onConfig := func(s aws.Config) *lambda.Client {
		return lambda.New(AWSConfig(s))
	}

	const whitelist = "../resources/AWSWhitelist.json"

	onClientWithWhitelist := func(s aws.Config) *lambda.Client {
		svc := lambda.New(s)
		AWSWithWhitelist(svc.Client, whitelist)
		return svc
	}

	onConfigWithWhitelist := func(s aws.Config) *lambda.Client {
		return lambda.New(AWSConfigWithWhitelist(s, whitelist))
	}

	type constructor func(aws.Config) *lambda.Client
	constructors := []struct {
		name        string
		constructor constructor
	}{
		{"AWS()", onClient},
		{"AWSConfig()", onConfig},
		{"AWSWithWhitelist()", onClientWithWhitelist},
		{"AWSConfigWithWhitelist()", onConfigWithWhitelist},
	}

	// Run all combinations of constructors + tests.
	for _, cons := range constructors {
		t.Run(cons.name, func(t *testing.T) {
			for _, test := range tests {
				t.Run(test.name, func(t *testing.T) {
					test.test(t, cons.constructor(fakeSession(t, test.failConn)))
				})
			}
		})
	}
}

func fakeSession(t *testing.T, failConn bool) aws.Config {
	cfg := defaults.Config()
	cfg.Region = "fake-moon-1"
	cfg.Credentials = aws.NewStaticCredentialsProvider("akid", "secret", "noop")
	cfg.Retryer = aws.DefaultRetryer{}

	if !failConn {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b := []byte(`{}`)
			w.WriteHeader(http.StatusOK)
			w.Write(b)
		}))
		cfg.EndpointResolver = aws.ResolveWithEndpointURL(ts.URL)
	} else {
		cfg.EndpointResolver = aws.ResolveWithEndpointURL("https://fake-moon-1.amazonaws.com")
	}
	return cfg
}

func testClientSuccessfulConnection(t *testing.T, svc *lambda.Client) {
	ctx, root := BeginSegment(context.Background(), "Test")
	_, err := svc.ListFunctionsRequest(&lambda.ListFunctionsInput{}).Send(ctx)
	root.Close(nil)
	assert.NoError(t, err)

	s, e := TestDaemon.Recv()
	assert.NoError(t, e)

	subseg := &Segment{}
	assert.NotEmpty(t, s.Subsegments)
	assert.NoError(t, json.Unmarshal(s.Subsegments[0], &subseg))
	assert.False(t, subseg.Fault)
	assert.NotEmpty(t, subseg.Subsegments)

	attemptSubseg := &Segment{}
	for _, sub := range subseg.Subsegments {
		tempSeg := &Segment{}
		assert.NoError(t, json.Unmarshal(sub, &tempSeg))
		if tempSeg.Name == "attempt" {
			attemptSubseg = tempSeg
			break
		}
	}

	assert.Equal(t, "attempt", attemptSubseg.Name)
	assert.Zero(t, attemptSubseg.openSegments)

	// Connect subsegment will contain multiple child subsegments.
	// The subsegment should fail since the endpoint is not valid,
	// and should not be InProgress.
	connectSubseg := &Segment{}
	assert.NotEmpty(t, attemptSubseg.Subsegments)
	assert.NoError(t, json.Unmarshal(attemptSubseg.Subsegments[0], &connectSubseg))
	assert.Equal(t, "connect", connectSubseg.Name)
	assert.False(t, connectSubseg.InProgress)
	assert.NotZero(t, connectSubseg.EndTime)
	assert.NotEmpty(t, connectSubseg.Subsegments)

	// Ensure that the 'connect' subsegments are completed.
	for _, sub := range connectSubseg.Subsegments {
		tempSeg := &Segment{}
		assert.NoError(t, json.Unmarshal(sub, &tempSeg))
		assert.False(t, tempSeg.InProgress)
		assert.NotZero(t, tempSeg.EndTime)
	}
}

func testClientFailedConnection(t *testing.T, svc *lambda.Client) {
	ctx, root := BeginSegment(context.Background(), "Test")
	_, err := svc.ListFunctionsRequest(&lambda.ListFunctionsInput{}).Send(ctx)
	root.Close(nil)
	assert.Error(t, err)

	s, e := TestDaemon.Recv()
	assert.NoError(t, e)

	subseg := &Segment{}
	assert.NotEmpty(t, s.Subsegments)
	assert.NoError(t, json.Unmarshal(s.Subsegments[0], &subseg))
	assert.True(t, subseg.Fault)
	// Should contain 'marshal', 'attempt' and 'wait' (for retries) subsegments only.
	assert.Len(t, subseg.Subsegments, 3)

	attemptSubseg := &Segment{}
	assert.NoError(t, json.Unmarshal(subseg.Subsegments[1], &attemptSubseg))
	assert.Equal(t, "attempt", attemptSubseg.Name)
	assert.Zero(t, attemptSubseg.openSegments)

	// Connect subsegment will contain multiple child subsegments.
	// The subsegment should fail since the endpoint is not valid,
	// and should not be InProgress.
	connectSubseg := &Segment{}
	assert.NotEmpty(t, attemptSubseg.Subsegments)
	assert.NoError(t, json.Unmarshal(attemptSubseg.Subsegments[0], &connectSubseg))
	assert.Equal(t, "connect", connectSubseg.Name)
	assert.False(t, connectSubseg.InProgress)
	assert.NotZero(t, connectSubseg.EndTime)
	assert.NotEmpty(t, connectSubseg.Subsegments)
}

func testClientWithoutSegment(t *testing.T, svc *lambda.Client) {
	Configure(Config{ContextMissingStrategy: &TestContextMissingStrategy{}})
	defer ResetConfig()

	ctx := context.Background()
	_, err := svc.ListFunctionsRequest(&lambda.ListFunctionsInput{}).Send(ctx)
	assert.NoError(t, err)
}
