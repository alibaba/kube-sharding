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

package memv1

import (
	"context"
	"errors"

	v1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/memkube/client"
	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
)

// rollingSets implements RollingSetInterface
type rollingSets struct {
	client client.Client
	ns     string
}

// newRollingSets returns a RollingSets
func newRollingSets(client client.Client, namespace string) *rollingSets {
	return &rollingSets{
		client: client,
		ns:     namespace,
	}
}

// Get takes name of the rollingSet, and returns the corresponding rollingSet object, and an error if there is any.
func (c *rollingSets) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.RollingSet, err error) {
	result = &v1.RollingSet{}
	obj, err := c.client.Get("rollingsets", c.ns, name, options, result)
	if err != nil {
		return nil, err
	}
	return obj.(*v1.RollingSet), err
}

// List takes label and field selectors, and returns the list of RollingSets that match those selectors.
func (c *rollingSets) List(ctx context.Context, opts metav1.ListOptions) (result *v1.RollingSetList, err error) {
	result = &v1.RollingSetList{}
	err = c.client.List("rollingsets", c.ns, opts, result)
	return
}

// Watch returns a watch.Interface that watches the requested rollingSets.
func (c *rollingSets) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.client.Watch("rollingsets", c.ns, &mem.WatchOptions{ListOptions: opts})
}

// Create takes the representation of a rollingSet and creates it.  Returns the server's representation of the rollingSet, and an error, if there is any.
func (c *rollingSets) Create(ctx context.Context, rollingSet *v1.RollingSet, opts metav1.CreateOptions) (*v1.RollingSet, error) {
	runtimeObj, err := c.client.Create("rollingsets", c.ns, rollingSet)
	if err != nil {
		return nil, err
	}
	return runtimeObj.(*v1.RollingSet), nil
}

// Update takes the representation of a rollingSet and updates it. Returns the server's representation of the rollingSet, and an error, if there is any.
func (c *rollingSets) Update(ctx context.Context, rollingSet *v1.RollingSet, opts metav1.UpdateOptions) (*v1.RollingSet, error) {
	runtimeObj, err := c.client.Update("rollingsets", c.ns, rollingSet)
	if err != nil {
		return nil, err
	}
	return runtimeObj.(*v1.RollingSet), nil
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *rollingSets) UpdateStatus(ctx context.Context, rollingSet *v1.RollingSet, opts metav1.UpdateOptions) (*v1.RollingSet, error) {
	runtimeObj, err := c.client.Update("rollingsets", c.ns, rollingSet)
	if err != nil {
		return nil, err
	}
	return runtimeObj.(*v1.RollingSet), nil
}

// Delete takes name of the rollingSet and deletes it. Returns an error if one occurs.
func (c *rollingSets) Delete(ctx context.Context, name string, options metav1.DeleteOptions) error {
	return c.client.Delete("rollingsets", c.ns, name, &options)
}

// DeleteCollection deletes a collection of objects.
func (c *rollingSets) DeleteCollection(ctx context.Context, options metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return errors.New("unsupport func")
}

// Patch applies the patch and returns the patched rollingSet.
func (c *rollingSets) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.RollingSet, err error) {
	return nil, errors.New("unsupport func")
}
