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
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io/ioutil"
)

// CompressStringToString  compress string to base64String used gzip
func CompressStringToString(content string) string {
	if content == "" {
		return content
	}
	var bytes bytes.Buffer
	w := gzip.NewWriter(&bytes)
	defer w.Close()
	w.Write([]byte(content))
	w.Flush()

	compressedContent := base64.StdEncoding.EncodeToString(bytes.Bytes())
	return compressedContent
}

// DecompressStringToString decompress base64String to string used gzip
func DecompressStringToString(compressedContent string) (string, error) {
	if compressedContent == "" {
		return compressedContent, nil
	}
	data, err := base64.StdEncoding.DecodeString(compressedContent)
	if err != nil {
		return "", err
	}
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer r.Close()
	content, _ := ioutil.ReadAll(r)
	return string(content), nil
}
