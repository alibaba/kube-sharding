/*
Copyright 2024 The Alibaba Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metric

import (
	"regexp"
	"time"
)

// Default kmon config
const (
	KmonDefaultAddress        = "localhost"
	KmonDefaultPort           = "4141"
	SummaryMaxAge             = time.Second * 30
	SummaryAgeBuckets  uint32 = 3
)

var _reg *regexp.Regexp = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// EscapeLabelKey escape key
func EscapeLabelKey(src string) string {
	dst := _reg.ReplaceAllString(src, "_")
	if "" == dst {
		dst = "null"
	}
	return dst
}
