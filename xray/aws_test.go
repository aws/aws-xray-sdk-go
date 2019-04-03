package xray

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/stretchr/testify/assert"
)

func TestClientSuccessfulConnection(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := []byte(`{}`)
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}))

	svc := lambda.New(session.Must(session.NewSession(&aws.Config{
		Endpoint:    aws.String(ts.URL),
		Region:      aws.String("fake-moon-1"),
		Credentials: credentials.NewStaticCredentials("akid", "secret", "noop")})))

	ctx, root := BeginSegment(context.Background(), "Test")

	AWS(svc.Client)

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

func TestClientFailedConnection(t *testing.T) {
	svc := lambda.New(session.Must(session.NewSession(&aws.Config{
		Region:      aws.String("fake-moon-1"),
		Credentials: credentials.NewStaticCredentials("akid", "secret", "noop")})))

	ctx, root := BeginSegment(context.Background(), "Test")

	AWS(svc.Client)

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
