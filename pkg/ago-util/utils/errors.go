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
	"errors"
	"strings"
)

// Errors used for collect errors of parallel process
type Errors []error

// NewErrors create errors
func NewErrors() Errors {
	errors := Errors(make([]error, 0))
	return errors
}

// FromErrors create Errors by a list of errors
func FromErrors(errs []error) Errors {
	es := NewErrors()
	for _, err := range errs {
		es.Add(err)
	}
	return es
}

// Add new error to errors
func (e *Errors) Add(err error) {
	if nil == *e {
		*e = NewErrors()
	}
	if nil == err {
		return
	}
	*e = append(*e, err)
}

// Empty is errors empty
func (e *Errors) Empty() bool {
	return 0 == len(*e)
}

// String is errors empty
func (e *Errors) String() string {
	if nil == *e {
		return ""
	}
	errors := []error(*e)
	if 0 == len(errors) {
		return ""
	}
	var b strings.Builder
	for i := range errors {
		b.WriteString(errors[i].Error())
		if i != (len(errors) - 1) {
			b.WriteString("\r\n")
		}
	}
	return b.String()
}

// String is errors empty
func (e *Errors) Error() error {
	if nil == *e {
		return nil
	}
	str := e.String()
	if "" == str {
		return nil
	}
	return errors.New(str)
}
