// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0

package main

import (
	"context"
	"database/sql"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-xray-sdk-go/xray"
	_ "github.com/go-sql-driver/mysql"
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
	http.Handle("/aws-mysql-call", xray.Handler(xray.NewFixedSegmentNamer("SQL Calls"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testSQL(r.Context())
		w.Write([]byte("Hello,SQL!"))
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

func testSQL(ctx context.Context) {
	dsnString := os.Getenv(dsn)
	if dsnString == "" {
		log.Println("Set DSN_STRING environment variable as a connection string to the database")
		return
	}

	db, _ := xray.SQLContext("mysql", dsnString)
	defer db.Close()

	if _, err := db.ExecContext(ctx, "CREATE TABLE ID1 (val int)"); err != nil {
		log.Println(err)
		return
	}

	if err := transaction(ctx, db); err != nil {
		log.Println(err)
		return
	}

	if _, err := db.ExecContext(ctx, "DROP TABLE ID1"); err != nil {
		log.Println(err)
		return
	}

	log.Println("Mysql SQL calls successfully")
}

func transaction(ctx context.Context, db *sql.DB) error {
	tx, _ := db.BeginTx(ctx, nil)
	defer tx.Rollback()

	if _, err := db.ExecContext(ctx, "INSERT INTO ID1 (val) VALUES (?)", 1); err != nil {
		return err
	}

	r, _ := tx.QueryContext(ctx, "SELECT val FROM ID1 WHERE val = ?", 1)
	defer r.Close()

	for r.Next() {
		var val int
		err := r.Scan(&val)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func main() {
	log.Println("SampleApp Starts")
	webServer()
}

