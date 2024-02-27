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

package transfer

import (
	"encoding/json"
	"fmt"
	"testing"
)

type matcher struct {
	out json.RawMessage
}

func (m *matcher) Matches(x interface{}) bool {
	cp, ok := x.(*json.RawMessage)
	if ok {
		*cp = m.out
	}
	return true
}

func (m *matcher) String() string {
	return fmt.Sprintf("%+v", string(m.out))
}

func Test_escape(t *testing.T) {

	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "sp",
			args: args{
				name: "demo_pod",
			},
			want: "demo-pod",
		},
		{
			name: "sp1",
			args: args{
				name: "demo.pod",
			},
			want: "demo.pod",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EscapeName(tt.args.name)
			if result != tt.want {
				t.Errorf("status trans not equal  %s ,want %s", result, tt.want)
			}
		})
	}
}
