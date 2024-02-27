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
	"sync"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	assert "github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

func newTestBatchObject(name string, count int, tweakFunc func(int, *spec.WorkerNode)) *BatchObject {
	pendings := make([]*pendingObject, 0)
	for i := 0; i < count; i++ {
		node := &spec.WorkerNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("node-%d", i),
			},
			Nick: fmt.Sprintf("nick-%d", i),
			Age:  i + 10,
			Status: spec.WorkerNodeStatus{
				Health: spec.HealthCondition{
					Metas:   map[string]string{"name": "emily willlis"},
					Success: true,
				},
			},
		}
		if tweakFunc != nil {
			tweakFunc(i, node)
		}
		pendings = append(pendings, &pendingObject{mutex: new(sync.RWMutex), current: node, event: &watch.Event{Object: node}})
	}
	batch := &BatchObject{
		Name:   name,
		Items:  pendings,
		Traits: map[string]string{"name": name},
	}
	return batch
}

// DeepEqual check if 2 BatchObject equals, for test only.
func (b *BatchObject) DeepEqual(b0 *BatchObject) (bool, error) {
	bs0, err := json.Marshal(b)
	if err != nil {
		return false, err
	}
	bs1, err := json.Marshal(b0)
	if err != nil {
		return false, err
	}
	fmt.Printf("batch0: %s\n", string(bs1))
	return string(bs0) == string(bs1), nil
}

func TestBatcherSerialize(t *testing.T) {
	assert := assert.New(t)
	batch := newTestBatchObject("b", 2, nil)
	bs, err := json.Marshal(batch)
	assert.Nil(err)
	fmt.Printf("%s\n", string(bs))

	decoder := &batchObjectUnserializer{creator: func() runtime.Object { return &spec.WorkerNode{} }}
	batch0, err := decoder.unmarshal(bs, encodingJSON)
	assert.Nil(err)
	assert.Equal(batch.Name, batch0.Name)
	eq, err := batch.DeepEqual(batch0)
	assert.Nil(err)
	assert.True(eq)
	// pb error
	_, err = batch.Marshal(encodingPB)
	assert.Error(err)
}

func TestBatcherKeyFunc(t *testing.T) {
	assert := assert.New(t)
	items := make([]*pendingObject, 0)
	label := "mkey"
	shardIDKey := "shardId"
	shardID := "1"
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			node := &spec.WorkerNode{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("node-%d-%d", i, j),
					Labels: map[string]string{
						label:      fmt.Sprintf("%d", i),
						shardIDKey: shardID,
					},
				},
			}
			items = append(items, newPendingObject(node, nil))
		}
	}
	traitsFunc := TraitsFuncLabels(shardIDKey)
	keyFunc := KeyFuncLabelValue(label)
	v, err := keyFunc(items[0].GetObject())
	assert.Nil(err)
	assert.Equal("0", v)
	// object missing label value.
	_, err = keyFunc(&spec.WorkerNode{})
	assert.NotNil(err)
	// append an invalid object into the list, we expect the batcher ignore the invalid object.
	items = append(items, newPendingObject(&spec.WorkerNode{}, nil))
	// append another trait failed object
	items = append(items, newPendingObject(&spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-no-traits",
			Labels: map[string]string{
				label: fmt.Sprintf("%d", 2),
			},
		},
	}, nil))
	// append a deleting object
	delPending := newPendingObject(&spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node-to-del",
			Labels: map[string]string{
				label: fmt.Sprintf("%d", 0),
			},
		},
	}, nil)
	delPending.RequestDelete(NewMetaInfo())
	items = append(items, delPending)

	batcher := &FuncBatcher{KeyFunc: keyFunc, TraitsFunc: traitsFunc}
	batches, err := batcher.Batch(items)
	assert.Nil(err)
	assert.Equal(2, len(batches))
	for _, batch := range batches {
		assert.Equal(1, len(batch.Traits))
		assert.Equal(shardID, batch.Traits[shardIDKey])
		var last string
		for _, item := range batch.Items {
			v, err = keyFunc(item.GetObject())
			assert.Nil(err)
			if last == "" {
				last = v
			} else {
				assert.Equal(last, v)
			}
		}
		if batch.Name == "0" { // see delPending batch label
			// verify the deleting item
			assert.Equal(1, len(batch.deletings))
			assert.Equal("node-to-del", batch.deletings[0].name)
		}
	}
}

func TestKeyFuncHashLabelValue(t *testing.T) {
	assert := assert.New(t)
	lblK := "lbl1"
	node := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node0",
			Labels: map[string]string{
				lblK: "hello",
			},
		},
	}
	fn := HashedNameFuncLabelValue(lblK, 1)
	k, err := fn(node)
	assert.Nil(err)
	fmt.Printf("keyFunc key: %s\n", k)
	assert.Equal("hello-0", k)
}

func TestBatcherSerializePB(t *testing.T) {
	assert := assert.New(t)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "p1",
			Generation: 111,
		},
	}
	pendings := make([]*pendingObject, 0)
	pendings = append(pendings, initPendingObject(pod))
	batch := &BatchObject{
		Name:   "b1",
		Items:  pendings,
		Traits: map[string]string{"name": "b1"},
	}
	bs, err := batch.marshalPB()
	assert.NoError(err)
	assert.True(isPBBytes(bs))

	un := &batchObjectUnserializer{
		creator: func() runtime.Object {
			return &corev1.Pod{}
		},
	}
	batch1, err := un.unmarshalPB(bs)
	assert.Nil(err)

	if utils.ObjJSON(batch) != utils.ObjJSON(batch1) {
		t.Errorf("Equal failed: \nE: %s\nG: %s\n", utils.ObjJSON(batch), utils.ObjJSON(batch1))
	}
	// compatible
	bs1, err := batch.Marshal(encodingJSON)
	assert.NoError(err)
	// failover to json
	batch1, err = un.unmarshal(bs1, encodingPB)
	assert.NoError(err)

	if utils.ObjJSON(batch) != utils.ObjJSON(batch1) {
		t.Errorf("Equal failed: \nE: %s\nG: %s\n", utils.ObjJSON(batch), utils.ObjJSON(batch1))
	}

	// empty pending objects
	batch = &BatchObject{
		Name: "b1",
	}
	bs, err = batch.Marshal(encodingPB)
	assert.NoError(err)
	batch1, err = un.unmarshalPB(bs)
	assert.Nil(err)

	if utils.ObjJSON(batch) != utils.ObjJSON(batch1) {
		t.Errorf("Equal failed: \nE: %s\nG: %s\n", utils.ObjJSON(batch), utils.ObjJSON(batch1))
	}
}
