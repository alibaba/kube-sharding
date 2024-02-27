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

// shardGroups implements shardGroups interface
type shardGroups struct {
	client client.Client
	ns     string
}

// newShardGroups returns a ShardGroups
func newShardGroups(client client.Client, namespace string) *shardGroups {
	return &shardGroups{
		client: client,
		ns:     namespace,
	}
}

// Get takes name of the ShardGroup, and returns the corresponding ShardGroups object, and an error if there is any.
func (c *shardGroups) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.ShardGroup, err error) {
	result = &v1.ShardGroup{}
	obj, err := c.client.Get("shardgroups", c.ns, name, options, result)
	if err != nil {
		return nil, err
	}
	return obj.(*v1.ShardGroup), err
}

// List takes label and field selectors, and returns the list of ShardGroups that match those selectors.
func (c *shardGroups) List(ctx context.Context, opts metav1.ListOptions) (result *v1.ShardGroupList, err error) {
	result = &v1.ShardGroupList{}
	err = c.client.List("shardgroups", c.ns, opts, result)
	return
}

// Watch returns a watch.Interface that watches the requested ShardGroups.
func (c *shardGroups) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return c.client.Watch("shardgroups", c.ns, &mem.WatchOptions{ListOptions: opts})
}

// Create takes the representation of a ShardGroups and creates it.  Returns the server's representation of the ShardGroups, and an error, if there is any.
func (c *shardGroups) Create(ctx context.Context, shardGroup *v1.ShardGroup, opts metav1.CreateOptions) (*v1.ShardGroup, error) {
	runtimeObj, err := c.client.Create("shardgroups", c.ns, shardGroup)
	if err != nil {
		return nil, err
	}
	return runtimeObj.(*v1.ShardGroup), nil
}

// Update takes the representation of a ShardGroups and updates it. Returns the server's representation of the ShardGroups, and an error, if there is any.
func (c *shardGroups) Update(ctx context.Context, shardGroup *v1.ShardGroup, opts metav1.UpdateOptions) (*v1.ShardGroup, error) {
	runtimeObj, err := c.client.Update("shardgroups", c.ns, shardGroup)
	if err != nil {
		return nil, err
	}
	return runtimeObj.(*v1.ShardGroup), nil
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().

func (c *shardGroups) UpdateStatus(ctx context.Context, shardGroup *v1.ShardGroup, opts metav1.UpdateOptions) (*v1.ShardGroup, error) {
	runtimeObj, err := c.client.Update("shardgroups", c.ns, shardGroup)
	if err != nil {
		return nil, err
	}
	return runtimeObj.(*v1.ShardGroup), nil
}

// Delete takes name of the ShardGroups and deletes it. Returns an error if one occurs.
func (c *shardGroups) Delete(ctx context.Context, name string, options metav1.DeleteOptions) error {
	return c.client.Delete("shardgroups", c.ns, name, &options)
}

// DeleteCollection deletes a collection of objects.
func (c *shardGroups) DeleteCollection(ctx context.Context, options metav1.DeleteOptions, listOptions metav1.ListOptions) error {
	return errors.New("unsupport func")
}

// Patch applies the patch and returns the patched ShardGroups.
func (c *shardGroups) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.ShardGroup, err error) {
	return nil, errors.New("unsupport func")
}
