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

package httph

import (
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils/restutil"
	"github.com/emicklei/go-restful"
)

var route = &restutil.EntityRoute{}

// InitRoute initialize basic route
func InitRoute(cluster string) {
	route.EncodeTimeFunc = func(request *restful.Request, response *restful.Response, latency time.Duration) {
		reportEncodeLatency(cluster, request.Request.URL.String(), request.Request.Method, latency.Nanoseconds())
	}
	route.WriteTimeFunc = func(request *restful.Request, response *restful.Response, latency time.Duration) {
		reportTransLatency(cluster, request.Request.URL.String(), request.Request.Method, latency.Nanoseconds())
	}
	route.RecoverRecorder = func(reason interface{}, request *restful.Request) {
		reportPanicCounter(cluster, request.Request.URL.String(), request.Request.Method)
	}
	route.ErrorFunc = func(request *restful.Request, response *restful.Response, e restutil.Entity) {
		reportResErrCounter(cluster, request.Request.URL.String(), request.Request.Method)
	}
}
