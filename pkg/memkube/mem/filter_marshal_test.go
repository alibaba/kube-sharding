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
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestMarshalFilter(t *testing.T) {
	assert := assert.New(t)
	node := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "node0",
			Namespace:  "ns1",
			Generation: 11,
		},
		Age: 11,
		Status: spec.WorkerNodeStatus{
			Health: spec.HealthCondition{
				Metas:   map[string]string{"name": "emily willlis"},
				Success: true,
			},
		},
	}
	bs, err := FilterMarshal(node)
	assert.Nil(err)
	fmt.Printf("sheriff marshal: %s\n", string(bs))
	node0 := &spec.WorkerNode{}
	assert.Nil(json.Unmarshal(bs, node0))
	assert.Equal(node.ObjectMeta, node0.ObjectMeta)
	assert.Equal(node.Age, node0.Age)
	assert.Nil(node0.Status.Health.Metas) // filtered
}

func TestMarshalFilterTypes(t *testing.T) {
	assert := assert.New(t)
	// Require sheriff 09745b36e78d92220bea2c59d02ab9d996e4c8f5 commit to fix string panic
	// type ResourceName string
	_, err := FilterMarshal(&v1.Pod{
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Resources: v1.ResourceRequirements{
						Requests: v1.ResourceList{
							v1.ResourceCPU: resource.Quantity{},
						},
					},
				},
			},
		},
	})
	assert.Nil(err)
}

type WorkerNodeInnnerSpec struct {
	metav1.ObjectMeta `json:"metadata,omitempty"`
}

type WorkerNodeInner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              WorkerNodeInnnerSpec `json:"spec"`
}

func (in *WorkerNodeInner) DeepCopyObject() runtime.Object {
	return nil
}

func TestMarshalFilterInnerEmbedded(t *testing.T) {
	assert := assert.New(t)
	n := &WorkerNodeInner{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"l1": "v1",
			},
		},
		Spec: WorkerNodeInnnerSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"l2": "v2",
				},
			},
		},
	}
	b, err := FilterMarshal(n)
	assert.Nil(err)
	n1 := WorkerNodeInner{}
	assert.Nil(json.Unmarshal(b, &n1))
	assert.Equal("v1", n1.Labels["l1"])
	assert.Equal("v2", n1.Spec.Labels["l2"])
}
