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

package main

import (
	"flag"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/alibaba/kube-sharding/pkg/memkube/rpc"
	"github.com/alibaba/kube-sharding/pkg/memkube/rpc/resolver"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
	glog "k8s.io/klog"
)

var opts struct {
	port      int
	namespace string
}

func init() {
	flag.IntVar(&opts.port, "p", 0, "server port")
	flag.StringVar(&opts.namespace, "n", "", "namespace")
}

func newResolver() rpc.EndpointResolver {
	namespace := opts.namespace

	staticResolver := resolver.StaticResolver{}
	nodeIndex := make(map[string]resolver.NamespaceNodes)
	nodeIndex[namespace] = resolver.NamespaceNodes{}
	nodeIndex[namespace]["1"] = &rpc.NodeInfo{IP: "127.0.0.1", Port: strconv.Itoa(opts.port)}
	staticResolver.SetIndex(nodeIndex)
	return &staticResolver
}

func bindEventHandler(informer cache.SharedInformer) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			glog.Infof("Handler on object added: %+v", obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			glog.Infof("Handler on object updated: %+v", newObj)
		},
		DeleteFunc: func(obj interface{}) {
			glog.Infof("Handle to delete object: %+v", obj)
		},
	})
}

func setupErrHandlers() {
	utilruntime.ErrorHandlers = append(utilruntime.ErrorHandlers, func(err error) {
		glog.ErrorDepth(2, err)
	})
}

func main() {
	setupErrHandlers()
	flag.Parse()
	rand.Seed(time.Now().Unix())
	stopCh := make(chan struct{})
	defer close(stopCh)
	defer glog.Flush()

	resolver := newResolver()
	serializer := spec.Codecs.WithoutConversion()
	clientOpts := &rpc.RemoteClientOpts{
		Src:     "c2-proxy",
		Group:   spec.SchemeGroupVersion.Group,
		Version: spec.SchemeGroupVersion.Version,
		Timeout: 30,
	}
	client := rpc.NewRemoteClient(resolver, serializer, clientOpts)
	informer := carbon.NewWorkerNodeInformer(client, opts.namespace, time.Duration(0))
	bindEventHandler(informer.Informer())
	go informer.Informer().Run(stopCh)
	glog.Info("Wait cache synced")
	cache.WaitForCacheSync(stopCh, informer.Informer().HasSynced)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGUSR1)

	<-sigChan
	glog.Infof("quit by signal")
}
