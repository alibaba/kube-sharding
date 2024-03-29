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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

// Scheme the schme
var Scheme = runtime.NewScheme()

// SchemeGroupVersion the group & version
var SchemeGroupVersion = schema.GroupVersion{Group: "carbon.taobao.com", Version: "v1"}
var Codecs = serializer.NewCodecFactory(Scheme)
var ParameterCodec = runtime.NewParameterCodec(Scheme)

// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func init() {
	Scheme.AddKnownTypes(SchemeGroupVersion,
		&WorkerNode{},
		&WorkerNodeList{},
	)
	metav1.AddToGroupVersion(Scheme, SchemeGroupVersion)
	// To register conversation functions.
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
}
