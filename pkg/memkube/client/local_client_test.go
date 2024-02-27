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
	"testing"
	"time"

	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	assert "github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClientDelete(t *testing.T) {
	assert := assert.New(t)
	stopCh := make(chan struct{})
	defer close(stopCh)
	store := mem.NewTestStore(stopCh, time.Millisecond*10)
	client := NewLocalClient(time.Millisecond * 20)
	ns := "ns"
	name := "w1"
	resource := "workernodes"
	client.Register(resource, ns, store)
	assert.Nil(client.Start(stopCh))
	node := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Age: 11,
	}
	_, err := client.Create(resource, ns, node)
	assert.Nil(err)
	// real deletion
	assert.Nil(client.Delete(resource, ns, name, &metav1.DeleteOptions{}))
	_, err = client.Get(resource, ns, name, metav1.GetOptions{}, nil)
	assert.True(errors.IsNotFound(err))

	// only set deletion timestamp
	node.Finalizers = []string{"a"}
	_, err = client.Create(resource, ns, node)
	var lastTm *metav1.Time
	for i := 0; i < 10; i++ {
		assert.Nil(client.Delete(resource, ns, name, &metav1.DeleteOptions{}))
		obj, err := client.Get(resource, ns, name, metav1.GetOptions{}, nil)
		assert.Nil(err)
		node0 := obj.(*spec.WorkerNode)
		assert.NotNil(node0.DeletionTimestamp)
		if lastTm == nil {
			lastTm = node0.DeletionTimestamp
		}
		// the deletion timestamp keep the same
		assert.True(lastTm.Equal(node0.DeletionTimestamp))
		assert.Equal(1, len(node0.Finalizers))
	}

	// clear the finalizers to do the real deletion
	obj, err := client.Get(resource, ns, name, metav1.GetOptions{}, nil)
	node0 := obj.(*spec.WorkerNode)
	node0.Finalizers = nil
	_, err = client.Update(resource, ns, node0)
	assert.Nil(err)
	assert.Nil(client.Delete(resource, ns, name, &metav1.DeleteOptions{}))
	_, err = client.Get(resource, ns, name, metav1.GetOptions{}, nil)
	assert.True(errors.IsNotFound(err))
}
