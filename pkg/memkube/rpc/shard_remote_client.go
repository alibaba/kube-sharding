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
	"net"
	"net/http"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/alibaba/kube-sharding/pkg/memkube/client"
	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	glog "k8s.io/klog"
)

// RemoteClientOpts remote client opts
type RemoteClientOpts struct {
	Timeout int // in second
	Src     string
	Cluster string
	Group   string
	Version string
	// set Accept http header, <resource, accept-header>
	Accepts map[string]string
}

type shardRemoteClient struct {
	resolver   EndpointResolver
	httpClient *utils.LiteHTTPClient
	rawclient  *http.Client
	src        string
	cluster    string
	pathFormat string

	serializer runtime.ClientNegotiator
	opts       *RemoteClientOpts
}

var _ client.Client = &shardRemoteClient{}

// NewRemoteClient new remote client
func NewRemoteClient(resolver EndpointResolver, serializer runtime.NegotiatedSerializer, opts *RemoteClientOpts) client.Client {
	var pathFormat string
	if opts.Version == "" {
		glog.Error("NewRemoteClient opts invaild, opts.version is nil")
		panic("newRemoteClient error")
	}
	if opts.Group == "" {
		pathFormat = fmt.Sprintf(versionURIFormat, opts.Version) + "/%s/%s"
	} else {
		pathFormat = fmt.Sprintf(groupVersionURIFormat, opts.Group, opts.Version) + "/%s/%s"
	}
	rawclient := &http.Client{
		Timeout: time.Duration(opts.Timeout) * time.Second,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
	gv := schema.GroupVersion{Group: opts.Group, Version: opts.Version}
	return &shardRemoteClient{
		resolver:   resolver,
		httpClient: utils.NewLiteHTTPClientWithClient(rawclient),
		rawclient:  rawclient,
		src:        opts.Src,
		cluster:    opts.Cluster,
		pathFormat: pathFormat,
		serializer: runtime.NewClientNegotiator(serializer, gv),
		opts:       opts,
	}
}

// Start remote client start
func (s *shardRemoteClient) Start(stopCh <-chan struct{}) error {
	return nil
}

func (s *shardRemoteClient) List(resource, ns string, opts metav1.ListOptions, into runtime.Object) error {
	if ns == "" {
		// HACK: At proxy startup, it List some resources with namespace='', actually memkube doesn't support to list all namespace resources,
		// to make the call success, return nil here. Otherwise the proxy will block on cache sync (List failed by resolve server with empty namespace).
		return nil
	}
	startTime := time.Now()
	defer Metrics.Get(MetricListRemoteTime).Elapsed(startTime, s.cluster, ns, resource)

	rpcNodes, err := s.resolver.ResolveAll(ns)
	if err != nil || len(rpcNodes) == 0 {
		return fmt.Errorf("resolve server addr failed, ns: %s, err: %v", ns, err)
	}
	accepts := s.getAccepts(resource)
	resourceVersion := uint64(0)
	allItems := make([]runtime.Object, 0)
	for _, rpcNode := range rpcNodes {
		invoker := NewSingleNodeInvoker(s.src, s.cluster, s.pathFormat, rpcNode, s.httpClient)
		invoker.SetAccepts(accepts)
		intoOne := into.DeepCopyObject()
		err := invoker.List(resource, ns, &opts, intoOne)
		if err != nil {
			return fmt.Errorf("list from remote %s failed, ns: %s, err: %v", rpcNode.String(), ns, err)
		}
		items, err := meta.ExtractList(intoOne)
		if err != nil {
			return fmt.Errorf("extract resource items error, ns: %s, err: %v", ns, err)
		}
		// NOTE: not really correct actually
		if len(items) > 0 {
			ver, err := mem.MetaResourceVersion(intoOne)
			if err != nil {
				return fmt.Errorf("not found resourceVersion in response: %v", err)
			}
			if ver >= resourceVersion {
				resourceVersion = ver
			}
		}
		allItems = append(allItems, items...)
	}
	mem.SetMetaResourceVersion(into, resourceVersion)
	meta.SetList(into, allItems)
	return nil
}

func (s *shardRemoteClient) Get(resource, ns, name string, opts metav1.GetOptions, into runtime.Object) (runtime.Object, error) {
	rpcNodes, err := s.resolver.ResolveAll(ns)
	if err != nil || len(rpcNodes) == 0 {
		return nil, fmt.Errorf("resolve server addr failed, ns: %s, err: %v", ns, err)
	}

	for _, rpcNode := range rpcNodes {
		invoker := NewSingleNodeInvoker(s.src, s.cluster, s.pathFormat, rpcNode, s.httpClient)
		intoOne := into.DeepCopyObject()
		result, err := invoker.Get(resource, ns, name, &opts, intoOne)
		if err != nil {
			if kerrors.IsNotFound(err) {
				glog.V(4).Infof("Get object from single node is not found: %s, %v", name, err)
				continue
			}
			glog.Warningf("Get object from single node failed: %s, %v", name, err)
			return nil, err
		}
		if result != nil {
			return result, nil
		}
	}
	return nil, kerrors.NewNotFound(schema.GroupResource{Resource: resource}, name)
}

func (s *shardRemoteClient) Create(resource, ns string, object runtime.Object) (runtime.Object, error) {
	rpcNode, err := s.resolver.Resolve(ns, object)
	if err != nil || rpcNode == nil {
		return nil, fmt.Errorf("resolve server addr failed, ns: %s, err: %v", ns, err)
	}
	invoker := NewSingleNodeInvoker(s.src, s.cluster, s.pathFormat, rpcNode, s.httpClient)
	result, err := invoker.Create(resource, ns, object)
	return result, err

}
func (s *shardRemoteClient) Update(resource, ns string, object runtime.Object) (runtime.Object, error) {
	rpcNode, err := s.resolver.Resolve(ns, object)
	if err != nil || rpcNode == nil {
		return nil, fmt.Errorf("resolve server addr failed, ns: %s, err: %v", ns, err)
	}

	invoker := NewSingleNodeInvoker(s.src, s.cluster, s.pathFormat, rpcNode, s.httpClient)
	name := mem.MetaName(object)
	if name == "" {
		return nil, fmt.Errorf("invaild obj, ns: %s", ns)
	}
	result, err := invoker.Update(resource, ns, name, object)
	return result, err

}
func (s *shardRemoteClient) Delete(resource, ns, name string, opts *metav1.DeleteOptions) error {
	rpcNodes, err := s.resolver.ResolveAll(ns)
	if err != nil || len(rpcNodes) == 0 {
		return fmt.Errorf("resolve server addr failed, ns: %s, err: %v", ns, err)
	}

	for _, rpcNode := range rpcNodes {
		invoker := NewSingleNodeInvoker(s.src, s.cluster, s.pathFormat, rpcNode, s.httpClient)
		err := invoker.Delete(resource, ns, name, opts)
		if err != nil {
			if kerrors.IsNotFound(err) {
				glog.V(4).Infof("Delete object on node not found: %s, %v", name, err)
				continue
			}
			return err
		}
	}
	return nil

}

func (s *shardRemoteClient) Watch(resource, ns string, opts *mem.WatchOptions) (watch.Interface, error) {
	if ns == "" {
		// See List comments
		return watch.NewEmptyWatch(), nil
	}
	rpcNodes, err := s.resolver.ResolveAll(ns)
	if err != nil || len(rpcNodes) == 0 {
		return nil, fmt.Errorf("resolve server addr failed, ns: %s, err: %v", ns, err)
	}
	// clone a http client without timeout
	httpclient := &http.Client{
		Transport: s.rawclient.Transport,
	}
	timeout := int(s.rawclient.Timeout.Seconds())
	if opts.TimeoutSeconds != nil && *opts.TimeoutSeconds > 0 {
		timeout = int(*opts.TimeoutSeconds)
	}
	accepts := s.getAccepts(resource)
	// TODO: support multiple shards watching
	rpcNode := rpcNodes[0]
	watcher := &watchClient{
		request: request{
			endpoint:   rpcNode.IP + ":" + rpcNode.Port,
			pathFormat: s.pathFormat,
			namespace:  ns,
			resource:   resource,
			source:     s.src,
			timeoutSec: timeout,
			watch:      true,

			selector:        opts.LabelSelector,
			resourceversion: opts.ResourceVersion,
		},
		client:     httpclient,
		negotiator: s.serializer,
		accepts:    accepts,
	}
	glog.Infof("Create watchClient: %+v", *watcher)
	return watcher.Watch()
}

func (s *shardRemoteClient) getAccepts(resource string) string {
	if s.opts != nil && s.opts.Accepts != nil && s.opts.Accepts[resource] != "" {
		return s.opts.Accepts[resource]
	}
	return ""
}
