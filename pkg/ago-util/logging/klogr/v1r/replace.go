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

package klog

import (
	"fmt"

	klogv2 "k8s.io/klog/v2"
)

// Symbols to replace
var (
	// only provdes by klogv2
	InitFlags    = klogv2.InitFlags
	Flush        = klogv2.Flush
	Info         = klogv2.Info
	InfoDepth    = klogv2.InfoDepth
	Infoln       = klogv2.Infoln
	Infof        = klogv2.Infof
	Warning      = klogv2.Warning
	WarningDepth = klogv2.WarningDepth
	Warningln    = klogv2.Warningln
	Warningf     = klogv2.Warningf
	Error        = klogv2.Error
	ErrorDepth   = klogv2.ErrorDepth
	Errorln      = klogv2.Errorln
	Errorf       = klogv2.Errorf
	Fatal        = klogv2.Fatal
	FatalDepth   = klogv2.FatalDepth
	Fatalln      = klogv2.Fatalln
	Fatalf       = klogv2.Fatalf
	Exit         = klogv2.Exit
	ExitDepth    = klogv2.ExitDepth
	Exitln       = klogv2.Exitln
	Exitf        = klogv2.Exitf
)

// Level log level
type Level = klogv2.Level

// Verbose adapter klogv2 `Verbose struct` to klog `Verbose bool`, without the klogv2 filter function.
type Verbose bool

// V returns a verbose
func V(level klogv2.Level) Verbose {
	return Verbose(klogv2.V(level).Enabled())
}

// Info writes args
func (v Verbose) Info(args ...interface{}) {
	if v {
		klogv2.InfoDepth(1, args...)
	}
}

// Infoln writes args
func (v Verbose) Infoln(args ...interface{}) {
	if v {
		klogv2.InfoDepth(1, args...)
	}
}

// Infof writes args
func (v Verbose) Infof(format string, args ...interface{}) {
	if v { // cpu-waste
		msg := fmt.Sprintf(format, args...)
		klogv2.InfoDepth(1, msg)
	}
}
