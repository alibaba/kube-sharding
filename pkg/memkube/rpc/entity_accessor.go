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

package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	restful "github.com/emicklei/go-restful"
	"github.com/gogo/protobuf/proto"
	"k8s.io/apimachinery/pkg/runtime"
)

type encoderDecoder interface {
	Decode(bs []byte, v interface{}) error
	Encode(v interface{}) ([]byte, error)
}

type jsonAccessor struct {
}

func (a *jsonAccessor) Decode(bs []byte, v interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(bs))
	decoder.UseNumber()
	return decoder.Decode(v)
}

func (a *jsonAccessor) Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

type pbAccessor struct {
}

func (a *pbAccessor) Decode(bs []byte, v interface{}) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return fmt.Errorf("%T is not a pb message", v)
	}
	return proto.Unmarshal(bs, msg)
}

func (a *pbAccessor) Encode(v interface{}) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("%T is not a pb message", v)
	}
	return proto.Marshal(msg)
}

type entityAccessors struct {
	accessors map[string]encoderDecoder
	defaults  encoderDecoder
}

var entityAccessRegistry = &entityAccessors{
	accessors: map[string]encoderDecoder{
		restful.MIME_JSON:           &jsonAccessor{},
		runtime.ContentTypeProtobuf: &pbAccessor{},
	},
	defaults: &jsonAccessor{},
}

var defaultMIMEType = restful.MIME_JSON

func (r *entityAccessors) accessorAt(mime string) encoderDecoder {
	er, ok := r.accessors[mime]
	if !ok {
		// retry with reverse lookup
		// more expensive but we are in an exceptional situation anyway
		for k, v := range r.accessors {
			if strings.Contains(mime, k) {
				return v
			}
		}
		return r.defaults
	}
	return er
}
