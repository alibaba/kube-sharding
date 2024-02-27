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

package glog

import (
	"fmt"
	"runtime"
	"strings"

	log "github.com/kevinlynx/log4go"
)

// This package is used to replace the existed glog references.
func init() {
	// change the default logger level
	log.Global["stdout"].Level = log.FINE
}

const calldepth = 2

func getTraceInfo() string {
	_, file, line, ok := runtime.Caller(calldepth)
	if !ok {
		file = "???"
		line = 0
	}
	lastIndex := strings.LastIndex(file, "/")
	if lastIndex > 0 {
		file = file[lastIndex+1:]
	}
	return fmt.Sprintf("%s:%d", file, line)
}

// Debugf print debug log.
func Debugf(format string, args ...interface{}) {
	log.Log(log.DEBUG, getTraceInfo(), fmt.Sprintf(format, args...))
}

// Debug print debug log.
func Debug(msg string) {
	log.Log(log.DEBUG, getTraceInfo(), msg)
}

// Infof print info log.
func Infof(format string, args ...interface{}) {
	log.Log(log.INFO, getTraceInfo(), fmt.Sprintf(format, args...))
}

// Info print info log.
func Info(msg string) {
	log.Log(log.INFO, getTraceInfo(), msg)
}

// Warningf prints warning log.
func Warningf(arg0 interface{}, args ...interface{}) {
	var msg string
	switch first := arg0.(type) {
	case string:
		msg = fmt.Sprintf(first, args...)
	default:
		msg = fmt.Sprintf(fmt.Sprint(first)+strings.Repeat(" %v", len(args)), args...)
	}
	log.Log(log.WARNING, getTraceInfo(), msg)
}

// Warning prints warning log.
func Warning(arg0 interface{}, args ...interface{}) {
	var msg string
	switch first := arg0.(type) {
	case string:
		msg = fmt.Sprintf(first, args...)
	default:
		msg = fmt.Sprintf(fmt.Sprint(first)+strings.Repeat(" %v", len(args)), args...)
	}
	log.Log(log.WARNING, getTraceInfo(), msg)
}

// Errorf prints error log.
func Errorf(arg0 interface{}, args ...interface{}) {
	var msg string
	switch first := arg0.(type) {
	case string:
		msg = fmt.Sprintf(first, args...)
	default:
		msg = fmt.Sprintf(fmt.Sprint(first)+strings.Repeat(" %v", len(args)), args...)
	}
	log.Log(log.ERROR, getTraceInfo(), msg)
}

// Error prints error log.
func Error(arg0 interface{}, args ...interface{}) {
	var msg string
	switch first := arg0.(type) {
	case string:
		msg = fmt.Sprintf(first, args...)
	default:
		msg = fmt.Sprintf(fmt.Sprint(first)+strings.Repeat(" %v", len(args)), args...)
	}
	log.Log(log.ERROR, getTraceInfo(), msg)
}

// Flush flushs the cached log message.
func Flush() {
	log.Close()
}
