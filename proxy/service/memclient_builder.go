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

package service

import (
	"fmt"
	"strings"

	c2controller "github.com/alibaba/kube-sharding/cmd/controllermanager/controller"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	carbonclientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	"github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/scheme"
	memclient "github.com/alibaba/kube-sharding/pkg/memkube/client"
	"github.com/alibaba/kube-sharding/pkg/memkube/rpc"
	"github.com/alibaba/kube-sharding/pkg/memkube/rpc/resolver"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	coordinationv1 "k8s.io/client-go/informers/coordination/v1"
	informesv1 "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/tools/cache"
	glog "k8s.io/klog"
)

const (
	defaultTimeoutSecond = 60
)

var (
	accepts = map[string]string{
		"pods":        runtime.ContentTypeProtobuf,
		"workernodes": runtime.ContentTypeProtobuf,
	}
)

func createMixClient(builder simpleControllerClientBuilder, configmapInformer informesv1.ConfigMapInformer, leaseInformers coordinationv1.LeaseInformer, cluster string) carbonclientset.Interface {
	var memObjArray []string
	if opts.memObjs != "" {
		memObjArray = strings.Split(strings.ToLower(opts.memObjs), ",")
	}

	var remoteClient memclient.Client
	if len(memObjArray) > 0 {
		var err error
		remoteClient, err = createMemRemoteClient(configmapInformer, leaseInformers, cluster, carbonv1.SchemeGroupVersion.Group, carbonv1.SchemeGroupVersion.Version)
		if err != nil {
			panic(err)
		}
	}

	mixClient := builder.RemoteCarbonMixClientOrDie("carbon-proxy", remoteClient, memObjArray)
	return mixClient
}

func createMemRemoteClient(configmapInformer informesv1.ConfigMapInformer, leaseInformers coordinationv1.LeaseInformer, cluster, group, version string) (memclient.Client, error) {
	glog.Infof("Create memkube remote client memObjs: %s memTimeoutSecond: %v", opts.memObjs, opts.memTimeout)

	resolver := getResolver(configmapInformer, leaseInformers)
	timeout := opts.memTimeout
	if timeout <= 0 {
		timeout = defaultTimeoutSecond
	}
	opts := rpc.RemoteClientOpts{Src: "c2-proxy", Cluster: cluster, Timeout: timeout, Group: group, Version: version, Accepts: accepts}
	client := rpc.NewRemoteClient(resolver, scheme.Codecs.WithoutConversion(), &opts)
	return client, nil
}

func getResolver(configMapInformer informesv1.ConfigMapInformer, leaseInformers coordinationv1.LeaseInformer) rpc.EndpointResolver {
	c := resolver.NewResolver([]cache.SharedIndexInformer{configMapInformer.Informer(), leaseInformers.Informer()}, c2controller.MemkubeRegisterPrefix)
	c.RegisterCalcShardIDFunc(func(obj interface{}) (string, error) {
		meta, err := meta.Accessor(obj)
		if err != nil {
			return "", err
		}

		lbls := meta.GetLabels()
		if lbls == nil || lbls[carbonv1.ControllersShardKey] == "" {
			return "", fmt.Errorf("object has no label: %s", meta.GetName())
		}
		return lbls[carbonv1.ControllersShardKey], nil

	})
	return c
}
