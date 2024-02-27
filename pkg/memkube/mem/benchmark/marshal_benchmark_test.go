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

package benchmark

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	carbon "github.com/alibaba/kube-sharding/pkg/memkube/mem/benchmark/testdata"
	assert "github.com/stretchr/testify/require"
)

func TestMarshalLoadingWorkerNode(t *testing.T) {
	file, _ := ioutil.ReadFile("testdata/workernode2.json")
	var workerNode carbon.WorkerNode
	err := json.Unmarshal(file, &workerNode)
	assert.Nil(t, err)
}

func Benchmark4JsonMarshal2(b *testing.B) {
	file, err := ioutil.ReadFile("testdata/workernode2.json")
	assert.Nil(b, err)
	var workerNode carbon.WorkerNode
	_ = json.Unmarshal(file, &workerNode)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(workerNode)
	}
}

func Benchmark4PbMarshal2(b *testing.B) {
	file, err := ioutil.ReadFile("testdata/workernode2.json")
	assert.Nil(b, err)
	var workerNode carbon.WorkerNode
	_ = json.Unmarshal(file, &workerNode)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs := make([]byte, 0, workerNode.XXX_Size())
		workerNode.XXX_Marshal(bs, false)
	}
}

func Benchmark4JsonMarshal1(b *testing.B) {
	file, err := ioutil.ReadFile("testdata/workernode.json")
	assert.Nil(b, err)
	var workerNode carbon.WorkerNode
	_ = json.Unmarshal(file, &workerNode)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		json.Marshal(workerNode)
	}
}

func Benchmark4PbMarshal1(b *testing.B) {
	file, err := ioutil.ReadFile("testdata/workernode.json")
	assert.Nil(b, err)
	var workerNode carbon.WorkerNode
	_ = json.Unmarshal(file, &workerNode)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bs := make([]byte, 0, workerNode.XXX_Size())
		workerNode.XXX_Marshal(bs, false)
	}
}

func TestBenchmark4JsonMarshal1(t *testing.T) {
	file, err := ioutil.ReadFile("testdata/workernode.json")
	assert.Nil(t, err)
	var workerNode carbon.WorkerNode
	_ = json.Unmarshal(file, &workerNode)
	start := time.Now()
	for i := 0; i < 10000; i++ {
		json.Marshal(workerNode)
	}
	fmt.Printf("json marshal 10000 times cost %d ms \n", time.Now().UnixMilli()-start.UnixMilli())
}

func TestBenchmark4PbMarshal1(t *testing.T) {
	file, err := ioutil.ReadFile("testdata/workernode.json")
	assert.Nil(t, err)
	var workerNode carbon.WorkerNode
	_ = json.Unmarshal(file, &workerNode)
	start := time.Now()
	for i := 0; i < 10000; i++ {
		bs := make([]byte, 0, workerNode.XXX_Size())
		workerNode.XXX_Marshal(bs, false)
	}
	fmt.Printf("protobuffer marshal 10000 times cost %d ms \n", time.Now().UnixMilli()-start.UnixMilli())
}
