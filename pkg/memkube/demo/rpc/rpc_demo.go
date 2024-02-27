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
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/alibaba/kube-sharding/pkg/memkube/rpc"
	"github.com/alibaba/kube-sharding/pkg/memkube/rpc/resolver"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/dummy"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	glog "k8s.io/klog"
)

const (
	demoPort = "34521"
)

// demo local obj
// curl -H "Content-Type:application/json"   -X POST -d '{"requestID":"xxxx","body":{"metadata":{"name":"test-add","namespace":"test"},"kind":"workernodes","nick":"add","Age":12}}' http://127.0.0.1:34521/api/memkubev1/namespaces/test/workernodes

func main() {
	flag.Parse()
	rand.Seed(time.Now().Unix())

	namespace := "test"
	staticResolver := resolver.StaticResolver{}
	nodeIndex := make(map[string]resolver.NamespaceNodes)
	nodeIndex[namespace] = resolver.NamespaceNodes{}
	nodeIndex[namespace]["1"] = &rpc.NodeInfo{IP: "127.0.0.1", Port: demoPort}
	nodeIndex[namespace]["2"] = &rpc.NodeInfo{IP: "127.0.0.1", Port: demoPort}
	staticResolver.SetIndex(nodeIndex)
	serializer := spec.Codecs.WithoutConversion()
	client := rpc.NewRemoteClient(&staticResolver, serializer, &rpc.RemoteClientOpts{Src: "c2-proxy", Version: "v1"})
	workerNodeInterface := carbon.NewWorkerNodeInterface(client, namespace)
	w1 := spec.WorkerNode{}
	w1.Name = "test-work1"
	w1.Namespace = "test"

	_, err := workerNodeInterface.Create(&w1)
	w2 := spec.WorkerNode{}
	w2.Name = "test-work2"
	w2.Namespace = "test"
	_, err = workerNodeInterface.Create(&w2)
	if err != nil {
		glog.Errorf("create error %v", err)
	}

	result, err := workerNodeInterface.List(metav1.ListOptions{})
	if err != nil {
		glog.Errorf("list error %v", err)
	}
	glog.Infof("interface result %v", utils.ObjJSON(result))

	dummyInformer := dummy.NewWorkerNodeInformer(client)
	lister := dummyInformer.Lister().WorkerNodes(namespace)
	result2, err := lister.List(labels.Everything())
	if err != nil {
		glog.Errorf("list error %v", err)
	}
	glog.Infof("lister result: %v", utils.ObjJSON(result2))
	for _, obj := range result2 {
		glog.Infof("lister obj: %v", utils.ObjJSON(obj))
	}
}
