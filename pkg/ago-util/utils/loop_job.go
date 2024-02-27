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
	"sync"
	"time"
)

// LoopJob do job periodic
type LoopJob struct {
	ch     chan struct{}
	waiter sync.WaitGroup
}

// Start job
func (l *LoopJob) Start() {
	l.ch = make(chan struct{}, 1)
	l.waiter.Add(1)
}

// Stop job
func (l *LoopJob) Stop() {
	if l.ch == nil { // not started before
		return
	}
	l.ch <- struct{}{}
	l.waiter.Wait()
}

// CheckStop executed in loop, check if it should exit
func (l *LoopJob) CheckStop(timeout int) bool {
	select {
	case <-l.ch:
		return true
	case <-time.After(time.Duration(timeout) * time.Millisecond):
		return false
	}
}

// MarkStop mark job stop, executed when you want to stop loopjob
func (l *LoopJob) MarkStop() {
	l.waiter.Done()
}
