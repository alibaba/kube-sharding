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
	"reflect"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/liip/sheriff"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// PersistedVersion all tag version < PersistedVersion will NOT be persisted.
// Some fields in runtime.Object doesn't need to persisted, here we use `"github.com/liip/sheriff"` library to filter these fields.
// e.g: Metas map[string]string `json:"metas" until:"1.0.0"`, when 1.0.0 < PersistedVersion, `Metas` will not be persisted.
var PersistedVersion = version.Must(version.NewVersion(MemObjectVersionAnnoValue))

var metaFields = getJSONTags(metav1.ObjectMeta{})

func getJSONTags(data interface{}) []string {
	v := reflect.ValueOf(data)
	t := v.Type()
	tags := []string{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := parseJSONTag(field.Tag.Get("json"))
		if jsonTag == "" {
			jsonTag = field.Name
		}
		tags = append(tags, jsonTag)
	}
	return tags
}

func parseJSONTag(tag string) string {
	if idx := strings.Index(tag, ","); idx != -1 {
		return tag[:idx]
	}
	return tag
}

// FilterMarshaler marshals object to a json string.
type FilterMarshaler interface {
	FilterMarshalJSON() ([]byte, error)
}

// FilterMarshal filter some fields for a runtime.Object
func FilterMarshal(obj runtime.Object) ([]byte, error) {
	if ut, ok := obj.(*unstructured.Unstructured); ok {
		return json.Marshal(ut)
	}
	if ms, ok := obj.(FilterMarshaler); ok {
		return ms.FilterMarshalJSON()
	}
	opts := &sheriff.Options{ApiVersion: PersistedVersion}
	raw, err := sheriff.Marshal(opts, obj)
	if err != nil {
		return nil, fmt.Errorf("sheriff marshal error: %v", err)
	}
	// Check for safety reason (make sure the sheriff library are updated the fixed version)
	if kvs, ok := raw.(map[string]interface{}); !ok || kvs["metadata"] == nil {
		panic("require sheriff library commit ca229732c4e19cf9fef1b95c180cee74b8145d9b")
	}
	return json.Marshal(raw)
}
