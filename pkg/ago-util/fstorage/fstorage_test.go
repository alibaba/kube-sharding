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

package fstorage

import (
	"testing"

	assert "github.com/stretchr/testify/require"
)

func testFStore(url string, t *testing.T) {
	assert := assert.New(t)
	fsl, err := CreateWithBackup(url, "")
	assert.Nil(err)
	key := "_test_key_"
	fs, err := fsl.Create(key)
	assert.Nil(err)
	str := "hello"
	bs := []byte(str)
	err = fs.Write(bs)
	assert.Nil(err)
	exists, err := fs.Exists()
	assert.Nil(err)
	assert.True(exists)
	rbs, err := fs.Read()
	assert.Nil(err)
	assert.Equal(str, string(rbs))
	// List
	list, err := fsl.List()
	assert.Nil(err)
	assert.Equal(1, len(list))
	assert.Equal(key, list[0])
	assert.Nil(fs.Remove())
	exists, err = fs.Exists()
	assert.Nil(err)
	assert.False(exists)
}

func TestFStorage(t *testing.T) {
	urls := []string{
		"file:///tmp/fstore_test",
	}
	for _, url := range urls {
		testFStore(url, t)
	}
}

func TestJoinPath(t *testing.T) {
	assert := assert.New(t)
	url := JoinPath("file:///tmp/test", "a")
	assert.Equal("file:///tmp/test/a", url)
	url = JoinPath("file:///tmp/test", "a", "b", "c")
	assert.Equal("file:///tmp/test/a/b/c", url)
	url = JoinPath("zfs://127.0.0.1:10181/test", "a", "b")
	assert.Equal("zfs://127.0.0.1:10181/test/a/b", url)
	url = JoinPath("zfs://127.0.0.1:10181/test")
	assert.Equal("zfs://127.0.0.1:10181/test", url)
	url = JoinPath("127.0.0.1:10181/test", "a")
	assert.Equal("127.0.0.1:10181/test", url)
}
