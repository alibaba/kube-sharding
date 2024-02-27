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

package httph

import (
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	glog "k8s.io/klog"
)

var (
	once                                sync.Once
	registeredCustomResDynamicInterface map[schema.GroupVersionResource]CustomeResourceDynamicInterfaceCreator
)

// CustomeResourceDynamicInterfaceCreator creator
type CustomeResourceDynamicInterfaceCreator func(schema.GroupVersionResource) dynamic.NamespaceableResourceInterface

func init() {
	once.Do(func() {
		if registeredCustomResDynamicInterface == nil {
			registeredCustomResDynamicInterface = map[schema.GroupVersionResource]CustomeResourceDynamicInterfaceCreator{}
		}
	})
}

// RegistereCustomResourceDynamicInterface register resource dynamic interface
func RegistereCustomResourceDynamicInterface(gvr schema.GroupVersionResource, creator CustomeResourceDynamicInterfaceCreator) {
	if _, ok := registeredCustomResDynamicInterface[gvr]; ok {
		return
	}
	registeredCustomResDynamicInterface[gvr] = creator
	glog.Infof("register custom res dynamic interface creator gvr%+v", gvr)
}
