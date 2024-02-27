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
	"os"
	"path/filepath"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"
)

func TestDiffLogger(t *testing.T) {
	assert := assert.New(t)
	fileCnt := 10
	logger, err := NewDiffLogger(1, func(msg string) { fmt.Println(msg) }, fileCnt, -1)
	assert.Nil(err)
	path := "/tmp/history"
	assert.Nil(os.RemoveAll(path))
	suffix := "status"
	k := "k1"
	obj := "hello"
	assert.Nil(logger.WriteHistoryFile(path, k, suffix, obj, nil))
	time.Sleep(1200 * time.Millisecond)
	assert.Nil(logger.WriteHistoryFile(path, k, suffix, obj, nil))
	verifyFileCount := func(n int) {
		files, err := filepath.Glob(path + "/*" + suffix)
		assert.Nil(err)
		assert.Equal(n, len(files))
	}
	// no more files because content is same
	verifyFileCount(1)
	// evict k, and 2 files written
	logger.Log("k2", "hello world")
	time.Sleep(1200 * time.Millisecond)
	assert.Nil(logger.WriteHistoryFile(path, k, suffix, obj, nil))
	verifyFileCount(2)
}
