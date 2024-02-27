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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/alibaba/kube-sharding/pkg/memkube/client"
	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	restful "github.com/emicklei/go-restful"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	getURIFormat = "apis/carbon.taobao.com/v1/namespaces/%s/%s"
)

func Test_defaultService(t *testing.T) {
	namespace := "ns"
	resource := "workernodes"
	stopCh := make(chan struct{})
	defer close(stopCh)
	syncPeriod := time.Millisecond * 300
	client := client.NewLocalClient(syncPeriod * 2)
	store := mem.NewStore(spec.Scheme, spec.SchemeGroupVersion.WithKind(mem.Kind(&spec.WorkerNode{})), namespace, &mem.OneBatcher{},
		mem.NewFakePersister(), mem.WithStorePersistPeriod(syncPeriod))
	client.Register("workernodes", namespace, store)
	store2 := mem.NewStore(spec.Scheme, spec.SchemeGroupVersion.WithKind(mem.Kind(&v1.Pod{})), namespace, &mem.OneBatcher{},
		mem.NewFakePersister(), mem.WithStorePersistPeriod(syncPeriod))
	client.Register("pods", namespace, store2)
	store.Start(stopCh)
	store2.Start(stopCh)
	w, _ := store.Watch(&mem.WatchOptions{})
	testutils.ConsumeWatchEvents(w, stopCh)
	w, _ = store2.Watch(&mem.WatchOptions{})
	testutils.ConsumeWatchEvents(w, stopCh)

	spec.Scheme.AddKnownTypes(v1.SchemeGroupVersion, &v1.Pod{}, &v1.PodList{})
	scheme := mem.NewResourceScheme(spec.Scheme)

	defaultService := NewService(client, scheme, nil, &ServiceOptions{})
	restful.Add(defaultService.GetHandlerService())
	restful.DefaultContainer.EnableContentEncoding(true)

	verifyList := func(rr *httptest.ResponseRecorder, code, size int) *spec.WorkerNodeList {
		var resp rpcResponse
		into := &spec.WorkerNodeList{}
		if size > 0 {
			resp.Data = into
		}
		assert.Equal(t, 200, rr.Code)
		assert.Nil(t, json.Unmarshal(rr.Body.Bytes(), &resp))
		assert.Equal(t, code, resp.Code)
		if resp.Code == 0 {
			assert.Equal(t, "", resp.Msg)
		}
		return into
	}

	verifyCreate := func(rr *httptest.ResponseRecorder, code int, name string) *spec.WorkerNode {
		into := &spec.WorkerNode{}
		resp := &rpcResponse{
			Data: into,
		}
		assert.Equal(t, 200, rr.Code)
		assert.Equal(t, restful.MIME_JSON, rr.Header().Get("Content-Type"))
		(&jsonAccessor{}).Decode(rr.Body.Bytes(), resp)
		assert.Equal(t, code, resp.Code)
		assert.Equal(t, "", resp.Msg)
		assert.NotNil(t, into)
		assert.NotNil(t, into.Name)
		assert.Equal(t, name, into.Name)
		assert.Equal(t, namespace, into.Namespace)
		return into
	}

	verifyUpdate := func(rr *httptest.ResponseRecorder, code int, expect *spec.WorkerNode) *spec.WorkerNode {
		into := &spec.WorkerNode{}
		resp := &rpcResponse{
			Data: into,
		}
		assert.Equal(t, 200, rr.Code)
		assert.Equal(t, restful.MIME_JSON, rr.Header().Get("Content-Type"))
		(&jsonAccessor{}).Decode(rr.Body.Bytes(), resp)
		assert.Equal(t, code, resp.Code)
		assert.Equal(t, "", resp.Msg)
		if utils.ObjJSON(into) != utils.ObjJSON(expect) {
			t.Errorf("Expect object failed, \nE: %s\nG: %s\n", utils.ObjJSON(expect), utils.ObjJSON(into))
		}
		return into
	}

	verifyDelete := func(rr *httptest.ResponseRecorder, code, size int) *spec.WorkerNodeList {
		var resp rpcResponse
		assert.Equal(t, 200, rr.Code)
		assert.Nil(t, json.Unmarshal(rr.Body.Bytes(), &resp))
		assert.Equal(t, code, resp.Code)
		assert.Equal(t, "", resp.Msg)
		return nil
	}

	httpReq, _ := http.NewRequest("GET", fmt.Sprintf("/"+getURIFormat, namespace, resource), strings.NewReader(""))
	httpWriter := httptest.NewRecorder()
	restful.DefaultContainer.ServeHTTP(httpWriter, httpReq)
	verifyList(httpWriter, 0, 0)

	// create
	w1 := &spec.WorkerNode{}
	w1.Name = "worker1"
	w1.Namespace = namespace
	request := &postRequest{Body: w1}
	httpReq, _ = http.NewRequest("POST", fmt.Sprintf("/"+getURIFormat, namespace, resource), strings.NewReader(utils.ObjJSON(request)))
	httpWriter = httptest.NewRecorder()
	restful.DefaultContainer.ServeHTTP(httpWriter, httpReq)
	verifyCreate(httpWriter, 0, w1.Name)

	// update
	w2 := &spec.WorkerNode{}
	obj, _ := client.Get("workernodes", namespace, "worker1", metav1.GetOptions{}, w2)
	w2 = obj.(*spec.WorkerNode)
	request = &postRequest{Body: w2}
	httpReq, _ = http.NewRequest("PUT", fmt.Sprintf("/"+getURIFormat+"/%s", namespace, resource, w1.Name), strings.NewReader(utils.ObjJSON(request)))
	httpWriter = httptest.NewRecorder()
	restful.DefaultContainer.ServeHTTP(httpWriter, httpReq)
	verifyUpdate(httpWriter, 0, w2)

	// list
	httpReq, _ = http.NewRequest("GET", fmt.Sprintf("/"+getURIFormat, namespace, resource), strings.NewReader(""))
	httpWriter = httptest.NewRecorder()
	restful.DefaultContainer.ServeHTTP(httpWriter, httpReq)
	verifyList(httpWriter, 0, 1)

	// list unregister error
	httpReq, _ = http.NewRequest("GET", fmt.Sprintf("/"+getURIFormat, namespace, "pods"), strings.NewReader(""))
	httpWriter = httptest.NewRecorder()
	restful.DefaultContainer.ServeHTTP(httpWriter, httpReq)
	verifyList(httpWriter, -1, 0)

	// list error ns
	httpReq, _ = http.NewRequest("GET", fmt.Sprintf("/"+getURIFormat, "ns2", resource), strings.NewReader(""))
	httpWriter = httptest.NewRecorder()
	restful.DefaultContainer.ServeHTTP(httpWriter, httpReq)
	verifyList(httpWriter, -1, 0)

	// list with 1 item response
	httpReq, _ = http.NewRequest("GET", fmt.Sprintf("/"+getURIFormat, namespace, resource), strings.NewReader(""))
	httpWriter = httptest.NewRecorder()
	restful.DefaultContainer.ServeHTTP(httpWriter, httpReq)
	verifyList(httpWriter, 0, 1)

	// delete
	httpReq, _ = http.NewRequest("DELETE", fmt.Sprintf("/"+getURIFormat+"/%s", namespace, resource, w1.Name), strings.NewReader(""))
	httpWriter = httptest.NewRecorder()
	restful.DefaultContainer.ServeHTTP(httpWriter, httpReq)
	verifyDelete(httpWriter, 0, 0)

	// list ?
	httpReq, _ = http.NewRequest("GET", fmt.Sprintf("/"+getURIFormat, namespace, resource), strings.NewReader(""))
	httpWriter = httptest.NewRecorder()
	restful.DefaultContainer.ServeHTTP(httpWriter, httpReq)
	verifyList(httpWriter, 0, 0)

	// list pod by pb
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "p1",
			Namespace:  namespace,
			Generation: 111,
		},
	}
	obj, err := client.Create("pods", namespace, pod)
	assert.NoError(t, err)
	pod = obj.(*v1.Pod)
	contentType := runtime.ContentTypeProtobuf
	httpReq, _ = http.NewRequest("GET", fmt.Sprintf("/apis/v1/namespaces/%s/%s", namespace, "pods"), strings.NewReader(""))
	httpReq.Header.Set("Accept", contentType)
	httpWriter = httptest.NewRecorder()
	restful.DefaultContainer.ServeHTTP(httpWriter, httpReq)
	assert.Equal(t, contentType, httpWriter.Header().Get("Content-Type"))
	podsList := &v1.PodList{}
	resp := rpcResponse{
		Data: podsList,
	}
	accessor := entityAccessRegistry.accessorAt(contentType)
	assert.NoError(t, accessor.Decode(httpWriter.Body.Bytes(), &resp))
	// No TypeMeta serialized in Pod, weird
	pod.TypeMeta = metav1.TypeMeta{}
	if utils.ObjJSON(pod) != utils.ObjJSON(podsList.Items[0]) {
		t.Errorf("Expect object failed, \nE: %s\nG: %s\n", utils.ObjJSON(pod), utils.ObjJSON(podsList.Items[0]))
	}
}
