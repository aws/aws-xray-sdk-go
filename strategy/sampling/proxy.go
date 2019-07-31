// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package sampling

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	xraySvc "github.com/aws/aws-sdk-go-v2/service/xray"
	"github.com/aws/aws-xray-sdk-go/daemoncfg"
	"github.com/aws/aws-xray-sdk-go/internal/logger"
)

// proxy is an implementation of svcProxy that forwards requests to the XRay daemon
type proxy struct {
	// XRay client for sending unsigned proxied requests to the daemon
	xray *xraySvc.Client
}

// NewProxy returns a Proxy
func newProxy(d *daemoncfg.DaemonEndpoints) (svcProxy, error) {

	if d == nil {
		d = daemoncfg.GetDaemonEndpoints()
	}
	logger.Infof("X-Ray proxy using address : %v", d.TCPAddr.String())
	url := "http://" + d.TCPAddr.String()

	// Dummy session for unsigned requests
	cfg := defaults.Config()
	cfg.Region = "us-west-1"
	cfg.Credentials = aws.NewStaticCredentialsProvider("", "", "")
	// Endpoint resolver for proxying requests through the daemon
	cfg.EndpointResolver = aws.ResolveWithEndpointURL(url)

	x := xraySvc.New(cfg)
	// Remove Signer and replace with No-Op handler
	x.Handlers.Sign.Clear()
	x.Handlers.Sign.PushBack(func(request *aws.Request) {
		// Do nothing
	})

	p := &proxy{xray: x}

	return p, nil
}

// GetSamplingTargets calls the XRay daemon for sampling targets
func (p *proxy) GetSamplingTargets(s []xraySvc.SamplingStatisticsDocument) (*xraySvc.GetSamplingTargetsOutput, error) {
	input := &xraySvc.GetSamplingTargetsInput{
		SamplingStatisticsDocuments: s,
	}

	output, err := p.xray.GetSamplingTargetsRequest(input).Send(context.TODO())
	if err != nil {
		return nil, err
	}

	return output.GetSamplingTargetsOutput, nil
}

// GetSamplingRules calls the XRay daemon for sampling rules
func (p *proxy) GetSamplingRules() ([]xraySvc.SamplingRuleRecord, error) {
	input := &xraySvc.GetSamplingRulesInput{}

	output, err := p.xray.GetSamplingRulesRequest(input).Send(context.TODO())
	if err != nil {
		return nil, err
	}

	rules := output.SamplingRuleRecords

	return rules, nil
}
