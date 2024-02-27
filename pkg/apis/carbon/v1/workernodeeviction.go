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

package v1

import (
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/hippo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkerNodeEviction WorkerNodeEviction
type WorkerNodeEviction struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkerNodeEvictionSpec   `json:"spec"`
	Status WorkerNodeEvictionStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// WorkerNodeEvictionList WorkerNodeEvictionList
type WorkerNodeEvictionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []WorkerNodeEviction `json:"items"`
}

// ReclaimWorkerNode ReclaimWorkerNode
type ReclaimWorkerNode struct {
	WorkerNodeID string       `json:"workerNodeId"`
	SlotID       hippo.SlotId `json:"slotId"`
}

// WorkerNodeEvictionSpec WorkerNodeEvictionSpec
type WorkerNodeEvictionSpec struct {
	AppName     string               `json:"appName"`
	WorkerNodes []*ReclaimWorkerNode `json:"workerNodes,omitempty"`
	Pref        *HippoPrefrence      `json:"preference,omitempty"`
}

// WorkerNodeEvictionStatus WorkerNodeEvictionStatus
type WorkerNodeEvictionStatus struct {
}

// HippoPrefrence HippoPrefrence
type HippoPrefrence struct {
	Scope *string `json:"scope,omitempty"`
	Type  *string `json:"type,omitempty"`
	TTL   *int32  `json:"ttl,omitempty"`
}
