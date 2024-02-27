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

package mem

import (
	"strings"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/fstorage"
	assert "github.com/stretchr/testify/require"
)

func TestNamespacePersisterFactory_Check(t *testing.T) {
	url := "file:///tmp/ut/test1"
	assert := assert.New(t)
	f, _ := NewNamespacePersisterFactory(url, "file:///tmp/ut/test2", "igraph")
	err := f.Check()
	assert.Error(err, "have err")
	assert.EqualError(err, "start error url:file:///tmp/ut/test1/igraph is not exist .meta")

	fsl, _ := fstorage.Create(url + "/igraph")
	defer fsl.Close()

	list, _ := fsl.List()
	assert.Equal(0, len(list))
	fs, _ := fsl.Create(TouchName)
	defer fs.Remove()
	str := "hello"
	fs.Write([]byte(str))

	err = f.Check()
	assert.NoError(err, "")
	list, _ = fsl.List()
	assert.Equal(1, len(list))
	assert.Equal(TouchName, list[0])
	assert.True(strings.HasPrefix(TouchName, "."))

	data, _ := fs.Read()
	assert.Equal(str, string(data))

	err = fs.Remove()
	assert.NoError(err, "")

	assert.True(strings.HasPrefix(".meta", "."))
	assert.False(strings.HasPrefix("d2_dc", "."))
	assert.True(strings.HasPrefix("..meta", "."))

}
