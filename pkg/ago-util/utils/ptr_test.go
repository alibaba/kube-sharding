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

package utils

import "testing"

func TestInt32Ptr(t *testing.T) {
	type args struct {
		v int32
	}
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "ptr32",
			args: args{v: 3},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Int32Ptr(tt.args.v); *got != tt.want {
				t.Errorf("Int32Ptr() = %v, want %v", *got, tt.want)
			}
		})
	}
}

func TestInt64Ptr(t *testing.T) {
	type args struct {
		v int64
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "ptr32",
			args: args{v: 3},
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Int64Ptr(tt.args.v); *got != tt.want {
				t.Errorf("Int64Ptr() = %v, want %v", got, tt.want)
			}
		})
	}
}
