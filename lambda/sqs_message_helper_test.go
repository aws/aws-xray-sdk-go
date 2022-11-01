package lambda

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestSQSMessageHelper(t *testing.T) {
	testTrue(t, "Root=1-632BB806-bd862e3fe1be46a994272793;Sampled=1")
	testTrue(t, "Root=1-5759e988-bd862e3fe1be46a994272793;Sampled=1")
	testTrue(t, "Root=1-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8;Sampled=1")

	testFalse(t, "Root=1-632BB806-bd862e3fe1be46a994272793")
	testFalse(t, "Root=1-632BB806-bd862e3fe1be46a994272793;Sampled=0")
	testFalse(t, "Root=1-5759e988-bd862e3fe1be46a994272793;Sampled=0")
	testFalse(t, "Root=1-5759e988-bd862e3fe1be46a994272793;Parent=53995c3f42cd8ad8;Sampled=0")
}

func testTrue(t *testing.T, header string) {
	var sqsMessage events.SQSMessage
	sqsMessage.Attributes = make(map[string]string)
	sqsMessage.Attributes["AWSTraceHeader"] = header
	assert.True(t, IsSampled(sqsMessage))
}

func testFalse(t *testing.T, header string) {
	var sqsMessage events.SQSMessage
	sqsMessage.Attributes = make(map[string]string)
	sqsMessage.Attributes["AWSTraceHeader"] = header
	assert.False(t, IsSampled(sqsMessage))
}
