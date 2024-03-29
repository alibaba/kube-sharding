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

package testutil

import (
	"bytes"
	"compress/gzip"

	"github.com/alibaba/kube-sharding/pkg/ago-util/fstorage"
)

// CreateStore create s storage.
func CreateStore(name string) (fstorage.FStorage, error) {
	fsl, err := fstorage.Create("file:///tmp/fstore")
	if err != nil {
		return nil, err
	}
	return fsl.Create(name)
}

// Compress compress the data, compatible with compressive storage.
func Compress(bs []byte) ([]byte, error) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(bs); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
