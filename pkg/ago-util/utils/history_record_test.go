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
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"
)

func TestHistoryRecord(t *testing.T) {
	assert := assert.New(t)
	path := "/tmp/history"
	assert.Nil(os.RemoveAll(path))
	k := "h"
	suffix := "target"
	type st struct {
		Idx int
	}
	for i := 0; i < 5; i++ {
		if err := WriteHistoryRecord(path, k, suffix, 1, 1, &st{i}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(1200 * time.Millisecond)
	}
	files, err := filepath.Glob(path + "/*" + suffix)
	assert.Nil(err)
	assert.Equal(1, len(files))
	dat, err := ioutil.ReadFile(files[0])
	assert.Nil(err)
	var s st
	assert.Nil(json.Unmarshal(dat, &s))
	assert.Equal(4, s.Idx)

	// remove old files
	for i := 0; i < 2; i++ {
		if err := WriteHistoryRecord(path, k, suffix, 10, 10, &st{i}); err != nil {
			t.Fatal(err)
		}
		time.Sleep(1200 * time.Millisecond)
	}
	files, err = filepath.Glob(path + "/*" + suffix)
	assert.Nil(err)
	assert.Equal(3, len(files)) // 2 + 1
	assert.Nil(WriteHistoryRecord(path, k, suffix, 10, 1, &st{7}))
	files, err = filepath.Glob(path + "/*" + suffix)
	assert.Nil(err)
	assert.Equal(1, len(files))
	// verify content
	dat, err = ioutil.ReadFile(files[0])
	assert.Nil(err)
	assert.Nil(json.Unmarshal(dat, &s))
	assert.Equal(7, s.Idx) // the latest file keeped
}
