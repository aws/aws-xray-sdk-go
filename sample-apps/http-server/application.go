package main

import (
	"context"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-xray-sdk-go/xray"
	"golang.org/x/net/context/ctxhttp"
	"log"
	"net/http"
	"os"
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
		listenAddress = "127.0.0.1:8080"
	}
	_ = http.ListenAndServe(listenAddress, nil)
	log.Printf("App is listening on %s !", listenAddress)
}

func testAWSCalls(ctx context.Context) {

	awsSess, err := session.NewSession()
	if err != nil {
		log.Fatalf("Failed to open aws session")
	}

	s3Client := s3.New(awsSess)

	xray.AWS(s3Client.Client)

	if _, err = s3Client.ListBucketsWithContext(ctx, nil); err != nil {
		log.Println(err)
		return
	}
	log.Println("Successfully traced aws sdk call")
}

func main() {
	webServer()
}

