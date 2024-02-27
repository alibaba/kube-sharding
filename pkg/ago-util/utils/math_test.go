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

func TestFloatRound(t *testing.T) {
	type args struct {
		f float64
		n int
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "test1",
			args: args{f: 0.3333333333333333, n: 6},
			want: 0.333333,
		},
		{
			name: "test1",
			args: args{f: 0.5, n: 6},
			want: 0.5,
		},
		{
			name: "test1",
			args: args{f: 0.333, n: 6},
			want: 0.333,
		},
		{
			name: "test1",
			args: args{f: 0.6666666666, n: 6},
			want: 0.666667,
		},
		{
			name: "test1",
			args: args{f: 0.333333, n: 6},
			want: 0.333333,
		},
		{
			name: "test1",
			args: args{f: 0.33333, n: 6},
			want: 0.33333,
		},
		{
			name: "test1",
			args: args{f: 0.111111, n: 6},
			want: 0.111111,
		},
		{
			name: "test1",
			args: args{f: 0.1111111, n: 6},
			want: 0.111111,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FloatRound(tt.args.f, tt.args.n); got != tt.want {
				t.Errorf("FloatRound() = %v, want %v", got, tt.want)
			}
		})
	}
}
