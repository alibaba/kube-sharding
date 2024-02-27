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

// Int32Ptr get ptr of value
func Int32Ptr(v int32) *int32 {
	return &v
}

// Int64Ptr get ptr of value
func Int64Ptr(v int64) *int64 {
	return &v
}

// StringPtr get ptr of value
func StringPtr(v string) *string {
	return &v
}

// BoolPtr get ptr of value
func BoolPtr(v bool) *bool {
	return &v
}
