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

package fstorage

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
)

// CompressStorage is a wrapper storage which compress the data.
type CompressStorage struct {
	fs FStorage
	cb func([]byte)
}

// Write writes data to storage.
func (c *CompressStorage) Write(bs []byte) error {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(bs); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	cbs := buf.Bytes()
	if c.cb != nil {
		c.cb(cbs)
	}
	return c.fs.Write(cbs)
}

// Read reads data from storage.
func (c *CompressStorage) Read() ([]byte, error) {
	bs, err := c.fs.Read()
	if err != nil {
		return nil, err
	}
	if len(bs) == 0 { // return success if empty
		return bs, nil
	}
	buf := bytes.NewBuffer(bs)
	zr, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return ioutil.ReadAll(zr)
}

// Exists check if the storage exists.
func (c *CompressStorage) Exists() (bool, error) {
	return c.fs.Exists()
}

// Create creates the storage.
func (c *CompressStorage) Create() error {
	return c.fs.Create()
}

// Remove removes the storage.
func (c *CompressStorage) Remove() error {
	return c.fs.Remove()
}

// Close close the storage.
func (c *CompressStorage) Close() error {
	return c.fs.Close()
}
