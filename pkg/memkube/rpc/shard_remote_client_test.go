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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	v1 "github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	"github.com/stretchr/testify/assert"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_shardRemoteClient_List(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/apis/carbon.taobao.com/v1/namespaces/test/workernodes?labelSelector=a%3Db%2CShardIDKey%3D1&src=testapp", req.URL.String())
		assert.Equal(t, "GET", req.Method)

		workernodeList := &v1.WorkerNodeList{}
		workernode1 := v1.WorkerNode{}
		workernode1.Name = "w1"
		workernode1.Status.EntityName = "worker1"
		workernode2 := v1.WorkerNode{}
		workernode2.Name = "w2"
		workernode2.Status.EntityName = "worker2"
		workernodeList.Items = append(workernodeList.Items, workernode1)
		workernodeList.Items = append(workernodeList.Items, workernode2)
		workernodeList.SetResourceVersion("2")
		response := rpcResponse{Data: workernodeList}
		rw.Write([]byte(utils.ObjJSON(response)))
	}))
	defer server.Close()
	t.Log(server)
	nodes := make([]*NodeInfo, 0)
	nodes = append(nodes, getNodeInfo(server.URL))
	nodes = append(nodes, getNodeInfo(server.URL))
	resolver := &MockResolver{}
	resolver.setNodeInfos(nodes)
	s := &shardRemoteClient{resolver: resolver,
		httpClient: utils.NewLiteHTTPClientWithClient(server.Client()),
		src:        "testapp",
		pathFormat: "apis/carbon.taobao.com/v1/namespaces/%s/%s",
	}
	opts := metav1.ListOptions{}
	opts.LabelSelector = "a=b,ShardIDKey=1"
	into := &v1.WorkerNodeList{}
	err := s.List("workernodes", "test", opts, into)
	assert.NoError(t, err, "list error")
	assert.Equal(t, 4, len(into.Items))
	assert.Equal(t, "w1", into.Items[0].Name)
	assert.Equal(t, "w2", into.Items[1].Name)
	assert.Equal(t, "w2", into.Items[3].Name)
	assert.Equal(t, "worker1", into.Items[0].Status.EntityName)
	assert.Equal(t, "worker2", into.Items[3].Status.EntityName)
}

func Test_shardRemoteClient_List_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/apis/carbon.taobao.com/v1/namespaces/test/workernodes?labelSelector=a%3Db%2CShardIDKey%3D1&src=testapp", req.URL.String())
		assert.Equal(t, "GET", req.Method)

		response := rpcResponse{}
		response.Code = 0
		rw.Write([]byte(utils.ObjJSON(response)))
	}))
	defer server.Close()
	t.Log(server)
	nodes := make([]*NodeInfo, 0)
	nodes = append(nodes, getNodeInfo(server.URL))
	nodes = append(nodes, getNodeInfo(server.URL))
	resolver := &MockResolver{}
	resolver.setNodeInfos(nodes)
	s := &shardRemoteClient{resolver: resolver,
		httpClient: utils.NewLiteHTTPClientWithClient(server.Client()),
		src:        "testapp",
		pathFormat: "apis/carbon.taobao.com/v1/namespaces/%s/%s"}
	opts := metav1.ListOptions{}
	opts.LabelSelector = "a=b,ShardIDKey=1"
	into := &v1.WorkerNodeList{}
	err := s.List("workernodes", "test", opts, into)
	assert.NoError(t, err, "list error")
	assert.Equal(t, 0, len(into.Items))
}

func Test_shardRemoteClient_EmptyNamespace(t *testing.T) {
	s := &shardRemoteClient{}
	into := &v1.WorkerNodeList{}
	assert.Nil(t, s.List("workernodes", "", metav1.ListOptions{}, into))
	w, err := s.Watch("workernodes", "", &mem.WatchOptions{})
	assert.Nil(t, err)
	assert.Equal(t, 0, len(w.ResultChan()))
}

func Test_shardRemoteClient_List_Error(t *testing.T) {
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
	nodes := make([]*NodeInfo, 0)
	nodes = append(nodes, getNodeInfo(server.URL))
	nodes = append(nodes, getNodeInfo(server.URL))
	resolver := &MockResolver{}
	resolver.setNodeInfos(nodes)
	s := &shardRemoteClient{resolver: resolver,
		httpClient: utils.NewLiteHTTPClientWithClient(server.Client()),
		src:        "testapp",
		pathFormat: "apis/carbon.taobao.com/v1/namespaces/%s/%s"}
	opts := metav1.ListOptions{}
	opts.LabelSelector = "a=b,ShardIDKey=1"
	into := &v1.WorkerNodeList{}
	err := s.List("workernodes", "test", opts, into)
	assert.Error(t, err, "list error:"+err.Error())
	assert.EqualError(t, err, fmt.Sprintf("list from remote %s failed, ns: test, err: response failed: -1, test error, host:%s", server.Listener.Addr(), server.Listener.Addr()))
}

func Test_shardRemoteClient_List_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/apis/carbon.taobao.com/v1/namespaces/test/workernodes?labelSelector=a%3Db%2CShardIDKey%3D1&src=testapp", req.URL.String())
		assert.Equal(t, "GET", req.Method)
		rw.WriteHeader(404)
		rw.Write([]byte("page not found"))
	}))
	defer server.Close()
	t.Log(server)
	nodes := make([]*NodeInfo, 0)
	nodes = append(nodes, getNodeInfo(server.URL))
	nodes = append(nodes, getNodeInfo(server.URL))
	resolver := &MockResolver{}
	resolver.setNodeInfos(nodes)
	s := &shardRemoteClient{resolver: resolver,
		httpClient: utils.NewLiteHTTPClientWithClient(server.Client()),
		src:        "testapp",
		pathFormat: "apis/carbon.taobao.com/v1/namespaces/%s/%s"}
	opts := metav1.ListOptions{}
	opts.LabelSelector = "a=b,ShardIDKey=1"
	into := &v1.WorkerNodeList{}
	err := s.List("workernodes", "test", opts, into)
	assert.Error(t, err, "list error:"+err.Error())
	assert.EqualError(t, err, fmt.Sprintf("list from remote %s failed, ns: test, err: response error, http statusCode:404, host:%s", server.Listener.Addr(), server.Listener.Addr()))
}

func Test_shardRemoteClient_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/apis/carbon.taobao.com/v1/namespaces/test/workernodes/w1?src=testapp", req.URL.String())
		assert.Equal(t, "GET", req.Method)
		response := rpcResponse{}
		workernode1 := v1.WorkerNode{}
		workernode1.Name = "w1"
		workernode1.Status.EntityName = "worker1"
		workernode1.Nick = "tony"
		workernode1.Age = 20
		response.Data = workernode1
		response.Code = 0
		response.RequestID = "111"
		rw.Write([]byte(utils.ObjJSON(response)))
	}))
	defer server.Close()
	t.Log(server)
	nodes := make([]*NodeInfo, 0)
	nodes = append(nodes, getNodeInfo(server.URL))
	resolver := &MockResolver{}
	resolver.setNodeInfos(nodes)
	s := &shardRemoteClient{resolver: resolver,
		httpClient: utils.NewLiteHTTPClientWithClient(server.Client()),
		src:        "testapp",
		pathFormat: "apis/carbon.taobao.com/v1/namespaces/%s/%s"}
	opts := metav1.GetOptions{}
	into := &v1.WorkerNode{}

	result, err := s.Get("workernodes", "test", "w1", opts, into)
	assert.NoError(t, err, "get error")
	assert.NotNil(t, result)
	w := result.(*v1.WorkerNode)
	assert.Equal(t, "w1", w.Name)
	assert.Equal(t, "tony", w.Nick)
	assert.Equal(t, 20, w.Age)

	nodes = append(nodes, getNodeInfo(server.URL))
	result, err = s.Get("workernodes", "test", "w1", opts, into)
	assert.NoError(t, err, "get error")
	assert.NotNil(t, result)
	w = result.(*v1.WorkerNode)
	assert.Equal(t, "w1", w.Name)
	assert.Equal(t, "tony", w.Nick)
	assert.Equal(t, 20, w.Age)
}

func Test_shardRemoteClient_Get_Empty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/apis/carbon.taobao.com/v1/namespaces/test/workernodes/w1?src=testapp", req.URL.String())
		assert.Equal(t, "GET", req.Method)

		response := rpcResponse{}

		response.Msg = "timeout"
		response.Code = -1
		response.RequestID = "111"
		rw.Write([]byte(utils.ObjJSON(response)))
	}))
	defer server.Close()
	t.Log(server)
	nodes := make([]*NodeInfo, 0)
	nodes = append(nodes, getNodeInfo(server.URL))
	resolver := &MockResolver{}
	resolver.setNodeInfos(nodes)
	s := &shardRemoteClient{resolver: resolver,
		httpClient: utils.NewLiteHTTPClientWithClient(server.Client()),
		src:        "testapp",
		pathFormat: "apis/carbon.taobao.com/v1/namespaces/%s/%s"}
	opts := metav1.GetOptions{}
	into := &v1.WorkerNode{}

	result, err := s.Get("workernodes", "test", "w1", opts, into)
	assert.Error(t, err, "get error")
	assert.True(t, strings.Contains(err.Error(), "response failed: -1, timeout, host:127.0.0.1:"))

	assert.False(t, kerrors.IsNotFound(err))
	assert.Nil(t, result)

	//not found
	server = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/apis/carbon.taobao.com/v1/namespaces/test/workernodes/w1?src=testapp", req.URL.String())
		assert.Equal(t, "GET", req.Method)

		response := rpcResponse{}

		response.Msg = "not found"
		response.Code = codeResourceNotFound
		response.RequestID = "111"
		rw.Write([]byte(utils.ObjJSON(response)))
	}))
	defer server.Close()
	t.Log(server)
	nodes = make([]*NodeInfo, 0)
	nodes = append(nodes, getNodeInfo(server.URL))
	resolver = &MockResolver{}
	resolver.setNodeInfos(nodes)
	s = &shardRemoteClient{resolver: resolver,
		httpClient: utils.NewLiteHTTPClientWithClient(server.Client()),
		src:        "testapp",
		pathFormat: "apis/carbon.taobao.com/v1/namespaces/%s/%s"}
	opts = metav1.GetOptions{}
	into = &v1.WorkerNode{}

	result, err = s.Get("workernodes", "test", "w1", opts, into)
	assert.Error(t, err, "get error")
	assert.Equal(t, "workernodes \"w1\" not found", err.Error())
	assert.True(t, kerrors.IsNotFound(err))
	assert.Nil(t, result)
}

func Test_shardRemoteClient_Create(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/apis/carbon.taobao.com/v1/namespaces/test/workernodes?src=testapp", req.URL.String())
		assert.Equal(t, "POST", req.Method)

		body, _ := ioutil.ReadAll(req.Body)
		into := &v1.WorkerNode{}
		postRequest := &postRequest{Body: into}
		json.Unmarshal(body, postRequest)

		assert.Equal(t, 20, into.Age)
		assert.Equal(t, "w1", into.Name)
		assert.Equal(t, "tony", into.Nick)

		response := rpcResponse{}

		workernode1 := v1.WorkerNode{}
		workernode1.Name = "w1"
		workernode1.Status.EntityName = "worker1"
		workernode1.Nick = "tony"
		workernode1.Age = 20

		response.Data = workernode1
		response.Code = 0
		response.RequestID = "111"
		rw.Write([]byte(utils.ObjJSON(response)))
	}))
	defer server.Close()
	t.Log(server)
	nodes := make([]*NodeInfo, 0)
	nodes = append(nodes, getNodeInfo(server.URL))
	resolver := &MockResolver{}
	resolver.setNodeInfos(nodes)
	s := &shardRemoteClient{resolver: resolver,
		httpClient: utils.NewLiteHTTPClientWithClient(server.Client()),
		src:        "testapp",
		pathFormat: "apis/carbon.taobao.com/v1/namespaces/%s/%s"}
	into := &v1.WorkerNode{}
	into.Name = "w1"
	into.Nick = "tony"
	into.Age = 20
	result, err := s.Create("workernodes", "test", into)

	assert.NoError(t, err, "get error")
	assert.NotNil(t, result)
	w := result.(*v1.WorkerNode)
	assert.Equal(t, "w1", w.Name)
	assert.Equal(t, "tony", w.Nick)
	assert.Equal(t, 20, w.Age)
}

func Test_shardRemoteClient_Update(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/apis/carbon.taobao.com/v1/namespaces/test/workernodes/w1?src=testapp", req.URL.String())
		assert.Equal(t, "PUT", req.Method)

		body, _ := ioutil.ReadAll(req.Body)
		into := &v1.WorkerNode{}
		postRequest := &postRequest{Body: into}
		json.Unmarshal(body, postRequest)

		assert.Equal(t, 10, into.Age)
		assert.Equal(t, "w1", into.Name)
		assert.Equal(t, "tony", into.Nick)

		response := rpcResponse{}

		workernode1 := v1.WorkerNode{}
		workernode1.Name = "w1"
		workernode1.Status.EntityName = "worker1"
		workernode1.Nick = into.Nick
		workernode1.Age = into.Age

		response.Data = workernode1
		response.Code = 0
		response.RequestID = "111"
		rw.Write([]byte(utils.ObjJSON(response)))
	}))
	defer server.Close()
	t.Log(server)
	nodes := make([]*NodeInfo, 0)
	nodes = append(nodes, getNodeInfo(server.URL))
	resolver := &MockResolver{}
	resolver.setNodeInfos(nodes)
	s := &shardRemoteClient{resolver: resolver,
		httpClient: utils.NewLiteHTTPClientWithClient(server.Client()),
		src:        "testapp",
		pathFormat: "apis/carbon.taobao.com/v1/namespaces/%s/%s"}
	into := &v1.WorkerNode{}
	into.Name = "w1"
	into.Nick = "tony"
	into.Age = 10

	result, err := s.Update("workernodes", "test", into)

	assert.NoError(t, err, "get error")
	assert.NotNil(t, result)
	w := result.(*v1.WorkerNode)
	assert.Equal(t, "w1", w.Name)
	assert.Equal(t, "tony", w.Nick)
	assert.Equal(t, 10, w.Age)
}

func Test_shardRemoteClient_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/apis/carbon.taobao.com/v1/namespaces/test/workernodes/w1?src=testapp", req.URL.String())
		assert.Equal(t, "DELETE", req.Method)

		response := rpcResponse{}

		response.Code = 0
		response.RequestID = "111"
		rw.Write([]byte(utils.ObjJSON(response)))
	}))
	defer server.Close()
	t.Log(server)
	nodes := make([]*NodeInfo, 0)
	nodes = append(nodes, getNodeInfo(server.URL))
	resolver := &MockResolver{}
	resolver.setNodeInfos(nodes)
	s := &shardRemoteClient{resolver: resolver,
		httpClient: utils.NewLiteHTTPClientWithClient(server.Client()),
		src:        "testapp",
		cluster:    "c",
		pathFormat: "apis/carbon.taobao.com/v1/namespaces/%s/%s"}

	err := s.Delete("workernodes", "test", "w1", nil)

	assert.NoError(t, err, "get error")
}

type MockResolver struct {
	nodeInfos []*NodeInfo
}

func (m *MockResolver) setNodeInfos(nodeInfos []*NodeInfo) {
	m.nodeInfos = nodeInfos
}

func (m *MockResolver) Resolve(namespace string, obj interface{}) (*NodeInfo, error) {
	return m.nodeInfos[0], nil
}

// ResolveAll resolve config
func (m *MockResolver) ResolveAll(namespace string) ([]*NodeInfo, error) {
	return m.nodeInfos, nil
}

// RegisterCalcShardIDFunc  register calc shardID func
func (m *MockResolver) RegisterCalcShardIDFunc(cf CalcShardIDFunc) {
}
