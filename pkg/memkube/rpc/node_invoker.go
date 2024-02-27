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
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	glog "k8s.io/klog"
)

// SingleNodeInvoker single node
type SingleNodeInvoker struct {
	rpcNode    *NodeInfo
	httpClient *utils.LiteHTTPClient
	src        string
	cluster    string
	pathFormat string
	accepts    string
}

// NewSingleNodeInvoker  create new single node http invoker
func NewSingleNodeInvoker(src, cluster, pathFormat string, rpcNode *NodeInfo, httpclient *utils.LiteHTTPClient) *SingleNodeInvoker {
	return &SingleNodeInvoker{
		src:        src,
		cluster:    cluster,
		pathFormat: pathFormat,
		rpcNode:    rpcNode,
		httpClient: httpclient,
	}
}

// SetAccepts set http accepts header
func (h *SingleNodeInvoker) SetAccepts(accepts string) {
	h.accepts = accepts
}

// List query list resource
func (h *SingleNodeInvoker) List(resource, ns string, opts *metav1.ListOptions, into runtime.Object) error {
	url := h.getRPCURL(h.rpcNode, resource, ns, "", opts.LabelSelector)
	glog.V(4).Infof("List RPC url %s", url)
	startTime := time.Now()
	headers := make(map[string]string)
	if h.accepts != "" {
		headers["Accept"] = h.accepts
	}
	resp, err := h.httpClient.Request(context.Background(), url, "", http.MethodGet, headers)
	Metrics.Get(MetricRPCReadTime).Elapsed(startTime, h.cluster, ns, resource)
	if err != nil {
		return err
	}
	return h.into(resp, resource, ns, "", into)
}

// Get get resource
func (h *SingleNodeInvoker) Get(resource, ns, name string, opts *metav1.GetOptions, into runtime.Object) (runtime.Object, error) {
	url := h.getRPCURL(h.rpcNode, resource, ns, name, "")
	glog.V(4).Infof("Get RPC url %s", url)
	startTime := time.Now()
	liteHTTPResp, err := h.doGet(url)
	Metrics.Get(MetricRPCReadTime).Elapsed(startTime, h.cluster, ns, resource)
	if err != nil {
		return nil, err
	}
	return into, h.into(liteHTTPResp, resource, ns, name, into)
}

// Create create resource
func (h *SingleNodeInvoker) Create(resource, ns string, object runtime.Object) (runtime.Object, error) {
	url := h.getRPCURL(h.rpcNode, resource, ns, "", "")
	glog.V(4).Infof("Create RPC url %s", url)

	postRequest := &postRequest{Resource: resource, Body: object}
	startTime := time.Now()
	liteHTTPResp, err := h.doPost(url, postRequest)
	if err != nil {
		Metrics.Get(MetricRPCWriteTime).Elapsed(startTime, h.cluster, ns, resource, "Create", strconv.FormatBool(false))
		return nil, err
	}
	err = h.into(liteHTTPResp, resource, ns, "", object)
	Metrics.Get(MetricRPCWriteTime).Elapsed(startTime, h.cluster, ns, resource, "Create", strconv.FormatBool(err == nil))
	return object, err
}

// Update update resource
func (h *SingleNodeInvoker) Update(resource, ns, name string, object runtime.Object) (runtime.Object, error) {
	url := h.getRPCURL(h.rpcNode, resource, ns, name, "")
	glog.V(4).Infof("Update RPC url %s", url)

	postRequest := &postRequest{Resource: resource, Body: object}
	startTime := time.Now()
	liteHTTPResp, err := h.doPut(url, postRequest)
	Metrics.Get(MetricRPCWriteTime).Elapsed(startTime, h.cluster, ns, resource, "Update")
	if err != nil {
		return nil, err
	}
	err = h.into(liteHTTPResp, resource, ns, name, object)
	return object, err
}

// Delete delete resource
func (h *SingleNodeInvoker) Delete(resource, ns, name string, opts *metav1.DeleteOptions) error {
	url := h.getRPCURL(h.rpcNode, resource, ns, name, "")
	glog.V(4).Infof("Delete RPC url %s", url)

	startTime := time.Now()
	liteHTTPResp, err := h.doDelete(url)
	Metrics.Get(MetricRPCWriteTime).Elapsed(startTime, h.cluster, ns, resource, "Delete")
	if err != nil {
		return err
	}
	err = h.into(liteHTTPResp, resource, ns, name, nil)
	return err
}

// getRPCURL  get rpc url by rpcNodeInfo, resource, ns...
func (h *SingleNodeInvoker) getRPCURL(rpcNodeInfo *NodeInfo, resource, ns, name, labelSelector string) string {
	return (&request{
		endpoint:   rpcNodeInfo.IP + ":" + rpcNodeInfo.Port,
		pathFormat: h.pathFormat,
		namespace:  ns,
		resource:   resource,
		name:       name,
		selector:   labelSelector,
		source:     h.src,
	}).URL()
}

func (h *SingleNodeInvoker) doGet(url string) (*utils.LiteHTTPResp, error) {
	liteHTTPResp, err := h.httpClient.Get(url)
	return liteHTTPResp, err
}

func (h *SingleNodeInvoker) doPost(url string, request *postRequest) (*utils.LiteHTTPResp, error) {
	accessor := entityAccessRegistry.accessorAt(defaultMIMEType)
	b, err := accessor.Encode(request)
	if err != nil {
		return nil, err
	}
	liteHTTPResp, err := h.httpClient.Post(url, utils.ByteCastString(b))
	return liteHTTPResp, err
}

func (h *SingleNodeInvoker) doPut(url string, request *postRequest) (*utils.LiteHTTPResp, error) {
	accessor := entityAccessRegistry.accessorAt(defaultMIMEType)
	b, err := accessor.Encode(request)
	if err != nil {
		return nil, err
	}
	liteHTTPResp, err := h.httpClient.Put(url, utils.ByteCastString(b))
	return liteHTTPResp, err
}

func (h *SingleNodeInvoker) doDelete(url string) (*utils.LiteHTTPResp, error) {
	liteHTTPResp, err := h.httpClient.Delete(url)
	return liteHTTPResp, err
}

func (h *SingleNodeInvoker) into(resp *utils.LiteHTTPResp, resource, ns, name string, into runtime.Object) error {
	startTime := time.Now()
	defer Metrics.Get(MetricRPCUnmarshalTime).Elapsed(startTime, h.cluster, ns, resource)
	if resp.StatusCode != 200 {
		return fmt.Errorf("response error, http statusCode:%d, host:%s", resp.StatusCode, h.rpcNode.String())
	}
	accessor := entityAccessRegistry.accessorAt(resp.Headers.Get("Content-Type"))
	result := &rpcResponse{
		Data: into,
	}
	if err := accessor.Decode(resp.Body, result); err != nil {
		glog.Errorf("Decode body error: %v, %s", err, string(resp.Body)) // what if binary data ?
		return fmt.Errorf("decode body error: %v", err)
	}
	if result.Code != 0 {
		if result.Code == codeResourceNotFound {
			return kerrors.NewNotFound(schema.GroupResource{Resource: resource}, name)
		}
		return fmt.Errorf("response failed: %v, %v, host:%s", result.Code, result.Msg, h.rpcNode.String())
	}
	return nil
}
