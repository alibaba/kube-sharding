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

package util

import (
	"strings"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// Patch is a patch that can be applied to a Kubernetes object.
type Patch interface {
	// Type is the PatchType of the patch.
	Type() types.PatchType
	// Data is the raw data representing the patch.
	Data(obj runtime.Object) ([]byte, error)
}

const (
	NULL_HOLDER     = "NULL_HOLDER"
	NULL_HOLDER_STR = "\"NULL_HOLDER\""
)

type CommonPatch struct {
	PatchType    types.PatchType        `json:"patchType"`
	PatchData    map[string]interface{} `json:"data"`
	SubResources []string
}

// Type implements Patch.
func (s *CommonPatch) Type() types.PatchType {
	return s.PatchType
}

// Data implements Patch.
func (s *CommonPatch) Data(obj runtime.Object) ([]byte, error) {
	return []byte(s.String()), nil
}

// SimpleData return data without error.
func (s *CommonPatch) SimpleData() []byte {
	return []byte(s.String())
}

func (s *CommonPatch) String() string {
	jsonStr := utils.ObjJSON(s.PatchData)
	return strings.Replace(jsonStr, NULL_HOLDER_STR, "null", -1)
}

func NewStrategicPatch() *CommonPatch {
	return &CommonPatch{PatchType: types.StrategicMergePatchType, PatchData: make(map[string]interface{})}
}

// 除 CRD 对象外，请使用上面的 NewStrategicPatch
func NewMergePatch() *CommonPatch {
	return &CommonPatch{PatchType: types.MergePatchType, PatchData: make(map[string]interface{})}
}

func (s *CommonPatch) AddFinalizer(item string) *CommonPatch {
	switch s.PatchType {
	case types.StrategicMergePatchType:
		if _, ok := s.PatchData["metadata"]; !ok {
			s.PatchData["metadata"] = make(map[string]interface{})
		}
		metadata := s.PatchData["metadata"].(map[string]interface{})

		if oldList, ok := metadata["finalizers"]; !ok {
			metadata["finalizers"] = []string{item}
		} else {
			metadata["finalizers"] = append(oldList.([]string), item)
		}
	}
	return s
}

func (s *CommonPatch) RemoveFinalizer(item string) *CommonPatch {
	switch s.PatchType {
	case types.StrategicMergePatchType:
		if _, ok := s.PatchData["metadata"]; !ok {
			s.PatchData["metadata"] = make(map[string]interface{})
		}
		metadata := s.PatchData["metadata"].(map[string]interface{})

		if oldList, ok := metadata["$deleteFromPrimitiveList/finalizers"]; !ok {
			metadata["$deleteFromPrimitiveList/finalizers"] = []string{item}
		} else {
			metadata["$deleteFromPrimitiveList/finalizers"] = append(oldList.([]string), item)
		}
	}
	return s
}

func (s *CommonPatch) OverrideFinalizer(items []string) *CommonPatch {
	switch s.PatchType {
	case types.MergePatchType:
		if _, ok := s.PatchData["metadata"]; !ok {
			s.PatchData["metadata"] = make(map[string]interface{})
		}
		metadata := s.PatchData["metadata"].(map[string]interface{})

		metadata["finalizers"] = items
	}
	return s
}

func (s *CommonPatch) InsertLabel(key, value string) *CommonPatch {
	switch s.PatchType {
	case types.StrategicMergePatchType:
		if _, ok := s.PatchData["metadata"]; !ok {
			s.PatchData["metadata"] = make(map[string]interface{})
		}
		metadata := s.PatchData["metadata"].(map[string]interface{})

		if oldMap, ok := metadata["labels"]; !ok {
			metadata["labels"] = map[string]string{key: value}
		} else {
			oldMap.(map[string]string)[key] = value
		}
	}
	return s
}

func (s *CommonPatch) DeleteLabel(key string) *CommonPatch {
	switch s.PatchType {
	case types.StrategicMergePatchType:
		if _, ok := s.PatchData["metadata"]; !ok {
			s.PatchData["metadata"] = make(map[string]interface{})
		}
		metadata := s.PatchData["metadata"].(map[string]interface{})

		if oldMap, ok := metadata["labels"]; !ok {
			metadata["labels"] = map[string]string{key: NULL_HOLDER}
		} else {
			oldMap.(map[string]string)[key] = NULL_HOLDER
		}
	}
	return s
}

// attention：不要修改覆盖写的特性
func (s *CommonPatch) InsertAnnotation(key, value string) *CommonPatch {
	switch s.PatchType {
	case types.StrategicMergePatchType:
		if _, ok := s.PatchData["metadata"]; !ok {
			s.PatchData["metadata"] = make(map[string]interface{})
		}
		metadata := s.PatchData["metadata"].(map[string]interface{})

		if oldMap, ok := metadata["annotations"]; !ok {
			metadata["annotations"] = map[string]string{key: value}
		} else {
			oldMap.(map[string]string)[key] = value
		}
	}
	return s
}

func (s *CommonPatch) DeleteAnnotation(key string) *CommonPatch {
	switch s.PatchType {
	case types.StrategicMergePatchType:
		if _, ok := s.PatchData["metadata"]; !ok {
			s.PatchData["metadata"] = make(map[string]interface{})
		}
		metadata := s.PatchData["metadata"].(map[string]interface{})

		if oldMap, ok := metadata["annotations"]; !ok {
			metadata["annotations"] = map[string]string{key: NULL_HOLDER}
		} else {
			oldMap.(map[string]string)[key] = NULL_HOLDER
		}
	}
	return s
}

func (s *CommonPatch) UpdatePodCondition(condition v1.PodCondition) *CommonPatch {
	switch s.PatchType {
	case types.StrategicMergePatchType:
		if _, ok := s.PatchData["status"]; !ok {
			s.PatchData["status"] = make(map[string]interface{})
		}
		status := s.PatchData["status"].(map[string]interface{})

		if oldList, ok := status["conditions"]; !ok {
			status["conditions"] = []v1.PodCondition{condition}
		} else {
			status["conditions"] = append(oldList.([]v1.PodCondition), condition)
		}
	}
	s.addSubResources("status")
	return s
}

func (s *CommonPatch) addSubResources(resource string) {
	if s.SubResources == nil {
		s.SubResources = []string{}
	}
	for i := range s.SubResources {
		if resource == s.SubResources[i] {
			return
		}
	}
	s.SubResources = append(s.SubResources, resource)
}
