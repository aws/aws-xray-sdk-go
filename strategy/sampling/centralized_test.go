// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package sampling

import (
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-xray-sdk-go/daemoncfg"

	xraySvc "github.com/aws/aws-sdk-go/service/xray"
	"github.com/aws/aws-xray-sdk-go/utils"
	"github.com/stretchr/testify/assert"
)

// Mock implementation of xray service proxy. Used for unit testing.
type mockProxy struct {
	samplingRules        []*xraySvc.SamplingRuleRecord
	samplingTargetOutput *xraySvc.GetSamplingTargetsOutput
}

func (p *mockProxy) GetSamplingRules() ([]*xraySvc.SamplingRuleRecord, error) {
	if p.samplingRules == nil {
		return nil, errors.New("Error encountered retrieving sampling rules")
	}

	return p.samplingRules, nil
}

func (p *mockProxy) GetSamplingTargets(s []*xraySvc.SamplingStatisticsDocument) (*xraySvc.GetSamplingTargetsOutput, error) {
	if p.samplingTargetOutput == nil {
		return nil, errors.New("Error encountered retrieving sampling targets")
	}

	targets := make([]*xraySvc.SamplingTargetDocument, 0, len(s))

	for _, s := range s {
		for _, t := range p.samplingTargetOutput.SamplingTargetDocuments {
			if *t.RuleName == *s.RuleName {
				targets = append(targets, t)
			}
		}
	}

	copy := *p.samplingTargetOutput
	copy.SamplingTargetDocuments = targets

	return &copy, nil
}

func getProperties(host string, method string, url string, serviceName string, rate float64, ft int) *Properties {
	return &Properties{
		Host:        host,
		HTTPMethod:  method,
		URLPath:     url,
		ServiceName: serviceName,
		Rate:        rate,
		FixedTarget: int64(ft),
	}
}

// Assert request matches against the correct sampling rule and gets sampled
func TestShouldTracePositive1(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	rand := &utils.MockRand{
		F64: 0.06,
	}

	host1 := "www.foo.com"
	method1 := "POST"
	url1 := "/resource/bar"
	serviceName1 := "localhost"
	servType1 := "AWS::EC2::Instance"

	sr := &Request{
		Host:        host1,
		URL:         url1,
		Method:      method1,
		ServiceName: serviceName1,
		ServiceType: servType1,
	}

	// Sampling rules with available quotas
	csr1 := &CentralizedRule{
		ruleName: "r1",
		reservoir: &CentralizedReservoir{
			quota:     10,
			expiresAt: 1500000050,
			reservoir: &reservoir{
				capacity:     50,
				used:         8,
				currentEpoch: 1500000000,
			},
		},
		Properties:  getProperties(host1, method1, url1, serviceName1, 0, 0),
		serviceType: servType1,
		clock:       clock,
		rand:        rand,
	}

	host2 := "www.bar.com"
	method2 := "POST"
	url2 := "/resource/foo"
	serviceName2 := ""

	csr2 := &CentralizedRule{
		ruleName: "r2",
		reservoir: &CentralizedReservoir{
			quota:     10,
			expiresAt: 1500000050,
			reservoir: &reservoir{
				capacity:     50,
				used:         8,
				currentEpoch: 1500000000,
			},
		},
		Properties: getProperties(host2, method2, url2, serviceName2, 0, 0),
		clock:      clock,
		rand:       rand,
	}

	rules := []*CentralizedRule{csr2, csr1}

	index := map[string]*CentralizedRule{
		"r1": csr1,
		"r2": csr2,
	}

	m := &CentralizedManifest{
		Rules:       rules,
		Index:       index,
		refreshedAt: 1500000000,
		clock:       clock,
	}

	s := &CentralizedStrategy{
		manifest: m,
		clock:    clock,
		rand:     rand,
	}

	// Make positive sampling decision against 'r1'
	sd := s.ShouldTrace(sr)

	assert.True(t, sd.Sample)
	assert.Equal(t, "r1", *sd.Rule)
	assert.Equal(t, int64(1), csr1.requests)
	assert.Equal(t, int64(1), csr1.sampled)
	assert.Equal(t, int64(9), csr1.reservoir.used)
}

// Assert request matches against the correct sampling rule and gets sampled
// ServiceType set to nil since not configured or passed in the request.
// r1 is matched because we do best effort matching
func TestShouldTracePositive2(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	rand := &utils.MockRand{
		F64: 0.06,
	}

	host1 := "www.foo.com"
	method1 := "POST"
	url1 := "/resource/bar"
	serviceName1 := "localhost"
	servType1 := "AWS::EC2::Instance"

	// serviceType missing
	sr := &Request{
		Host:        host1,
		URL:         url1,
		Method:      method1,
		ServiceName: serviceName1,
	}

	// Sampling rules with available quotas
	csr1 := &CentralizedRule{
		ruleName: "r1",
		reservoir: &CentralizedReservoir{
			quota:     10,
			expiresAt: 1500000050,
			reservoir: &reservoir{
				capacity:     50,
				used:         8,
				currentEpoch: 1500000000,
			},
		},
		Properties:  getProperties(host1, method1, url1, serviceName1, 0, 0),
		serviceType: servType1,
		clock:       clock,
		rand:        rand,
	}

	host2 := "www.bar.com"
	method2 := "POST"
	url2 := "/resource/foo"
	serviceName2 := ""

	csr2 := &CentralizedRule{
		ruleName: "r2",
		reservoir: &CentralizedReservoir{
			quota:     10,
			expiresAt: 1500000050,
			reservoir: &reservoir{
				capacity:     50,
				used:         8,
				currentEpoch: 1500000000,
			},
		},
		Properties: getProperties(host2, method2, url2, serviceName2, 0, 0),
		clock:      clock,
		rand:       rand,
	}

	rules := []*CentralizedRule{csr2, csr1}

	index := map[string]*CentralizedRule{
		"r1": csr1,
		"r2": csr2,
	}

	m := &CentralizedManifest{
		Rules:       rules,
		Index:       index,
		refreshedAt: 1500000000,
		clock:       clock,
	}

	strategy, _ := NewLocalizedStrategy()
	s := &CentralizedStrategy{
		manifest: m,
		clock:    clock,
		rand:     rand,
		fallback: strategy,
	}

	// Make positive sampling decision against 'r1'
	sd := s.ShouldTrace(sr)

	assert.True(t, sd.Sample)
	assert.Equal(t, "r1", *sd.Rule)
	assert.Equal(t, int64(1), csr1.requests)
	assert.Equal(t, int64(1), csr1.sampled)
	assert.Equal(t, int64(9), csr1.reservoir.used)
}

// Assert request matches against the default sampling rule and gets sampled
func TestShouldTraceDefaultPositive(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	rand := &utils.MockRand{
		F64: 0.06,
	}

	// Sampling rule with available quota
	csr := &CentralizedRule{
		ruleName: "r1",
		reservoir: &CentralizedReservoir{
			quota:     10,
			expiresAt: 1500000050,
			reservoir: &reservoir{
				capacity:     50,
				used:         8,
				currentEpoch: 1500000000,
			},
		},
		Properties: getProperties("www.foo.com", "POST", "/resource/bar", "", 0, 0),
		clock:      clock,
		rand:       rand,
	}

	// Default sampling rule
	def := &CentralizedRule{
		ruleName: "Default",
		reservoir: &CentralizedReservoir{
			quota:     10,
			expiresAt: 1500000050,
			reservoir: &reservoir{
				capacity:     50,
				used:         8,
				currentEpoch: 1500000000,
			},
		},
		clock: clock,
		rand:  rand,
	}

	rules := []*CentralizedRule{csr}

	index := map[string]*CentralizedRule{
		"r1": csr,
	}

	m := &CentralizedManifest{
		Default:     def,
		Rules:       rules,
		Index:       index,
		refreshedAt: 1500000000,
		clock:       clock,
	}

	s := &CentralizedStrategy{
		manifest: m,
		clock:    clock,
		rand:     rand,
	}

	sr := &Request{
		Host:   "www.foo.bar.com",
		URL:    "/resource/bat",
		Method: "GET",
	}

	// Make positive sampling decision against 'Default' rule
	sd := s.ShouldTrace(sr)

	// Assert 'Default' rule was used
	assert.True(t, sd.Sample)
	assert.Equal(t, "Default", *sd.Rule)
	assert.Equal(t, int64(1), m.Default.requests)
	assert.Equal(t, int64(1), m.Default.sampled)
	assert.Equal(t, int64(9), m.Default.reservoir.used)

	// Assert 'r1' was not used
	assert.Equal(t, int64(0), csr.requests)
	assert.Equal(t, int64(0), csr.sampled)
	assert.Equal(t, int64(8), csr.reservoir.used)
}

// Assert fallback strategy was used for expired manifest
func TestShouldTraceExpiredManifest(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500003601,
	}

	rand := &utils.MockRand{
		F64: 0.05,
	}

	// Sampling rule with available quota
	csr := &CentralizedRule{
		ruleName: "r1",
		reservoir: &CentralizedReservoir{
			quota:     10,
			expiresAt: 1500000050,
			reservoir: &reservoir{
				capacity:     50,
				used:         8,
				currentEpoch: 1500000000,
			},
		},
		Properties: getProperties("www.foo.com", "POST", "/resource/bar", "", 0, 0),
		clock:      clock,
		rand:       rand,
	}

	rules := []*CentralizedRule{csr}

	index := map[string]*CentralizedRule{
		"r1": csr,
	}

	centralManifest := &CentralizedManifest{
		Rules:       rules,
		Index:       index,
		refreshedAt: 1500000000,
		clock:       clock,
	}

	// Local Manifest with Default rule with available reservoir.
	defaultRule := &Rule{
		reservoir: &Reservoir{
			clock: clock,
			reservoir: &reservoir{
				capacity:     int64(10),
				used:         int64(4),
				currentEpoch: int64(1500003601),
			},
		},
		Properties: &Properties{
			FixedTarget: int64(10),
			Rate:        float64(0.05),
		},
	}
	localManifest := &RuleManifest{
		Version: 1,
		Default: defaultRule,
		Rules:   []*Rule{},
	}

	fb := &LocalizedStrategy{
		manifest: localManifest,
	}

	s := &CentralizedStrategy{
		manifest: centralManifest,
		fallback: fb,
		clock:    clock,
		rand:     rand,
	}

	sr := &Request{
		Host:   "www.foo.bar.com",
		URL:    "/resource/bar",
		Method: "POST",
	}

	// Fallback to local sampling strategy and make positive decision
	sd := s.ShouldTrace(sr)

	// Assert fallback 'Default' rule was sampled
	assert.True(t, sd.Sample)
	assert.Nil(t, sd.Rule)

	// Assert 'r1' was not used
	assert.Equal(t, int64(0), csr.requests)
	assert.Equal(t, int64(0), csr.sampled)
	assert.Equal(t, int64(8), csr.reservoir.used)
}

// Assert that snapshots returns an array of valid sampling statistics
func TestSnapshots(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	id := "c1"
	time := clock.Now()

	name1 := "r1"
	requests1 := int64(1000)
	sampled1 := int64(100)
	borrows1 := int64(5)
	r1 := &CentralizedReservoir{
		interval: 10,
	}
	csr1 := &CentralizedRule{
		ruleName:  name1,
		requests:  requests1,
		sampled:   sampled1,
		borrows:   borrows1,
		usedAt:    1500000000,
		reservoir: r1,
		clock:     clock,
	}

	name2 := "r2"
	requests2 := int64(500)
	sampled2 := int64(10)
	borrows2 := int64(0)
	r2 := &CentralizedReservoir{
		interval: 10,
	}
	csr2 := &CentralizedRule{
		ruleName:  name2,
		requests:  requests2,
		sampled:   sampled2,
		borrows:   borrows2,
		usedAt:    1500000000,
		reservoir: r2,
		clock:     clock,
	}

	rules := []*CentralizedRule{csr1, csr2}

	m := &CentralizedManifest{
		Rules: rules,
	}

	strategy := &CentralizedStrategy{
		manifest: m,
		clientID: id,
		clock:    clock,
	}

	// Expected SamplingStatistics structs
	ss1 := xraySvc.SamplingStatisticsDocument{
		ClientID:     &id,
		RequestCount: &requests1,
		RuleName:     &name1,
		SampledCount: &sampled1,
		BorrowCount:  &borrows1,
		Timestamp:    &time,
	}

	ss2 := xraySvc.SamplingStatisticsDocument{
		ClientID:     &id,
		RequestCount: &requests2,
		RuleName:     &name2,
		SampledCount: &sampled2,
		BorrowCount:  &borrows2,
		Timestamp:    &time,
	}

	statistics := strategy.snapshots()

	assert.Equal(t, ss1, *statistics[0])
	assert.Equal(t, ss2, *statistics[1])
}

// Assert that fresh and inactive rules are not included in a snapshot
func TestMixedSnapshots(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	id := "c1"
	time := clock.Now()

	// Stale and active rule
	name1 := "r1"
	requests1 := int64(1000)
	sampled1 := int64(100)
	borrows1 := int64(5)
	r1 := &CentralizedReservoir{
		interval:    20,
		refreshedAt: 1499999980,
	}
	csr1 := &CentralizedRule{
		ruleName:  name1,
		requests:  requests1,
		sampled:   sampled1,
		borrows:   borrows1,
		usedAt:    1499999999,
		reservoir: r1,
		clock:     clock,
	}

	// Stale and inactive rule
	name2 := "r2"
	requests2 := int64(0)
	sampled2 := int64(0)
	borrows2 := int64(0)
	r2 := &CentralizedReservoir{
		interval:    20,
		refreshedAt: 1499999970,
	}
	csr2 := &CentralizedRule{
		ruleName:  name2,
		requests:  requests2,
		sampled:   sampled2,
		borrows:   borrows2,
		usedAt:    1499999999,
		reservoir: r2,
		clock:     clock,
	}

	// Fresh rule
	name3 := "r3"
	requests3 := int64(1000)
	sampled3 := int64(100)
	borrows3 := int64(5)
	r3 := &CentralizedReservoir{
		interval:    20,
		refreshedAt: 1499999990,
	}
	csr3 := &CentralizedRule{
		ruleName:  name3,
		requests:  requests3,
		sampled:   sampled3,
		borrows:   borrows3,
		usedAt:    1499999999,
		reservoir: r3,
		clock:     clock,
	}

	rules := []*CentralizedRule{csr1, csr2, csr3}

	m := &CentralizedManifest{
		Rules: rules,
	}

	strategy := &CentralizedStrategy{
		manifest: m,
		clientID: id,
		clock:    clock,
	}

	// Expected SamplingStatistics structs
	ss1 := xraySvc.SamplingStatisticsDocument{
		ClientID:     &id,
		RequestCount: &requests1,
		RuleName:     &name1,
		SampledCount: &sampled1,
		BorrowCount:  &borrows1,
		Timestamp:    &time,
	}

	statistics := strategy.snapshots()

	assert.Equal(t, 1, len(statistics))
	assert.Equal(t, ss1, *statistics[0])
}

// Assert that a valid sampling target updates its rule
func TestUpdateTarget(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	// Sampling target received from centralized sampling backend
	rate := float64(0.05)
	quota := int64(10)
	ttl := time.Unix(1500000060, 0)
	name := "r1"
	st := &xraySvc.SamplingTargetDocument{
		FixedRate:         &rate,
		ReservoirQuota:    &quota,
		ReservoirQuotaTTL: &ttl,
		RuleName:          &name,
	}

	// Sampling rule about to be updated with new target
	csr := &CentralizedRule{
		ruleName: "r1",
		Properties: &Properties{
			Rate: 0.10,
		},
		reservoir: &CentralizedReservoir{
			quota:       8,
			refreshedAt: 1499999990,
			expiresAt:   1500000010,
			reservoir: &reservoir{
				capacity:     50,
				used:         7,
				currentEpoch: 1500000000,
			},
		},
	}

	rules := []*CentralizedRule{csr}

	index := map[string]*CentralizedRule{
		"r1": csr,
	}

	m := &CentralizedManifest{
		Rules: rules,
		Index: index,
	}

	s := &CentralizedStrategy{
		manifest: m,
		clock:    clock,
	}

	err := s.updateTarget(st)
	assert.Nil(t, err)

	// Updated sampling rule
	exp := &CentralizedRule{
		ruleName: "r1",
		Properties: &Properties{
			Rate: 0.05,
		},
		reservoir: &CentralizedReservoir{
			quota:       10,
			refreshedAt: 1500000000,
			expiresAt:   1500000060,
			reservoir: &reservoir{
				capacity:     50,
				used:         7,
				currentEpoch: 1500000000,
			},
		},
	}

	act := s.manifest.Rules[0]

	assert.Equal(t, exp, act)
}

// Assert that a missing sampling rule returns an error
func TestUpdateTargetMissingRule(t *testing.T) {
	// Sampling target received from centralized sampling backend
	rate := float64(0.05)
	quota := int64(10)
	ttl := time.Unix(1500000060, 0)
	name := "r1"
	st := &xraySvc.SamplingTargetDocument{
		FixedRate:         &rate,
		ReservoirQuota:    &quota,
		ReservoirQuotaTTL: &ttl,
		RuleName:          &name,
	}

	rules := []*CentralizedRule{}

	index := map[string]*CentralizedRule{}

	m := &CentralizedManifest{
		Rules: rules,
		Index: index,
	}

	s := &CentralizedStrategy{
		manifest: m,
	}

	err := s.updateTarget(st)
	assert.NotNil(t, err)
}

// Assert that an invalid sampling target returns an error and does not panic
func TestUpdateTargetPanicRecovery(t *testing.T) {
	// Invalid sampling target missing FixedRate.
	quota := int64(10)
	ttl := time.Unix(1500000060, 0)
	name := "r1"
	st := &xraySvc.SamplingTargetDocument{
		ReservoirQuota:    &quota,
		ReservoirQuotaTTL: &ttl,
		RuleName:          &name,
	}

	// Sampling rule about to be updated with new target
	csr := &CentralizedRule{
		ruleName: "r1",
		Properties: &Properties{
			Rate: 0.10,
		},
		reservoir: &CentralizedReservoir{
			quota:     8,
			expiresAt: 1500000010,
			reservoir: &reservoir{
				capacity:     50,
				used:         7,
				currentEpoch: 1500000000,
			},
		},
	}

	rules := []*CentralizedRule{csr}

	index := map[string]*CentralizedRule{
		"r1": csr,
	}

	m := &CentralizedManifest{
		Rules: rules,
		Index: index,
	}

	s := &CentralizedStrategy{
		manifest: m,
	}

	err := s.updateTarget(st)
	assert.NotNil(t, err)

	// Unchanged sampling rule
	exp := &CentralizedRule{
		ruleName: "r1",
		Properties: &Properties{
			Rate: 0.10,
		},
		reservoir: &CentralizedReservoir{
			quota:     8,
			expiresAt: 1500000010,
			reservoir: &reservoir{
				capacity:     50,
				used:         7,
				currentEpoch: 1500000000,
			},
		},
	}

	act := s.manifest.Rules[0]

	assert.Equal(t, exp, act)
}

// Assert that manifest refresh updates the manifest and leaves it in a
// consistent state.
func TestRefreshManifestRuleAddition(t *testing.T) {
	serviceTye := ""
	resourceARN := "*"
	// Rule 'r1'
	r1 := &CentralizedRule{
		ruleName: "r1",
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 50,
			},
		},
		Properties:  &Properties{},
		priority:    4,
		resourceARN: resourceARN,
	}

	// Rule 'r3'
	r3 := &CentralizedRule{
		ruleName: "r3",
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 50,
			},
		},
		Properties: &Properties{
			Host:        "www.bar.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/foo",
			FixedTarget: 40,
			Rate:        0.10,
			ServiceName: "www.bar.com",
		},
		priority:    8,
		resourceARN: resourceARN,
	}

	// Sorted array
	rules := []*CentralizedRule{r1, r3}

	index := map[string]*CentralizedRule{
		"r1": r1,
		"r3": r3,
	}

	manifest := &CentralizedManifest{
		Rules:       rules,
		Index:       index,
		refreshedAt: 1500000000,
	}

	// Valid no-op update for rule 'r1'
	name1 := "r1"
	fixedRate1 := 0.05
	httpMethod1 := "POST"
	priority1 := int64(4)
	reservoirSize1 := int64(50)
	serviceName1 := "www.foo.com"
	urlPath1 := "/resource/bar"
	version1 := int64(1)
	u1 := &xraySvc.SamplingRuleRecord{
		SamplingRule: &xraySvc.SamplingRule{
			RuleName:      &name1,
			ServiceName:   &serviceName1,
			URLPath:       &urlPath1,
			HTTPMethod:    &httpMethod1,
			Priority:      &priority1,
			ReservoirSize: &reservoirSize1,
			FixedRate:     &fixedRate1,
			Version:       &version1,
			Host:          &serviceName1,
			ServiceType:   &serviceTye,
			ResourceARN:   &resourceARN,
		},
	}

	// New valid rule 'r2'
	name2 := "r2"
	fixedRate2 := 0.04
	httpMethod2 := "PUT"
	priority2 := int64(5)
	reservoirSize2 := int64(60)
	serviceName2 := "www.fizz.com"
	urlPath2 := "/resource/fizz"
	version2 := int64(1)
	u2 := &xraySvc.SamplingRuleRecord{
		SamplingRule: &xraySvc.SamplingRule{
			RuleName:      &name2,
			ServiceName:   &serviceName2,
			URLPath:       &urlPath2,
			HTTPMethod:    &httpMethod2,
			Priority:      &priority2,
			ReservoirSize: &reservoirSize2,
			FixedRate:     &fixedRate2,
			Version:       &version2,
			Host:          &serviceName2,
			ServiceType:   &serviceTye,
			ResourceARN:   &resourceARN,
		},
	}

	// Valid no-op update for rule 'r3'
	name3 := "r3"
	fixedRate3 := 0.10
	httpMethod3 := "POST"
	priority3 := int64(8)
	reservoirSize3 := int64(40)
	serviceName3 := "www.bar.com"
	urlPath3 := "/resource/foo"
	version3 := int64(1)
	u3 := &xraySvc.SamplingRuleRecord{
		SamplingRule: &xraySvc.SamplingRule{
			RuleName:      &name3,
			ServiceName:   &serviceName3,
			URLPath:       &urlPath3,
			HTTPMethod:    &httpMethod3,
			Priority:      &priority3,
			ReservoirSize: &reservoirSize3,
			FixedRate:     &fixedRate3,
			Version:       &version3,
			Host:          &serviceName3,
			ServiceType:   &serviceTye,
			ResourceARN:   &resourceARN,
		},
	}

	// Mock proxy with updates u1, u2, and u3
	proxy := &mockProxy{
		samplingRules: []*xraySvc.SamplingRuleRecord{u1, u2, u3},
	}

	// Mock clock with time incremented to 60 seconds past current
	// manifest refreshedAt timestamp.
	clock := &utils.MockClock{
		NowTime: 1500000060,
	}

	ss := &CentralizedStrategy{
		manifest: manifest,
		proxy:    proxy,
		clock:    clock,
	}

	// Refresh manifest with updates from mock proxy
	err := ss.refreshManifest()
	assert.Nil(t, err)

	// Expected 'r2'
	r2 := &CentralizedRule{
		ruleName: "r2",
		reservoir: &CentralizedReservoir{
			reservoir: &reservoir{
				capacity: 60,
			},
			interval: 10,
		},
		Properties: &Properties{
			Host:        "www.fizz.com",
			HTTPMethod:  "PUT",
			URLPath:     "/resource/fizz",
			FixedTarget: 60,
			Rate:        0.04,
			ServiceName: "www.fizz.com",
		},
		priority:    5,
		clock:       &utils.DefaultClock{},
		rand:        &utils.DefaultRand{},
		resourceARN: resourceARN,
	}

	// Assert on addition of new rule
	assert.Equal(t, r2, ss.manifest.Index["r2"])
	assert.Equal(t, r2, ss.manifest.Rules[1])

	// Assert on sorting order
	assert.Equal(t, r1, ss.manifest.Rules[0])
	assert.Equal(t, r2, ss.manifest.Rules[1])
	assert.Equal(t, r3, ss.manifest.Rules[2])

	// Assert on size of manifest
	assert.Equal(t, 3, len(ss.manifest.Rules))
	assert.Equal(t, 3, len(ss.manifest.Index))

	// Assert on refreshedAt timestamp
	assert.Equal(t, int64(1500000060), ss.manifest.refreshedAt)
}

func TestRefreshManifestRuleAdditionInvalidRule1(t *testing.T) { // ResourceARN has invalid value
	serviceTye := ""
	resourceARN := "XYZ" // invalid
	// Rule 'r1'
	r1 := &CentralizedRule{
		ruleName: "r1",
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 50,
			},
		},
		Properties:  &Properties{},
		priority:    4,
		resourceARN: resourceARN,
	}

	// Sorted array
	rules := []*CentralizedRule{r1}

	index := map[string]*CentralizedRule{
		"r1": r1,
	}

	manifest := &CentralizedManifest{
		Rules:       rules,
		Index:       index,
		refreshedAt: 1500000000,
	}

	// Valid no-op update for rule 'r1'
	name1 := "r1"
	fixedRate1 := 0.05
	httpMethod1 := "POST"
	priority1 := int64(4)
	reservoirSize1 := int64(50)
	serviceName1 := "www.foo.com"
	urlPath1 := "/resource/bar"
	version1 := int64(1)
	u1 := &xraySvc.SamplingRuleRecord{
		SamplingRule: &xraySvc.SamplingRule{
			RuleName:      &name1,
			ServiceName:   &serviceName1,
			URLPath:       &urlPath1,
			HTTPMethod:    &httpMethod1,
			Priority:      &priority1,
			ReservoirSize: &reservoirSize1,
			FixedRate:     &fixedRate1,
			Version:       &version1,
			Host:          &serviceName1,
			ServiceType:   &serviceTye,
			ResourceARN:   &resourceARN,
		},
	}

	// Mock proxy with updates u1
	proxy := &mockProxy{
		samplingRules: []*xraySvc.SamplingRuleRecord{u1},
	}

	// Mock clock with time incremented to 60 seconds past current
	// manifest refreshedAt timestamp.
	clock := &utils.MockClock{
		NowTime: 1500000060,
	}

	ss := &CentralizedStrategy{
		manifest: manifest,
		proxy:    proxy,
		clock:    clock,
	}
	err := ss.refreshManifest()
	assert.Nil(t, err)
	// Refresh manifest with updates from mock proxy
	assert.Equal(t, 0, len(ss.manifest.Rules)) // Rule not added
}

func TestRefreshManifestRuleAdditionInvalidRule2(t *testing.T) { // non nil Attributes
	serviceTye := ""
	resourceARN := "*"
	attributes := make(map[string]*string)
	attributes["a"] = &resourceARN

	// Rule 'r1'
	r1 := &CentralizedRule{
		ruleName: "r1",
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 50,
			},
		},
		Properties:  &Properties{},
		priority:    4,
		resourceARN: resourceARN,
	}

	// Sorted array
	rules := []*CentralizedRule{r1}

	index := map[string]*CentralizedRule{
		"r1": r1,
	}

	manifest := &CentralizedManifest{
		Rules:       rules,
		Index:       index,
		refreshedAt: 1500000000,
	}

	// Valid no-op update for rule 'r1'
	name1 := "r1"
	fixedRate1 := 0.05
	httpMethod1 := "POST"
	priority1 := int64(4)
	reservoirSize1 := int64(50)
	serviceName1 := "www.foo.com"
	urlPath1 := "/resource/bar"
	version1 := int64(1)
	u1 := &xraySvc.SamplingRuleRecord{
		SamplingRule: &xraySvc.SamplingRule{
			RuleName:      &name1,
			ServiceName:   &serviceName1,
			URLPath:       &urlPath1,
			HTTPMethod:    &httpMethod1,
			Priority:      &priority1,
			ReservoirSize: &reservoirSize1,
			FixedRate:     &fixedRate1,
			Version:       &version1,
			Host:          &serviceName1,
			ServiceType:   &serviceTye,
			ResourceARN:   &resourceARN,
			Attributes:    attributes, // invalid
		},
	}

	// Mock proxy with updates u1
	proxy := &mockProxy{
		samplingRules: []*xraySvc.SamplingRuleRecord{u1},
	}

	// Mock clock with time incremented to 60 seconds past current
	// manifest refreshedAt timestamp.
	clock := &utils.MockClock{
		NowTime: 1500000060,
	}

	ss := &CentralizedStrategy{
		manifest: manifest,
		proxy:    proxy,
		clock:    clock,
	}

	err := ss.refreshManifest()
	assert.Nil(t, err)
	assert.Equal(t, 0, len(ss.manifest.Rules)) // rule not added
}

func TestRefreshManifestRuleAdditionInvalidRule3(t *testing.T) { // 1 valid and 1 invalid rule
	serviceTye := ""
	resourceARN := "*"
	attributes := make(map[string]*string)
	attributes["a"] = &resourceARN

	// Rule 'r1'
	r1 := &CentralizedRule{
		ruleName: "r1",
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 50,
			},
		},
		Properties:  &Properties{},
		priority:    4,
		resourceARN: resourceARN,
	}

	r2 := &CentralizedRule{
		ruleName: "r2",
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 50,
			},
		},
		Properties:  &Properties{},
		priority:    4,
		resourceARN: resourceARN,
	}

	// Sorted array
	rules := []*CentralizedRule{r1}

	index := map[string]*CentralizedRule{
		"r1": r1,
	}

	manifest := &CentralizedManifest{
		Rules:       rules,
		Index:       index,
		refreshedAt: 1500000000,
	}

	// Valid no-op update for rule 'r1'
	name1 := "r1"
	fixedRate1 := 0.05
	httpMethod1 := "POST"
	priority1 := int64(4)
	reservoirSize1 := int64(50)
	serviceName1 := "www.foo.com"
	urlPath1 := "/resource/bar"
	version1 := int64(1)
	u1 := &xraySvc.SamplingRuleRecord{
		SamplingRule: &xraySvc.SamplingRule{
			RuleName:      &name1,
			ServiceName:   &serviceName1,
			URLPath:       &urlPath1,
			HTTPMethod:    &httpMethod1,
			Priority:      &priority1,
			ReservoirSize: &reservoirSize1,
			FixedRate:     &fixedRate1,
			Version:       &version1,
			Host:          &serviceName1,
			ServiceType:   &serviceTye,
			ResourceARN:   &resourceARN,
			Attributes:    attributes, // invalid
		},
	}

	name2 := "r2"
	u2 := &xraySvc.SamplingRuleRecord{ // valid rule
		SamplingRule: &xraySvc.SamplingRule{
			RuleName:      &name2,
			ServiceName:   &serviceName1,
			URLPath:       &urlPath1,
			HTTPMethod:    &httpMethod1,
			Priority:      &priority1,
			ReservoirSize: &reservoirSize1,
			FixedRate:     &fixedRate1,
			Version:       &version1,
			Host:          &serviceName1,
			ServiceType:   &serviceTye,
			ResourceARN:   &resourceARN,
		},
	}

	// Mock proxy with updates u1
	proxy := &mockProxy{
		samplingRules: []*xraySvc.SamplingRuleRecord{u1, u2},
	}

	// Mock clock with time incremented to 60 seconds past current
	// manifest refreshedAt timestamp.
	clock := &utils.MockClock{
		NowTime: 1500000060,
	}

	ss := &CentralizedStrategy{
		manifest: manifest,
		proxy:    proxy,
		clock:    clock,
	}

	err := ss.refreshManifest()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(ss.manifest.Rules)) // u1 not added
	assert.Equal(t, r2.ruleName, ss.manifest.Rules[0].ruleName)
	// Assert on refreshedAt timestamp
	assert.Equal(t, int64(1500000060), ss.manifest.refreshedAt)
}

// Assert that rules missing from GetSamplingRules are pruned
func TestRefreshManifestRuleRemoval(t *testing.T) {
	resARN := "*"
	serviceTye := ""
	attributes := make(map[string]*string)
	// Rule 'r1'
	r1 := &CentralizedRule{
		ruleName: "r1",
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 50,
			},
		},
		Properties: &Properties{
			ServiceName: "www.foo.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/bar",
			FixedTarget: 50,
			Rate:        0.05,
		},
		priority:    4,
		resourceARN: resARN,
	}

	// Rule 'r2'
	r2 := &CentralizedRule{
		ruleName: "r2",
		reservoir: &CentralizedReservoir{
			quota: 20,
			reservoir: &reservoir{
				capacity: 60,
			},
		},
		Properties: &Properties{
			ServiceName: "www.fizz.com",
			HTTPMethod:  "PUT",
			URLPath:     "/resource/fizz",
			FixedTarget: 60,
			Rate:        0.04,
		},
		priority:    5,
		resourceARN: resARN,
	}

	// Rule 'r3'
	r3 := &CentralizedRule{
		ruleName: "r3",
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 50,
			},
		},
		Properties: &Properties{
			ServiceName: "www.bar.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/foo",
			FixedTarget: 40,
			Rate:        0.10,
		},
		priority:    8,
		resourceARN: resARN,
	}

	// Sorted array
	rules := []*CentralizedRule{r1, r2, r3}

	index := map[string]*CentralizedRule{
		"r1": r1,
		"r2": r2,
		"r3": r3,
	}

	manifest := &CentralizedManifest{
		Rules:       rules,
		Index:       index,
		refreshedAt: 1500000000,
	}

	// Valid no-op update for rule 'r1'
	name1 := "r1"
	fixedRate1 := 0.05
	httpMethod1 := "POST"
	priority1 := int64(4)
	reservoirSize1 := int64(50)
	serviceName1 := "www.foo.com"
	urlPath1 := "/resource/bar"
	version1 := int64(1)
	host1 := "h1"
	u1 := &xraySvc.SamplingRuleRecord{
		SamplingRule: &xraySvc.SamplingRule{
			RuleName:      &name1,
			ServiceName:   &serviceName1,
			URLPath:       &urlPath1,
			HTTPMethod:    &httpMethod1,
			Priority:      &priority1,
			ReservoirSize: &reservoirSize1,
			FixedRate:     &fixedRate1,
			Version:       &version1,
			ResourceARN:   &resARN,
			Host:          &host1,
			ServiceType:   &serviceTye,
			Attributes:    attributes,
		},
	}

	// Rule 'r2' is missing from GetSamplingRules response

	// Valid no-op update for rule 'r3'
	name3 := "r3"
	fixedRate3 := 0.10
	httpMethod3 := "POST"
	priority3 := int64(8)
	reservoirSize3 := int64(40)
	serviceName3 := "www.bar.com"
	urlPath3 := "/resource/foo"
	version3 := int64(1)
	host3 := "h3"
	u3 := &xraySvc.SamplingRuleRecord{
		SamplingRule: &xraySvc.SamplingRule{
			RuleName:      &name3,
			ServiceName:   &serviceName3,
			URLPath:       &urlPath3,
			HTTPMethod:    &httpMethod3,
			Priority:      &priority3,
			ReservoirSize: &reservoirSize3,
			FixedRate:     &fixedRate3,
			Version:       &version3,
			ResourceARN:   &resARN,
			Host:          &host3,
			ServiceType:   &serviceTye,
			Attributes:    attributes,
		},
	}

	// Mock proxy with updates u1 and u3
	proxy := &mockProxy{
		samplingRules: []*xraySvc.SamplingRuleRecord{u1, u3},
	}

	// Mock clock with time incremented to 60 seconds past current
	// manifest refreshedAt timestamp.
	clock := &utils.MockClock{
		NowTime: 1500000060,
	}

	ss := &CentralizedStrategy{
		manifest: manifest,
		proxy:    proxy,
		clock:    clock,
	}

	// Refresh manifest with updates from mock proxy
	err := ss.refreshManifest()
	assert.Nil(t, err)

	// Assert on removal of rule
	assert.Equal(t, 2, len(ss.manifest.Rules))
	assert.Equal(t, 2, len(ss.manifest.Index))

	// Assert on sorting order
	assert.Equal(t, r1, ss.manifest.Rules[0])
	assert.Equal(t, r3, ss.manifest.Rules[1])

	// Assert on refreshedAt timestamp
	assert.Equal(t, int64(1500000060), ss.manifest.refreshedAt)
}

// Assert that an invalid rule update does not update the rule
func TestRefreshManifestInvalidRuleUpdate(t *testing.T) {
	resARN := "*"
	serviceTye := ""
	attributes := make(map[string]*string)
	// Rule 'r1'
	r1 := &CentralizedRule{
		ruleName: "r1",
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 50,
			},
		},
		Properties: &Properties{
			ServiceName: "www.foo.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/bar",
			FixedTarget: 50,
			Rate:        0.05,
		},
		priority:    4,
		resourceARN: resARN,
	}

	h3 := "h3"
	// Rule 'r3'
	r3 := &CentralizedRule{
		ruleName: "r3",
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 50,
			},
		},
		Properties: &Properties{
			ServiceName: "www.bar.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/foo",
			FixedTarget: 40,
			Rate:        0.10,
			Host:        h3,
		},
		priority:    8,
		resourceARN: resARN,
	}

	// Sorted array
	rules := []*CentralizedRule{r1, r3}

	index := map[string]*CentralizedRule{
		"r1": r1,
		"r3": r3,
	}

	manifest := &CentralizedManifest{
		Rules:       rules,
		Index:       index,
		refreshedAt: 1500000000,
	}

	// Invalid update for rule 'r1' (missing fixedRate)
	name1 := "r1"
	httpMethod1 := "GET"
	priority1 := int64(9)
	reservoirSize1 := int64(50)
	serviceName1 := "www.foo.com"
	urlPath1 := "/resource/bar"
	version1 := int64(1)
	u1 := &xraySvc.SamplingRuleRecord{
		SamplingRule: &xraySvc.SamplingRule{
			RuleName:      &name1,
			ServiceName:   &serviceName1,
			URLPath:       &urlPath1,
			HTTPMethod:    &httpMethod1,
			Priority:      &priority1,
			ReservoirSize: &reservoirSize1,
			Version:       &version1,
			ResourceARN:   &resARN,
			ServiceType:   &serviceTye,
			Attributes:    attributes,
		},
	}

	// Valid update for rule 'r3'
	name3 := "r3"
	fixedRate3 := 0.10
	httpMethod3 := "POST"
	priority3 := int64(8)
	reservoirSize3 := int64(40)
	serviceName3 := "www.bar.com"
	urlPath3 := "/resource/foo"
	version3 := int64(1)
	u3 := &xraySvc.SamplingRuleRecord{
		SamplingRule: &xraySvc.SamplingRule{
			RuleName:      &name3,
			ServiceName:   &serviceName3,
			URLPath:       &urlPath3,
			HTTPMethod:    &httpMethod3,
			Priority:      &priority3,
			ReservoirSize: &reservoirSize3,
			FixedRate:     &fixedRate3,
			Version:       &version3,
			ResourceARN:   &resARN,
			Host:          &h3,
			ServiceType:   &serviceTye,
			Attributes:    attributes,
		},
	}

	// Mock proxy with updates u1 and u3
	proxy := &mockProxy{
		samplingRules: []*xraySvc.SamplingRuleRecord{u1, u3},
	}

	// Mock clock with time incremented to 60 seconds past current
	// manifest refreshedAt timestamp.
	clock := &utils.MockClock{
		NowTime: 1500000060,
	}

	ss := &CentralizedStrategy{
		manifest: manifest,
		proxy:    proxy,
		clock:    clock,
	}

	// Refresh manifest with updates from mock proxy
	err := ss.refreshManifest()
	assert.NotNil(t, err)

	// Assert on size of manifest
	assert.Equal(t, 1, len(ss.manifest.Rules))
	assert.Equal(t, 1, len(ss.manifest.Index))

	// Assert on sorting order
	assert.Equal(t, r3, ss.manifest.Rules[0])

	// Assert on index consistency
	assert.Equal(t, r3, ss.manifest.Index["r3"])

	// Assert on refreshedAt timestamp not changing
	assert.Equal(t, int64(1500000060), ss.manifest.refreshedAt)
}

// Assert that a new invalid rule does not get added to manifest
func TestRefreshManifestInvalidNewRule(t *testing.T) {
	resARN := "*"
	h := "h"
	serviceTye := ""
	// Rule 'r1'
	r1 := &CentralizedRule{
		ruleName: "r1",
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 40,
			},
		},
		Properties: &Properties{
			ServiceName: "www.foo.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/bar",
			FixedTarget: 40,
			Rate:        0.05,
			Host:        h,
		},
		priority:    4,
		resourceARN: resARN,
	}

	// Rule 'r3'
	r3 := &CentralizedRule{
		ruleName: "r3",
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 50,
			},
		},
		Properties: &Properties{
			ServiceName: "www.bar.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/foo",
			FixedTarget: 50,
			Rate:        0.10,
			Host:        h,
		},
		priority:    8,
		resourceARN: resARN,
	}

	// Sorted array
	rules := []*CentralizedRule{r1, r3}

	index := map[string]*CentralizedRule{
		"r1": r1,
		"r3": r3,
	}

	manifest := &CentralizedManifest{
		Rules:       rules,
		Index:       index,
		refreshedAt: 1500000000,
	}

	// Valid no-op update for rule 'r1'
	name1 := "r1"
	fixedRate1 := 0.05
	httpMethod1 := "POST"
	priority1 := int64(4)
	reservoirSize1 := int64(50)
	serviceName1 := "www.foo.com"
	urlPath1 := "/resource/bar"
	version1 := int64(1)
	u1 := &xraySvc.SamplingRuleRecord{
		SamplingRule: &xraySvc.SamplingRule{
			RuleName:      &name1,
			ServiceName:   &serviceName1,
			URLPath:       &urlPath1,
			HTTPMethod:    &httpMethod1,
			Priority:      &priority1,
			ReservoirSize: &reservoirSize1,
			FixedRate:     &fixedRate1,
			Version:       &version1,
			ResourceARN:   &resARN,
			Host:          &h,
			ServiceType:   &serviceTye,
		},
	}

	// New Invalid rule 'r2' (missing priority)
	name2 := "r2"
	fixedRate2 := 0.04
	httpMethod2 := "PUT"
	reservoirSize2 := int64(60)
	serviceName2 := "www.fizz.com"
	urlPath2 := "/resource/fizz"
	version2 := int64(1)
	u2 := &xraySvc.SamplingRuleRecord{
		SamplingRule: &xraySvc.SamplingRule{
			RuleName:      &name2,
			ServiceName:   &serviceName2,
			URLPath:       &urlPath2,
			HTTPMethod:    &httpMethod2,
			ReservoirSize: &reservoirSize2,
			FixedRate:     &fixedRate2,
			Version:       &version2,
			ResourceARN:   &resARN,
		},
	}

	// Valid no-op update for rule 'r3'
	name3 := "r3"
	fixedRate3 := 0.10
	httpMethod3 := "POST"
	priority3 := int64(8)
	reservoirSize3 := int64(40)
	serviceName3 := "www.bar.com"
	urlPath3 := "/resource/foo"
	version3 := int64(1)
	u3 := &xraySvc.SamplingRuleRecord{
		SamplingRule: &xraySvc.SamplingRule{
			RuleName:      &name3,
			ServiceName:   &serviceName3,
			URLPath:       &urlPath3,
			HTTPMethod:    &httpMethod3,
			Priority:      &priority3,
			ReservoirSize: &reservoirSize3,
			FixedRate:     &fixedRate3,
			Version:       &version3,
			ResourceARN:   &resARN,
			Host:          &h,
			ServiceType:   &serviceTye,
		},
	}

	// New Invalid rule 'r4' (missing version)
	name4 := "r4"
	fixedRate4 := 0.04
	httpMethod4 := "PUT"
	priority4 := int64(8)
	reservoirSize4 := int64(60)
	serviceName4 := "www.fizz.com"
	urlPath4 := "/resource/fizz"
	u4 := &xraySvc.SamplingRuleRecord{
		SamplingRule: &xraySvc.SamplingRule{
			RuleName:      &name4,
			ServiceName:   &serviceName4,
			URLPath:       &urlPath4,
			HTTPMethod:    &httpMethod4,
			Priority:      &priority4,
			ReservoirSize: &reservoirSize4,
			FixedRate:     &fixedRate4,
			ResourceARN:   &resARN,
		},
	}

	// New Invalid rule 'r5' (invalid version)
	name5 := "r5"
	fixedRate5 := 0.04
	httpMethod5 := "PUT"
	priority5 := int64(8)
	reservoirSize5 := int64(60)
	serviceName5 := "www.fizz.com"
	urlPath5 := "/resource/fizz"
	version5 := int64(0)
	u5 := &xraySvc.SamplingRuleRecord{
		SamplingRule: &xraySvc.SamplingRule{
			RuleName:      &name5,
			ServiceName:   &serviceName5,
			URLPath:       &urlPath5,
			HTTPMethod:    &httpMethod5,
			Priority:      &priority5,
			ReservoirSize: &reservoirSize5,
			FixedRate:     &fixedRate5,
			Version:       &version5,
			ResourceARN:   &resARN,
		},
	}

	// Mock proxy with updates u1 and u3
	proxy := &mockProxy{
		samplingRules: []*xraySvc.SamplingRuleRecord{u1, u2, u3, u4, u5},
	}

	// Mock clock with time incremented to 60 seconds past current
	// manifest refreshedAt timestamp.
	clock := &utils.MockClock{
		NowTime: 1500000060,
	}

	ss := &CentralizedStrategy{
		manifest: manifest,
		proxy:    proxy,
		clock:    clock,
	}

	// Refresh manifest with updates from mock proxy
	err := ss.refreshManifest()
	assert.NotNil(t, err)

	// Assert on size of manifest
	assert.Equal(t, 2, len(ss.manifest.Rules))
	assert.Equal(t, 2, len(ss.manifest.Index))

	// Assert on sorting order
	assert.Equal(t, r1, ss.manifest.Rules[0])
	assert.Equal(t, r3, ss.manifest.Rules[1])

	// Assert on index consistency
	assert.Equal(t, r1, ss.manifest.Index["r1"])
	assert.Equal(t, r3, ss.manifest.Index["r3"])

	// Assert on refreshedAt timestamp not changing
	assert.Equal(t, int64(1500000060), ss.manifest.refreshedAt)
}

// Assert that a proxy error results in an early return
func TestRefreshManifestProxyError(t *testing.T) {
	rules := []*CentralizedRule{}

	index := map[string]*CentralizedRule{}

	manifest := &CentralizedManifest{
		Rules:       rules,
		Index:       index,
		refreshedAt: 1500000000,
	}

	// Mock proxy. Will return error.
	proxy := &mockProxy{}

	// Mock clock with time incremented to 60 seconds past current
	// manifest refreshedAt timestamp.
	clock := &utils.MockClock{
		NowTime: 1500000060,
	}

	ss := &CentralizedStrategy{
		manifest: manifest,
		proxy:    proxy,
		clock:    clock,
	}

	// Refresh manifest with updates from mock proxy
	err := ss.refreshManifest()
	assert.NotNil(t, err)

	// Assert on size of manifest
	assert.Equal(t, 0, len(ss.manifest.Rules))
	assert.Equal(t, 0, len(ss.manifest.Index))

	// Assert on refreshedAt timestamp not changing
	assert.Equal(t, int64(1500000000), ss.manifest.refreshedAt)
}

// Assert that valid targets from proxy result in updated quotas for sampling rules
func TestRefreshTargets(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	// Rule 'r1'
	r1 := &CentralizedRule{
		ruleName: "r1",
		requests: 100,
		sampled:  6,
		borrows:  0,
		usedAt:   1500000000,
		reservoir: &CentralizedReservoir{
			quota: 5,
			reservoir: &reservoir{
				capacity: 30,
			},
			expiresAt:   1500000050,
			refreshedAt: 1499999990,
			interval:    10,
		},
		Properties: &Properties{
			Host:        "www.foo.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/bar",
			FixedTarget: 30,
			Rate:        0.05,
		},
		priority: 4,
		clock:    clock,
	}

	// Rule 'r3'
	r3 := &CentralizedRule{
		ruleName: "r3",
		requests: 50,
		sampled:  50,
		borrows:  0,
		usedAt:   1500000000,
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 50,
			},
			expiresAt:   1500000050,
			refreshedAt: 1499999990,
			interval:    10,
		},
		Properties: &Properties{
			Host:        "www.bar.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/foo",
			FixedTarget: 50,
			Rate:        0.10,
		},
		priority: 8,
		clock:    clock,
	}

	// Sorted array
	rules := []*CentralizedRule{r1, r3}

	index := map[string]*CentralizedRule{
		"r1": r1,
		"r3": r3,
	}

	manifest := &CentralizedManifest{
		Rules:       rules,
		Index:       index,
		refreshedAt: 1499999990,
	}

	// Sampling Target for 'r1'
	rate1 := 0.07
	quota1 := int64(3)
	quotaTTL1 := time.Unix(1500000060, 0)
	name1 := "r1"
	t1 := &xraySvc.SamplingTargetDocument{
		FixedRate:         &rate1,
		ReservoirQuota:    &quota1,
		ReservoirQuotaTTL: &quotaTTL1,
		RuleName:          &name1,
	}

	// Sampling Target for 'r3'
	rate3 := 0.11
	quota3 := int64(15)
	quotaTTL3 := time.Unix(1500000060, 0)
	name3 := "r3"
	t3 := &xraySvc.SamplingTargetDocument{
		FixedRate:         &rate3,
		ReservoirQuota:    &quota3,
		ReservoirQuotaTTL: &quotaTTL3,
		RuleName:          &name3,
	}

	// 'LastRuleModification' attribute
	modifiedAt := time.Unix(1499999900, 0)

	// Mock proxy with targets for 'r1' and 'r3'
	proxy := &mockProxy{
		samplingTargetOutput: &xraySvc.GetSamplingTargetsOutput{
			LastRuleModification: &modifiedAt,
			SamplingTargetDocuments: []*xraySvc.SamplingTargetDocument{
				t1,
				t3,
			},
		},
	}

	ss := &CentralizedStrategy{
		manifest: manifest,
		proxy:    proxy,
		clientID: "c1",
		clock:    clock,
	}

	// Expected state of 'r1' after refresh
	expR1 := &CentralizedRule{
		ruleName: "r1",
		usedAt:   1500000000,
		reservoir: &CentralizedReservoir{
			quota:       3,
			refreshedAt: 1500000000,
			reservoir: &reservoir{
				capacity: 30,
			},
			expiresAt: 1500000060,
			interval:  10,
		},
		Properties: &Properties{
			Host:        "www.foo.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/bar",
			FixedTarget: 30,
			Rate:        0.07,
		},
		priority: 4,
		clock:    clock,
	}

	// Expected state of 'r3' after refresh
	expR3 := &CentralizedRule{
		ruleName: "r3",
		usedAt:   1500000000,
		reservoir: &CentralizedReservoir{
			quota:       15,
			refreshedAt: 1500000000,
			reservoir: &reservoir{
				capacity: 50,
			},
			expiresAt: 1500000060,
			interval:  10,
		},
		Properties: &Properties{
			Host:        "www.bar.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/foo",
			FixedTarget: 50,
			Rate:        0.11,
		},
		priority: 8,
		clock:    clock,
	}

	err := ss.refreshTargets()
	assert.Nil(t, err)

	// Assert on size of manifest not changing
	assert.Equal(t, 2, len(ss.manifest.Rules))
	assert.Equal(t, 2, len(ss.manifest.Index))

	// Assert on updated sampling rules
	assert.Equal(t, expR1, ss.manifest.Index["r1"])
	assert.Equal(t, expR3, ss.manifest.Index["r3"])
}

func TestRefreshTargetsVariableIntervals(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	// Rule 'r1'. Interval of 20 seconds
	r1 := &CentralizedRule{
		ruleName: "r1",
		requests: 100,
		sampled:  6,
		borrows:  0,
		usedAt:   1499999999,
		reservoir: &CentralizedReservoir{
			quota: 5,
			reservoir: &reservoir{
				capacity: 30,
			},
			expiresAt:   1500000100,
			refreshedAt: 1499999990,
			interval:    20,
		},
		Properties: &Properties{
			Host:        "www.foo.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/bar",
			FixedTarget: 30,
			Rate:        0.05,
		},
		priority: 4,
		clock:    clock,
	}

	// Rule 'r3'. Interval of 30 seconds.
	r3 := &CentralizedRule{
		ruleName: "r3",
		requests: 50,
		sampled:  50,
		borrows:  0,
		usedAt:   1500000000,
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 50,
			},
			expiresAt:   1500000200,
			refreshedAt: 1499999990,
			interval:    30,
		},
		Properties: &Properties{
			Host:        "www.bar.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/foo",
			FixedTarget: 50,
			Rate:        0.10,
		},
		priority: 8,
		clock:    clock,
	}

	// Sorted array
	rules := []*CentralizedRule{r1, r3}

	index := map[string]*CentralizedRule{
		"r1": r1,
		"r3": r3,
	}

	manifest := &CentralizedManifest{
		Rules:       rules,
		Index:       index,
		refreshedAt: 1499999990,
	}

	// Sampling Target for 'r1'
	rate1 := 0.07
	quota1 := int64(3)
	quotaTTL1 := time.Unix(1500000060, 0)
	name1 := "r1"
	t1 := &xraySvc.SamplingTargetDocument{
		FixedRate:         &rate1,
		ReservoirQuota:    &quota1,
		ReservoirQuotaTTL: &quotaTTL1,
		RuleName:          &name1,
	}

	// Sampling Target for 'r3'
	rate3 := 0.11
	quota3 := int64(15)
	quotaTTL3 := time.Unix(1500000060, 0)
	name3 := "r3"
	t3 := &xraySvc.SamplingTargetDocument{
		FixedRate:         &rate3,
		ReservoirQuota:    &quota3,
		ReservoirQuotaTTL: &quotaTTL3,
		RuleName:          &name3,
	}

	// 'LastRuleModification' attribute
	modifiedAt := time.Unix(1499999900, 0)

	// Mock proxy with targets for 'r1' and 'r3'
	proxy := &mockProxy{
		samplingTargetOutput: &xraySvc.GetSamplingTargetsOutput{
			LastRuleModification: &modifiedAt,
			SamplingTargetDocuments: []*xraySvc.SamplingTargetDocument{
				t1,
				t3,
			},
		},
	}

	ss := &CentralizedStrategy{
		manifest: manifest,
		proxy:    proxy,
		clientID: "c1",
		clock:    clock,
	}

	// No targets should be refreshed
	err := ss.refreshTargets()
	assert.Nil(t, err)
	assert.Equal(t, r1, ss.manifest.Index["r1"])
	assert.Equal(t, r3, ss.manifest.Index["r3"])

	// Increment time to 1500000010
	clock.Increment(10, 0)

	// Expected state of 'r1' after refresh
	expR1 := &CentralizedRule{
		ruleName: "r1",
		usedAt:   1499999999,
		reservoir: &CentralizedReservoir{
			quota:       3,
			refreshedAt: 1500000010,
			reservoir: &reservoir{
				capacity: 30,
			},
			expiresAt: 1500000060,
			interval:  20,
		},
		Properties: &Properties{
			Host:        "www.foo.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/bar",
			FixedTarget: 30,
			Rate:        0.07,
		},
		priority: 4,
		clock:    clock,
	}

	// Expected state of 'r3' after refresh
	expR3 := &CentralizedRule{
		ruleName: "r3",
		requests: 50,
		sampled:  50,
		borrows:  0,
		usedAt:   1500000000,
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 50,
			},
			expiresAt:   1500000200,
			refreshedAt: 1499999990,
			interval:    30,
		},
		Properties: &Properties{
			Host:        "www.bar.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/foo",
			FixedTarget: 50,
			Rate:        0.10,
		},
		priority: 8,
		clock:    clock,
	}

	// Only r1 should be refreshed
	err = ss.refreshTargets()
	assert.Nil(t, err)
	assert.Equal(t, expR1, ss.manifest.Index["r1"])
	assert.Equal(t, expR3, ss.manifest.Index["r3"])

	// Increment time to 1500000020
	clock.Increment(10, 0)

	// Expected state of 'r3' after refresh
	expR3 = &CentralizedRule{
		ruleName: "r3",
		usedAt:   1500000000,
		reservoir: &CentralizedReservoir{
			quota:       15,
			refreshedAt: 1500000020,
			reservoir: &reservoir{
				capacity: 50,
			},
			expiresAt: 1500000060,
			interval:  30,
		},
		Properties: &Properties{
			Host:        "www.bar.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/foo",
			FixedTarget: 50,
			Rate:        0.11,
		},
		priority: 8,
		clock:    clock,
	}

	// r3 should be refreshed
	err = ss.refreshTargets()
	assert.Nil(t, err)

	// Assert on size of manifest not changing
	assert.Equal(t, 2, len(ss.manifest.Rules))
	assert.Equal(t, 2, len(ss.manifest.Index))

	// Assert on updated sampling rules
	assert.Equal(t, expR1, ss.manifest.Index["r1"])
	assert.Equal(t, expR3, ss.manifest.Index["r3"])
}

// Assert that an invalid sampling target does not leave the manifest in an
// inconsistent state. The invalid sampling target should be ignored.
func TestRefreshTargetsInvalidTarget(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	// Rule 'r1'
	r1 := &CentralizedRule{
		ruleName: "r1",
		requests: 100,
		sampled:  6,
		borrows:  0,
		usedAt:   1500000000,
		reservoir: &CentralizedReservoir{
			quota: 5,
			reservoir: &reservoir{
				capacity: 30,
			},
			interval:  10,
			expiresAt: 1500000050,
		},
		Properties: &Properties{
			Host:        "www.foo.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/bar",
			FixedTarget: 30,
			Rate:        0.05,
		},
		priority: 4,
		clock:    clock,
	}

	// Rule 'r3'
	r3 := &CentralizedRule{
		ruleName: "r3",
		requests: 50,
		sampled:  50,
		borrows:  0,
		usedAt:   1500000000,
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 50,
			},
			expiresAt: 1500000050,
			interval:  10,
		},
		Properties: &Properties{
			Host:        "www.bar.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/foo",
			FixedTarget: 50,
			Rate:        0.10,
		},
		priority: 8,
		clock:    clock,
	}

	// Sorted array
	rules := []*CentralizedRule{r1, r3}

	index := map[string]*CentralizedRule{
		"r1": r1,
		"r3": r3,
	}

	manifest := &CentralizedManifest{
		Rules:       rules,
		Index:       index,
		refreshedAt: 1499999990,
	}

	// Invalid sampling Target for 'r1' (missing fixed rate)
	quota1 := int64(3)
	quotaTTL1 := time.Unix(1500000060, 0)
	name1 := "r1"
	t1 := &xraySvc.SamplingTargetDocument{
		RuleName:          &name1,
		ReservoirQuota:    &quota1,
		ReservoirQuotaTTL: &quotaTTL1,
	}

	// Valid sampling Target for 'r3'
	rate3 := 0.11
	quota3 := int64(15)
	quotaTTL3 := time.Unix(1500000060, 0)
	name3 := "r3"
	t3 := &xraySvc.SamplingTargetDocument{
		FixedRate:         &rate3,
		ReservoirQuota:    &quota3,
		ReservoirQuotaTTL: &quotaTTL3,
		RuleName:          &name3,
	}

	// 'LastRuleModification' attribute
	modifiedAt := time.Unix(1499999900, 0)

	// Mock proxy with targets for 'r1' and 'r3'
	proxy := &mockProxy{
		samplingTargetOutput: &xraySvc.GetSamplingTargetsOutput{
			LastRuleModification: &modifiedAt,
			SamplingTargetDocuments: []*xraySvc.SamplingTargetDocument{
				t1,
				t3,
			},
		},
	}

	ss := &CentralizedStrategy{
		manifest: manifest,
		proxy:    proxy,
		clientID: "c1",
		clock:    clock,
	}

	// Expected state of 'r1' after refresh.
	// Unchanged except for reset counters.
	expR1 := &CentralizedRule{
		ruleName: "r1",
		usedAt:   1500000000,
		reservoir: &CentralizedReservoir{
			quota: 5,
			reservoir: &reservoir{
				capacity: 30,
			},
			expiresAt: 1500000050,
			interval:  10,
		},
		Properties: &Properties{
			Host:        "www.foo.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/bar",
			FixedTarget: 30,
			Rate:        0.05,
		},
		priority: 4,
		clock:    clock,
	}

	// Expected state of 'r3' after refresh
	expR3 := &CentralizedRule{
		ruleName: "r3",
		usedAt:   1500000000,
		reservoir: &CentralizedReservoir{
			quota:       15,
			refreshedAt: 1500000000,
			reservoir: &reservoir{
				capacity: 50,
			},
			expiresAt: 1500000060,
			interval:  10,
		},
		Properties: &Properties{
			Host:        "www.bar.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/foo",
			FixedTarget: 50,
			Rate:        0.11,
		},
		priority: 8,
		clock:    clock,
	}

	err := ss.refreshTargets()
	assert.NotNil(t, err)

	// Assert on size of manifest not changing
	assert.Equal(t, 2, len(ss.manifest.Rules))
	assert.Equal(t, 2, len(ss.manifest.Index))

	// Assert on updated sampling rules
	assert.Equal(t, expR1, ss.manifest.Index["r1"])
	assert.Equal(t, expR3, ss.manifest.Index["r3"])
}

// Assert that target refresh triggers a manifest refresh if `LastRuleModification`
// attribute is greater than the manifest's refreshedAt attribute
func TestRefreshTargetsOutdatedManifest(t *testing.T) {
	serviceType := ""
	resARN := "*"
	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	// Existing Rule 'r3'
	r3 := &CentralizedRule{
		ruleName: "r3",
		requests: 50,
		sampled:  50,
		borrows:  0,
		usedAt:   1500000000,
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 50,
			},
			expiresAt: 1500000050,
			interval:  10,
		},
		Properties: &Properties{
			ServiceName: "www.bar.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/foo",
			FixedTarget: 50,
			Rate:        0.10,
			Host:        "www.bar.com",
		},
		priority:    8,
		clock:       clock,
		resourceARN: resARN,
	}

	// Sorted array
	rules := []*CentralizedRule{r3}

	index := map[string]*CentralizedRule{
		"r3": r3,
	}

	manifest := &CentralizedManifest{
		Rules:       rules,
		Index:       index,
		refreshedAt: 1499999800,
	}

	// Valid sampling Target for 'r3'
	rate3 := 0.11
	quota3 := int64(15)
	quotaTTL3 := time.Unix(1500000060, 0)
	name3 := "r3"
	t3 := &xraySvc.SamplingTargetDocument{
		FixedRate:         &rate3,
		ReservoirQuota:    &quota3,
		ReservoirQuotaTTL: &quotaTTL3,
		RuleName:          &name3,
	}

	// New rule 'r1'
	name := "r1"
	fixedRate := 0.05
	httpMethod := "POST"
	priority := int64(4)
	reservoirSize := int64(50)
	serviceName := "www.foo.com"
	urlPath := "/resource/bar"
	version := int64(1)

	new := &xraySvc.SamplingRuleRecord{
		SamplingRule: &xraySvc.SamplingRule{
			RuleName:      &name,
			ServiceName:   &serviceName,
			URLPath:       &urlPath,
			HTTPMethod:    &httpMethod,
			Priority:      &priority,
			ReservoirSize: &reservoirSize,
			FixedRate:     &fixedRate,
			Version:       &version,
			Host:          &serviceName,
			ServiceType:   &serviceType,
			ResourceARN:   &resARN,
		},
	}

	// 'LastRuleModification' attribute
	modifiedAt := time.Unix(1499999900, 0)

	// Mock proxy with `LastRuleModification` attribute and sampling rules
	proxy := &mockProxy{
		samplingTargetOutput: &xraySvc.GetSamplingTargetsOutput{
			LastRuleModification:    &modifiedAt,
			SamplingTargetDocuments: []*xraySvc.SamplingTargetDocument{t3},
		},
		samplingRules: []*xraySvc.SamplingRuleRecord{new},
	}

	ss := &CentralizedStrategy{
		manifest: manifest,
		proxy:    proxy,
		clientID: "c1",
		clock:    clock,
	}

	err := ss.refreshTargets()
	assert.Nil(t, err)

	timer := time.NewTimer(1 * time.Second)

	// Assert that manifest is refreshed. The refresh is async so we timeout
	// after one second.
A:
	for {
		select {
		case <-timer.C:
			assert.Fail(t, "Timed out waiting for async manifest refresh")
			break A
		default:
			// Assert that rule was added to manifest and the timestamp refreshed
			ss.manifest.mu.Lock()
			if len(ss.manifest.Rules) == 1 &&
				len(ss.manifest.Index) == 1 &&
				ss.manifest.refreshedAt == 1500000000 {
				ss.manifest.mu.Unlock()
				break A
			}
			ss.manifest.mu.Unlock()
		}
	}
}

// Assert that a proxy error results in an early return with the manifest unchanged.
func TestRefreshTargetsProxyError(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	// Existing Rule 'r3'
	r3 := &CentralizedRule{
		ruleName: "r3",
		requests: 50,
		sampled:  50,
		borrows:  0,
		usedAt:   1500000000,
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 50,
			},
			expiresAt: 1500000050,
		},
		Properties: &Properties{
			Host:        "www.bar.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/foo",
			FixedTarget: 50,
			Rate:        0.10,
		},
		priority: 8,
		clock:    clock,
	}

	// Sorted array
	rules := []*CentralizedRule{r3}

	index := map[string]*CentralizedRule{
		"r3": r3,
	}

	manifest := &CentralizedManifest{
		Rules:       rules,
		Index:       index,
		refreshedAt: 1499999800,
	}

	// Mock proxy. Will return error.
	proxy := &mockProxy{}

	ss := &CentralizedStrategy{
		manifest: manifest,
		proxy:    proxy,
		clientID: "c1",
		clock:    clock,
	}

	err := ss.refreshTargets()
	assert.NotNil(t, err)

	// Assert on size of manifest not changing
	assert.Equal(t, 1, len(ss.manifest.Rules))
	assert.Equal(t, 1, len(ss.manifest.Index))
}

func TestLoadDaemonEndpoints1(t *testing.T) {

	host1 := "www.foo.com"
	method1 := "POST"
	url1 := "/resource/bar"
	serviceName1 := "localhost"
	servType1 := "AWS::EC2::Instance"

	sr := &Request{
		Host:        host1,
		URL:         url1,
		Method:      method1,
		ServiceName: serviceName1,
		ServiceType: servType1,
	}

	s, _ := NewCentralizedStrategy()
	d, _ := daemoncfg.GetDaemonEndpointsFromString("127.0.0.0:3000")
	s.LoadDaemonEndpoints(d)

	// Make positive sampling decision against 'r1'
	s.ShouldTrace(sr)
	assert.Equal(t, d, s.daemonEndpoints)

}

func TestLoadDaemonEndpoints2(t *testing.T) {
	host1 := "www.foo.com"
	method1 := "POST"
	url1 := "/resource/bar"
	serviceName1 := "localhost"
	servType1 := "AWS::EC2::Instance"

	sr := &Request{
		Host:        host1,
		URL:         url1,
		Method:      method1,
		ServiceName: serviceName1,
		ServiceType: servType1,
	}

	s, _ := NewCentralizedStrategy()
	s.LoadDaemonEndpoints(nil) // if nil, env variable or default endpoint is set to proxy

	// Make positive sampling decision against 'r1'
	s.ShouldTrace(sr)
	assert.Nil(t, s.daemonEndpoints)
}

// Benchmarks
func BenchmarkCentralizedStrategy_ShouldTrace(b *testing.B) {
	s, _ := NewCentralizedStrategy()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			s.ShouldTrace(&Request{})
		}
	})
}

func BenchmarkNewCentralizedStrategy_refreshManifest(b *testing.B) {
	serviceTye := ""
	resourceARN := "*"
	// Valid no-op update for rule 'r1'
	name1 := "r1"
	fixedRate1 := 0.05
	httpMethod1 := "POST"
	priority1 := int64(4)
	reservoirSize1 := int64(50)
	serviceName1 := "www.foo.com"
	urlPath1 := "/resource/bar"
	version1 := int64(1)
	u1 := &xraySvc.SamplingRuleRecord{
		SamplingRule: &xraySvc.SamplingRule{
			RuleName:      &name1,
			ServiceName:   &serviceName1,
			URLPath:       &urlPath1,
			HTTPMethod:    &httpMethod1,
			Priority:      &priority1,
			ReservoirSize: &reservoirSize1,
			FixedRate:     &fixedRate1,
			Version:       &version1,
			Host:          &serviceName1,
			ServiceType:   &serviceTye,
			ResourceARN:   &resourceARN,
		},
	}

	// Rule 'r1'
	r1 := &CentralizedRule{
		ruleName: "r1",
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 50,
			},
		},
		Properties:  &Properties{},
		priority:    4,
		resourceARN: resourceARN,
	}

	// Rule 'r3'
	r3 := &CentralizedRule{
		ruleName: "r3",
		reservoir: &CentralizedReservoir{
			quota: 10,
			reservoir: &reservoir{
				capacity: 50,
			},
		},
		Properties: &Properties{
			Host:        "www.bar.com",
			HTTPMethod:  "POST",
			URLPath:     "/resource/foo",
			FixedTarget: 40,
			Rate:        0.10,
			ServiceName: "www.bar.com",
		},
		priority:    8,
		resourceARN: resourceARN,
	}

	// Sorted array
	rules := []*CentralizedRule{r1, r3}

	index := map[string]*CentralizedRule{
		"r1": r1,
		"r3": r3,
	}

	manifest := &CentralizedManifest{
		Rules:       rules,
		Index:       index,
		refreshedAt: 1500000000,
	}

	// Mock proxy with updates u1, u2, and u3
	proxy := &mockProxy{
		samplingRules: []*xraySvc.SamplingRuleRecord{u1},
	}

	// Mock clock with time incremented to 60 seconds past current
	// manifest refreshedAt timestamp.
	clock := &utils.MockClock{
		NowTime: 1500000060,
	}

	ss := &CentralizedStrategy{
		manifest: manifest,
		proxy:    proxy,
		clock:    clock,
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := ss.refreshManifest()
			if err != nil {
				return
			}
		}
	})
}

func BenchmarkCentralizedStrategy_refreshTargets(b *testing.B) {
	s, _ := NewCentralizedStrategy()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := s.refreshTargets()
			if err != nil {
				return
			}
		}
	})
}
