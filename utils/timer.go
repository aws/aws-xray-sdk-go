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

type timer struct {
	t      *time.Timer
	d      time.Duration
	jitter time.Duration
}

func NewTimer(d, jitter time.Duration) *timer {
	t := time.NewTimer(d - time.Duration(rand.Int63n(int64(jitter))))

	jitteredTimer := timer{
		t:      t,
		d:      d,
		jitter: jitter,
	}

	return &jitteredTimer
}

func (j *timer) C() <-chan time.Time {
	return j.t.C
}

func (j *timer) Reset() {
	j.t.Reset(j.d - time.Duration(rand.Int63n(int64(j.jitter))))
}
