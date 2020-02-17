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

type Sanitizer struct{}

// NewDefaultSanitizingStrategy initializes
// an instance of DefaultSanitizer.
func NewSanitizingStrategy() (*Sanitizer, error) {
	return &Sanitizer{}, nil
}

// DefaultSanitizer provides basic sanitizing implementation
func (ds *Sanitizer) DefaultSanitizer(san string, value string) string {
	switch san {
	case "SQL":
		buf := bytes.Buffer{}
		i := strings.Index(value, ":")
		j := strings.Index(value, "@")

		if i < j {
			str1 := value[0:i]
			str2 := value[j:]

			buf.WriteString(str1)
			buf.WriteString(str2)

			value = buf.String()
		}

		return value

	case "HTTP":
		// Code for HTTP Sanitizer
		return value

	case "AWS":
		// Code for AWS Sanitizer
		return value

	case "Metadata":
		// Code for Metadata Sanitizer
		return value

	case "Annotation":
		// Code for Annotation Sanitizer
		return value

	default:
		return "We do not support sanitizing this value :" + value

	}
}

// CustomSanitizer hooks Sanitizer logic provided by customer
func (ds *Sanitizer) CustomSanitizer(cs CustomSanitizeFunction, value string) string {

	return cs(value)
}
