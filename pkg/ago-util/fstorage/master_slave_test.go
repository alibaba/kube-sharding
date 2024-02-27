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
	"strconv"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"
)

func testBackupFStore(url, backupURL string, count int, writeAfterClose bool, t *testing.T) {
	assert := assert.New(t)
	fsl, err := CreateWithBackup(url, backupURL)
	assert.Nil(err)
	fsl2URL := backupURL
	if fsl2URL == "" {
		fsl2URL = url
	}
	fsl2, err := CreateWithBackup(fsl2URL, "")

	key := "_test_key_"
	fs, err := fsl.Create(key)
	assert.Nil(err)
	str := "hello"
	bs := []byte(str)
	err = fs.Write(bs)
	for i := 0; i < count; i++ {
		fs, _ = fsl.Create(key)
		if writeAfterClose {
			defer fs.Close()
		}
		str = "hello:" + strconv.Itoa(i) + "\n"
		bs := []byte(str)
		err = fs.Write(bs)
	}
	assert.Nil(err)
	exists, err := fs.Exists()
	assert.Nil(err)
	assert.True(exists)
	rbs, err := fs.Read()
	assert.Nil(err)
	assert.Equal(str, string(rbs))
	// wait async write finished
	time.Sleep(time.Millisecond * 100)
	fs2, err := fsl2.Create(key)
	rbs2, err := fs2.Read()
	assert.Nil(err)
	assert.Equal(str, string(rbs2))
	// List
	list, err := fsl.List()
	assert.Nil(err)
	assert.Equal(1, len(list))
	assert.Equal(key, list[0])
	assert.Nil(fs.Remove())
	exists, err = fs.Exists()
	assert.Nil(err)
	assert.False(exists)
	fsl.Close()
}

func TestBackupFStorage(t *testing.T) {
	urls := [][]string{
		[]string{"file:///tmp/fstore_test", ""},
		[]string{"file:///tmp/fstore_test", "file:///tmp/fstore_test2"},
		[]string{"file:///tmp/fstore_test", "file:///tmp/fstore_test"},
	}
	for _, url := range urls {
		testBackupFStore(url[0], url[1], 0, false, t)
		testBackupFStore(url[0], url[1], 100, false, t)
		testBackupFStore(url[0], url[1], 100, true, t)
	}
}

func TestBackupFStorageGoroutineLeak(t *testing.T) {
	assert := assert.New(t)
	urls := []string{"file:///tmp/fstore_test", "file:///tmp/fstore_test2"}
	fsl, err := CreateWithBackup(urls[0], urls[1])
	assert.Nil(err)
	defer fsl.Close()
	fname := "a"
	fs, err := fsl.Create(fname)
	assert.Nil(err)
	assert.Nil(fs.Write([]byte("hello")))
	fs.Close()  // nop
	fs.Remove() // remove the cache, stop the goroutine
	// re-create
	fs, err = fsl.Create(fname)
	assert.Nil(err)
	assert.Nil(fs.Write([]byte("world")))
	b, err := fs.Read()
	assert.Nil(err)
	assert.Equal("world", string(b))
	fs.Remove()
	fs.Close()
}
