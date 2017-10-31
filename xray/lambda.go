package xray

import (
	"os"
	"time"
	"context"
	"path/filepath"
	log "github.com/cihub/seelog"
	"github.com/aws/aws-xray-sdk-go/header"
)

var LambdaTraceHeaderKey string = "x-amzn-trace-id"

var LambdaTaskRootKey string = "LAMBDA_TASK_ROOT"
var SDKInitializedFileFolder string = "/tmp/.aws-xray"
var SDKInitializedFileName string = "initialized"

func getTraceHeaderFromContext(ctx context.Context) *header.Header {
	var traceHeader string

	if traceHeaderValue := ctx.Value(LambdaTraceHeaderKey); traceHeaderValue != nil {
		traceHeader = traceHeaderValue.(string)
		return header.FromString(traceHeader)
	}
	return nil
}

func newFacadeSegment(ctx context.Context) (context.Context, *Segment) {
	traceHeader := getTraceHeaderFromContext(ctx)
	return BeginFacadeSegment(ctx, "facade", traceHeader)
}

func getLambdaTaskRoot() string {
	return os.Getenv(LambdaTaskRootKey)
}

func initLambda() {
	if getLambdaTaskRoot() != "" {
		now := time.Now()
		filePath, err := createFile(SDKInitializedFileFolder, SDKInitializedFileName)
		if err != nil {
			log.Tracef("unable to create file at %s. failed to signal SDK initialization with error: %v", filePath, err)
		} else {
			e := os.Chtimes(filePath, now, now)
			if e != nil {
				log.Tracef("unable to write to %s. failed to signal SDK initialization with error: %v", filePath, e)
			}
		}
	}
}

func createFile(dir string, name string) (string, error) {
	fileDir := filepath.FromSlash(dir)
	filePath := fileDir + string(os.PathSeparator) + name

	// detect if file exists
	var _, err = os.Stat(filePath)

	// create file if not exists
	if os.IsNotExist(err) {
		e := os.MkdirAll(dir, os.ModePerm)
		if e != nil {
			return filePath, e
		} else {
			var file, err = os.Create(filePath)
			if err != nil {
				return filePath, err
			}
			file.Close()
			return filePath, nil
		}
	} else {
		return filePath, err
	}
	return filePath, nil
}