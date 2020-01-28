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

	type test func(context.Context, *TestDaemon, *testing.T, *lambda.Lambda)
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
				ctx, td := NewTestDaemon()
				defer td.Close()

				t.Run(test.name, func(t *testing.T) {
					session, cleanup := fakeSession(t, test.failConn)
					defer cleanup()
					test.test(ctx, td, t, cons.constructor(session))
				})
			}
		})
	}
}

func fakeSession(t *testing.T, failConn bool) (*session.Session, func()) {
	cfg := &aws.Config{
		Region:      aws.String("fake-moon-1"),
		Credentials: credentials.NewStaticCredentials("akid", "secret", "noop"),
	}

	var ts *httptest.Server
	if !failConn {
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b := []byte(`{}`)
			w.WriteHeader(http.StatusOK)
			w.Write(b)
		}))
		cfg.Endpoint = aws.String(ts.URL)
	}
	s, err := session.NewSession(cfg)
	assert.NoError(t, err)
	return s, func() {
		if ts != nil {
			ts.Close()
		}
	}
}

func testClientSuccessfulConnection(ctx context.Context, td *TestDaemon, t *testing.T, svc *lambda.Lambda) {
	ctx, root := BeginSegment(ctx, "Test")
	_, err := svc.ListFunctionsWithContext(ctx, &lambda.ListFunctionsInput{})
	root.Close(nil)
	assert.NoError(t, err)

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}

	var subseg *Segment
	if !assert.NoError(t, json.Unmarshal(seg.Subsegments[0], &subseg)) {
		return
	}
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

func testClientFailedConnection(ctx context.Context, td *TestDaemon, t *testing.T, svc *lambda.Lambda) {
	ctx, root := BeginSegment(ctx, "Test")
	_, err := svc.ListFunctionsWithContext(ctx, &lambda.ListFunctionsInput{})
	root.Close(nil)
	assert.Error(t, err)

	seg, err := td.Recv()
	if !assert.NoError(t, err) {
		return
	}

	var subseg *Segment
	if !assert.NoError(t, json.Unmarshal(seg.Subsegments[0], &subseg)) {
		return
	}
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

func testClientWithoutSegment(ctx context.Context, td *TestDaemon, t *testing.T, svc *lambda.Lambda) {
	_, err := svc.ListFunctionsWithContext(ctx, &lambda.ListFunctionsInput{})
	assert.NoError(t, err)
}

func testAWSDataRace(ctx context.Context, td *TestDaemon, t *testing.T, svc *lambda.Lambda) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	ctx, seg := BeginSegment(ctx, "TestSegment")

	var wg sync.WaitGroup
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
