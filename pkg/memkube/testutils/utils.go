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

package testutils

import (
	"github.com/alibaba/kube-sharding/pkg/ago-util/fstorage"
	"k8s.io/apimachinery/pkg/watch"
)

// ClearLocation clear all files in location.
func ClearLocation(fsl fstorage.Location) error {
	names, err := fsl.List()
	if err != nil {
		return err
	}
	for _, name := range names {
		fs, err := fsl.New(name)
		if err != nil {
			return err
		}
		if err = fs.Remove(); err != nil {
			return err
		}
	}
	return nil
}

// ConsumeWatchEvents consumes watch event from a watch.Interface
func ConsumeWatchEvents(w watch.Interface, stopCh chan struct{}) {
	CollectWatchEvents(w, nil, stopCh)
}

// CollectWatchEvents collect events to specified channel.
func CollectWatchEvents(w watch.Interface, outch chan watch.Event, stopCh chan struct{}) {
	ch := w.ResultChan()
	go func() {
		defer w.Stop()
		defer func() {
			if outch != nil {
				close(outch)
			}
		}()
		for {
			select {
			case <-stopCh:
				return
			case w, ok := <-ch:
				if !ok {
					return
				}
				if outch != nil {
					outch <- w
				}
			}
		}
	}()
}

// FuncMockMatcher used in gomock as a matcher.
type FuncMockMatcher struct {
	Func func(x interface{}) bool
	Name string
}

// Matches check if the argument matches.
func (m FuncMockMatcher) Matches(x interface{}) bool {
	return m.Func(x)
}

// String return the stringify for this matcher
func (m FuncMockMatcher) String() string {
	return m.Name
}
