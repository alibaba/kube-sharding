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

package utils

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"hash"
	"hash/fnv"
)

// Signature compute signature for objs
func Signature(ins ...interface{}) (string, error) {
	h := fnv.New64a()
	return sign(h, ins...)
}

// SignatureShort compute signature for objs and return a short signature
func SignatureShort(ins ...interface{}) (string, error) {
	h := fnv.New32a()
	return sign(h, ins...)
}

// SignatureWithMD5 compute signature for objs with md5
func SignatureWithMD5(ins ...interface{}) (string, error) {
	h := md5.New()
	return sign(h, ins...)
}

// HashString64 hash string to uint64
func HashString64(b []byte) uint64 {
	h := fnv.New64a()
	h.Write(b)
	return h.Sum64()
}

func sign(h hash.Hash, ins ...interface{}) (string, error) {
	for i := range ins {
		if str, ok := ins[i].(string); ok {
			_, err := h.Write(StringCastByte(str))
			if nil != err {
				return "", err
			}
		} else if bs, ok := ins[i].([]byte); ok {
			_, err := h.Write(bs)
			if nil != err {
				return "", err
			}
		} else {
			b, err := json.Marshal(ins[i])
			if nil != err {
				return "", err
			}
			_, err = h.Write(b)
			if nil != err {
				return "", err
			}
		}
	}
	cipherStr := h.Sum(nil)
	value := hex.EncodeToString(cipherStr)
	return value, nil
}
