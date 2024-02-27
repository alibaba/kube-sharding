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

package utils

import (
	"encoding/json"
	"log"
	"strings"
	"unsafe"
)

// AppendString link strings
func AppendString(strs ...string) string {
	var b strings.Builder
	for i := range strs {
		b.WriteString(strs[i])
	}
	return b.String()
}

// StringCastByte change string to byte
func StringCastByte(s string) []byte {
	return *(*[]byte)(unsafe.Pointer(&s))
}

// ByteCastString change byte to string
func ByteCastString(b []byte) string {
	s := *(*string)(unsafe.Pointer(&b))
	return s
}

// ObjJSON get obj json string. easy to use but not safety, used for log
func ObjJSON(o interface{}) string {
	b, err := json.Marshal(o)
	if nil != err {
		log.Printf("marshal error :%v", err)
		return ""
	}
	return ByteCastString(b)
}

// GetMaxLengthCommonString GetMaxLengthCommonString
func GetMaxLengthCommonString(str1, str2 string) string {
	chs1 := len(str1)
	chs2 := len(str2)

	maxLength := 0 //记录最大长度
	end := 0       //记录最大长度的结尾位置

	rows := 0
	cols := chs2 - 1
	for rows < chs1 {
		i, j := rows, cols
		length := 0 //记录长度
		for i < chs1 && j < chs2 {
			if str1[i] != str2[j] {
				length = 0
			} else {
				length++
			}
			if length > maxLength {
				end = i
				maxLength = length
			}
			i++
			j++
		}
		if cols > 0 {
			cols--
		} else {
			rows++
		}
	}
	return string(str1[(end - maxLength + 1):(end + 1)])
}

// GetMaxLengthSubRepeatString GetMaxLengthSubRepeatString
func GetMaxLengthSubRepeatString(str string) string {
	strn := []rune(str)
	n := len(strn)
	if 0 == n {
		return ""
	}
	start := 0
	maxLength := 0
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			length := 0
			for k := 0; k+j < n; k++ {
				if strn[j+k] == strn[i+k] {
					length++
				} else {
					break
				}
			}
			if length > maxLength {
				maxLength = length
				start = i
			}
		}
	}
	return str[start : start+maxLength]
}
