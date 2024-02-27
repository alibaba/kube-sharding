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

package rpc

import (
	"reflect"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/gogo/protobuf/proto"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPBMarshalPod(t *testing.T) {
	assert := assert.New(t)
	pod := &v1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind: "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:       "p1",
			Generation: 111,
		},
	}
	bs, err := proto.Marshal(pod)
	assert.Nil(err)
	var pod2 v1.Pod
	proto.Unmarshal(bs, &pod2)
	// NOTE: see vendor/k8s.io/api/core/v1/generated.proto Pod, there's no TypeMeta in pb response and i don't know why
	pod.Kind = ""
	if utils.ObjJSON(pod) != utils.ObjJSON(pod2) {
		t.Errorf("Equal failed: \nE: %s\nG: %s\n", utils.ObjJSON(pod), utils.ObjJSON(pod2))
	}
}

func Test_rpcResponse_XXX_Marshal(t *testing.T) {
	type fields struct {
		baseResponse baseResponse
		Data         interface{}
		OutData      interface{}
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			"Pod",
			fields{
				baseResponse: baseResponse{
					Code:      1,
					SubCode:   0,
					Msg:       "success",
					RequestID: "req-1",
				},
				Data: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "p1",
						Generation: 111,
					},
				},
				OutData: &v1.Pod{},
			},
		},
		{
			"Nil",
			fields{
				baseResponse: baseResponse{
					Code:      1,
					SubCode:   128,
					Msg:       "failed",
					RequestID: "req-1",
				},
				Data:    nil,
				OutData: nil,
			},
		},
		{
			"TypeMeta",
			fields{
				baseResponse: baseResponse{
					Code:      1,
					SubCode:   128,
					Msg:       "failed",
					RequestID: "req-1",
				},
				Data: &metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				OutData: &metav1.TypeMeta{},
			},
		},
		{
			"ListPod",
			fields{
				baseResponse: baseResponse{
					Code:      -1,
					SubCode:   128,
					Msg:       "failed",
					RequestID: "req-1",
				},
				Data: &v1.PodList{
					Items: []v1.Pod{
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:       "p1",
								Generation: 111,
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:       "p2",
								Generation: 113,
							},
						},
					},
				},
				OutData: &v1.PodList{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &rpcResponse{
				baseResponse: tt.fields.baseResponse,
				Data:         tt.fields.Data,
			}
			size := r.XXX_Size()
			bs, err := proto.Marshal(r)
			if err != nil {
				t.Errorf("Marshal error: %v", err)
			}
			if size != len(bs) {
				t.Errorf("Size not equal: %d %d", size, len(bs))
			}
			r2 := &rpcResponse{Data: tt.fields.OutData}
			if err := proto.Unmarshal(bs, r2); err != nil {
				t.Errorf("Unmarshal error: %v", err)
			}
			if !reflect.DeepEqual(r, r2) {
				t.Errorf("Equal failed: \nE: %s\nG: %s\n", utils.ObjJSON(r), utils.ObjJSON(r2))
			}
		})
	}
}

func Test_rpcResponse_json_Marshal(t *testing.T) {
	type fields struct {
		baseResponse baseResponse
		Data         interface{}
		OutData      interface{}
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			"Pod",
			fields{
				baseResponse: baseResponse{
					Code:      1,
					SubCode:   0,
					Msg:       "success",
					RequestID: "req-1",
				},
				Data: &v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:       "p1",
						Generation: 111,
					},
				},
				OutData: &v1.Pod{},
			},
		},
		{
			"Nil",
			fields{
				baseResponse: baseResponse{
					Code:      1,
					SubCode:   128,
					Msg:       "failed",
					RequestID: "req-1",
				},
				Data:    nil,
				OutData: nil,
			},
		},
		{
			"TypeMeta",
			fields{
				baseResponse: baseResponse{
					Code:      1,
					SubCode:   128,
					Msg:       "failed",
					RequestID: "req-1",
				},
				Data: &metav1.TypeMeta{
					Kind:       "Pod",
					APIVersion: "v1",
				},
				OutData: &metav1.TypeMeta{},
			},
		},
		{
			"ListPod",
			fields{
				baseResponse: baseResponse{
					Code:      -1,
					SubCode:   128,
					Msg:       "failed",
					RequestID: "req-1",
				},
				Data: &v1.PodList{
					Items: []v1.Pod{
						{
							TypeMeta: metav1.TypeMeta{
								Kind: "Pod",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name:       "p1",
								Generation: 111,
							},
						},
						{
							ObjectMeta: metav1.ObjectMeta{
								Name:       "p2",
								Generation: 113,
							},
						},
					},
				},
				OutData: &v1.PodList{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &rpcResponse{
				baseResponse: tt.fields.baseResponse,
				Data:         tt.fields.Data,
			}
			ed := &jsonAccessor{}
			bs, err := ed.Encode(r)
			if err != nil {
				t.Errorf("Marshal error: %v", err)
			}
			r2 := &rpcResponse{Data: tt.fields.OutData}
			if err := ed.Decode(bs, r2); err != nil {
				t.Errorf("Unmarshal error: %v", err)
			}
			if !reflect.DeepEqual(r, r2) {
				t.Errorf("Equal failed: \nE: %s\nG: %s\n", utils.ObjJSON(r), utils.ObjJSON(r2))
			}
		})
	}
}
