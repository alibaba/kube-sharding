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

package spec

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// WorkerNode sample object spec.
type WorkerNode struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Nick string `json:"nick"`
	Age  int    `json:"age"`

	Kvs    map[string]string `json:"kvs,omitempty"`
	Status WorkerNodeStatus  `json:"status"`
}

// WorkerNodeStatus the status for the worker.
type WorkerNodeStatus struct {
	Health     HealthCondition `json:"health"`
	EntityName string          `json:"entityName"`
}

// HealthCondition holds the health checking result.
type HealthCondition struct {
	Success bool              `json:"success"`
	Metas   map[string]string `json:"metas" until:"1.0.0"`
}

// WorkerNodeList list object for WorkerNode.
type WorkerNodeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []WorkerNode `json:"items"`
}

// DeepCopyObject deep copy
func (in *WorkerNode) DeepCopyObject() runtime.Object {
	out := new(WorkerNode)
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	return out
}

// DeepCopyObject deep copy
func (in *WorkerNodeList) DeepCopyObject() runtime.Object {
	out := new(WorkerNodeList)
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]WorkerNode, len(*in))
		for i := range *in {
			(*out)[i] = *((*in)[i].DeepCopyObject().(*WorkerNode))
		}
	}
	return out
}
