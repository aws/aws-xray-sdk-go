// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package sampling

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	xraySvc "github.com/aws/aws-sdk-go/service/xray"
	"github.com/aws/aws-xray-sdk-go/daemoncfg"
	log "github.com/cihub/seelog"
)

// proxy is an implementation of svcProxy that forwards requests to the XRay daemon
type proxy struct {
	// XRay client for sending unsigned proxied requests to the daemon
	xray *xraySvc.XRay
}

// NewProxy returns a Proxy
func NewProxy(d *daemoncfg.DaemonEndpoints) (svcProxy, error) {

	if d == nil {
		d = daemoncfg.GetDaemonEndpoints()
	}
	log.Infof("X-Ray proxy using address : %v", d.TCPAddr.String())
	url := "http://" + d.TCPAddr.String()

	// Endpoint resolver for proxying requests through the daemon
	f := func(service, region string, optFns ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
		return endpoints.ResolvedEndpoint{
			URL: url,
		}, nil
	}

	// Dummy session for unsigned requests
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String("us-west-1"),
		Credentials:      credentials.NewStaticCredentials("", "", ""),
		EndpointResolver: endpoints.ResolverFunc(f),
	})

	if err != nil {
		return nil, err
	}

	x := xraySvc.New(sess)

	// Remove Signer and replace with No-Op handler
	x.Handlers.Sign.Clear()
	x.Handlers.Sign.PushBack(func(*request.Request) {
		// Do nothing
	})

	p := &proxy{xray: x}

	return p, nil
}

// GetSamplingTargets calls the XRay daemon for sampling targets
func (p *proxy) GetSamplingTargets(s []*xraySvc.SamplingStatisticsDocument) (*xraySvc.GetSamplingTargetsOutput, error) {
	input := &xraySvc.GetSamplingTargetsInput{
		SamplingStatisticsDocuments: s,
	}

	output, err := p.xray.GetSamplingTargets(input)
	if err != nil {
		return nil, err
	}

	return output, nil
}

// GetSamplingRules calls the XRay daemon for sampling rules
func (p *proxy) GetSamplingRules() ([]*xraySvc.SamplingRuleRecord, error) {
	input := &xraySvc.GetSamplingRulesInput{}

	output, err := p.xray.GetSamplingRules(input)
	if err != nil {
		return nil, err
	}

	rules := output.SamplingRuleRecords

	return rules, nil
}
