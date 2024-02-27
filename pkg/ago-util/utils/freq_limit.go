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
	"sync/atomic"
)

// FreqLimiter limits to run a function.
type FreqLimiter struct {
	counter int32
	max     int32
}

// NewFreqLimiter creates a limiter
func NewFreqLimiter(max int32) *FreqLimiter {
	return &FreqLimiter{
		counter: 0,
		max:     max,
	}
}

// Run runs the function
func (f *FreqLimiter) Run(fn func()) bool {
	curr := atomic.LoadInt32(&f.counter)
	if f.max != 0 && curr >= f.max {
		return false
	}
	atomic.AddInt32(&f.counter, 1)
	defer atomic.AddInt32(&f.counter, -1)
	fn()
	return true
}
