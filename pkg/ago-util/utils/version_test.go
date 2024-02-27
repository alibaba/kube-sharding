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

import (
	"strings"
	"testing"
)

func TestVersion_String(t *testing.T) {
	type fields struct {
		GitBranch string
		GitCommit string
		BuildDate string
		GoVersion string
		Compiler  string
		Platform  string
		Version   string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "version",
			fields: fields{
				GitBranch: "master",
				GitCommit: "ml01",
				BuildDate: "2019-05-05",
				GoVersion: "go1.12.5",
				Compiler:  "gc",
				Platform:  "darwin/amd64",
				Version:   "v0.0.1",
			},
			want: `
			Version:
			branch:                 master
			commit:                 ml01
			build-date:             2019-05-05
			compiler:               go1.12.5/gc/darwin/amd64
			version:                v0.0.1`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Version{
				GitBranch: tt.fields.GitBranch,
				GitCommit: tt.fields.GitCommit,
				BuildDate: tt.fields.BuildDate,
				GoVersion: tt.fields.GoVersion,
				Compiler:  tt.fields.Compiler,
				Platform:  tt.fields.Platform,
				Version:   tt.fields.Version,
			}
			if got := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(v.String(), "\n", ""), " ", ""), "\t", ""); got != strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(tt.want, "\n", ""), " ", ""), "\t", "") {
				t.Errorf("Version.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
