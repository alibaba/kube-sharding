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

package mem

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	assert "github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPendingCommit(t *testing.T) {
	assert := assert.New(t)
	node := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("node"),
		},
	}
	pending := newPendingObject(node, nil)
	assert.Nil(pending.current)
	pending.Commit(false)
	assert.Nil(pending.GetEvent())
	assert.NotNil(pending.current)
}

// Test Delete before add committed.
func TestPendingAddDelete(t *testing.T) {
	assert := assert.New(t)
	node := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("node"),
		},
	}
	// the new created object is in pending
	pending := newPendingObject(node, nil)
	// now delete it
	pending.RequestDelete(NewMetaInfo())
	// verify the object is not nil
	assert.NotNil(pending.GetObject())
	// the version increase
	assert.Equal("1", node.GetResourceVersion())
}

func TestPendingMarshal(t *testing.T) {
	assert := assert.New(t)
	node := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("node"),
		},
		Age: 11,
	}
	pending := newPendingObject(node, nil)
	assert.True(pending.IsPending())
	// expect marshal the latest state.
	bs, err := json.Marshal(pending)
	assert.Nil(err)
	wn := &spec.WorkerNode{}
	assert.Nil(json.Unmarshal(bs, wn))
	p1 := initPendingObject(wn)
	assert.NotNil(p1.GetCurrent())
	w1 := p1.GetCurrent().(*spec.WorkerNode)
	assert.Equal(node.Age, w1.Age)
	assert.Equal(node.Name, w1.Name)
	assert.False(p1.IsPending())
}

func TestPendingMergeReadOnlyFields(t *testing.T) {
	assert := assert.New(t)
	node := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("node"),
		},
		Age: 11,
	}
	pending := newPendingObject(node, nil)
	node0 := node.DeepCopyObject().(*spec.WorkerNode)
	now := metav1.Now()
	node0.DeletionTimestamp = &now
	_, err := pending.RequestUpdate(node0, &MetaInfo{})
	assert.Nil(err)
	latest := pending.getLatest().(*spec.WorkerNode)
	assert.NotNil(latest.DeletionTimestamp)

	node1 := node.DeepCopyObject().(*spec.WorkerNode)
	node1.ResourceVersion = "3"
	node1.DeletionTimestamp = nil
	_, err = pending.RequestUpdate(node1, &MetaInfo{})
	assert.Nil(err)
	latest1 := pending.getLatest().(*spec.WorkerNode)
	assert.Equal(*latest.DeletionTimestamp, *latest1.DeletionTimestamp)
}
