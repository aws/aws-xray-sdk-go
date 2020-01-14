package xray

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/stretchr/testify/assert"
)

func TestAWS(t *testing.T) {
	// Runs a suite of tests against two different methods of registering
	// handlers on an AWS client.

	type test func(*testing.T, *lambda.Lambda)
	tests := []struct {
		name     string
		test     test
		failConn bool
	}{
		{"failed connection", testClientFailedConnection, true},
		{"successful connection", testClientSuccessfulConnection, false},
		{"without segment", testClientWithoutSegment, false},
		{"test data race", testAWSDataRace, false},
	}

	onClient := func(s *session.Session) *lambda.Lambda {
		svc := lambda.New(s)
		AWS(svc.Client)
		return svc
	}

	onSession := func(s *session.Session) *lambda.Lambda {
		return lambda.New(AWSSession(s))
	}

	const whitelist = "../resources/AWSWhitelist.json"

	onClientWithWhitelist := func(s *session.Session) *lambda.Lambda {
		svc := lambda.New(s)
		AWSWithWhitelist(svc.Client, whitelist)
		return svc
	}

	onSessionWithWhitelist := func(s *session.Session) *lambda.Lambda {
		return lambda.New(AWSSessionWithWhitelist(s, whitelist))
	}

	type constructor func(*session.Session) *lambda.Lambda
	constructors := []struct {
		name        string
		constructor constructor
	}{
		{"AWS()", onClient},
		{"AWSSession()", onSession},
		{"AWSWithWhitelist()", onClientWithWhitelist},
		{"AWSSessionWithWhitelist()", onSessionWithWhitelist},
	}

	// Run all combinations of constructors + tests.
	for _, cons := range constructors {
		cons := cons
		t.Run(cons.name, func(t *testing.T) {
			for _, test := range tests {
				test := test
				t.Run(test.name, func(t *testing.T) {
					test.test(t, cons.constructor(fakeSession(t, test.failConn)))
				})
			}
		})
	}
}

func fakeSession(t *testing.T, failConn bool) *session.Session {
	cfg := &aws.Config{
		Region:      aws.String("fake-moon-1"),
		Credentials: credentials.NewStaticCredentials("akid", "secret", "noop"),
	}
	if !failConn {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b := []byte(`{}`)
			w.WriteHeader(http.StatusOK)
			w.Write(b)
		}))
		cfg.Endpoint = aws.String(ts.URL)
	}
	s, err := session.NewSession(cfg)
	assert.NoError(t, err)
	return s
}

func testClientSuccessfulConnection(t *testing.T, svc *lambda.Lambda) {
	ctx, root := BeginSegment(context.Background(), "Test")
	_, err := svc.ListFunctionsWithContext(ctx, &lambda.ListFunctionsInput{})
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

func testClientFailedConnection(t *testing.T, svc *lambda.Lambda) {
	ctx, root := BeginSegment(context.Background(), "Test")
	_, err := svc.ListFunctionsWithContext(ctx, &lambda.ListFunctionsInput{})
	root.Close(nil)
	assert.Error(t, err)

	s, e := TestDaemon.Recv()
	assert.NoError(t, e)

	subseg := &Segment{}
	assert.NotEmpty(t, s.Subsegments)
	assert.NoError(t, json.Unmarshal(s.Subsegments[0], &subseg))
	assert.True(t, subseg.Fault)
	// Should contain 'marshal' and 'attempt' subsegments only.
	assert.Len(t, subseg.Subsegments, 2)

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

func testClientWithoutSegment(t *testing.T, svc *lambda.Lambda) {
	Configure(Config{ContextMissingStrategy: &TestContextMissingStrategy{}})
	defer ResetConfig()

	ctx := context.Background()
	_, err := svc.ListFunctionsWithContext(ctx, &lambda.ListFunctionsInput{})
	assert.NoError(t, err)
}

func testAWSDataRace(t *testing.T, svc *lambda.Lambda) {
	Configure(Config{ContextMissingStrategy: &TestContextMissingStrategy{}, DaemonAddr: "localhost:3000"})
	defer ResetConfig()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx, seg := BeginSegment(ctx, "TestSegment")

	wg := sync.WaitGroup{}

	for i := 0; i < 5; i++ {
		if i != 3 && i != 2 {
			wg.Add(1)
		}
		go func(i int) {
			if i != 3 && i != 2 {
				time.Sleep(time.Nanosecond)
				defer wg.Done()
			}
			_, seg := BeginSubsegment(ctx, "TestSubsegment1")
			time.Sleep(time.Nanosecond)
			seg.Close(nil)
			svc.ListFunctionsWithContext(ctx, &lambda.ListFunctionsInput{})
			if i == 3 || i == 2 {
				cancel() // cancel context
			}
		}(i)
	}

	wg.Wait()
	seg.Close(nil)
}
