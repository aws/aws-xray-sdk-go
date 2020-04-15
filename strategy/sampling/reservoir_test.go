// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package sampling

import (
	"math"
	"testing"

	"github.com/aws/aws-xray-sdk-go/utils"
	"github.com/stretchr/testify/assert"
)

const Interval = 100

func takeOverTime(r *Reservoir, millis int) int {
	taken := 0
	for i := 0; i < millis/Interval; i++ {
		if r.Take() {
			taken++
		}
		r.clock.Increment(0, 1e6*Interval)
	}
	return taken
}

const TestDuration = 1500

// Asserts consumption from reservoir once per second
func TestOnePerSecond(t *testing.T) {
	clock := &utils.MockClock{}
	cap := 1
	res := &Reservoir{
		clock: clock,
		reservoir: &reservoir{
			capacity: int64(cap),
		},
	}
	taken := takeOverTime(res, TestDuration)
	assert.True(t, int(math.Ceil(TestDuration/1000.0)) == taken)
}

// Asserts consumption from reservoir ten times per second
func TestTenPerSecond(t *testing.T) {
	clock := &utils.MockClock{}
	cap := 10
	res := &Reservoir{
		clock: clock,
		reservoir: &reservoir{
			capacity: int64(cap),
		},
	}
	taken := takeOverTime(res, TestDuration)
	assert.True(t, int(math.Ceil(float64(TestDuration*cap)/1000.0)) == taken)
}

func TestTakeQuotaAvailable(t *testing.T) {
	capacity := int64(100)
	used := int64(0)
	quota := int64(9)

	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	r := &CentralizedReservoir{
		quota: quota,
		reservoir: &reservoir{
			capacity:     capacity,
			used:         used,
			currentEpoch: clock.Now().Unix(),
		},
	}

	s := r.Take(clock.Now().Unix())
	assert.Equal(t, true, s)
	assert.Equal(t, int64(1), r.used)
}

func TestTakeQuotaUnavailable(t *testing.T) {
	capacity := int64(100)
	used := int64(100)
	quota := int64(9)

	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	r := &CentralizedReservoir{
		quota: quota,
		reservoir: &reservoir{
			capacity:     capacity,
			used:         used,
			currentEpoch: clock.Now().Unix(),
		},
	}

	s := r.Take(clock.Now().Unix())
	assert.Equal(t, false, s)
	assert.Equal(t, int64(100), r.used)
}

func TestExpiredReservoir(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500000001,
	}

	r := &CentralizedReservoir{
		expiresAt: 1500000000,
	}

	expired := r.expired(clock.Now().Unix())

	assert.Equal(t, true, expired)
}

// Assert that the borrow flag is reset every second
func TestBorrowFlagReset(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	r := &CentralizedReservoir{
		reservoir: &reservoir{
			capacity: 10,
		},
	}

	s := r.borrow(clock.Now().Unix())
	assert.True(t, s)

	s = r.borrow(clock.Now().Unix())
	assert.False(t, s)

	// Increment clock by 1
	clock = &utils.MockClock{
		NowTime: 1500000001,
	}

	// Reset borrow flag
	r.Take(clock.Now().Unix())

	s = r.borrow(clock.Now().Unix())
	assert.True(t, s)
}

// Assert that the reservoir does not allow borrowing if the reservoir capacity
// is zero.
func TestBorrowZeroCapacity(t *testing.T) {
	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	r := &CentralizedReservoir{
		reservoir: &reservoir{
			capacity: 0,
		},
	}

	s := r.borrow(clock.Now().Unix())
	assert.False(t, s)
}

func TestResetQuotaUsageRotation(t *testing.T) {
	capacity := int64(100)
	used := int64(0)
	quota := int64(5)

	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	r := &CentralizedReservoir{
		quota: quota,
		reservoir: &reservoir{
			capacity:     capacity,
			used:         used,
			currentEpoch: clock.Now().Unix(),
		},
	}

	// Consume quota for second
	for i := 0; i < 5; i++ {
		taken := r.Take(clock.Now().Unix())
		assert.Equal(t, true, taken)
		assert.Equal(t, int64(i+1), r.used)
	}

	// Take() should be false since no unused quota left
	taken := r.Take(clock.Now().Unix())
	assert.Equal(t, false, taken)
	assert.Equal(t, int64(5), r.used)

	// Increment epoch to reset unused quota
	clock = &utils.MockClock{
		NowTime: 1500000001,
	}

	// Take() should be true since ununsed quota is available
	taken = r.Take(clock.Now().Unix())
	assert.Equal(t, int64(1500000001), r.currentEpoch)
	assert.Equal(t, true, taken)
	assert.Equal(t, int64(1), r.used)
}

func TestResetReservoirUsageRotation(t *testing.T) {
	capacity := int64(5)
	used := int64(0)

	clock := &utils.MockClock{
		NowTime: 1500000000,
	}

	r := &Reservoir{
		clock: clock,
		reservoir: &reservoir{
			capacity:     capacity,
			used:         used,
			currentEpoch: clock.Now().Unix(),
		},
	}

	// Consume reservoir for second
	for i := 0; i < 5; i++ {
		taken := r.Take()
		assert.Equal(t, true, taken)
		assert.Equal(t, int64(i+1), r.used)
	}

	// Take() should be false since no reservoir left
	taken := r.Take()
	assert.Equal(t, false, taken)
	assert.Equal(t, int64(5), r.used)

	// Increment epoch to reset reservoir
	clock = &utils.MockClock{
		NowTime: 1500000001,
	}
	r.clock = clock

	// Take() should be true since reservoir is available
	taken = r.Take()
	assert.Equal(t, int64(1500000001), r.currentEpoch)
	assert.Equal(t, true, taken)
	assert.Equal(t, int64(1), r.used)
}
