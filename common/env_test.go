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
	"reflect"
	"testing"
)

func TestDeleteFromSlice(t *testing.T) {
	type args struct {
		src  string
		from []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "delete",
			args: args{
				src:  "a",
				from: []string{"a", "b"},
			},
			want: []string{"b"},
		},
		{
			name: "delete to nil",
			args: args{
				src:  "a",
				from: []string{"a"},
			},
			want: []string{},
		},
		{
			name: "no delete",
			args: args{
				src:  "c",
				from: []string{"a", "b"},
			},
			want: []string{"a", "b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeleteFromSlice(tt.args.src, tt.args.from); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DeleteFromSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddToUniqueSlice(t *testing.T) {
	type args struct {
		src string
		to  []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "no add",
			args: args{
				src: "a",
				to:  []string{"a", "b"},
			},
			want: []string{"a", "b"},
		},
		{
			name: "add",
			args: args{
				src: "a",
				to:  []string{"b"},
			},
			want: []string{"b", "a"},
		},
		{
			name: "add to nil",
			args: args{
				src: "a",
				to:  nil,
			},
			want: []string{"a"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AddToUniqueSlice(tt.args.src, tt.args.to); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AddToUniqueSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCorrecteShellExpr(t *testing.T) {
	type args struct {
		src string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "no need",
			args: args{
				src: "sh /home/admin/carbontest/script/start.sh normal tests/*",
			},
			want: "sh /home/admin/carbontest/script/start.sh normal tests/*",
		},
		{
			name: "correct end",
			args: args{
				src: "ip=$HIPPO_SLAVE_IP",
			},
			want: "ip=$(HIPPO_SLAVE_IP)",
		},
		{
			name: "correct start and multi",
			args: args{
				src: "$HIPPO_APP_INST_ROOT/usr/lib64/:$HIPPO_APP_INST_ROOT/usr/lib64/nvidia/:$HIPPO_APP_INST_ROOT/usr/local/cuda/lib64:$HIPPO_APP_INST_ROOT/usr/local/java/jdk/jre/lib/amd64/server:$HIPPO_APP_INST_ROOT/usr/local/lib64:$HIPPO_APP_INST_ROOT/home/admin/sap/lib/:$HIPPO_APP_INST_ROOT/home/admin/sap/lib64/:$HIPPO_APP_INST_ROOT/home/admin/eagleeye-core/lib/:$HIPPO_APP_INST_ROOT/home/admin/diamond-client4cpp/lib/",
			},
			want: "$(HIPPO_APP_INST_ROOT)/usr/lib64/:$(HIPPO_APP_INST_ROOT)/usr/lib64/nvidia/:$(HIPPO_APP_INST_ROOT)/usr/local/cuda/lib64:$(HIPPO_APP_INST_ROOT)/usr/local/java/jdk/jre/lib/amd64/server:$(HIPPO_APP_INST_ROOT)/usr/local/lib64:$(HIPPO_APP_INST_ROOT)/home/admin/sap/lib/:$(HIPPO_APP_INST_ROOT)/home/admin/sap/lib64/:$(HIPPO_APP_INST_ROOT)/home/admin/eagleeye-core/lib/:$(HIPPO_APP_INST_ROOT)/home/admin/diamond-client4cpp/lib/",
		},
		{
			name: "correct double $",
			args: args{
				src: "ip=$$HIPPO_SLAVE_IP",
			},
			want: "ip=$(HIPPO_SLAVE_IP)",
		},
		{
			name: "correct {} with end",
			args: args{
				src: "ip=$$${HIPPO_SLAVE_IP}",
			},
			want: "ip=$(HIPPO_SLAVE_IP)",
		},
		{
			name: "correct {} with start",
			args: args{
				src: "${HIPPO_APP_INST_ROOT}/usr/lib64/:${HIPPO_APP_INST_ROOT}/usr/lib64/nvidia/:$HIPPO_APP_INST_ROOT/usr/local/cuda/lib64:$HIPPO_APP_INST_ROOT/usr/local/java/jdk/jre/lib/amd64/server:$HIPPO_APP_INST_ROOT/usr/local/lib64:$HIPPO_APP_INST_ROOT/home/admin/sap/lib/:$HIPPO_APP_INST_ROOT/home/admin/sap/lib64/:$HIPPO_APP_INST_ROOT/home/admin/eagleeye-core/lib/:$HIPPO_APP_INST_ROOT/home/admin/diamond-client4cpp/lib/",
			},
			want: "$(HIPPO_APP_INST_ROOT)/usr/lib64/:$(HIPPO_APP_INST_ROOT)/usr/lib64/nvidia/:$(HIPPO_APP_INST_ROOT)/usr/local/cuda/lib64:$(HIPPO_APP_INST_ROOT)/usr/local/java/jdk/jre/lib/amd64/server:$(HIPPO_APP_INST_ROOT)/usr/local/lib64:$(HIPPO_APP_INST_ROOT)/home/admin/sap/lib/:$(HIPPO_APP_INST_ROOT)/home/admin/sap/lib64/:$(HIPPO_APP_INST_ROOT)/home/admin/eagleeye-core/lib/:$(HIPPO_APP_INST_ROOT)/home/admin/diamond-client4cpp/lib/",
		},
		{
			name: "correct () with end",
			args: args{
				src: "ip=$$$(HIPPO_SLAVE_IP)",
			},
			want: "ip=$(HIPPO_SLAVE_IP)",
		},
		{
			name: "correct () with start",
			args: args{
				src: "$(HIPPO_APP_INST_ROOT)/usr/lib64/:$(HIPPO_APP_INST_ROOT)/usr/lib64/nvidia/:$(HIPPO_APP_INST_ROOT)/usr/local/cuda/lib64:$(HIPPO_APP_INST_ROOT)/usr/local/java/jdk/jre/lib/amd64/server:$(HIPPO_APP_INST_ROOT)/usr/local/lib64:$(HIPPO_APP_INST_ROOT)/home/admin/sap/lib/:$(HIPPO_APP_INST_ROOT)/home/admin/sap/lib64/:$(HIPPO_APP_INST_ROOT)/home/admin/eagleeye-core/lib/:$(HIPPO_APP_INST_ROOT)/home/admin/diamond-client4cpp/lib/",
			},
			want: "$(HIPPO_APP_INST_ROOT)/usr/lib64/:$(HIPPO_APP_INST_ROOT)/usr/lib64/nvidia/:$(HIPPO_APP_INST_ROOT)/usr/local/cuda/lib64:$(HIPPO_APP_INST_ROOT)/usr/local/java/jdk/jre/lib/amd64/server:$(HIPPO_APP_INST_ROOT)/usr/local/lib64:$(HIPPO_APP_INST_ROOT)/home/admin/sap/lib/:$(HIPPO_APP_INST_ROOT)/home/admin/sap/lib64/:$(HIPPO_APP_INST_ROOT)/home/admin/eagleeye-core/lib/:$(HIPPO_APP_INST_ROOT)/home/admin/diamond-client4cpp/lib/",
		},
		{
			name: "correct splicted $",
			args: args{
				src: "$HIPPO_APP_IP $HIPPO_SLAVE_IP",
			},
			want: "$(HIPPO_APP_IP) $(HIPPO_SLAVE_IP)",
		},
		{
			name: "correct splicted $",
			args: args{
				src: "$HIPPO_APP_IP:$HIPPO_SLAVE_IP",
			},
			want: "$(HIPPO_APP_IP):$(HIPPO_SLAVE_IP)",
		},
		{
			name: "correct splicted $ and {}",
			args: args{
				src: "${HIPPO_APP_IP}:${HIPPO_SLAVE_IP}",
			},
			want: "$(HIPPO_APP_IP):$(HIPPO_SLAVE_IP)",
		},
		{
			name: "correct not end",
			args: args{
				src: "${HIPPO_APP_IP}:${HIPPO_SLAVE_IP",
			},
			want: "$(HIPPO_APP_IP):$(HIPPO_SLAVE_IP)",
		},
		{
			name: "empty",
			args: args{
				src: "",
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CorrecteShellExpr(tt.args.src); got != tt.want {
				t.Errorf("CorrecteShellExpr() = %v, want %v", got, tt.want)
			}
		})
	}
}
