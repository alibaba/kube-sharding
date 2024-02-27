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
	"fmt"
	"testing"
)

func TestSignature(t *testing.T) {
	var obj = map[string]interface{}{
		"key1": 1,
		"key2": "test",
	}
	result, err := Signature(obj, 1, "test")
	if nil != err {
		t.Errorf("sign error %v", err)
	}
	if result != "8fe5d4720365d22a" {
		t.Errorf(" want %s , got %s", "8fe5d4720365d22a", result)
	}
}

func TestSignatureStringBytes(t *testing.T) {
	s := "hello"
	h0, err := Signature(s)
	if nil != err {
		t.Errorf("sign error %v", err)
	}
	h1, err := Signature([]byte(s))
	if nil != err {
		t.Errorf("sign error %v", err)
	}
	fmt.Printf("%s - %s\n", h0, h1)
	if h0 != h1 {
		t.Errorf("signature not equal")
	}
}

func TestSignatureShort(t *testing.T) {
	var obj = map[string]interface{}{
		"key1": 1,
		"key2": "test",
	}
	result, err := SignatureShort(obj, 1, "test")
	if nil != err {
		t.Errorf("sign error %v", err)
	}
	if result != "724d06ea" {
		t.Errorf(" want %s , got %s", "724d06ea", result)
	}
}

func TestSignatureMd5(t *testing.T) {
	var obj = map[string]interface{}{
		"key1": 1,
		"key2": "test",
	}
	result, err := SignatureWithMD5(obj, 1, "test")
	if nil != err {
		t.Errorf("sign error %v", err)
	}
	if result != "0eb7c34e5ce868eb3a1c6fa40867b28f" {
		t.Errorf(" want %s , got %s", "0eb7c34e5ce868eb3a1c6fa40867b28f", result)
	}
}
