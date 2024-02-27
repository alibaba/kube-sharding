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

package set

// StringSet is a string set.
type StringSet map[string]struct{}

// StringSetFromList create StringSet from a list of strings
func StringSetFromList(ss []string) StringSet {
	set := NewStringSet()
	for _, s := range ss {
		set.Insert(s)
	}
	return set
}

// NewStringSet builds a string set.
func NewStringSet() StringSet {
	return make(map[string]struct{})
}

// Exist checks whether `val` exists in `s`.
func (s StringSet) Exist(val string) bool {
	_, ok := s[val]
	return ok
}

// Insert inserts `val` into `s`.
func (s StringSet) Insert(val string) {
	s[val] = struct{}{}
}
