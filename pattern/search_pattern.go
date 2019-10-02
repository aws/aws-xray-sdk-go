// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.

// Package pattern provides a basic pattern matching utility.
// Patterns may contain fixed text, and/or special characters (`*`, `?`).
// `*` represents 0 or more wildcard characters. `?` represents a single wildcard character.
package pattern

import "strings"

// WildcardMatchCaseInsensitive returns true if text matches pattern (case-insensitive); returns false otherwise.
func WildcardMatchCaseInsensitive(pattern string, text string) bool {
	return WildcardMatch(pattern, text, true)
}

// WildcardMatch returns true if text matches pattern at the given case-sensitivity; returns false otherwise.
func WildcardMatch(pattern string, text string, caseInsensitive bool) bool {
	patternLen := len(pattern)
	textLen := len(text)
	if 0 == patternLen {
		return 0 == textLen
	}

	if pattern == "*" {
		return true
	}

	if caseInsensitive {
		pattern = strings.ToLower(pattern)
		text = strings.ToLower(text)
	}

	i := 0
	p := 0
	iStar := textLen
	pStar := 0

	for i < textLen {
		if p < patternLen && pattern[p] == text[i] {
			i++
			p++
		} else if p < patternLen && '?' == pattern[p] {
			i++
			p++
		} else if p < patternLen && pattern[p] == '*' {
			iStar = i
			pStar = p
			p++
		} else if iStar != textLen {
			iStar++
			i = iStar
			p = pStar + 1
		} else {
			return false
		}
	}

	for p < patternLen && pattern[p] == '*' {
		p++
	}

	return p == patternLen && i == textLen
}

func simpleWildcardMatch(pattern string, text string) bool {
	j := 0
	patternLen := len(pattern)
	textLen := len(text)
	for i := 0; i < patternLen; i++ {
		p := pattern[i]
		if '*' == p {
			return true
		} else if '?' == p {
			if textLen == j {
				return false
			}
			j++
		} else {
			if j >= textLen {
				return false
			}
			t := text[j]
			if p != t {
				return false
			}
			j++
		}
	}
	return j == textLen
}
