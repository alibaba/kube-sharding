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

package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/emicklei/go-restful"
	proto "github.com/gogo/protobuf/proto"
	glog "k8s.io/klog"
)

// TestCallHTTPDecoder test func
func TestCallHTTPDecoder(method, uri string, body []byte, headers map[string]string,
	decoder func(rr *httptest.ResponseRecorder) (interface{}, error)) (interface{}, error) {
	httpReq, _ := http.NewRequest(method, uri, bytes.NewReader(body))
	for k, v := range headers {
		httpReq.Header.Add(k, v)
	}
	httpWriter := httptest.NewRecorder()
	restful.DefaultContainer.ServeHTTP(httpWriter, httpReq)
	return decoder(httpWriter)
}

// TestCallHTTPJson test func
func TestCallHTTPJson(method, uri string, body []byte, headers map[string]string, resp interface{}) (interface{}, error) {
	return TestCallHTTPDecoder(method, uri, body, headers, func(rr *httptest.ResponseRecorder) (interface{}, error) {
		if err := json.Unmarshal(rr.Body.Bytes(), resp); err != nil {
			glog.Errorf("unmarshal response error: %s, %d", string(rr.Body.Bytes()), rr.Code)
			return nil, fmt.Errorf("unmarshal response error: %s, %d, err %v", string(rr.Body.Bytes()), rr.Code, err)
		}
		return resp, nil
	})
}

// TestCallHTTPPB test func
func TestCallHTTPPB(method, uri string, body []byte, headers map[string]string, resp proto.Message) (interface{}, error) {
	if nil == headers {
		headers = map[string]string{}
	}
	headers["accept"] = "application/x-protobuf"
	//headers[restful.HEADER_ContentType] = "application/x-protobuf"
	return TestCallHTTPDecoder(method, uri, body, headers, func(rr *httptest.ResponseRecorder) (interface{}, error) {
		if err := proto.Unmarshal(rr.Body.Bytes(), resp); err != nil {
			glog.Errorf("unmarshal response error: %s, %d", string(rr.Body.Bytes()), rr.Code)
			return nil, fmt.Errorf("unmarshal response error: %s, %d, err %v", string(rr.Body.Bytes()), rr.Code, err)
		}
		return resp, nil
	})
}

// TestCallHTTPPBInput test func
func TestCallHTTPPBInput(method, uri string, body []byte, headers map[string]string, resp interface{}) (interface{}, error) {
	if nil == headers {
		headers = map[string]string{}
	}
	headers[restful.HEADER_ContentType] = "application/x-protobuf"
	return TestCallHTTPDecoder(method, uri, body, headers, func(rr *httptest.ResponseRecorder) (interface{}, error) {
		if err := json.Unmarshal(rr.Body.Bytes(), resp); err != nil {
			glog.Errorf("unmarshal response error: %s, %d", string(rr.Body.Bytes()), rr.Code)
			return nil, fmt.Errorf("unmarshal response error: %s, %d, err %v", string(rr.Body.Bytes()), rr.Code, err)
		}
		return resp, nil
	})
}

// matcher for gomock
type JsonMatcher struct {
	b string
}

func NewJSONMatcher(b []byte, obj interface{}) *JsonMatcher {
	// format to go json
	json.Unmarshal(b, obj)
	b, _ = json.Marshal(obj)
	return &JsonMatcher{string(b)}
}

func (m *JsonMatcher) String() string {
	return m.b
}

func (m *JsonMatcher) Matches(x interface{}) bool {
	b, _ := json.Marshal(x)
	return string(b) == m.b
}
