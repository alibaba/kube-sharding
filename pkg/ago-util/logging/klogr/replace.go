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

// Package glog is used to replace glog by klog.
package glog

import (
	"fmt"

	"k8s.io/klog/v2"
)

// Symbols to replace
var (
	// only provdes by klog
	InitFlags    = klog.InitFlags
	Flush        = klog.Flush
	Info         = klog.Info
	InfoDepth    = klog.InfoDepth
	Infoln       = klog.Infoln
	Infof        = klog.Infof
	Warning      = klog.Warning
	WarningDepth = klog.WarningDepth
	Warningln    = klog.Warningln
	Warningf     = klog.Warningf
	Error        = klog.Error
	ErrorDepth   = klog.ErrorDepth
	Errorln      = klog.Errorln
	Errorf       = klog.Errorf
	Fatal        = klog.Fatal
	FatalDepth   = klog.FatalDepth
	Fatalln      = klog.Fatalln
	Fatalf       = klog.Fatalf
	Exit         = klog.Exit
	ExitDepth    = klog.ExitDepth
	Exitln       = klog.Exitln
	Exitf        = klog.Exitf
)

// Level log level
type Level = klog.Level

// Verbose adapter klog `Verbose struct` to glog `Verbose bool`, without the klog filter function.
type Verbose bool

// V returns a verbose
func V(level klog.Level) Verbose {
	return Verbose(klog.V(level).Enabled())
}

// Info writes args
func (v Verbose) Info(args ...interface{}) {
	if v {
		klog.InfoDepth(1, args...)
	}
}

// Infoln writes args
func (v Verbose) Infoln(args ...interface{}) {
	if v {
		klog.InfoDepth(1, args...)
	}
}

// Infof writes args
func (v Verbose) Infof(format string, args ...interface{}) {
	if v { // cpu-waste
		msg := fmt.Sprintf(format, args...)
		klog.InfoDepth(1, msg)
	}
}
