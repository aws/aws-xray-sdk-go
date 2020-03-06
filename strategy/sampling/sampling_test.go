// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

package sampling

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLocalizedStrategy(t *testing.T) {
	ss, err := NewLocalizedStrategy()
	assert.NotNil(t, ss)
	assert.Nil(t, err)
}

func TestNewLocalizedStrategyFromFilePath1(t *testing.T) { // V1 sampling
	ruleString :=
		`{
	  "version": 1,
	  "default": {
	    "fixed_target": 1,
	    "rate": 0.05
	  },
	  "rules": [
       {
        "description": "Example path-based rule below. Rules are evaluated in id-order, the default rule will be used if none match the incoming request. This is a rule for the checkout page.",
        "id": "1",
        "service_name": "*",
        "http_method": "*",
        "url_path": "/checkout",
        "fixed_target": 10,
        "rate": 0.05
       }
	  ]
	}`
	goPath := os.Getenv("PWD")
	testFile := goPath + "/test_rule.json"
	f, err := os.Create(testFile)
	if err != nil {
		panic(err)
	}
	f.WriteString(ruleString)
	f.Close()
	ss, err := NewLocalizedStrategyFromFilePath(testFile)
	assert.NotNil(t, ss)
	assert.Equal(t, 1, ss.manifest.Version)
	assert.Equal(t, 1, len(ss.manifest.Rules))
	assert.Equal(t, "", ss.manifest.Rules[0].ServiceName)
	assert.Equal(t, "*", ss.manifest.Rules[0].Host) // always host set for V1 and V2 sampling rule
	assert.Equal(t, "*", ss.manifest.Rules[0].HTTPMethod)
	assert.Equal(t, "/checkout", ss.manifest.Rules[0].URLPath)
	assert.Equal(t, int64(10), ss.manifest.Rules[0].FixedTarget)
	assert.Equal(t, 0.05, ss.manifest.Rules[0].Rate)

	assert.Nil(t, err)
	os.Remove(testFile)
}

func TestNewLocalizedStrategyFromFilePath2(t *testing.T) { // V2 sampling
	ruleString :=
		`{
	  "version": 2,
	  "default": {
	    "fixed_target": 1,
	    "rate": 0.05
	  },
	  "rules": [
       {
        "description": "Example path-based rule below. Rules are evaluated in id-order, the default rule will be used if none match the incoming request. This is a rule for the checkout page.",
        "id": "1",
        "host": "*",
        "http_method": "*",
        "url_path": "/checkout",
        "fixed_target": 10,
        "rate": 0.05
       }
	  ]
	}`
	goPath := os.Getenv("PWD")
	testFile := goPath + "/test_rule.json"
	f, err := os.Create(testFile)
	if err != nil {
		panic(err)
	}
	f.WriteString(ruleString)
	f.Close()
	ss, err := NewLocalizedStrategyFromFilePath(testFile)
	assert.NotNil(t, ss)
	assert.Equal(t, 2, ss.manifest.Version)
	assert.Equal(t, 1, len(ss.manifest.Rules))
	assert.Equal(t, "", ss.manifest.Rules[0].ServiceName)
	assert.Equal(t, "*", ss.manifest.Rules[0].Host)
	assert.Equal(t, "*", ss.manifest.Rules[0].HTTPMethod)
	assert.Equal(t, "/checkout", ss.manifest.Rules[0].URLPath)
	assert.Equal(t, int64(10), ss.manifest.Rules[0].FixedTarget)
	assert.Equal(t, 0.05, ss.manifest.Rules[0].Rate)

	assert.Nil(t, err)
	os.Remove(testFile)
}

func TestNewLocalizedStrategyFromFilePathInvalidRulesV1(t *testing.T) { // V1 contains host
	ruleString :=
		`{
	  "version": 1,
	  "default": {
	    "fixed_target": 1,
	    "rate": 0.05
	  },
	  "rules": [
       {
        "description": "Example path-based rule below. Rules are evaluated in id-order, the default rule will be used if none match the incoming request. This is a rule for the checkout page.",
        "id": "1",
        "host": "*",
        "http_method": "*",
        "url_path": "/checkout",
        "fixed_target": 10,
        "rate": 0.05
       }
	  ]
	}`
	goPath := os.Getenv("PWD")
	testFile := goPath + "/test_rule.json"
	f, err := os.Create(testFile)
	if err != nil {
		panic(err)
	}
	f.WriteString(ruleString)
	f.Close()
	ss, err := NewLocalizedStrategyFromFilePath(testFile)
	assert.Nil(t, ss)
	assert.NotNil(t, err)
	os.Remove(testFile)
}

func TestNewLocalizedStrategyFromFilePathInvalidRulesV2(t *testing.T) { // V2 contains service_name
	ruleString :=
		`{
	  "version": 2,
	  "default": {
	    "fixed_target": 1,
	    "rate": 0.05
	  },
	  "rules": [
       {
        "description": "Example path-based rule below. Rules are evaluated in id-order, the default rule will be used if none match the incoming request. This is a rule for the checkout page.",
        "id": "1",
        "service_name": "*",
        "http_method": "*",
        "url_path": "/checkout",
        "fixed_target": 10,
        "rate": 0.05
       }
	  ]
	}`
	goPath := os.Getenv("PWD")
	testFile := goPath + "/test_rule.json"
	f, err := os.Create(testFile)
	if err != nil {
		panic(err)
	}
	f.WriteString(ruleString)
	f.Close()
	ss, err := NewLocalizedStrategyFromFilePath(testFile)
	assert.Nil(t, ss)
	assert.NotNil(t, err)
	os.Remove(testFile)
}

func TestNewLocalizedStrategyFromFilePathWithInvalidJSON(t *testing.T) { // Test V1 sampling rule
	ruleString :=
		`{
	  "version": 1,
	  "default": {
	    "fixed_target": 1,
	    "rate":
	  },
	  "rules": [
	  ]
	}`
	goPath := os.Getenv("PWD")
	testFile := goPath + "/test_rule.json"
	f, err := os.Create(testFile)
	if err != nil {
		panic(err)
	}
	f.WriteString(ruleString)
	f.Close()
	ss, err := NewLocalizedStrategyFromFilePath(testFile)
	assert.Nil(t, ss)
	assert.NotNil(t, err)
	os.Remove(testFile)
}

func TestNewLocalizedStrategyFromJSONBytes(t *testing.T) {
	ruleBytes := []byte(`{
	  "version": 1,
	  "default": {
	    "fixed_target": 1,
	    "rate": 0.05
	  },
	  "rules": [
	  ]
	}`)
	ss, err := NewLocalizedStrategyFromJSONBytes(ruleBytes)
	assert.NotNil(t, ss)
	assert.Nil(t, err)
}

func TestNewLocalizedStrategyFromInvalidJSONBytes(t *testing.T) {
	ruleBytes := []byte(`{
	  "version": 1,
	  "default": {
	    "fixed_target": 1,
	    "rate":
	  },
	  "rules": [
	  ]
	}`)
	ss, err := NewLocalizedStrategyFromJSONBytes(ruleBytes)
	assert.Nil(t, ss)
	assert.NotNil(t, err)
}

// Benchmarks
func BenchmarkNewLocalizedStrategyFromJSONBytes(b *testing.B) {
	ruleBytes := []byte(`{
	  "version": 1,
	  "default": {
	    "fixed_target": 1,
	    "rate":
	  },
	  "rules": [
	  ]
	}`)
	for i := 0; i < b.N; i++ {
		_, err := NewLocalizedStrategyFromJSONBytes(ruleBytes)
		if err != nil {
			return
		}
	}
}
