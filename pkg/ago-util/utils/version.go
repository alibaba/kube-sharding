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
	"fmt"
	"runtime"
)

var (
	branch  = "----"
	commit  = "----"
	date    = "00000000_0000"
	version = "v0.0.0"
)

// Version is the pkg build version
type Version struct {
	GitBranch string
	GitCommit string
	BuildDate string
	GoVersion string
	Compiler  string
	Platform  string
	Version   string
}

func (v *Version) String() string {
	return fmt.Sprintf(`
	Version:
	%-24s%s
	%-24s%s
	%-24s%s
	%-24s%s/%s/%s
	%-24s%s`,
		"branch:", v.GitBranch,
		"commit:", v.GitCommit,
		"build-date:", v.BuildDate,
		"compiler:", v.GoVersion, v.Compiler, v.Platform,
		"version:", v.Version)
}

// GetVersion get pkg build version
func GetVersion() *Version {
	return &Version{
		GitBranch: branch,
		GitCommit: commit,
		BuildDate: date,
		Version:   version,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}
