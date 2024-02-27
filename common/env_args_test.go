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
	"testing"
	"time"
)

func Test_initBatcherTimeout(t *testing.T) {
	os.Setenv(batcherTimeoutKey, "1")
	initBatcherTimeout()
	timeout := GetBatcherTimeout(10 * time.Second)
	if timeout != 1*time.Second {
		t.Errorf("failed init")
	}

	os.Setenv(batcherTimeoutKey, "0")
	initBatcherTimeout()
	timeout = GetBatcherTimeout(10 * time.Second)
	if timeout != 10*time.Second {
		t.Errorf("failed init")
	}

	os.Setenv(batcherTimeoutKey, "aa")
	initBatcherTimeout()
	timeout = GetBatcherTimeout(10 * time.Second)
	if timeout != 10*time.Second {
		t.Errorf("failed init")
	}

	os.Setenv(batcherTimeoutKey, "")
	initBatcherTimeout()
	timeout = GetBatcherTimeout(10 * time.Second)
	if timeout != 10*time.Second {
		t.Errorf("failed init")
	}
}
