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
	"fmt"
	"io/ioutil"

	restful "github.com/emicklei/go-restful"
	proto "github.com/gogo/protobuf/proto"
)

// NewEntityAccessorPB returns a new EntityReaderWriter for accessing PB content.
func NewEntityAccessorPB(contentType string) restful.EntityReaderWriter {
	return entityPBAccess{ContentType: contentType}
}

// entityPBAccess is a EntityReaderWriter for PB encoding
type entityPBAccess struct {
	// This is used for setting the Content-Type header when writing
	ContentType string
}

// Read unmarshalls the value from PB
func (e entityPBAccess) Read(req *restful.Request, v interface{}) error {
	message, ok := v.(proto.Message)
	if !ok {
		return getTypeError(v)
	}
	b, err := ioutil.ReadAll(req.Request.Body)
	if nil != err {
		return err
	}
	return proto.Unmarshal(b, message)
}

// Write marshalls the value to PB and set the Content-Type Header.
func (e entityPBAccess) Write(resp *restful.Response, status int, v interface{}) error {
	return writePB(resp, status, e.ContentType, v)
}

// write marshalls the value to PB and set the Content-Type Header.
func writePB(resp *restful.Response, status int, contentType string, v interface{}) error {
	if v == nil {
		resp.WriteHeader(status)
		// do not write a nil representation
		return nil
	}
	resp.Header().Set(restful.HEADER_ContentType, contentType)
	resp.WriteHeader(status)
	var b []byte
	var err error
	switch entity := v.(type) {
	case *timeRecordEncoder:
		b, err = entity.MarshalPB()
	case proto.Message:
		b, err = proto.Marshal(entity)
	default:
		err = getTypeError(v)
	}
	if nil != err {
		return err
	}
	_, err = resp.Write(b)
	return err
}

func getTypeError(v interface{}) error {
	return fmt.Errorf("%T is not proto Message", v)
}
