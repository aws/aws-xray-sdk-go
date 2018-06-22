// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Rand is an interface for a set of methods that return random value.
type Rand interface {
	Int63n(n int64) int64
	Intn(n int) int
	Float64() float64
}

// DefaultRand is an implementation of Rand interface.
type DefaultRand struct{}

// Int63n returns, as an int64, a non-negative pseudo-random number in [0,n)
// from the default Source.
func (r *DefaultRand) Int63n(n int64) int64 {
	return rand.Int63n(n)
}

// Intn returns, as an int, a non-negative pseudo-random number in [0,n)
// from the default Source.
func (r *DefaultRand) Intn(n int) int {
	return rand.Intn(n)
}

// Float64 returns, as a float64, a pseudo-random number in [0.0,1.0)
// from the default Source.
func (r *DefaultRand) Float64() float64 {
	return rand.Float64()
}
