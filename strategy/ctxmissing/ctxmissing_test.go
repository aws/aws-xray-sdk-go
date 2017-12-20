// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package ctxmissing

import (
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/aws/aws-xray-sdk-go/logger"
	"github.com/stretchr/testify/assert"
)

type LogWriter struct {
	logger.Logger
	Logs []string
}

func (l *LogWriter) Debug(msg string)                          {}
func (l *LogWriter) Debugf(format string, args ...interface{}) {}
func (l *LogWriter) Info(msg string)                           {}
func (l *LogWriter) Infof(format string, args ...interface{})  {}
func (l *LogWriter) Warn(msg string)                           {}
func (l *LogWriter) Warnf(format string, args ...interface{})  {}
func (l *LogWriter) Error(msg string)                          {}
func (l *LogWriter) Errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args)
	log.Println(msg)
	l.Logs = append(l.Logs, msg)
}

func (sw *LogWriter) Write(p []byte) (n int, err error) {
	sw.Logs = append(sw.Logs, string(p))
	return len(p), nil
}

func LogSetup() *LogWriter {
	writer := &LogWriter{}
	/*
		logger, err := log.LoggerFromWriterWithMinLevelAndFormat(writer, log.TraceLvl, "%Ns [%Level] %Msg")
		if err != nil {
			panic(err)
		}
		log.ReplaceLogger(logger)
	*/
	logger.InjectLogger(writer)
	return writer
}

func TestDefaultRuntimeErrorStrategy(t *testing.T) {
	defer func() {
		if p := recover(); p != nil {
			assert.Equal(t, "TestRuntimeError", p.(string))
		}
	}()
	r := NewDefaultRuntimeErrorStrategy()
	r.ContextMissing("TestRuntimeError")
}

func TestDefaultLogErrorStrategy(t *testing.T) {
	logger := LogSetup()
	l := NewDefaultLogErrorStrategy()
	l.ContextMissing("TestLogError")
	assert.True(t, strings.Contains(logger.Logs[0], "Suppressing AWS X-Ray context missing panic: [[TestLogError]]"))
}
