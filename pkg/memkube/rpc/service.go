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
	"fmt"
	"net/http"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils/restutil"
	"github.com/alibaba/kube-sharding/pkg/memkube/client"
	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	restful "github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/handlers/responsewriters"
	glog "k8s.io/klog"
)

const (
	any                  = "*/*"
	codeResourceNotFound = 404
)

// Service represents a rpc server to serve mem kube resources.
type Service interface {
	GetHandlerService() *restful.WebService
}

// ServiceOptions represents rpc service options, e.g, listen port.
type ServiceOptions struct {
}

type defaultService struct {
	client         client.Client
	resourceScheme *mem.ResourceScheme
	serializer     runtime.NegotiatedSerializer
}

// NewService create a new rpc service.
// If we use http, we may pass a http container here, e.g: NewService(webContainer, ...)
func NewService(client client.Client, resourceScheme *mem.ResourceScheme, ns runtime.NegotiatedSerializer, opts *ServiceOptions) Service {
	return &defaultService{
		client:         client,
		resourceScheme: resourceScheme,
		serializer:     ns,
	}
}

func (svc *defaultService) GetHandlerService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Produces(any)
	ws.Consumes(any)

	ws.Path(`/apis`)

	ws.Route(ws.GET("/{group}/{version}/namespaces/{namespace}/{resource}").To(svc.listWatch))
	ws.Route(ws.GET("/{group}/{version}/namespaces/{namespace}/{resource}/{name}").To(svc.to(svc.getResource)))
	ws.Route(ws.POST("/{group}/{version}/namespaces/{namespace}/{resource}").To(svc.to(svc.createResource)))
	ws.Route(ws.PUT("/{group}/{version}/namespaces/{namespace}/{resource}/{name}").To(svc.to(svc.updateResource)))
	ws.Route(ws.DELETE("/{group}/{version}/namespaces/{namespace}/{resource}/{name}").To(svc.to(svc.deleteResource)))

	ws.Route(ws.GET("/{version}/namespaces/{namespace}/{resource}").To(svc.listWatch))
	ws.Route(ws.GET("/{version}/namespaces/{namespace}/{resource}/{name}").To(svc.to(svc.getResource)))
	ws.Route(ws.POST("/{version}/namespaces/{namespace}/{resource}").To(svc.to(svc.createResource)))
	ws.Route(ws.PUT("/{version}/namespaces/{namespace}/{resource}/{name}").To(svc.to(svc.updateResource)))
	ws.Route(ws.DELETE("/{version}/namespaces/{namespace}/{resource}/{name}").To(svc.to(svc.deleteResource)))

	restful.DefaultRequestContentType(restful.MIME_JSON)
	restful.DefaultResponseContentType(restful.MIME_JSON)
	restful.PrettyPrintResponses = false // otherwise cpu waste

	restful.RegisterEntityAccessor(runtime.ContentTypeProtobuf, restutil.NewEntityAccessorPB(runtime.ContentTypeProtobuf))
	return ws
}

// ResourceRequest contains restful.Request
type ResourceRequest struct {
	*request
	restRequest *restful.Request
	group       string
	version     string
}

// newResourceRequest construct newResourceRequest
func newResourceRequest(request *restful.Request) *ResourceRequest {
	req := &ResourceRequest{
		request:     parseRequest(request),
		restRequest: request,
		group:       request.PathParameter("group"),
		version:     request.PathParameter("version"),
	}
	return req
}

type httpHandler func(rpcRequest *ResourceRequest, response *restful.Response) *rpcResponse

func (svc *defaultService) to(handler httpHandler) restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		resourceRequest := newResourceRequest(request)
		err := svc.checkRequest(resourceRequest)
		if err != nil {
			sendAPIResponse(request, response, newResponse(nil, err))
			return
		}
		glog.V(4).Infof("Rpc request %s, pathParam:%s, invoker:%s", request.Request.Method,
			utils.ObjJSON(request.PathParameters()), resourceRequest.source)
		resp := handler(resourceRequest, response)
		sendAPIResponse(request, response, resp)
	}
}

func (svc *defaultService) checkRequest(resourceRequest *ResourceRequest) error {
	if resourceRequest.resource == "" || resourceRequest.namespace == "" {
		return fmt.Errorf("illegal request: %s", utils.ObjJSON(resourceRequest.restRequest.PathParameters()))
	}
	gvr := schema.GroupVersionResource{Group: resourceRequest.group, Version: resourceRequest.version, Resource: resourceRequest.resource}
	_, err := svc.resourceScheme.New(gvr)
	if err != nil {
		return fmt.Errorf("unregister resource: %s %v", utils.ObjJSON(resourceRequest.restRequest.PathParameters()), err)
	}
	return nil
}

func (svc *defaultService) listWatch(request *restful.Request, response *restful.Response) {
	watch := request.QueryParameter("watch")
	if watch == "1" {
		svc.watch(request, response)
		return
	}
	resourceRequest := newResourceRequest(request)
	err := svc.checkRequest(resourceRequest)
	if err != nil {
		sendAPIResponse(request, response, newResponse(nil, err))
		return
	}
	glog.V(4).Infof("List request %s, pathParam:%s, invoker:%s", request.Request.Method,
		utils.ObjJSON(request.PathParameters()), resourceRequest.source)
	resp := svc.listResource(resourceRequest, response)
	sendAPIResponse(request, response, resp)
}

func (svc *defaultService) watch(request *restful.Request, response *restful.Response) {
	resReq := newResourceRequest(request)
	gvr := schema.GroupVersionResource{Group: resReq.group, Version: resReq.version, Resource: resReq.resource}
	gvk, err := svc.resourceScheme.KindFor(gvr)
	if err != nil {
		responsewriters.ErrorNegotiated(err, svc.serializer, gvk.GroupVersion(), response.ResponseWriter, request.Request)
		return
	}
	host := getRequestHost(request.Request)
	glog.Infof("Starting watch for %s, %s, args: %+v", host, request.Request.URL.Path, *resReq.request)
	opts := &mem.WatchOptions{
		ListOptions: metav1.ListOptions{
			LabelSelector:   resReq.selector,
			ResourceVersion: resReq.resourceversion,
			TimeoutSeconds:  utils.Int64Ptr(int64(resReq.timeoutSec)),
		},
		LocalWatch: utils.BoolPtr(false),
		Source:     host,
	}
	watch, err := svc.client.Watch(gvr.Resource, resReq.namespace, opts)
	if err != nil {
		glog.Errorf("Watch failed: %v, %v", err, gvk.String())
		responsewriters.ErrorNegotiated(err, svc.serializer, gvk.GroupVersion(), response.ResponseWriter, request.Request)
		return
	}
	serveWatch(watch, svc.serializer, gvk, request.Request, response.ResponseWriter, time.Second*time.Duration(resReq.timeoutSec))
}

func (svc *defaultService) listResource(resourceRequest *ResourceRequest, response *restful.Response) *rpcResponse {
	opts := metav1.ListOptions{}
	opts.LabelSelector = resourceRequest.selector
	gvr := schema.GroupVersionResource{Group: resourceRequest.group, Version: resourceRequest.version, Resource: resourceRequest.resource}
	into, err := svc.resourceScheme.NewList(gvr)
	if err != nil {
		return newResponse(nil, err)
	}
	err = svc.client.List(resourceRequest.resource, resourceRequest.namespace, opts, into)
	if err != nil {
		glog.Warningf("List resource error: %s, err: %v", resourceRequest.resource, err)
	}
	return newResponse(into, err)
}

func (svc *defaultService) getResource(resourceRequest *ResourceRequest, response *restful.Response) *rpcResponse {
	name := resourceRequest.name
	gvr := schema.GroupVersionResource{Group: resourceRequest.group, Version: resourceRequest.version, Resource: resourceRequest.resource}
	resourceObj, err := svc.resourceScheme.New(gvr)
	if err != nil {
		return newResponse(nil, err)
	}
	resourceObj, err = svc.client.Get(resourceRequest.resource, resourceRequest.namespace, name, metav1.GetOptions{}, resourceObj)
	if err != nil {
		glog.Warningf("Get resource error: %s, err: %v", resourceRequest.resource, err)
	}
	return newResponse(resourceObj, err)
}

func (svc *defaultService) createResource(resourceRequest *ResourceRequest, response *restful.Response) *rpcResponse {
	gvr := schema.GroupVersionResource{Group: resourceRequest.group, Version: resourceRequest.version, Resource: resourceRequest.resource}
	resourceObj, err := svc.resourceScheme.New(gvr)
	if err != nil {
		return newResponse(nil, err)
	}
	err = loadResource(resourceRequest.restRequest, resourceObj)
	if err != nil {
		glog.Warningf("Load resource error: %s, err: %v", resourceRequest.resource, err)
		return newResponse(nil, err)
	}
	glog.V(4).Infof("CreateResource %v", utils.ObjJSON(resourceObj))
	_, err = svc.client.Create(resourceRequest.resource, resourceRequest.namespace, resourceObj)
	if err != nil {
		glog.Warningf("Create resource error: %s, err: %v", resourceRequest.resource, err)
	}
	return newResponse(resourceObj, err)
}

func (svc *defaultService) updateResource(resourceRequest *ResourceRequest, response *restful.Response) *rpcResponse {
	gvr := schema.GroupVersionResource{Group: resourceRequest.group, Version: resourceRequest.version, Resource: resourceRequest.resource}
	resourceObj, err := svc.resourceScheme.New(gvr)
	if err != nil {
		return newResponse(nil, err)
	}
	err = loadResource(resourceRequest.restRequest, resourceObj)
	if err != nil {
		glog.Warningf("Load resource error: %s, err: %v", resourceRequest.resource, err)
		return newResponse(nil, err)
	}
	_, err = svc.client.Update(resourceRequest.resource, resourceRequest.namespace, resourceObj)
	if err != nil {
		glog.Warningf("Update resource error: %s, %s, err: %v", resourceRequest.resource, mem.Describe(resourceObj), err)
	}
	return newResponse(resourceObj, err)
}

func (svc *defaultService) deleteResource(resourceRequest *ResourceRequest, response *restful.Response) *rpcResponse {
	name := resourceRequest.name
	err := svc.client.Delete(resourceRequest.resource, resourceRequest.namespace, name, &metav1.DeleteOptions{})
	if err != nil {
		glog.Warningf("Delete resource error: %s/%s, err: %v", resourceRequest.resource, name, err)
	}
	return newResponse(nil, err)
}

func sendAPIResponse(request *restful.Request, hresponse *restful.Response, rpcResp *rpcResponse) {
	err := hresponse.WriteEntity(rpcResp)
	if nil != err {
		utils.RenderPlain(hresponse, http.StatusInternalServerError, err.Error())
	}
}

func newResponse(data interface{}, err error) *rpcResponse {
	resp := &rpcResponse{baseResponse: baseResponse{Code: 0}, Data: data}
	if err != nil {
		if errors.IsNotFound(err) {
			resp.Code = codeResourceNotFound
		} else {
			resp.Code = -1
		}
		resp.Msg = err.Error()
	}
	return resp
}

func loadResource(request *restful.Request, into runtime.Object) error {
	postRequest := &postRequest{Body: into}
	err := request.ReadEntity(postRequest)
	if err != nil {
		glog.Warningf("Load resource unmarshal error: %v", err)
		return err
	}
	return nil
}
