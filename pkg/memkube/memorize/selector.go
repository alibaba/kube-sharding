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

package memorize

import (
	"errors"

	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// OwnerLabelKey the label key in ListMetas
	OwnerLabelKey = "owner-label"
	// OwnerHashLabelKey get the owner hash
	OwnerHashLabelKey = "owner-hash-label"
)

// SelectorProvider provides selector in ListOptions
type SelectorProvider interface {
	Selector(ListMetas, runtime.Object) (string, error)
}

type nameSelector struct {
}

func (s *nameSelector) Selector(metas ListMetas, parent runtime.Object) (string, error) {
	set := labels.Set{}
	for k, v := range metas.Selectors {
		set[k] = v
	}
	if parent != nil && metas.Config != nil {
		hashLbl := metas.Config[OwnerHashLabelKey]
		var h string
		if hashLbl == "" {
			h = mem.MetaName(parent)
		} else { // otherwise get a hash value from parent
			mem.MetaAccessors(func(accessor ...metav1.Object) error {
				lbls := accessor[0].GetLabels()
				if lbls == nil || lbls[hashLbl] == "" {
					return errors.New("not found label hash in parent: " + hashLbl)
				}
				h = lbls[hashLbl]
				return nil
			}, parent)
		}
		k, ok := metas.Config[OwnerLabelKey]
		if ok {
			set[k] = h
		}
	}
	return set.String(), nil
}
