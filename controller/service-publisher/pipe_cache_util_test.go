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

package publisher

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func Test_Fetch(t *testing.T) {
	var wg sync.WaitGroup
	callNum := 0
	for i := 0; i < 100000; i++ {
		go fetch(&callNum, &wg, t)
	}
	wg.Wait()
	if callNum != 1 {
		t.Errorf("callNum is %d", callNum)
	}
}

func fetch(callNum *int, wg *sync.WaitGroup, t *testing.T) {
	wg.Add(1)
	defer wg.Done()
	loadFunc := func(key interface{}) (interface{}, error) {
		fmt.Println(key)
		*callNum = *callNum + 1
		return key, nil
	}
	key := getPipeCacheKey("test", "daily", "test")
	pipeCache := getPipeCache()
	result, _ := pipeCache.Fetch(key, time.Second*time.Duration(cm2CacheSeconds), loadFunc)
	if result != key {
		fmt.Println(result, key)
	}
}
