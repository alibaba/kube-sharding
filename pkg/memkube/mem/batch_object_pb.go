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

package mem

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gogo/protobuf/proto"
)

var (
	// To be compatible with old serialized json data
	header = []byte{0x9, 0x9, 0x6}
)

type pbSizer interface {
	XXX_Size() int
}

type filterCopyer interface {
	Copy4PBPersist() (proto.Message, error)
}

// Marshal marshal this batch object
func (b *BatchObject) Marshal(encodig encodingType) ([]byte, error) {
	if encodig == encodingPB {
		return b.marshalPB()
	}
	return json.Marshal(b)
}

func (b *BatchObject) marshalPB() ([]byte, error) {
	size := b.pbSize()
	if size < 0 {
		return nil, errors.New("object is not XX_Size")
	}
	bs := make([]byte, 0, size)
	buf := proto.NewBuffer(bs)
	buf.EncodeRawBytes(header)
	buf.EncodeStringBytes(b.Name)
	buf.EncodeVarint(uint64(len(b.Traits)))
	for k, v := range b.Traits {
		buf.EncodeStringBytes(k)
		buf.EncodeStringBytes(v)
	}
	buf.EncodeVarint(uint64(len(b.Items)))
	for _, item := range b.Items {
		o := item.GetLatestObject()
		m, ok := o.(proto.Message)
		if !ok {
			return nil, errors.New("object is not a pb Message")
		}
		if f, ok := o.(filterCopyer); ok {
			copy, err := f.Copy4PBPersist()
			if err != nil {
				return nil, err
			}
			m = copy
		}
		if err := buf.EncodeMessage(m); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (b *BatchObject) pbSize() int {
	n := len(header)
	n += proto.SizeVarint(uint64(len(header)))
	n += proto.SizeVarint(uint64(len(b.Name)))
	n += len(b.Name)
	n += proto.SizeVarint(uint64(len(b.Traits)))
	for k, v := range b.Traits {
		n += proto.SizeVarint(uint64(len(k)))
		n += len(k)
		n += proto.SizeVarint(uint64(len(v)))
		n += len(v)
	}
	n += proto.SizeVarint(uint64(len(b.Items)))
	for _, item := range b.Items {
		o := item.GetLatestObject()
		sizer, ok := o.(pbSizer)
		if !ok {
			return -1
		}
		size := sizer.XXX_Size()
		n += proto.SizeVarint(uint64(size))
		n += size
	}
	return n
}

func (s *batchObjectUnserializer) unmarshalPB(bs []byte) (*BatchObject, error) {
	if !isPBBytes(bs) {
		return nil, errors.New("invalid pb bytes")
	}
	b := &BatchObject{
		Traits: make(map[string]string),
	}
	n := proto.SizeVarint(uint64(len(header)))
	bs = bs[n+len(header):]
	buf := proto.NewBuffer(bs)
	str, err := buf.DecodeStringBytes()
	if err != nil {
		return nil, err
	}
	b.Name = str
	v, err := buf.DecodeVarint()
	if err != nil {
		return nil, err
	}
	for i := 0; i < int(v); i++ {
		k, err := buf.DecodeStringBytes()
		if err != nil {
			return nil, err
		}
		v, err := buf.DecodeStringBytes()
		if err != nil {
			return nil, err
		}
		b.Traits[k] = v
	}
	v, err = buf.DecodeVarint()
	if err != nil {
		return nil, err
	}
	for i := 0; i < int(v); i++ {
		o := s.creator()
		msg, ok := o.(proto.Message)
		if !ok {
			return nil, fmt.Errorf("%T is not pb message", o)
		}
		if err := buf.DecodeMessage(msg); err != nil {
			return nil, err
		}
		po := initPendingObject(o)
		b.Items = append(b.Items, po)
	}
	return b, nil
}

func isPBBytes(bs []byte) bool {
	n := proto.SizeVarint(uint64(len(header)))
	if len(bs) < n+len(header) {
		return false
	}
	for i, b := range header {
		if bs[n+i] != b {
			return false
		}
	}
	return true
}
