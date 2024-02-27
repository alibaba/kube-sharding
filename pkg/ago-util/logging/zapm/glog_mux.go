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

package zapm

import (
	"go.uber.org/zap/zapcore"
	glog "k8s.io/klog"
)

type glogMux struct {
}

var muxFuncs = map[zapcore.Level]func(...interface{}){
	zapcore.DebugLevel: glog.V(4).Info,
	zapcore.InfoLevel:  glog.Info,
	zapcore.WarnLevel:  glog.Warning,
	zapcore.ErrorLevel: glog.Error,
}

func (m *glogMux) Write(lvl zapcore.Level, b []byte) {
	fn, ok := muxFuncs[lvl]
	if ok {
		fn(string(b[:len(b)-1])) // trim line end
	}
}
