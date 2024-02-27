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

package rpc

import (
	"compress/gzip"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	v1 "github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func Test_singleNodeInvoker_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/apis/carbon.taobao.com/v1/namespaces/test/workernodes?labelSelector=a%3Db%2CShardIDKey%3D1&src=testapp", req.URL.String())
		assert.Equal(t, "GET", req.Method)
		workernodeList := &v1.WorkerNodeList{}
		workernode1 := v1.WorkerNode{}
		workernode1.Name = "w1"
		workernode1.Status.EntityName = "worker1"
		workernode2 := v1.WorkerNode{}
		workernode2.Name = "w2"
		workernodeList.Items = append(workernodeList.Items, workernode1)
		workernodeList.Items = append(workernodeList.Items, workernode2)
		response := rpcResponse{Data: workernodeList}
		rd := []byte(utils.ObjJSON(response))
		rw.Write(rd)
	}))
	defer server.Close()
	t.Log(server)

	nodeInfo := getNodeInfo(server.URL)
	s := NewSingleNodeInvoker("testapp", "testCluster", getURIFormat, nodeInfo, utils.NewLiteHTTPClientWithClient(server.Client()))

	opts := &metav1.ListOptions{}
	opts.LabelSelector = "a=b,ShardIDKey=1"
	into := &v1.WorkerNodeList{}
	err := s.List("workernodes", "test", opts, into)
	assert.NoError(t, err, "")
	assert.Equal(t, 2, len(into.Items))
	assert.Equal(t, "w1", into.Items[0].Name)
	assert.Equal(t, "w2", into.Items[1].Name)
	assert.Equal(t, "worker1", into.Items[0].Status.EntityName)
	if err != nil {
		t.Errorf("List err %v", err)
	}
}

func Test_singleNodeInvoker_ListPB(t *testing.T) {
	podsList := &corev1.PodList{
		Items: []corev1.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "p1",
					Generation: 111,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "p2",
					Generation: 113,
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, runtime.ContentTypeProtobuf, req.Header.Get("Accept"))
		response := &rpcResponse{Data: podsList}
		bs, err := (&pbAccessor{}).Encode(response)
		assert.NoError(t, err)
		rw.Header().Set("Content-Type", runtime.ContentTypeProtobuf)
		rw.Write(bs)
	}))
	defer server.Close()
	t.Log(server)

	nodeInfo := getNodeInfo(server.URL)
	s := NewSingleNodeInvoker("testapp", "testCluster", getURIFormat, nodeInfo, utils.NewLiteHTTPClientWithClient(server.Client()))
	s.SetAccepts(runtime.ContentTypeProtobuf)

	opts := &metav1.ListOptions{}
	into := &corev1.PodList{}
	err := s.List("pods", "test", opts, into)
	assert.NoError(t, err, "")
	if !reflect.DeepEqual(into, podsList) {
		t.Errorf("Equal failed: expect: %s\ngot: %s\n", utils.ObjJSON(podsList), utils.ObjJSON(into))
	}
}
func Test_singleNodeInvoker_gzip(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/apis/carbon.taobao.com/v1/namespaces/test/workernodes?labelSelector=a%3Db%2CShardIDKey%3D1&src=testapp", req.URL.String())
		assert.Equal(t, "GET", req.Method)
		workernodeList := &v1.WorkerNodeList{}
		workernode1 := v1.WorkerNode{}
		workernode1.Name = "w1"
		workernode1.Status.EntityName = "worker1"
		workernode2 := v1.WorkerNode{}
		workernode2.Name = "w2"
		workernodeList.Items = append(workernodeList.Items, workernode1)
		workernodeList.Items = append(workernodeList.Items, workernode2)
		response := rpcResponse{Data: workernodeList}
		rd := []byte(utils.ObjJSON(response))
		rw.Header().Set("Content-Encoding", "gzip")
		zw := gzip.NewWriter(rw)
		zw.Write(rd)
		zw.Close()
	}))
	defer server.Close()
	t.Log(server)

	nodeInfo := getNodeInfo(server.URL)
	s := NewSingleNodeInvoker("testapp", "testCluster", getURIFormat, nodeInfo, utils.NewLiteHTTPClientWithClient(server.Client()))

	opts := &metav1.ListOptions{}
	opts.LabelSelector = "a=b,ShardIDKey=1"
	into := &v1.WorkerNodeList{}
	err := s.List("workernodes", "test", opts, into)
	assert.NoError(t, err, "")
	assert.Equal(t, 2, len(into.Items))
	assert.Equal(t, "w1", into.Items[0].Name)
	assert.Equal(t, "w2", into.Items[1].Name)
	assert.Equal(t, "worker1", into.Items[0].Status.EntityName)
	if err != nil {
		t.Errorf("List err %v", err)
	}
}

func Test_singleNodeInvoker_List_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/apis/carbon.taobao.com/v1/namespaces/test/workernodes?labelSelector=a%3Db%2CShardIDKey%3D1&src=testapp", req.URL.String())
		assert.Equal(t, "GET", req.Method)
		response := rpcResponse{}
		response.Code = -1
		response.Msg = "test error"
		rw.Write([]byte(utils.ObjJSON(response)))
	}))
	defer server.Close()
	t.Log(server)

	nodeInfo := getNodeInfo(server.URL)
	s := NewSingleNodeInvoker("testapp", "testCluster", getURIFormat, nodeInfo, utils.NewLiteHTTPClientWithClient(server.Client()))

	opts := &metav1.ListOptions{}
	opts.LabelSelector = "a=b,ShardIDKey=1"
	into := &v1.WorkerNodeList{}
	err := s.List("workernodes", "test", opts, into)
	assert.Equal(t, 0, len(into.Items))
	if err == nil {
		t.Errorf("List empty %v", err)
	}
	assert.True(t, strings.Contains(err.Error(), "response failed: -1, test error, host:127.0.0.1:"))

	nodeInfo = getNodeInfo("http://0:3333")
	s = NewSingleNodeInvoker("testapp", "testCluster", getURIFormat, nodeInfo, utils.NewLiteHTTPClientWithClient(server.Client()))

	opts = &metav1.ListOptions{}
	opts.LabelSelector = "a=b,ShardIDKey=1"
	into = &v1.WorkerNodeList{}
	err = s.List("workernodes", "test", opts, into)
	assert.Equal(t, 0, len(into.Items))
	if err == nil {
		t.Errorf("List empty %v", err)
	}
	assert.NotNil(t, err)

	server404 := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/apis/carbon.taobao.com/v1/namespaces/test/workernodes?labelSelector=a%3Db%2CShardIDKey%3D1&src=testapp", req.URL.String())
		assert.Equal(t, "GET", req.Method)
		rw.WriteHeader(404)
		rw.Write([]byte("test error"))
	}))
	defer server.Close()
	t.Log(server)

	nodeInfo = getNodeInfo(server404.URL)
	s = NewSingleNodeInvoker("testapp", "testCluster", getURIFormat, nodeInfo, utils.NewLiteHTTPClientWithClient(server.Client()))

	opts = &metav1.ListOptions{}
	opts.LabelSelector = "a=b,ShardIDKey=1"
	into = &v1.WorkerNodeList{}
	err = s.List("workernodes", "test", opts, into)
	assert.Equal(t, 0, len(into.Items))
	if err == nil {
		t.Errorf("List empty %v", err)
	}
	assert.True(t, strings.Contains(err.Error(), "response error, http statusCode:404, host:127.0.0.1:"))
}

func getNodeInfo(url string) *NodeInfo {
	return &NodeInfo{IP: "127.0.0.1", Port: strings.Split(url, ":")[2]}
}
