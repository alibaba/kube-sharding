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

package client

import (
	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

// ReadClient represents a client to read a mem resource.
type ReadClient interface {
	List(resource, ns string, opts metav1.ListOptions, into runtime.Object) error
	Get(resource, ns, name string, opts metav1.GetOptions, into runtime.Object) (runtime.Object, error)
}

// Client is a generic client to read/write mem objects.
type Client interface {
	ReadClient
	Start(stopCh <-chan struct{}) error
	Create(resource, ns string, object runtime.Object) (runtime.Object, error)
	Update(resource, ns string, object runtime.Object) (runtime.Object, error)
	Delete(resource, ns, name string, opts *metav1.DeleteOptions) error
	Watch(resource, ns string, opts *mem.WatchOptions) (watch.Interface, error)
}
