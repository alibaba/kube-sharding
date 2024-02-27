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
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
)

const (
	groupVersionURIFormat = "apis/%s/%s/namespaces"
	versionURIFormat      = "apis/%s/namespaces"
)

type baseRequest struct {
	RequestID string `json:"requestID"`
	Src       string `json:"src"`
}

type baseResponse struct {
	Code      int    `json:"code"`
	SubCode   int    `json:"subCode"`
	Msg       string `json:"msg"`
	RequestID string `json:"requestID"`
}

type postRequest struct {
	baseRequest
	Resource string      `json:"kind"`
	Body     interface{} `json:"body"`
}

type rpcResponse struct {
	baseResponse
	Data interface{} `json:"data,omitempty"`
}
type newMarshaler interface {
	XXX_Size() int
	XXX_Marshal(b []byte, deterministic bool) ([]byte, error)
}

func (r *rpcResponse) Reset() {
	m, ok := r.Data.(proto.Message)
	if ok {
		m.Reset()
	}
	*r = rpcResponse{}
	// keep the schema to Unmarshal
	r.Data = m
}

func (r *rpcResponse) String() string {
	return fmt.Sprintf("%+v", *r)
}

func (r *rpcResponse) ProtoMessage() {
}

func (r *rpcResponse) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	buf := proto.NewBuffer(b)
	buf.EncodeVarint(uint64(r.Code))
	buf.EncodeVarint(uint64(r.SubCode))
	buf.EncodeStringBytes(r.Msg)
	buf.EncodeStringBytes(r.RequestID)
	if r.Data != nil {
		buf.EncodeVarint(uint64(1))
		m, ok := r.Data.(proto.Message)
		if !ok {
			return nil, errors.New("data is not a pb Message")
		}
		if err := buf.EncodeMessage(m); err != nil {
			return nil, err
		}
	} else {
		buf.EncodeVarint(uint64(0))
	}
	return buf.Bytes(), nil
}

func (r *rpcResponse) XXX_Unmarshal(b []byte) error {
	buf := proto.NewBuffer(b)
	v, err := buf.DecodeVarint()
	if err != nil {
		return err
	}
	r.Code = int(v)
	v, err = buf.DecodeVarint()
	if err != nil {
		return err
	}
	r.SubCode = int(v)
	s, err := buf.DecodeStringBytes()
	if err != nil {
		return err
	}
	r.Msg = s
	s, err = buf.DecodeStringBytes()
	if err != nil {
		return err
	}
	r.RequestID = s
	if v, err := buf.DecodeVarint(); err != nil {
		return err
	} else if v == 1 {
		msg, ok := r.Data.(proto.Message)
		if !ok {
			return errors.New("require data set pb Message")
		}
		if err := buf.DecodeMessage(msg); err != nil {
			return err
		}
		r.Data = msg
	}
	return nil
}

func (r *rpcResponse) XXX_Size() int {
	n := proto.SizeVarint(uint64(1))
	if r.Data != nil {
		marshaler := r.Data.(newMarshaler)
		size := marshaler.XXX_Size()
		n += proto.SizeVarint(uint64(size))
		n += size
	}
	n += proto.SizeVarint(uint64(r.Code))
	n += proto.SizeVarint(uint64(r.SubCode))
	n += proto.SizeVarint(uint64(len(r.Msg)))
	n += len(r.Msg)
	n += proto.SizeVarint(uint64(len(r.RequestID)))
	n += len(r.RequestID)
	return n
}
