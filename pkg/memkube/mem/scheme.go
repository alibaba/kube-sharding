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
	"errors"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	glog "k8s.io/klog"
)

// ResourceScheme tracks the resource to kind mapping.
type ResourceScheme struct {
	scheme *runtime.Scheme
	mapper meta.RESTMapper
}

func createRESTMapper(scheme *runtime.Scheme) meta.RESTMapper {
	mapper := meta.NewDefaultRESTMapper([]schema.GroupVersion{})
	for gvk := range scheme.AllKnownTypes() {
		if !strings.HasSuffix(gvk.Kind, "List") { // hack
			// DefaultRESTMapper ToLower Kind and append `s` to a resource.
			mapper.Add(gvk, meta.RESTScopeNamespace)
		}
	}
	return mapper
}

// NewResourceScheme create a new ResourceScheme
func NewResourceScheme(scheme *runtime.Scheme) *ResourceScheme {
	return &ResourceScheme{
		scheme: scheme,
		mapper: createRESTMapper(scheme),
	}
}

// KindFor get the kind for a resource.
func (s *ResourceScheme) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	return s.mapper.KindFor(resource)
}

// New create an runtime.Object by resource type, similar with schema.Scheme.New
func (s *ResourceScheme) New(gvr schema.GroupVersionResource) (runtime.Object, error) {
	gvk, err := s.KindFor(gvr)
	if err != nil {
		return nil, errors.New("invalid resource: " + err.Error())
	}
	return s.scheme.New(gvk)
}

// NewList create a XXXList runtime.Obejct for resource.
func (s *ResourceScheme) NewList(gvr schema.GroupVersionResource) (runtime.Object, error) {
	gvk, err := s.KindFor(gvr)
	if err != nil {
		return nil, errors.New("invalid resource: " + err.Error())
	}
	gvk.Kind = gvk.Kind + "List" // hack
	return s.scheme.New(gvk)
}

// Scheme returns the runtime scheme.
func (s *ResourceScheme) Scheme() *runtime.Scheme {
	return s.scheme
}

// LogInfo log details of this scheme.
func (s *ResourceScheme) LogInfo() {
	var gvks []string
	for gvk := range s.scheme.AllKnownTypes() {
		gvks = append(gvks, gvk.String())
	}
	sort.Strings(gvks)
	glog.Infof("ResourceScheme details: %s", strings.Join(gvks, "\n"))
}
