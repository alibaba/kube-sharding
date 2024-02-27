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

package controller

import (
	"testing"

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
)

func Test_hashedNameFuncLabelValue(t *testing.T) {
	type args struct {
		key     string
		hashNum uint32
		obj     *carbonv1.WorkerNode
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "normal make hash",
			args: args{
				key:     carbonv1.LabelKeyCarbonJobName,
				hashNum: 10,
				obj: func() *carbonv1.WorkerNode {
					w := &carbonv1.WorkerNode{}
					w.Name = "test1"
					w.Labels = map[string]string{
						carbonv1.LabelKeyCarbonJobName: "c1",
					}
					return w
				}(),
			},
			want:    "c1-4",
			wantErr: false,
		},
		{
			name: "normal make hash",
			args: args{
				key:     carbonv1.LabelKeyCarbonJobName,
				hashNum: 10,
				obj: func() *carbonv1.WorkerNode {
					w := &carbonv1.WorkerNode{}
					w.Labels = map[string]string{
						carbonv1.LabelKeyCarbonJobName: "c1",
					}
					return w
				}(),
			},
			want:    "c1-1",
			wantErr: false,
		},
		{
			name: "failed on no label",
			args: args{
				key:     carbonv1.LabelKeyCarbonJobName,
				hashNum: 10,
				obj: func() *carbonv1.WorkerNode {
					w := &carbonv1.WorkerNode{}
					w.Labels = map[string]string{}
					return w
				}(),
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mem.HashedNameFuncLabelValue(tt.args.key, tt.args.hashNum)
			v, err := got(tt.args.obj)
			if (!tt.wantErr && nil != err) || (tt.wantErr && nil == err) {
				t.Errorf("hashedNameFuncLabelValue() err %v", err)
				return
			}
			if v != tt.want {
				t.Errorf("hashedNameFuncLabelValue() = %v, want %v", v, tt.want)
			}
		})
	}
}
