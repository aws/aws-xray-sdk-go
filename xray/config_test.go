// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package xray

import (
	"fmt"
	"net"
	"testing"

	"github.com/aws/aws-xray-sdk-go/strategy/ctxmissing"
	"github.com/aws/aws-xray-sdk-go/strategy/exception"
	"github.com/aws/aws-xray-sdk-go/strategy/sampling"
	"github.com/stretchr/testify/assert"
)

type TestSamplingStrategy struct{}

type TestExceptionFormattingStrategy struct{}

type TestStreamingStrategy struct{}

type TestContextMissingStrategy struct{}

type TestEmitter struct{}

func (tss *TestSamplingStrategy) ShouldTrace(request *sampling.Request) *sampling.Decision {
	return &sampling.Decision{
		Sample: true,
	}
}

func (tefs *TestExceptionFormattingStrategy) Error(message string) *exception.XRayError {
	return &exception.XRayError{}
}

func (tefs *TestExceptionFormattingStrategy) Errorf(message string, args ...interface{}) *exception.XRayError {
	return &exception.XRayError{}
}

func (tefs *TestExceptionFormattingStrategy) Panic(message string) *exception.XRayError {
	return &exception.XRayError{}
}

func (tefs *TestExceptionFormattingStrategy) Panicf(message string, args ...interface{}) *exception.XRayError {
	return &exception.XRayError{}
}

func (tefs *TestExceptionFormattingStrategy) ExceptionFromError(err error) exception.Exception {
	return exception.Exception{}
}

func (sms *TestStreamingStrategy) RequiresStreaming(seg *Segment) bool {
	return false
}

func (sms *TestStreamingStrategy) StreamCompletedSubsegments(seg *Segment) [][]byte {
	var test [][]byte
	return test
}

func (te *TestEmitter) Emit(seg *Segment) {}

func (te *TestEmitter) RefreshEmitterWithAddress(raddr *net.UDPAddr) {}

func (cms *TestContextMissingStrategy) ContextMissing(v interface{}) {
	fmt.Printf("Test ContextMissing Strategy %v\n", v)
}

// TODO: @shogo82148 think how do we write tests for the environment values.
// TestEnvironmentDaemonAddress is commented out because it touches the environment values.
// https://github.com/aws/aws-xray-sdk-go/pull/177#issuecomment-575416696

// func TestEnvironmentDaemonAddress(t *testing.T) {
// 	os.Setenv("AWS_XRAY_DAEMON_ADDRESS", "192.168.2.100:2000")
// 	defer os.Unsetenv("AWS_XRAY_DAEMON_ADDRESS")

// 	cfg := newGlobalConfig()

// 	daemonAddr := &net.UDPAddr{IP: net.IPv4(192, 168, 2, 100), Port: 2000}
// 	assert.Equal(t, daemonAddr, cfg.daemonAddr)
// }

// TODO: @shogo82148 think how do we write tests for the environment values.
// TestInvalidEnvironmentDaemonAddress is commented out because it touches the environment values.
// https://github.com/aws/aws-xray-sdk-go/pull/177#issuecomment-575416696

// func TestInvalidEnvironmentDaemonAddress(t *testing.T) {
// 	os.Setenv("AWS_XRAY_DAEMON_ADDRESS", "This is not a valid address")
// 	defer os.Unsetenv("AWS_XRAY_DAEMON_ADDRESS")

// 	assert.Panics(t, func() { _ = newGlobalConfig() })
// }

func TestDefaultConfigureParameters(t *testing.T) {
	daemonAddr := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 2000}
	efs, _ := exception.NewDefaultFormattingStrategy()
	sms, _ := NewDefaultStreamingStrategy()
	cms := ctxmissing.NewDefaultRuntimeErrorStrategy()

	assert.Equal(t, daemonAddr, globalCfg.daemonAddr)
	assert.Equal(t, efs, globalCfg.exceptionFormattingStrategy)
	assert.Equal(t, "", globalCfg.serviceVersion)
	assert.Equal(t, sms, globalCfg.streamingStrategy)
	assert.Equal(t, cms, globalCfg.contextMissingStrategy)
}

// TODO: @shogo82148 think how do we write tests for the environment values.
// TestSetConfigureParameters is commented out because it touches the environment values.
// https://github.com/aws/aws-xray-sdk-go/pull/177#issuecomment-575416696

// func TestSetConfigureParameters(t *testing.T) {
// 	daemonAddr := "127.0.0.1:3000"
// 	logLevel := "error"
// 	logFormat := "[%Level] %Msg%n"
// 	serviceVersion := "TestVersion"

// 	ss := &TestSamplingStrategy{}
// 	efs := &TestExceptionFormattingStrategy{}
// 	sms := &TestStreamingStrategy{}
// 	cms := &TestContextMissingStrategy{}

// 	Configure(Config{
// 		DaemonAddr:                  daemonAddr,
// 		ServiceVersion:              serviceVersion,
// 		SamplingStrategy:            ss,
// 		ExceptionFormattingStrategy: efs,
// 		StreamingStrategy:           sms,
// 		ContextMissingStrategy:      cms,
// 		LogLevel:                    logLevel,
// 		LogFormat:                   logFormat,
// 	})

// 	assert.Equal(t, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 3000}, globalCfg.daemonAddr)
// 	assert.Equal(t, ss, globalCfg.samplingStrategy)
// 	assert.Equal(t, efs, globalCfg.exceptionFormattingStrategy)
// 	assert.Equal(t, sms, globalCfg.streamingStrategy)
// 	assert.Equal(t, cms, globalCfg.contextMissingStrategy)
// 	assert.Equal(t, serviceVersion, globalCfg.serviceVersion)

// 	ResetConfig()
// }

// TODO: @shogo82148 think how do we write tests for the environment values.
// TestSetDaemonAddressEnvironmentVariable is commented out because it touches the environment values.
// https://github.com/aws/aws-xray-sdk-go/pull/177#issuecomment-575416696

// func TestSetDaemonAddressEnvironmentVariable(t *testing.T) {
// 	env := stashEnv()
// 	defer popEnv(env)
// 	daemonAddr := "127.0.0.1:3000"
// 	os.Setenv("AWS_XRAY_DAEMON_ADDRESS", "127.0.0.1:4000")
// 	Configure(Config{DaemonAddr: daemonAddr})
// 	assert.Equal(t, &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 4000}, globalCfg.daemonAddr)
// 	os.Unsetenv("AWS_XRAY_DAEMON_ADDRESS")

// 	ResetConfig()
// }

// TODO: @shogo82148 think how do we write tests for the environment values.
// TestSetContextMissingEnvironmentVariable is commented out because it touches the environment values.
// https://github.com/aws/aws-xray-sdk-go/pull/177#issuecomment-575416696

// func TestSetContextMissingEnvironmentVariable(t *testing.T) {
// 	env := stashEnv()
// 	defer popEnv(env)
// 	cms := ctxmissing.NewDefaultLogErrorStrategy()
// 	r := ctxmissing.NewDefaultRuntimeErrorStrategy()
// 	os.Setenv("AWS_XRAY_CONTEXT_MISSING", "RUNTIME_ERROR")
// 	Configure(Config{ContextMissingStrategy: cms})
// 	assert.Equal(t, r, globalCfg.contextMissingStrategy)
// 	os.Unsetenv("AWS_XRAY_CONTEXT_MISSING")

// 	ResetConfig()
// }

// TODO: @shogo82148 think how do we write tests for the environment values.
// TestConfigureWithContext is commented out because it touches the environment values.
// https://github.com/aws/aws-xray-sdk-go/pull/177#issuecomment-575416696

// func TestConfigureWithContext(t *testing.T) {
// 	daemonAddr := "127.0.0.1:3000"
// 	logLevel := "error"
// 	logFormat := "[%Level] %Msg%n"
// 	serviceVersion := "TestVersion"

// 	ss := &TestSamplingStrategy{}
// 	efs := &TestExceptionFormattingStrategy{}
// 	sms := &TestStreamingStrategy{}
// 	cms := &TestContextMissingStrategy{}
// 	de := &TestEmitter{}

// 	ctx, err := ContextWithConfig(context.Background(), Config{
// 		DaemonAddr:                  daemonAddr,
// 		ServiceVersion:              serviceVersion,
// 		SamplingStrategy:            ss,
// 		ExceptionFormattingStrategy: efs,
// 		StreamingStrategy:           sms,
// 		Emitter:                     de,
// 		ContextMissingStrategy:      cms,
// 		LogLevel:                    logLevel,
// 		LogFormat:                   logFormat,
// 	})

// 	cfg := GetRecorder(ctx)
// 	assert.Nil(t, err)
// 	assert.Equal(t, daemonAddr, cfg.DaemonAddr)
// 	assert.Equal(t, logLevel, cfg.LogLevel)
// 	assert.Equal(t, logFormat, cfg.LogFormat)
// 	assert.Equal(t, ss, cfg.SamplingStrategy)
// 	assert.Equal(t, efs, cfg.ExceptionFormattingStrategy)
// 	assert.Equal(t, sms, cfg.StreamingStrategy)
// 	assert.Equal(t, de, cfg.Emitter)
// 	assert.Equal(t, cms, cfg.ContextMissingStrategy)
// 	assert.Equal(t, serviceVersion, cfg.ServiceVersion)

// 	ResetConfig()
// }

// TODO: @shogo82148 think how do we write tests for the environment values.
// TestSelectiveConfigWithContext is commented out because it touches the environment values.
// https://github.com/aws/aws-xray-sdk-go/pull/177#issuecomment-575416696

// func TestSelectiveConfigWithContext(t *testing.T) {
// 	daemonAddr := "127.0.0.1:3000"
// 	serviceVersion := "TestVersion"
// 	cms := &TestContextMissingStrategy{}

// 	ctx, err := ContextWithConfig(context.Background(), Config{
// 		DaemonAddr:             daemonAddr,
// 		ServiceVersion:         serviceVersion,
// 		ContextMissingStrategy: cms,
// 	})

// 	cfg := GetRecorder(ctx)
// 	assert.Nil(t, err)
// 	assert.Equal(t, daemonAddr, cfg.DaemonAddr)
// 	assert.Equal(t, cms, cfg.ContextMissingStrategy)
// 	assert.Equal(t, serviceVersion, cfg.ServiceVersion)

// 	ResetConfig()
// }
