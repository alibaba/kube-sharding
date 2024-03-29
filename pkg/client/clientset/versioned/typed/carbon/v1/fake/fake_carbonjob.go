/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeCarbonJobs implements CarbonJobInterface
type FakeCarbonJobs struct {
	Fake *FakeCarbonV1
	ns   string
}

var carbonjobsResource = schema.GroupVersionResource{Group: "carbon.taobao.com", Version: "v1", Resource: "carbonjobs"}

var carbonjobsKind = schema.GroupVersionKind{Group: "carbon.taobao.com", Version: "v1", Kind: "CarbonJob"}

// Get takes name of the carbonJob, and returns the corresponding carbonJob object, and an error if there is any.
func (c *FakeCarbonJobs) Get(ctx context.Context, name string, options v1.GetOptions) (result *carbonv1.CarbonJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(carbonjobsResource, c.ns, name), &carbonv1.CarbonJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*carbonv1.CarbonJob), err
}

// List takes label and field selectors, and returns the list of CarbonJobs that match those selectors.
func (c *FakeCarbonJobs) List(ctx context.Context, opts v1.ListOptions) (result *carbonv1.CarbonJobList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(carbonjobsResource, carbonjobsKind, c.ns, opts), &carbonv1.CarbonJobList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &carbonv1.CarbonJobList{ListMeta: obj.(*carbonv1.CarbonJobList).ListMeta}
	for _, item := range obj.(*carbonv1.CarbonJobList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested carbonJobs.
func (c *FakeCarbonJobs) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(carbonjobsResource, c.ns, opts))

}

// Create takes the representation of a carbonJob and creates it.  Returns the server's representation of the carbonJob, and an error, if there is any.
func (c *FakeCarbonJobs) Create(ctx context.Context, carbonJob *carbonv1.CarbonJob, opts v1.CreateOptions) (result *carbonv1.CarbonJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(carbonjobsResource, c.ns, carbonJob), &carbonv1.CarbonJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*carbonv1.CarbonJob), err
}

// Update takes the representation of a carbonJob and updates it. Returns the server's representation of the carbonJob, and an error, if there is any.
func (c *FakeCarbonJobs) Update(ctx context.Context, carbonJob *carbonv1.CarbonJob, opts v1.UpdateOptions) (result *carbonv1.CarbonJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(carbonjobsResource, c.ns, carbonJob), &carbonv1.CarbonJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*carbonv1.CarbonJob), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeCarbonJobs) UpdateStatus(ctx context.Context, carbonJob *carbonv1.CarbonJob, opts v1.UpdateOptions) (*carbonv1.CarbonJob, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(carbonjobsResource, "status", c.ns, carbonJob), &carbonv1.CarbonJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*carbonv1.CarbonJob), err
}

// Delete takes name of the carbonJob and deletes it. Returns an error if one occurs.
func (c *FakeCarbonJobs) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(carbonjobsResource, c.ns, name), &carbonv1.CarbonJob{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeCarbonJobs) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(carbonjobsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &carbonv1.CarbonJobList{})
	return err
}

// Patch applies the patch and returns the patched carbonJob.
func (c *FakeCarbonJobs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *carbonv1.CarbonJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(carbonjobsResource, c.ns, name, pt, data, subresources...), &carbonv1.CarbonJob{})

	if obj == nil {
		return nil, err
	}
	return obj.(*carbonv1.CarbonJob), err
}
