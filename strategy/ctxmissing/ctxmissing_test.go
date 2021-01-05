// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package ctxmissing

import (
	"bytes"
	"strings"
	"testing"

	"github.com/aws/aws-xray-sdk-go/internal/logger"
	"github.com/aws/aws-xray-sdk-go/xraylog"
	"github.com/stretchr/testify/assert"
)

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
	oldLogger := logger.Logger
	defer func() { logger.Logger = oldLogger }()

	var buf bytes.Buffer
	logger.Logger = xraylog.NewDefaultLogger(&buf, xraylog.LogLevelDebug)

	l := NewDefaultLogErrorStrategy()
	l.ContextMissing("TestLogError")
	assert.True(t, strings.Contains(buf.String(), "Suppressing AWS X-Ray context missing panic: TestLogError"))
}

func TestDefaultIgnoreErrorStrategy(t *testing.T) {
	defer func() {
		p := recover()
		assert.Equal(t, p, nil)
	}()
	r := NewDefaultIgnoreErrorStrategy()
	r.ContextMissing("TestIgnoreError")
}
