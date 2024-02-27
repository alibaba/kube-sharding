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
	"fmt"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"
)

func TestFreqLimiter(t *testing.T) {
	assert := assert.New(t)
	max := 5
	l := NewFreqLimiter(int32(max))
	ch := make(chan struct{}, max)
	worker := func(i int) func() {
		return func() {
			fmt.Printf("goroutin %d start\n", i)
			<-ch
			fmt.Printf("goroutin %d exit\n", i)
		}
	}
	for i := 0; i < max; i++ {
		go l.Run(worker(i))
	}
	time.Sleep(100 * time.Millisecond)
	assert.Equal(int32(max), l.counter)
	assert.False(l.Run(func() {}))
	for i := 0; i < max; i++ {
		ch <- struct{}{}
	}
	time.Sleep(100 * time.Millisecond)
	assert.Equal(int32(0), l.counter)
	assert.True(l.Run(func() {}))
	close(ch)
}
