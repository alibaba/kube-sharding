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
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPatchObject(t *testing.T) {
	type args struct {
		object metav1.Object
		patch  Patch
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		wantObject metav1.Object
	}{
		{
			name: "patch",
			args: args{
				object: &RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"k":        "v1",
							"kkk":      "vvv",
							"todelete": "todelete",
						},
					},
				},
				patch: Patch{
					Type:  PatchStrategyOverride,
					Patch: json.RawMessage(`{"metadata":{"labels":{"kk":"vv","k":"v","todelete":null}}}`),
				},
			},
			wantErr: false,
			wantObject: &RollingSet{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"k":        "v",
						"kk":       "vv",
						"kkk":      "vvv",
						"todelete": "",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := patchObject(tt.args.object, tt.args.patch); (err != nil) != tt.wantErr {
				t.Errorf("PatchObject() error = %v, wantErr %v", err, tt.wantErr)
			}
			if utils.ObjJSON(tt.args.object) != utils.ObjJSON(tt.wantObject) {
				t.Errorf("PatchObject() want = %v, got %v", utils.ObjJSON(tt.args.object), utils.ObjJSON(tt.wantObject))
			}
		})
	}
}
