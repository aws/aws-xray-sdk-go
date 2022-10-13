package lambda

import (
	"github.com/aws/aws-lambda-go/events"
	"strings"
)

func IsSampled(sqsMessge *events.SQSMessage) bool {
	value, ok := sqsMessge.Attributes["AWSTraceHeader"]

	if !ok {
		return false
	} else {
		return strings.Contains(value, "Sampled=1")
	}
}
