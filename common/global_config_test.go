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

package common

import (
	"testing"

	"github.com/alibaba/kube-sharding/common/config"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_getConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockConfiger := config.NewMockConfiger(ctrl)
	SetGlobalConfiger(mockConfiger, "hippo-c2-system")
	mockConfiger.EXPECT().GetString("staragent-sidecar_hippo-c2-system").Return("sigma", nil).Times(1)
	val, err := GetGlobalConfigVal("staragent-sidecar", "")
	assert.Nil(t, err)
	assert.Equal(t, "sigma", val)
}

func TestGetGlobalConfigValForObj(t *testing.T) {
	type args struct {
		confKey string
		kind    string
		object  metav1.Object
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	tests := []struct {
		name     string
		args     args
		configer config.Configer
		want     string
		wantErr  bool
	}{
		{
			name: "get",
			args: args{
				confKey: "interpose",
				kind:    "Pod",
				object: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"aa": "aaa",
							"bb": "bbb",
						},
						Namespace: "test",
					},
				},
			},
			configer: func() config.Configer {
				mockConfiger := config.NewMockConfiger(ctrl)
				SetGlobalConfiger(mockConfiger, "hippo-c2-system")
				mockConfiger.EXPECT().GetString("interpose_hippo-c2-system").Return(`
				[ 
				    {"matchRule":{"labelSelector":{"matchLabels":{"aa":"aaa"}},"kind":"Pod"},"data":"interpose-1"},
					{"matchRule":{"labelSelector":{"matchLabels":{"aa":"aaa"}},"namespaces":["test"],"kind":"Pod"},"data":"interpose-1"},
					{"matchRule":{"labelSelector":{"matchLabels":{"aa":"aaa"}},"namespaces":["test"]},"data":"interpose-2"}
				]`, nil).Times(1)
				return mockConfiger
			}(),
			want:    `"interpose-1"`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetGlobalConfiger(tt.configer, "hippo-c2-system")
			got, err := GetGlobalConfigValForObj(tt.args.confKey, tt.args.kind, tt.args.object)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGlobalConfigValForObj() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetGlobalConfigValForObj() = %v, want %v", got, tt.want)
			}
		})
	}
}
