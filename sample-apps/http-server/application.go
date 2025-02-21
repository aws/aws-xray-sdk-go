package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-xray-sdk-go/v2/instrumentation/awsv2"
	"github.com/aws/aws-xray-sdk-go/v2/xray"
	"golang.org/x/net/context/ctxhttp"
)

func webServer() {
	http.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("healthcheck"))
	}))

	//test http instrumentation
	http.Handle("/outgoing-http-call", xray.Handler(xray.NewFixedSegmentNamer("/outgoing-http-call"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := ctxhttp.Get(r.Context(), xray.Client(nil), "https://aws.amazon.com")
		if err != nil {
			log.Println(err)
			return
		}
		_, _ = w.Write([]byte("Tracing http call!"))
	})))

	//test aws sdk instrumentation
	http.Handle("/aws-sdk-call", xray.Handler(xray.NewFixedSegmentNamer("/aws-sdk-call"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testAWSCalls(r.Context())
		_, _ = w.Write([]byte("Tracing aws sdk call!"))
	})))

	listenAddress := os.Getenv("LISTEN_ADDRESS")
	if listenAddress == "" {
		listenAddress = "127.0.0.1:5000"
	}
	_ = http.ListenAndServe(listenAddress, nil)
	log.Printf("App is listening on %s !", listenAddress)
}

func testAWSCalls(ctx context.Context) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-west-2"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	// Instrumenting AWS SDK v2
	awsv2.AWSV2Instrumentor(&cfg.APIOptions)
	// Using the Config value, create the S3 client
	svc := s3.NewFromConfig(cfg)
	// Build the request with its input parameters
	_, err = svc.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		log.Fatalf("failed to list buckets, %v", err)
	}

	log.Println("Successfully traced aws sdk call")
}

func main() {
	webServer()
}
