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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

// DynamicMixResource mix kubernetes and mem-kube dynamic resource interface
type DynamicMixResource struct {
	scheme      *mem.ResourceScheme
	resources   map[string]bool
	kubeDynamic dynamic.Interface
	memDynamic  dynamic.Interface
}

func gvkMap(gvks []schema.GroupVersionKind) map[string]bool {
	m := make(map[string]bool)
	for _, gvk := range gvks {
		m[gvk.String()] = true
	}
	return m
}

// NewDynamicMixResource creates a new DynamicMixResource
func NewDynamicMixResource(client Client, config *rest.Config, scheme *mem.ResourceScheme, gvks []schema.GroupVersionKind) (*DynamicMixResource, error) {
	i, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &DynamicMixResource{
		scheme:      scheme,
		memDynamic:  NewDynamicResource(NewDynamicClient(client, scheme)),
		kubeDynamic: i,
		resources:   gvkMap(gvks),
	}, nil
}

// Resource returns dynamic.NamespaceableResourceInterface by resources
func (d *DynamicMixResource) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	gvk, err := d.scheme.KindFor(resource)
	if err != nil {
		panic("not found kind for resource " + resource.String())
	}
	_, ok := d.resources[gvk.String()]
	if ok {
		return d.memDynamic.Resource(resource)
	}
	return d.kubeDynamic.Resource(resource)
}
