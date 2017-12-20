// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package xray

import (
	"net"
	"os"
	"sync"

	"github.com/aws/aws-xray-sdk-go/logger"
	"github.com/aws/aws-xray-sdk-go/strategy/ctxmissing"
	"github.com/aws/aws-xray-sdk-go/strategy/exception"
	"github.com/aws/aws-xray-sdk-go/strategy/sampling"
)

var privateCfg = newPrivateConfig()

func newPrivateConfig() *privateConfig {
	ret := &privateConfig{
		daemonAddr: &net.UDPAddr{
			IP:   net.IPv4(127, 0, 0, 1),
			Port: 2000,
		},
	}

	ss, err := sampling.NewLocalizedStrategy()
	if err != nil {
		panic(err)
	}
	ret.samplingStrategy = ss

	efs, err := exception.NewDefaultFormattingStrategy()
	if err != nil {
		panic(err)
	}
	ret.exceptionFormattingStrategy = efs

	sts, err := NewDefaultStreamingStrategy()
	if err != nil {
		panic(err)
	}
	ret.streamingStrategy = sts

	cm := ctxmissing.NewDefaultRuntimeErrorStrategy()

	ret.contextMissingStrategy = cm

	return ret
}

type privateConfig struct {
	sync.RWMutex

	daemonAddr                  *net.UDPAddr
	serviceVersion              string
	samplingStrategy            sampling.Strategy
	streamingStrategy           StreamingStrategy
	exceptionFormattingStrategy exception.FormattingStrategy
	contextMissingStrategy      ctxmissing.Strategy
	logger                      logger.Logger
}

// Config is a set of X-Ray configurations.
type Config struct {
	DaemonAddr                  string
	ServiceVersion              string
	SamplingStrategy            sampling.Strategy
	StreamingStrategy           StreamingStrategy
	ExceptionFormattingStrategy exception.FormattingStrategy
	ContextMissingStrategy      ctxmissing.Strategy
	Logger                      logger.Logger
}

// Configure overrides default configuration options with customer-defined values.
func Configure(c Config) error {
	privateCfg.Lock()
	defer privateCfg.Unlock()

	var errors exception.MultiError

	var daemonAddress string
	if addr := os.Getenv("AWS_XRAY_DAEMON_ADDRESS"); addr != "" {
		daemonAddress = addr
	} else if c.DaemonAddr != "" {
		daemonAddress = c.DaemonAddr
	}

	if daemonAddress != "" {
		addr, err := net.ResolveUDPAddr("udp", daemonAddress)
		if err == nil {
			privateCfg.daemonAddr = addr
			go refreshEmitter()
		} else {
			errors = append(errors, err)
		}
	}

	if c.SamplingStrategy != nil {
		privateCfg.samplingStrategy = c.SamplingStrategy
	}

	if c.ExceptionFormattingStrategy != nil {
		privateCfg.exceptionFormattingStrategy = c.ExceptionFormattingStrategy
	}

	if c.StreamingStrategy != nil {
		privateCfg.streamingStrategy = c.StreamingStrategy
	}

	cms := os.Getenv("AWS_XRAY_CONTEXT_MISSING")
	if cms != "" {
		if cms == ctxmissing.RuntimeErrorStrategy {
			cm := ctxmissing.NewDefaultRuntimeErrorStrategy()
			privateCfg.contextMissingStrategy = cm
		} else if cms == ctxmissing.LogErrorStrategy {
			cm := ctxmissing.NewDefaultLogErrorStrategy()
			privateCfg.contextMissingStrategy = cm
		}
	} else if c.ContextMissingStrategy != nil {
		privateCfg.contextMissingStrategy = c.ContextMissingStrategy
	}

	if c.ServiceVersion != "" {
		privateCfg.serviceVersion = c.ServiceVersion
	}

	if c.Logger != nil {
		privateCfg.logger = c.Logger
		logger.InjectLogger(privateCfg.logger)
	}

	switch len(errors) {
	case 0:
		return nil
	case 1:
		return errors[0]
	default:
		return errors
	}
}

func (c *privateConfig) DaemonAddr() *net.UDPAddr {
	c.RLock()
	defer c.RUnlock()
	return c.daemonAddr
}

func (c *privateConfig) SamplingStrategy() sampling.Strategy {
	c.RLock()
	defer c.RUnlock()
	return c.samplingStrategy
}

func (c *privateConfig) StreamingStrategy() StreamingStrategy {
	c.RLock()
	defer c.RUnlock()
	return c.streamingStrategy
}

func (c *privateConfig) ExceptionFormattingStrategy() exception.FormattingStrategy {
	c.RLock()
	defer c.RUnlock()
	return c.exceptionFormattingStrategy
}

func (c *privateConfig) ContextMissingStrategy() ctxmissing.Strategy {
	c.RLock()
	defer c.RUnlock()
	return c.contextMissingStrategy
}

func (c *privateConfig) ServiceVersion() string {
	c.RLock()
	defer c.RUnlock()
	return c.serviceVersion
}

func (c *privateConfig) Logger() logger.Logger {
	c.RLock()
	defer c.RUnlock()
	return c.logger
}
