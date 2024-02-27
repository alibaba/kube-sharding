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
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

// DynamicResource dynamic interfaces compatible with client-go dynamic.Interface
type DynamicResource struct {
	client   *DynamicClient
	ns       string
	resource schema.GroupVersionResource
}

// NewDynamicResource creates a dynamic.Interface
func NewDynamicResource(client *DynamicClient) dynamic.Interface {
	return &DynamicResource{client: client}
}

// Resource creates a dynamic resource interface
func (d *DynamicResource) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	return &DynamicResource{client: d.client, resource: resource}
}

// Namespace creates a dynamic namespace resource interface
func (d *DynamicResource) Namespace(ns string) dynamic.ResourceInterface {
	return &DynamicResource{client: d.client, resource: d.resource, ns: ns}
}

// Create create a resource
func (d *DynamicResource) Create(ctx context.Context, obj *unstructured.Unstructured, options metav1.CreateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	d.apiCheck(subresources...)
	return d.client.Create(d.ns, d.resource, obj)
}

// Update updates a resource
func (d *DynamicResource) Update(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions, subresources ...string) (*unstructured.Unstructured, error) {
	d.apiCheck(subresources...)
	return d.client.Update(d.ns, d.resource, obj)
}

// UpdateStatus update status for a resource
func (d *DynamicResource) UpdateStatus(ctx context.Context, obj *unstructured.Unstructured, options metav1.UpdateOptions) (*unstructured.Unstructured, error) {
	panic("not implemented yet")
}

// Delete deletes a resource
func (d *DynamicResource) Delete(ctx context.Context, name string, options metav1.DeleteOptions, subresources ...string) error {
	d.apiCheck(subresources...)
	return d.client.Delete(d.ns, name, d.resource, &options)
}

// DeleteCollection delete a collection of resources.
func (d *DynamicResource) DeleteCollection(ctx context.Context, options metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	panic("not implemented yet")
}

// Get gets a resource
func (d *DynamicResource) Get(ctx context.Context, name string, options metav1.GetOptions, subresources ...string) (*unstructured.Unstructured, error) {
	d.apiCheck(subresources...)
	return d.client.Get(d.ns, name, d.resource, options)
}

// List lists resources
func (d *DynamicResource) List(ctx context.Context, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	return d.client.List(d.ns, d.resource, opts)
}

// Watch returns a watch interface for a resource
func (d *DynamicResource) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	panic("not implemented yet")
}

// Patch patches a resource
func (d *DynamicResource) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, options metav1.PatchOptions, subresources ...string) (*unstructured.Unstructured, error) {
	d.apiCheck(subresources...)
	panic("not implemented yet")
}

func (d *DynamicResource) apiCheck(subresources ...string) {
	if len(subresources) != 0 {
		panic("not implemented for sub resources")
	}
	if len(d.ns) == 0 {
		panic("require namespace")
	}
	if d.resource.Empty() {
		panic("require GroupVersionResource")
	}
}
