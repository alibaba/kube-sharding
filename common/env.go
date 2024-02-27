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

package common

import (
	"os"
	"strings"

	"github.com/pborman/uuid"
)

// GetRequestID 请求id
func GetRequestID() string {
	return GetCommonID()
}

// GetCommonID 通用id
func GetCommonID() string {
	uuidString := uuid.NewUUID().String()
	return uuidString[0:13]
}

// DeleteFromSlice delete a string from slice
func DeleteFromSlice(src string, from []string) []string {
	var result = make([]string, 0, len(from))
	for i := range from {
		if from[i] != src {
			result = append(result, from[i])
		}
	}
	return result
}

// AddToUniqueSlice add a string to slice unique
func AddToUniqueSlice(src string, to []string) []string {
	var result = make([]string, 0, len(to)+1)
	contains := false
	for i := range to {
		if to[i] == src {
			contains = true
			break
		}
	}
	if !contains {
		result = append(to, src)
	} else {
		result = to
	}
	return result
}

// GetEnvWithDefault get env if default is nil
func GetEnvWithDefault(defaultValue, envkey string) string {
	if "" != defaultValue {
		return defaultValue
	}
	env := os.Getenv(envkey)
	return env
}

// CorrecteShellExpr CorrecteShellExpr
func CorrecteShellExpr(src string) string {
	var dstSB strings.Builder
	var srcBytes = []byte(src)
	var foundStart = false
	var startPos = 0
	var len = len(srcBytes)
	var pre byte
	var cur byte
	for i := range srcBytes {
		cur = srcBytes[i]
		if !foundStart {
			if cur == '$' || i == len-1 {
				index := i
				if i == len-1 {
					index++
				}
				dstSB.Write(srcBytes[startPos:index])
				startPos = i + 1
				foundStart = true
			}
		} else {
			var foundEnd = false
			if cur == ':' || cur == '/' || cur == ' ' || cur == '}' || cur == ')' || (pre != '$' && cur == '$') {
				foundEnd = true
			}
			if i == len-1 || foundEnd {
				index := i
				if !foundEnd {
					index++
				}
				key := srcBytes[startPos:index]
				dstSB.Write([]byte{'$', '('})
				dstSB.Write(key)
				dstSB.WriteByte(')')
				foundStart = false
				startPos = i
				if cur == '$' {
					foundStart = true
				}
				if cur == '$' || cur == '}' || cur == ')' {
					startPos = i + 1
				}
			} else if pre == '$' && (cur == '{' || cur == '(' || cur == '$') {
				startPos = i + 1
				continue
			}
		}
		pre = cur
	}
	return dstSB.String()
}
