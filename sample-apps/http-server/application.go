package main

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-xray-sdk-go/xray"
	"golang.org/x/net/context/ctxhttp"
	"net/http"
)

func main() {
	http.Handle("/aws-sdk-call", xray.Handler(xray.NewFixedSegmentNamer("/aws-sdk-call"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := session.Must(session.NewSession(&aws.Config{
			Region: aws.String("us-west-2")},))
		dynamo := dynamodb.New(sess)
		xray.AWS(dynamo.Client)
		_, _ = dynamo.ListTablesWithContext(r.Context(), &dynamodb.ListTablesInput{})

		_, _ = w.Write([]byte("Ok! tracing aws sdk call"))
	})))

	http.Handle("/outgoing-http-call", xray.Handler(xray.NewFixedSegmentNamer("/outgoing-http-call"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := ctxhttp.Get(r.Context(), xray.Client(nil), "https://aws.amazon.com/")
		if err != nil {
			return
		}

		_, _ = w.Write([]byte("Ok! tracing outgoing http call"))
	})))

	http.Handle("/annotation-metadata", xray.Handler(xray.NewFixedSegmentNamer("/annotation-metadata"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, subSeg := xray.BeginSubsegment(r.Context(), "subsegment")
		_ = subSeg.AddAnnotation("one", "1")
		_ = subSeg.AddMetadata("integration-test", "true")
		subSeg.Close(nil)
		_, _ = w.Write([]byte("Ok! annotation-metadata testing"))
	})))


	_ = http.ListenAndServe(":5000", nil)
}