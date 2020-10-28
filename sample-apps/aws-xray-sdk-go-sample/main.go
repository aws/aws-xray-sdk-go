// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0

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

const (
	dsn = "DSN_STRING"
)

func webServer(){
	http.Handle("/outgoing-http-call", xray.Handler(xray.NewFixedSegmentNamer("SampleApplication"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := ctxhttp.Get(r.Context(), xray.Client(nil), "https://aws.amazon.com/")
		if err != nil {
			log.Println(err)
			return
		}
		w.Write([]byte("Hello, http!"))
	})))
	http.Handle("/aws-sdk-call", xray.Handler(xray.NewFixedSegmentNamer("AWS SDK Calls"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testAWSCalls(r.Context())
		w.Write([]byte("Hello,aws!"))
	})))

	listenAddress := os.Getenv("LISTEN_ADDRESS")
	if listenAddress == "" {
		listenAddress = "127.0.0.1:8080"
	}
	http.ListenAndServe(listenAddress, nil)
	log.Printf("SampleApp is listening on %s !", listenAddress)
}



func testAWSCalls(ctx context.Context) {

	awsSess, err := session.NewSession()
	if err != nil {
		log.Fatalf("failed to open aws session")
	}

	s3Client := s3.New(awsSess)

	xray.AWS(s3Client.Client)

	if _, err = s3Client.ListBucketsWithContext(ctx, nil); err != nil {
		log.Println(err)
		return
	}
	log.Println("downstream aws calls successfully{}", )
}

func main() {
	log.Println("SampleApp Starts")
	webServer()
}

