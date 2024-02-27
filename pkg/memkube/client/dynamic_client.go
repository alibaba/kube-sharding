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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	glog "k8s.io/klog"
)

// DynamicClient operates on client using *unstructured* type.
type DynamicClient struct {
	client Client
	scheme *mem.ResourceScheme
}

// NewDynamicClient create a new dynamic client.
func NewDynamicClient(client Client, scheme *mem.ResourceScheme) *DynamicClient {
	return &DynamicClient{client: client, scheme: scheme}
}

// List lists objects.
func (c *DynamicClient) List(ns string, resource schema.GroupVersionResource, opts metav1.ListOptions) (*unstructured.UnstructuredList, error) {
	rtList, err := c.scheme.NewList(resource)
	if err != nil {
		return nil, err
	}
	err = c.client.List(resource.Resource, ns, opts, rtList)
	if err != nil {
		return nil, err
	}
	rtObjs, err := meta.ExtractList(rtList)
	if err != nil {
		return nil, err
	}
	ulist := &unstructured.UnstructuredList{}
	for _, obj := range rtObjs {
		ut := &unstructured.Unstructured{}
		err = c.scheme.Scheme().Convert(obj, ut, nil)
		if err != nil {
			return nil, err
		}
		ulist.Items = append(ulist.Items, *ut)
	}
	return ulist, err
}

// Get gets an object.
func (c *DynamicClient) Get(ns, name string, resource schema.GroupVersionResource, opts metav1.GetOptions) (*unstructured.Unstructured, error) {
	ut := &unstructured.Unstructured{}
	rtObj, err := c.scheme.New(resource)
	if err != nil {
		return nil, err
	}
	rtObj, err = c.client.Get(resource.Resource, ns, name, opts, rtObj)
	if err != nil {
		return nil, err
	}
	err = c.scheme.Scheme().Convert(rtObj, ut, nil)
	return ut, err
}

// Create creates an object.
func (c *DynamicClient) Create(ns string, resource schema.GroupVersionResource, ut *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	rtObj, err := c.scheme.New(resource)
	if err != nil {
		glog.Warningf("New resource by scheme failed: %s, %v", resource.String(), err)
		return nil, err
	}
	if err = c.scheme.Scheme().Convert(ut, rtObj, nil); err != nil {
		glog.Warningf("Convert to runtime object failed: %v", err)
		return nil, err
	}
	ret, err := c.client.Create(resource.Resource, ns, rtObj)
	if err != nil {
		glog.Warningf("Create object failed: %v", err)
		return nil, err
	}
	uto := &unstructured.Unstructured{}
	if err = c.scheme.Scheme().Convert(ret, uto, nil); err != nil {
		glog.Warningf("Convert to unstructured object failed: %v", err)
		return nil, err
	}
	return uto, nil
}

// Update updates an object.
func (c *DynamicClient) Update(ns string, resource schema.GroupVersionResource, ut *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	rtObj, err := c.scheme.New(resource)
	if err != nil {
		glog.Warningf("New resource by scheme failed: %s, %v", resource.String(), err)
		return nil, err
	}
	if err = c.scheme.Scheme().Convert(ut, rtObj, nil); err != nil {
		glog.Warningf("Convert to runtime object failed: %v", err)
		return nil, err
	}
	ret, err := c.client.Update(resource.Resource, ns, rtObj)
	if err != nil {
		glog.Warningf("Update object failed: %v", err)
		return nil, err
	}
	uto := &unstructured.Unstructured{}
	if err = c.scheme.Scheme().Convert(ret, uto, nil); err != nil {
		glog.Warningf("Convert to unstructured object failed: %v", err)
		return nil, err
	}
	return uto, nil

}

// Delete deletes an object.
func (c *DynamicClient) Delete(ns, name string, resource schema.GroupVersionResource, opts *metav1.DeleteOptions) error {
	return c.client.Delete(resource.Resource, ns, name, opts)
}
