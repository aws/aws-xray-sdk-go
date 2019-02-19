// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package sampling

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	xraySvc "github.com/aws/aws-sdk-go/service/xray"
	"github.com/aws/aws-xray-sdk-go/utils"
)

const defaultRule = "Default"
const defaultInterval = int64(10)

const manifestTTL = 3600 // Seconds

// CentralizedManifest represents a full sampling ruleset, with a list of
// custom rules and default values for incoming requests that do
// not match any of the provided rules.
type CentralizedManifest struct {
	Default     *CentralizedRule
	Rules       []*CentralizedRule
	Index       map[string]*CentralizedRule
	refreshedAt int64
	clock       utils.Clock
	sync.RWMutex
}

// putRule updates the named rule if it already exists or creates it if it does not.
// May break ordering of the sorted rules array if it creates a new rule.
func (m *CentralizedManifest) putRule(svcRule *xraySvc.SamplingRule) (r *CentralizedRule, err error) {
	defer func() {
		if x := recover(); x != nil {
			err = fmt.Errorf("%v", x)
		}
	}()

	name := *svcRule.RuleName

	// Default rule
	if name == defaultRule {
		m.RLock()
		r = m.Default
		m.RUnlock()

		// Update rule if already exists
		if r != nil {
			m.updateDefaultRule(svcRule)

			return
		}

		// Create Default rule
		r = m.createDefaultRule(svcRule)

		return
	}

	// User-defined rule
	m.RLock()
	r, ok := m.Index[name]
	m.RUnlock()

	// Create rule if it does not exist
	if !ok {
		r = m.createUserRule(svcRule)

		return
	}

	// Update existing rule
	m.updateUserRule(r, svcRule)

	return
}

// createUserRule creates a user-defined CentralizedRule, appends it to the sorted array,
// adds it to the index, and returns the newly created rule.
// Appends new rule to the sorted array which may break its ordering.
// Panics if svcRule contains nil pointers
func (m *CentralizedManifest) createUserRule(svcRule *xraySvc.SamplingRule) *CentralizedRule {
	// Create CentralizedRule from xraySvc.SamplingRule
	clock := &utils.DefaultClock{}
	rand := &utils.DefaultRand{}

	p := &Properties{
		ServiceName: *svcRule.ServiceName,
		HTTPMethod:  *svcRule.HTTPMethod,
		URLPath:     *svcRule.URLPath,
		FixedTarget: *svcRule.ReservoirSize,
		Rate:        *svcRule.FixedRate,
		Host:        *svcRule.Host,
	}

	r := &reservoir{
		capacity: *svcRule.ReservoirSize,
	}

	cr := &CentralizedReservoir{
		reservoir: r,
		interval:  defaultInterval,
	}

	csr := &CentralizedRule{
		ruleName:    *svcRule.RuleName,
		priority:    *svcRule.Priority,
		reservoir:   cr,
		Properties:  p,
		serviceType: *svcRule.ServiceType,
		resourceARN: *svcRule.ResourceARN,
		attributes:  svcRule.Attributes,
		clock:       clock,
		rand:        rand,
	}

	m.Lock()
	defer m.Unlock()

	// Return early if rule already exists
	if r, ok := m.Index[*svcRule.RuleName]; ok {
		return r
	}

	// Update sorted array
	m.Rules = append(m.Rules, csr)

	// Update index
	m.Index[*svcRule.RuleName] = csr

	return csr
}

// updateUserRule updates the properties of the user-defined CentralizedRule using the given
// xraySvc.SamplingRule.
// Panics if svcRule contains nil pointers.
func (m *CentralizedManifest) updateUserRule(r *CentralizedRule, svcRule *xraySvc.SamplingRule) {
	// Preemptively dereference xraySvc.SamplingRule fields and panic early on nil pointers.
	// A panic in the middle of an update may leave the rule in an inconsistent state.
	pr := &Properties{
		ServiceName: *svcRule.ServiceName,
		HTTPMethod:  *svcRule.HTTPMethod,
		URLPath:     *svcRule.URLPath,
		FixedTarget: *svcRule.ReservoirSize,
		Rate:        *svcRule.FixedRate,
		Host:        *svcRule.Host,
	}

	p, c := *svcRule.Priority, *svcRule.ReservoirSize

	r.Lock()
	defer r.Unlock()

	r.Properties = pr
	r.priority = p
	r.reservoir.capacity = c
	r.serviceType = *svcRule.ServiceType
	r.resourceARN = *svcRule.ResourceARN
	r.attributes = svcRule.Attributes
}

// createDefaultRule creates a default CentralizedRule and adds it to the manifest.
// Panics if svcRule contains nil values for FixedRate and ReservoirSize.
func (m *CentralizedManifest) createDefaultRule(svcRule *xraySvc.SamplingRule) *CentralizedRule {
	// Create CentralizedRule from xraySvc.SamplingRule
	clock := &utils.DefaultClock{}
	rand := &utils.DefaultRand{}

	p := &Properties{
		FixedTarget: *svcRule.ReservoirSize,
		Rate:        *svcRule.FixedRate,
	}

	r := &reservoir{
		capacity: *svcRule.ReservoirSize,
	}

	cr := &CentralizedReservoir{
		reservoir: r,
		interval:  defaultInterval,
	}

	csr := &CentralizedRule{
		ruleName:   *svcRule.RuleName,
		reservoir:  cr,
		Properties: p,
		clock:      clock,
		rand:       rand,
	}

	m.Lock()
	defer m.Unlock()

	// Return early if rule already exists
	if d := m.Default; d != nil {
		return d
	}

	// Update manifest if rule does not exist
	m.Default = csr

	// Update index
	m.Index[*svcRule.RuleName] = csr

	return csr
}

// updateDefaultRule updates the properties of the default CentralizedRule using the given
// xraySvc.SamplingRule.
// Panics if svcRule contains nil values for FixedRate and ReservoirSize.
func (m *CentralizedManifest) updateDefaultRule(svcRule *xraySvc.SamplingRule) {
	r := m.Default

	// Preemptively dereference xraySvc.SamplingRule fields and panic early on nil pointers.
	// A panic in the middle of an update may leave the rule in an inconsistent state.
	p := &Properties{
		FixedTarget: *svcRule.ReservoirSize,
		Rate:        *svcRule.FixedRate,
	}

	c := *svcRule.ReservoirSize

	r.Lock()
	defer r.Unlock()

	r.Properties = p
	r.reservoir.capacity = c
}

// prune removes all rules in the manifest not present in the given list of active rules.
// Preserves ordering of sorted array.
func (m *CentralizedManifest) prune(actives map[*CentralizedRule]bool) {
	m.Lock()
	defer m.Unlock()

	// Iterate in reverse order to avoid adjusting index for each deleted rule
	for i := len(m.Rules) - 1; i >= 0; i-- {
		r := m.Rules[i]

		if _, ok := actives[r]; !ok {
			m.deleteRule(i)
		}
	}
}

// deleteRule deletes the rule from the array, and the index.
// Assumes write lock is already held.
// Preserves ordering of sorted array.
func (m *CentralizedManifest) deleteRule(idx int) {
	// Remove from index
	delete(m.Index, m.Rules[idx].ruleName)

	// Delete by reslicing without index
	a := append(m.Rules[:idx], m.Rules[idx+1:]...)

	// Set pointer to nil to free capacity from underlying array
	m.Rules[len(m.Rules)-1] = nil

	// Assign resliced rules
	m.Rules = a
}

// sort sorts the rule array first by priority and then by rule name.
func (m *CentralizedManifest) sort() {
	// Comparison function
	less := func(i, j int) bool {
		if m.Rules[i].priority == m.Rules[j].priority {
			return strings.Compare(m.Rules[i].ruleName, m.Rules[j].ruleName) < 0
		}
		return m.Rules[i].priority < m.Rules[j].priority
	}

	m.Lock()
	defer m.Unlock()

	sort.Slice(m.Rules, less)
}

// expired returns true if the manifest has not been successfully refreshed in
// 'manifestTTL' seconds.
func (m *CentralizedManifest) expired() bool {
	m.RLock()
	defer m.RUnlock()

	return m.refreshedAt < m.clock.Now().Unix()-manifestTTL
}
