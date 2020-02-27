// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package sampling

import (
	"testing"

	"github.com/stretchr/testify/assert"

	xraySvc "github.com/aws/aws-sdk-go/service/xray"
	"github.com/aws/aws-xray-sdk-go/utils"
)

// Assert that putRule() creates a new user-defined rule and adds to manifest
func TestCreateUserRule(t *testing.T) {
	resARN := "*"
	r1 := &CentralizedRule{
		ruleName: "r1",
		priority: 5,
	}

	r3 := &CentralizedRule{
		ruleName: "r3",
		priority: 7,
	}

	rules := []*CentralizedRule{r1, r3}

	index := map[string]*CentralizedRule{
		"r1": r1,
		"r3": r3,
	}

	m := &CentralizedManifest{
		Rules: rules,
		Index: index,
	}

	// New xraySvc.CentralizedSamplingRule. Input to putRule().
	serviceName := "www.foo.com"
	httpMethod := "POST"
	urlPath := "/bar/*"
	reservoirSize := int64(10)
	fixedRate := float64(0.05)
	ruleName := "r2"
	host := "local"
	priority := int64(6)
	serviceTye := "*"
	new := &xraySvc.SamplingRule{
		ServiceName:   &serviceName,
		HTTPMethod:    &httpMethod,
		URLPath:       &urlPath,
		ReservoirSize: &reservoirSize,
		FixedRate:     &fixedRate,
		RuleName:      &ruleName,
		Priority:      &priority,
		Host:          &host,
		ServiceType:   &serviceTye,
		ResourceARN:   &resARN,
	}

	// Expected centralized sampling rule
	clock := &utils.DefaultClock{}
	rand := &utils.DefaultRand{}

	p := &Properties{
		ServiceName: serviceName,
		HTTPMethod:  httpMethod,
		URLPath:     urlPath,
		FixedTarget: reservoirSize,
		Rate:        fixedRate,
		Host:        host,
	}

	cr := &CentralizedReservoir{
		reservoir: &reservoir{
			capacity: 10,
		},
		interval: 10,
	}

	exp := &CentralizedRule{
		reservoir:   cr,
		ruleName:    ruleName,
		priority:    priority,
		Properties:  p,
		clock:       clock,
		rand:        rand,
		serviceType: serviceTye,
		resourceARN: resARN,
	}

	// Add to manifest, index, and sort
	r2, err := m.putRule(new)
	assert.Nil(t, err)
	assert.Equal(t, exp, r2)

	// Assert new rule is present in index
	r2, ok := m.Index["r2"]
	assert.True(t, ok)
	assert.Equal(t, exp, r2)

	// Assert new rule present at end of array. putRule() does not preserve order.
	r2 = m.Rules[2]
	assert.Equal(t, exp, r2)
}

// Assert that putRule() creates a new default rule and adds to manifest
func TestCreateDefaultRule(t *testing.T) {
	m := &CentralizedManifest{
		Index: map[string]*CentralizedRule{},
	}

	// New xraySvc.CentralizedSamplingRule. Input to putRule().
	reservoirSize := int64(10)
	fixedRate := float64(0.05)
	ruleName := "Default"
	new := &xraySvc.SamplingRule{
		ReservoirSize: &reservoirSize,
		FixedRate:     &fixedRate,
		RuleName:      &ruleName,
	}

	// Expected centralized sampling rule
	clock := &utils.DefaultClock{}
	rand := &utils.DefaultRand{}

	p := &Properties{
		FixedTarget: reservoirSize,
		Rate:        fixedRate,
	}

	cr := &CentralizedReservoir{
		reservoir: &reservoir{
			capacity: reservoirSize,
		},
		interval: 10,
	}

	exp := &CentralizedRule{
		reservoir:  cr,
		ruleName:   ruleName,
		Properties: p,
		clock:      clock,
		rand:       rand,
	}

	// Add to manifest
	r, err := m.putRule(new)
	assert.Nil(t, err)
	assert.Equal(t, exp, r)
	assert.Equal(t, exp, m.Default)
}

// Assert that putRule() creates a new default rule and adds to manifest
func TestUpdateDefaultRule(t *testing.T) {
	clock := &utils.DefaultClock{}
	rand := &utils.DefaultRand{}

	// Original default sampling rule
	r := &CentralizedRule{
		ruleName: "Default",
		Properties: &Properties{
			FixedTarget: 10,
			Rate:        0.05,
		},
		reservoir: &CentralizedReservoir{
			reservoir: &reservoir{
				capacity: 10,
			},
		},
		clock: clock,
		rand:  rand,
	}

	m := &CentralizedManifest{
		Default: r,
	}

	// Updated xraySvc.CentralizedSamplingRule. Input to putRule().
	reservoirSize := int64(20)
	fixedRate := float64(0.06)
	ruleName := "Default"
	updated := &xraySvc.SamplingRule{
		ReservoirSize: &reservoirSize,
		FixedRate:     &fixedRate,
		RuleName:      &ruleName,
	}

	// Expected centralized sampling rule
	p := &Properties{
		FixedTarget: reservoirSize,
		Rate:        fixedRate,
	}

	cr := &CentralizedReservoir{
		reservoir: &reservoir{
			capacity: reservoirSize,
		},
	}

	exp := &CentralizedRule{
		reservoir:  cr,
		ruleName:   ruleName,
		Properties: p,
		clock:      clock,
		rand:       rand,
	}

	// Update default rule in manifest
	r, err := m.putRule(updated)
	assert.Nil(t, err)
	assert.Equal(t, exp, r)
	assert.Equal(t, exp, m.Default)
}

// Assert that creating a user-defined rule which already exists is a no-op
func TestCreateUserRuleNoOp(t *testing.T) {
	resARN := "*"
	serviceTye := ""
	attributes := make(map[string]*string)
	r1 := &CentralizedRule{
		ruleName: "r1",
		priority: 5,
	}

	r3 := &CentralizedRule{
		ruleName: "r3",
		priority: 7,
		reservoir: &CentralizedReservoir{
			reservoir: &reservoir{},
		},
	}

	rules := []*CentralizedRule{r1, r3}

	index := map[string]*CentralizedRule{
		"r1": r1,
		"r3": r3,
	}

	m := &CentralizedManifest{
		Rules: rules,
		Index: index,
	}

	// Duplicate xraySvc.CentralizedSamplingRule. 'r3' already exists. Input to updateRule().
	serviceName := "www.foo.com"
	httpMethod := "POST"
	urlPath := "/bar/*"
	reservoirSize := int64(10)
	fixedRate := float64(0.05)
	ruleName := "r3"
	priority := int64(6)
	host := "h"
	new := &xraySvc.SamplingRule{
		ServiceName:   &serviceName,
		HTTPMethod:    &httpMethod,
		URLPath:       &urlPath,
		ReservoirSize: &reservoirSize,
		FixedRate:     &fixedRate,
		RuleName:      &ruleName,
		Priority:      &priority,
		Host:          &host,
		ResourceARN:   &resARN,
		ServiceType:   &serviceTye,
		Attributes:    attributes,
	}

	// Assert manifest has not changed
	r, err := m.putRule(new)
	assert.Nil(t, err)
	assert.Equal(t, r3, r)
	assert.Equal(t, 2, len(m.Rules))
	assert.Equal(t, 2, len(m.Index))
	assert.Equal(t, r1, m.Rules[0])
	assert.Equal(t, r3, m.Rules[1])
}

// Assert that putRule() updates the user-defined rule in the manifest
func TestUpdateUserRule(t *testing.T) {
	resARN := "*"
	serviceTye := ""
	attributes := make(map[string]*string)
	// Original rule
	r1 := &CentralizedRule{
		ruleName: "r1",
		priority: 5,
		Properties: &Properties{
			ServiceName: "*.foo.com",
			HTTPMethod:  "GET",
			URLPath:     "/resource/*",
			FixedTarget: 15,
			Rate:        0.04,
		},
		reservoir: &CentralizedReservoir{
			reservoir: &reservoir{
				capacity: 5,
			},
		},
		resourceARN: resARN,
		serviceType: serviceTye,
		attributes:  attributes,
	}

	rules := []*CentralizedRule{r1}

	index := map[string]*CentralizedRule{
		"r1": r1,
	}

	m := &CentralizedManifest{
		Rules: rules,
		Index: index,
	}

	// Updated xraySvc.CentralizedSamplingRule. Input to updateRule().
	serviceName := "www.foo.com"
	httpMethod := "POST"
	urlPath := "/bar/*"
	reservoirSize := int64(10)
	fixedRate := float64(0.05)
	ruleName := "r1"
	priority := int64(6)
	host := "h"
	updated := &xraySvc.SamplingRule{
		ServiceName:   &serviceName,
		HTTPMethod:    &httpMethod,
		URLPath:       &urlPath,
		ReservoirSize: &reservoirSize,
		FixedRate:     &fixedRate,
		RuleName:      &ruleName,
		Priority:      &priority,
		Host:          &host,
		ResourceARN:   &resARN,
		ServiceType:   &serviceTye,
		Attributes:    attributes,
	}

	// Expected updated centralized sampling rule
	p := &Properties{
		ServiceName: serviceName,
		HTTPMethod:  httpMethod,
		URLPath:     urlPath,
		FixedTarget: reservoirSize,
		Rate:        fixedRate,
		Host:        host,
	}

	cr := &CentralizedReservoir{
		reservoir: &reservoir{
			capacity: 10,
		},
	}

	exp := &CentralizedRule{
		reservoir:   cr,
		ruleName:    ruleName,
		priority:    priority,
		Properties:  p,
		resourceARN: resARN,
		serviceType: serviceTye,
		attributes:  attributes,
	}

	// Assert that rule has been updated
	r, err := m.putRule(updated)
	assert.Nil(t, err)
	assert.Equal(t, exp, r)
	assert.Equal(t, exp, m.Index["r1"])
	assert.Equal(t, exp, m.Rules[0])
	assert.Equal(t, 1, len(m.Rules))
	assert.Equal(t, 1, len(m.Index))
}

// Assert that putRule() recovers from panic.
func TestPutRuleRecovery(t *testing.T) {
	resARN := "*"
	serviceTye := ""
	attributes := make(map[string]*string)
	rules := []*CentralizedRule{}

	index := map[string]*CentralizedRule{}

	m := &CentralizedManifest{
		Rules: rules,
		Index: index,
	}

	// Invalid xraySvc.CentralizedSamplingRule with nil fileds. Input to putRule().
	serviceName := "www.foo.com"
	httpMethod := "POST"
	fixedRate := float64(0.05)
	ruleName := "r2"
	priority := int64(6)
	new := &xraySvc.SamplingRule{
		ServiceName: &serviceName,
		HTTPMethod:  &httpMethod,
		FixedRate:   &fixedRate,
		RuleName:    &ruleName,
		Priority:    &priority,
		ResourceARN: &resARN,
		ServiceType: &serviceTye,
		Attributes:  attributes,
	}

	// Attempt to add to manifest
	r, err := m.putRule(new)
	assert.NotNil(t, err)
	assert.Nil(t, r)
	assert.Nil(t, m.Default)

	// Assert index is unchanged
	assert.Equal(t, 0, len(m.Index))

	// Assert sorted array is unchanged
	assert.Equal(t, 0, len(m.Rules))
}

// Assert that deleting a rule from the end of the array removes the rule
// and preserves ordering of the sorted array
func TestDeleteLastRule(t *testing.T) {
	r1 := &CentralizedRule{
		ruleName: "r1",
		priority: 5,
	}

	r2 := &CentralizedRule{
		ruleName: "r2",
		priority: 6,
	}

	r3 := &CentralizedRule{
		ruleName: "r3",
		priority: 7,
	}

	rules := []*CentralizedRule{r1, r2, r3}

	index := map[string]*CentralizedRule{
		"r1": r1,
		"r2": r2,
		"r3": r3,
	}

	m := &CentralizedManifest{
		Rules: rules,
		Index: index,
	}

	// Active rules to exclude from deletion
	a := map[*CentralizedRule]bool{
		r1: true,
		r2: true,
	}

	// Delete r3
	m.prune(a)

	// Assert size of manifest
	assert.Equal(t, 2, len(m.Rules))
	assert.Equal(t, 2, len(m.Index))

	// Assert index consistency
	_, ok := m.Index["r3"]
	assert.False(t, ok)
	assert.Equal(t, r1, m.Index["r1"])
	assert.Equal(t, r2, m.Index["r2"])

	// Assert ordering of array
	assert.Equal(t, r1, m.Rules[0])
	assert.Equal(t, r2, m.Rules[1])
}

// Assert that deleting a rule from the middle of the array removes the rule
// and preserves ordering of the sorted array
func TestDeleteMiddleRule(t *testing.T) {
	r1 := &CentralizedRule{
		ruleName: "r1",
		priority: 5,
	}

	r2 := &CentralizedRule{
		ruleName: "r2",
		priority: 6,
	}

	r3 := &CentralizedRule{
		ruleName: "r3",
		priority: 7,
	}

	rules := []*CentralizedRule{r1, r2, r3}

	index := map[string]*CentralizedRule{
		"r1": r1,
		"r2": r2,
		"r3": r3,
	}

	m := &CentralizedManifest{
		Rules: rules,
		Index: index,
	}

	// Active rules to exclude from deletion
	a := map[*CentralizedRule]bool{
		r1: true,
		r3: true,
	}

	// Delete r2
	m.prune(a)

	// Assert size of manifest
	assert.Equal(t, 2, len(m.Rules))
	assert.Equal(t, 2, len(m.Index))

	// Assert index consistency
	_, ok := m.Index["r2"]
	assert.False(t, ok)
	assert.Equal(t, r1, m.Index["r1"])
	assert.Equal(t, r3, m.Index["r3"])

	// Assert ordering of array
	assert.Equal(t, r1, m.Rules[0])
	assert.Equal(t, r3, m.Rules[1])
}

// Assert that deleting a rule from the beginning of the array removes the rule
// and preserves ordering of the sorted array
func TestDeleteFirstRule(t *testing.T) {
	r1 := &CentralizedRule{
		ruleName: "r1",
		priority: 5,
	}

	r2 := &CentralizedRule{
		ruleName: "r2",
		priority: 6,
	}

	r3 := &CentralizedRule{
		ruleName: "r3",
		priority: 7,
	}

	rules := []*CentralizedRule{r1, r2, r3}

	index := map[string]*CentralizedRule{
		"r1": r1,
		"r2": r2,
		"r3": r3,
	}

	m := &CentralizedManifest{
		Rules: rules,
		Index: index,
	}

	// Active rules to exclude from deletion
	a := map[*CentralizedRule]bool{
		r2: true,
		r3: true,
	}

	// Delete r1
	m.prune(a)

	// Assert size of manifest
	assert.Equal(t, 2, len(m.Rules))
	assert.Equal(t, 2, len(m.Index))

	// Assert index consistency
	_, ok := m.Index["r1"]
	assert.False(t, ok)
	assert.Equal(t, r2, m.Index["r2"])
	assert.Equal(t, r3, m.Index["r3"])

	// Assert ordering of array
	assert.Equal(t, r2, m.Rules[0])
	assert.Equal(t, r3, m.Rules[1])
}

// Assert that deleting the only rule from the array removes the rule
func TestDeleteOnlyRule(t *testing.T) {
	r1 := &CentralizedRule{
		ruleName: "r1",
		priority: 5,
	}

	rules := []*CentralizedRule{r1}

	index := map[string]*CentralizedRule{
		"r1": r1,
	}

	m := &CentralizedManifest{
		Rules: rules,
		Index: index,
	}

	// Active rules to exclude from deletion
	a := map[*CentralizedRule]bool{}

	// Delete r1
	m.prune(a)

	// Assert size of manifest
	assert.Equal(t, 0, len(m.Rules))
	assert.Equal(t, 0, len(m.Index))

	// Assert index consistency
	_, ok := m.Index["r1"]
	assert.False(t, ok)
}

// Assert that deleting rules from an empty array does not panic
func TestDeleteEmptyRulesArray(t *testing.T) {
	rules := []*CentralizedRule{}

	index := map[string]*CentralizedRule{}

	m := &CentralizedManifest{
		Rules: rules,
		Index: index,
	}

	// Active rules to exclude from deletion
	a := map[*CentralizedRule]bool{}

	// Delete from empty array
	m.prune(a)

	// Assert size of manifest
	assert.Equal(t, 0, len(m.Rules))
	assert.Equal(t, 0, len(m.Index))
}

// Assert that deleting all rules results in an empty array and does not panic
func TestDeleteAllRules(t *testing.T) {
	r1 := &CentralizedRule{
		ruleName: "r1",
		priority: 5,
	}

	r2 := &CentralizedRule{
		ruleName: "r2",
		priority: 6,
	}

	r3 := &CentralizedRule{
		ruleName: "r3",
		priority: 7,
	}

	rules := []*CentralizedRule{r1, r2, r3}

	index := map[string]*CentralizedRule{
		"r1": r1,
		"r2": r2,
		"r3": r3,
	}

	m := &CentralizedManifest{
		Rules: rules,
		Index: index,
	}

	// Active rules to exclude from deletion
	a := map[*CentralizedRule]bool{}

	// Delete r3
	m.prune(a)

	// Assert size of manifest
	assert.Equal(t, 0, len(m.Rules))
	assert.Equal(t, 0, len(m.Index))
}

// Assert that sorting an unsorted array results in a sorted array - check priority
func TestSort1(t *testing.T) {
	r1 := &CentralizedRule{
		ruleName: "r1",
		priority: 5,
	}

	r2 := &CentralizedRule{
		ruleName: "r2",
		priority: 6,
	}

	r3 := &CentralizedRule{
		ruleName: "r3",
		priority: 7,
	}

	// Unsorted rules array
	rules := []*CentralizedRule{r2, r1, r3}

	m := &CentralizedManifest{
		Rules: rules,
	}

	// Sort array
	m.sort()

	// Assert on order
	assert.Equal(t, r1, m.Rules[0])
	assert.Equal(t, r2, m.Rules[1])
	assert.Equal(t, r3, m.Rules[2])
}

// Assert that sorting an unsorted array results in a sorted array - check priority and rule name
func TestSort2(t *testing.T) {
	r1 := &CentralizedRule{
		ruleName: "r1",
		priority: 5,
	}

	r2 := &CentralizedRule{
		ruleName: "r2",
		priority: 5,
	}

	r3 := &CentralizedRule{
		ruleName: "r3",
		priority: 7,
	}

	// Unsorted rules array
	rules := []*CentralizedRule{r2, r1, r3}

	m := &CentralizedManifest{
		Rules: rules,
	}

	// Sort array
	m.sort() // r1 should precede r2

	// Assert on order
	assert.Equal(t, r1, m.Rules[0])
	assert.Equal(t, r2, m.Rules[1])
	assert.Equal(t, r3, m.Rules[2])
}

// Assert that an expired manifest is recognized as such
func TestExpired(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500003601,
	}

	m := &CentralizedManifest{
		refreshedAt: 1500000000,
		clock:       clock,
	}

	assert.True(t, m.expired())
}

// Assert that a fresh manifest is recognized as such
func TestFresh(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500003600,
	}

	m := &CentralizedManifest{
		refreshedAt: 1500000000,
		clock:       clock,
	}

	assert.False(t, m.expired())
}

// benchmarks
func BenchmarkCentralizedManifest_putRule(b *testing.B) {

	r1 := &CentralizedRule{
		ruleName: "r1",
		priority: 5,
	}

	r3 := &CentralizedRule{
		ruleName: "r3",
		priority: 7,
	}

	rules := []*CentralizedRule{r1, r3}

	index := map[string]*CentralizedRule{
		"r1": r1,
		"r3": r3,
	}

	m := &CentralizedManifest{
		Rules: rules,
		Index: index,
	}
	// New xraySvc.CentralizedSamplingRule. Input to putRule().
	resARN := "*"
	serviceName := "www.foo.com"
	httpMethod := "POST"
	urlPath := "/bar/*"
	reservoirSize := int64(10)
	fixedRate := float64(0.05)
	ruleName := "r2"
	host := "local"
	priority := int64(6)
	serviceTye := "*"
	new := &xraySvc.SamplingRule{
		ServiceName:   &serviceName,
		HTTPMethod:    &httpMethod,
		URLPath:       &urlPath,
		ReservoirSize: &reservoirSize,
		FixedRate:     &fixedRate,
		RuleName:      &ruleName,
		Priority:      &priority,
		Host:          &host,
		ServiceType:   &serviceTye,
		ResourceARN:   &resARN,
	}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := m.putRule(new)
			if err != nil {
				return
			}
		}
	})

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.createUserRule(new)
		}
	})
}
