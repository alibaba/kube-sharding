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
	"encoding/json"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	assert "github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func createUnstructured() (*unstructured.Unstructured, *spec.WorkerNode) {
	node := &spec.WorkerNode{
		TypeMeta: metav1.TypeMeta{
			APIVersion: spec.SchemeGroupVersion.String(),
			Kind:       mem.Kind(&spec.WorkerNode{}),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "n1",
			CreationTimestamp: metav1.Now(),
		},
		Age: 11,
	}
	ut := &unstructured.Unstructured{}
	spec.Scheme.Convert(node, ut, nil)
	return ut, node
}

func TestUnstructuredMarshal(t *testing.T) {
	assert := assert.New(t)
	ut, node := createUnstructured()
	bs, err := mem.FilterMarshal(ut)
	assert.Nil(err)
	node0 := &spec.WorkerNode{}
	assert.Nil(json.Unmarshal(bs, node0))
	bs1, err := mem.FilterMarshal(node0)
	bs0, err := mem.FilterMarshal(node)
	assert.Nil(err)
	assert.Equal(string(bs1), string(bs0))
}
