// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package xraylog

import (
	"fmt"
	"io"
	"time"
)

// Logger is the logging interface used by xray. fmt.Stringer is used to
// defer expensive serialization operations until the message is actually
// logged (i.e. don't bother serializing debug messages if they aren't going
// to show up).
type Logger interface {
	Log(level LogLevel, msg fmt.Stringer)
}

// LogLevel represents the severity of a log message, where a higher value
// means more severe. The integer value should not be serialized as it is
// subject to change.
type LogLevel int

const (
	LogLevelDebug LogLevel = iota + 1
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

func (ll LogLevel) String() string {
	switch ll {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return fmt.Sprintf("UNKNOWN<%d>", ll)
	}
}

// NewDefaultLogger makes a Logger object that writes newline separated
// messages to w, if the level of the message is at least minLogLevel.
func NewDefaultLogger(w io.Writer, minLogLevel LogLevel) Logger {
	return &defaultLogger{w, minLogLevel}
}

type defaultLogger struct {
	w        io.Writer
	minLevel LogLevel
}

func (l *defaultLogger) Log(ll LogLevel, msg fmt.Stringer) {
	if ll < l.minLevel {
		return
	}

	fmt.Fprintf(l.w, "%s [%s] %s\n", time.Now().Format(time.RFC3339), ll, msg)
}

// NullLogger can be used to disable logging (pass to xray.SetLogger()).
var NullLogger = nullLogger{}

type nullLogger struct{}

func (nl nullLogger) Log(ll LogLevel, msg fmt.Stringer) {
}
