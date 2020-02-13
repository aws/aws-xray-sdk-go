// Copyright 2017-2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance with the License. A copy of the License is located at
//
//     http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
package sanitizing

import (
	"bytes"
	"strings"
)

type DefaultSanitizer struct{}

// NewDefaultSanitizingStrategy initializes
// an instance of DefaultSanitizer.
func NewDefaultSanitizingStrategy() (*DefaultSanitizer, error) {
	return &DefaultSanitizer{}, nil
}

// returns sanitized sql dsn
func (ds *DefaultSanitizer) SQLSanitizer(san string) string {
	buf := bytes.Buffer{}
	i := strings.Index(san, ":")
	j := strings.Index(san, "@")

	if i < j {
		str1 := san[0:i]
		str2 := san[j:len(san)]

		buf.WriteString(str1)
		buf.WriteString(str2)

		san = buf.String()
	}

	return san
}

func (ds *DefaultSanitizer) HTTPSanitizer(san string) string {
	// Code for HTTP Sanitizer
	return ""
}

func (ds *DefaultSanitizer) AWSSanitizer(san string) string {
	// Code for AWS Sanitizer
	return ""
}

func (ds *DefaultSanitizer) MetadataSanitizer(san string) string {
	// Code for metadata Sanitizer
	return ""
}
