// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package logger

import (
	"bytes"
	"strings"
	"testing"

	"github.com/aws/aws-xray-sdk-go/xraylog"
)

func TestLogger(t *testing.T) {
	oldLogger := Logger
	defer func() { Logger = oldLogger }()

	var buf bytes.Buffer

	// filter properly by level
	Logger = xraylog.NewDefaultLogger(&buf, xraylog.LogLevelWarn)

	Debug("debug")
	Info("info")
	Warn("warn")
	Error("error")

	gotLines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(gotLines) != 2 {
		t.Fatalf("got %d lines", len(gotLines))
	}

	if !strings.Contains(gotLines[0], "[WARN] warn") {
		t.Error("expected first line to be warn")
	}

	if !strings.Contains(gotLines[1], "[ERROR] error") {
		t.Error("expected second line to be warn")
	}
}

func TestDeferredDebug(t *testing.T) {
	oldLogger := Logger
	defer func() { Logger = oldLogger }()

	var buf bytes.Buffer

	Logger = xraylog.NewDefaultLogger(&buf, xraylog.LogLevelInfo)

	var called bool
	DebugDeferred(func() string {
		called = true
		return "deferred"
	})

	if called {
		t.Error("deferred should not have been called")
	}

	if buf.String() != "" {
		t.Errorf("unexpected log contents: %s", buf.String())
	}

	Logger = xraylog.NewDefaultLogger(&buf, xraylog.LogLevelDebug)

	DebugDeferred(func() string {
		called = true
		return "deferred"
	})

	if !called {
		t.Error("deferred should have been called")
	}

	if !strings.Contains(buf.String(), "[DEBUG] deferred") {
		t.Errorf("expected deferred message, got %s", buf.String())
	}
}
