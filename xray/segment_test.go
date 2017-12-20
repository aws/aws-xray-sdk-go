// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package xray

import (
	"context"
	"testing"
)

func TestSegmentDataRace(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	for i := 0; i < 10; i += 1 { // flakey data race test, so we run it multiple times
		_, seg := BeginSegment(ctx, "TestSegment")

		go seg.Close(nil)
		cancel()
	}
}
