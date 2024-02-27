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

import "testing"

func TestIsInterfaceNil(t *testing.T) {
	type args struct {
		i interface{}
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nil",
			args: args{
				i: nil,
			},
			want: true,
		},
		{
			name: "nil obj",
			args: args{
				i: func() interface{} {
					var o *int
					return o
				}(),
			},
			want: true,
		},
		{
			name: "not nil",
			args: args{
				i: func() interface{} {
					var i = 5
					var o *int = &i
					return o
				}(),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInterfaceNil(tt.args.i); got != tt.want {
				t.Errorf("IsInterfaceNil() = %v, want %v", got, tt.want)
			}
		})
	}
}
