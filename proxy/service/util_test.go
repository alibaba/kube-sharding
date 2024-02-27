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

package service

import "testing"

func TestSliceContainsSlice(t *testing.T) {
	type args struct {
		parent []string
		child  []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "normal case",
			args: args{
				parent: nil,
				child:  nil,
			},
			want: true,
		},
		{
			name: "normal case",
			args: args{
				parent: nil,
				child:  []string{},
			},
			want: false,
		},
		{
			name: "normal case",
			args: args{
				parent: []string{},
				child:  nil,
			},
			want: false,
		},
		{
			name: "normal case",
			args: args{
				parent: []string{"a", "b", "c"},
				child:  []string{"a", "c"},
			},
			want: true,
		},
		{
			name: "normal case",
			args: args{
				parent: []string{"a", "b", "c"},
				child:  []string{"a", "d"},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SliceContainsSlice(tt.args.parent, tt.args.child); got != tt.want {
				t.Errorf("SliceContainsSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}
