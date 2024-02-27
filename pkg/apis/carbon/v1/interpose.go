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
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/alibaba/kube-sharding/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	glog "k8s.io/klog"
)

// PatchStrategy PatchStrategy
type PatchStrategy string

// PatchStrategy
const (
	PatchStrategyOverride PatchStrategy = "override"
	PatchStrategyMerge    PatchStrategy = "merge"
)

type Patch struct {
	Type  PatchStrategy   `json:"type"`
	Patch json.RawMessage `json:"patch"`
}

func PatchObject(kind string, object metav1.Object) error {
	val, err := common.GetGlobalConfigValForObj(common.ConfInterposeObject, kind, object)
	if err != nil {
		glog.Errorf("PatchObject with error: %s, %v", kind, err)
		return err
	}
	glog.V(5).Infof("GetGlobalConfigValForObj %s, %s", object.GetName(), val)
	if val == "" {
		return nil
	}
	return patchObjectFromString(object, val)
}

func patchObjectFromString(object metav1.Object, patchStr string) error {
	var patch Patch
	err := json.Unmarshal([]byte(patchStr), &patch)
	if err != nil {
		glog.Errorf("patchObjectFromString with error: %v", err)
		return err
	}
	return patchObject(object, patch)
}

func patchObject(object metav1.Object, patch Patch) error {
	vi := reflect.ValueOf(object)
	if vi.IsNil() {
		return nil
	}
	switch patch.Type {
	case PatchStrategyOverride:
		err := json.Unmarshal(patch.Patch, object)
		if err != nil {
			glog.Errorf("patchObjectFromString with error: %v", err)
		}
		return err
	default:
		return fmt.Errorf("not support patch type %s", patch.Type)
	}
}
