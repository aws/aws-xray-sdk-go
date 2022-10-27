package lambda

import (
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

func IsSampled(sqsMessge events.SQSMessage) bool {
	value, ok := sqsMessge.Attributes["AWSTraceHeader"]

	if !ok {
		return false
	}

	return strings.Contains(value, "Sampled=1")
}
