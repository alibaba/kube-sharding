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

	assert "github.com/stretchr/testify/require"
)

func TestCompress(t *testing.T) {
	assert := assert.New(t)
	url := "file:///tmp/fstore_comp"
	fsl, err := Create(url)
	assert.Nil(err)
	key := "_test_key_"
	fs, err := fsl.Create(key)
	assert.Nil(err)
	cs := CompressStorage{fs: fs}
	str := "hello"
	for i := 0; i < 2000; i++ {
		str += strconv.Itoa(i)
	}
	assert.Nil(cs.Write([]byte(str)))
	bs, err := cs.Read()
	assert.Nil(err)
	assert.Equal(str, string(bs))
}
