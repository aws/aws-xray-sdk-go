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
	"time"

	"github.com/aws/aws-xray-sdk-go/utils"
	"github.com/stretchr/testify/assert"
)

func TestStaleRule(t *testing.T) {
	cr := &CentralizedRule{
		requests: 5,
		reservoir: &CentralizedReservoir{
			refreshedAt: 1500000000,
			interval:    10,
		},
	}

	s := cr.stale(1500000010)
	assert.True(t, s)
}

func TestFreshRule(t *testing.T) {
	cr := &CentralizedRule{
		requests: 5,
		reservoir: &CentralizedReservoir{
			refreshedAt: 1500000000,
			interval:    10,
		},
	}

	s := cr.stale(1500000009)
	assert.False(t, s)
}

func TestInactiveRule(t *testing.T) {
	cr := &CentralizedRule{
		requests: 0,
		reservoir: &CentralizedReservoir{
			refreshedAt: 1500000000,
			interval:    10,
		},
	}

	s := cr.stale(1500000011)
	assert.False(t, s)
}

func TestExpiredReservoirBernoulliSample(t *testing.T) {
	// One second past expiration
	clock := &utils.MockClock{
		NowTime: 1500000061,
	}

	// Set random to be within sampling rate
	rand := &utils.MockRand{
		F64: 0.05,
	}

	p := &Properties{
		Rate: 0.06,
	}

	// Expired reservoir
	cr := &CentralizedReservoir{
		expiresAt: 1500000060,
		borrowed:  true,
		reservoir: &reservoir{
			used:         0,
			capacity:     10,
			currentEpoch: 1500000061,
		},
	}

	csr := &CentralizedRule{
		ruleName:   "r1",
		reservoir:  cr,
		Properties: p,
		clock:      clock,
		rand:       rand,
	}

	sd := csr.Sample()

	assert.True(t, sd.Sample)
	assert.Equal(t, "r1", *sd.Rule)
	assert.Equal(t, int64(1), csr.sampled)
	assert.Equal(t, int64(1), csr.requests)
}

func TestTakeFromQuotaSample(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	// Reservoir with unused quota
	r := &reservoir{
		currentEpoch: clock.Now().Unix(),
		used:         0,
	}

	cr := &CentralizedReservoir{
		quota:     10,
		expiresAt: 1500000060,
		reservoir: r,
	}

	csr := &CentralizedRule{
		ruleName:  "r1",
		reservoir: cr,
		clock:     clock,
	}

	sd := csr.Sample()

	assert.True(t, sd.Sample)
	assert.Equal(t, "r1", *sd.Rule)
	assert.Equal(t, int64(1), csr.sampled)
	assert.Equal(t, int64(1), csr.requests)
	assert.Equal(t, int64(1), csr.reservoir.used)
}

func TestBernoulliSamplePositve(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	// Reservoir with unused quota
	r := &reservoir{
		currentEpoch: clock.Now().Unix(),
		used:         10,
	}

	// Set random to be within sampling rate
	rand := &utils.MockRand{
		F64: 0.05,
	}

	p := &Properties{
		Rate: 0.06,
	}

	cr := &CentralizedReservoir{
		quota:     10,
		expiresAt: 1500000060,
		reservoir: r,
	}

	csr := &CentralizedRule{
		ruleName:   "r1",
		reservoir:  cr,
		Properties: p,
		rand:       rand,
		clock:      clock,
	}

	sd := csr.Sample()

	assert.True(t, sd.Sample)
	assert.Equal(t, "r1", *sd.Rule)
	assert.Equal(t, int64(1), csr.sampled)
	assert.Equal(t, int64(1), csr.requests)
	assert.Equal(t, int64(10), csr.reservoir.used)
}

func TestBernoulliSampleNegative(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	// Reservoir with unused quota
	r := &reservoir{
		currentEpoch: clock.Now().Unix(),
		used:         10,
	}

	// Set random to be outside sampling rate
	rand := &utils.MockRand{
		F64: 0.07,
	}

	p := &Properties{
		Rate: 0.06,
	}

	cr := &CentralizedReservoir{
		quota:     10,
		expiresAt: 1500000060,
		reservoir: r,
	}

	csr := &CentralizedRule{
		ruleName:   "r1",
		reservoir:  cr,
		Properties: p,
		rand:       rand,
		clock:      clock,
	}

	sd := csr.Sample()

	assert.False(t, sd.Sample)
	assert.Equal(t, "r1", *sd.Rule)
	assert.Equal(t, int64(0), csr.sampled)
	assert.Equal(t, int64(1), csr.requests)
	assert.Equal(t, int64(10), csr.reservoir.used)
}

// Test sampling from local reservoir
func TestReservoirSample(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	r := &reservoir{
		capacity:     10,
		used:         5,
		currentEpoch: 1500000000,
	}

	lr := &Reservoir{
		clock:     clock,
		reservoir: r,
	}

	lsr := &Rule{
		reservoir: lr,
	}

	sd := lsr.Sample()

	assert.True(t, sd.Sample)
	assert.Nil(t, sd.Rule)
	assert.Equal(t, int64(6), lsr.reservoir.used)
}

// Test bernoulli sampling for local sampling rule
func TestLocalBernoulliSample(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	r := &reservoir{
		capacity:     10,
		used:         10,
		currentEpoch: 1500000000,
	}

	lr := &Reservoir{
		clock:     clock,
		reservoir: r,
	}

	// 6% sampling rate
	p := &Properties{
		Rate: 0.06,
	}

	// Set random to be outside sampling rate
	rand := &utils.MockRand{
		F64: 0.07,
	}

	lsr := &Rule{
		reservoir:  lr,
		rand:       rand,
		Properties: p,
	}

	sd := lsr.Sample()

	assert.False(t, sd.Sample)
	assert.Nil(t, sd.Rule)
	assert.Equal(t, int64(10), lsr.reservoir.used)
}

func TestSnapshot(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	csr := &CentralizedRule{
		ruleName: "rule1",
		requests: 100,
		sampled:  12,
		borrows:  2,
		clock:    clock,
	}

	ss := csr.snapshot()

	// Assert counters were reset
	assert.Equal(t, int64(0), csr.requests)
	assert.Equal(t, int64(0), csr.sampled)
	assert.Equal(t, int64(0), csr.borrows)

	// Assert on SamplingStatistics counters
	now := time.Unix(1500000000, 0)

	assert.Equal(t, int64(100), *ss.RequestCount)
	assert.Equal(t, int64(12), *ss.SampledCount)
	assert.Equal(t, int64(2), *ss.BorrowCount)
	assert.Equal(t, "rule1", *ss.RuleName)
	assert.Equal(t, now, *ss.Timestamp)
}

// Benchmarks
func BenchmarkCentralizedRule_Sample(b *testing.B) {

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			clock := &utils.MockClock{
				NowTime: 1500000000,
			}

			// Reservoir with unused quota
			r := &reservoir{
				currentEpoch: clock.Now().Unix(),
				used:         0,
			}

			cr := &CentralizedReservoir{
				quota:     10,
				expiresAt: 1500000060,
				reservoir: r,
			}

			csr := &CentralizedRule{
				ruleName:  "r1",
				reservoir: cr,
				clock:     clock,
			}
			csr.Sample()
		}
	})
}
