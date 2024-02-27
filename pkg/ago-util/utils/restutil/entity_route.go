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

package restutil

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/emicklei/go-restful"
	proto "github.com/gogo/protobuf/proto"
	glog "k8s.io/klog"
)

// Entity represents response result.
type Entity interface {
	HasError() bool
}

// EntityHandler entity handler to process requests.
type EntityHandler func(request *restful.Request, response *restful.Response) Entity

// RecordTimeFunc function to record latency time
type RecordTimeFunc func(request *restful.Request, response *restful.Response, latency time.Duration)

// EntityRoute generates handler to restful service.
type EntityRoute struct {
	// Record the entity encoding time
	EncodeTimeFunc RecordTimeFunc
	// Usually to report panic metric
	RecoverRecorder func(reason interface{}, request *restful.Request)
	// logging
	LoggingFunc RecordTimeFunc
	// Record io write time
	WriteTimeFunc RecordTimeFunc
	// Called if got error entity
	ErrorFunc func(request *restful.Request, response *restful.Response, e Entity)
}

// To returns handler
func (r *EntityRoute) To(h EntityHandler) restful.RouteFunction {
	return func(request *restful.Request, response *restful.Response) {
		var err error
		if r.RecoverRecorder != nil {
			defer func() {
				if err := recover(); err != nil {
					r.RecoverRecorder(err, request)
					utils.HTTPRecover(err, response)
					glog.Errorf("recover from panic %s, %v", request.Request.URL.String(), err)
					return
				}
			}()
		}
		if r.WriteTimeFunc != nil {
			response.ResponseWriter = &timeRecordWriter{ResponseWriter: response.ResponseWriter, recorder: r.WriteTimeFunc, req: request, resp: response}
		}
		start := time.Now()
		e := h(request, response)
		if r.ErrorFunc != nil && e.HasError() {
			r.ErrorFunc(request, response, e)
		}
		if r.EncodeTimeFunc != nil {
			err = response.WriteEntity(&timeRecordEncoder{Entity: e, fn: r.EncodeTimeFunc, req: request, resp: response})
		} else {
			err = response.WriteEntity(e)
		}
		if nil != err {
			response.WriteError(http.StatusInternalServerError, err)
			glog.Errorf("write entity with error: %v", err)
		}
		if r.LoggingFunc != nil {
			r.LoggingFunc(request, response, time.Now().Sub(start))
		}
	}
}

type timeRecordWriter struct {
	http.ResponseWriter
	recorder RecordTimeFunc
	req      *restful.Request
	resp     *restful.Response
}

func (w *timeRecordWriter) Write(bs []byte) (int, error) {
	start := time.Now()
	len, err := w.ResponseWriter.Write(bs)
	w.recorder(w.req, w.resp, time.Now().Sub(start))
	return len, err
}

type timeRecordEncoder struct {
	// Export this field to make it can be encoded default for xml/pb etc.
	Entity interface{}
	fn     RecordTimeFunc
	req    *restful.Request
	resp   *restful.Response
}

func (s *timeRecordEncoder) MarshalJSON() ([]byte, error) {
	start := time.Now()
	b, err := json.Marshal(s.Entity)
	s.fn(s.req, s.resp, time.Now().Sub(start))
	return b, err
}

func (s *timeRecordEncoder) MarshalPB() ([]byte, error) {
	start := time.Now()
	message, ok := s.Entity.(proto.Message)
	if !ok { // 返回非pb类型时，默认json编码
		return s.MarshalJSON()
	}
	b, err := proto.Marshal(message)
	s.fn(s.req, s.resp, time.Now().Sub(start))
	return b, err
}
