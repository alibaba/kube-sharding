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
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
)

// LocalFile local file implementation.
type LocalFile struct {
	path string
}

func init() {
	Register("file", &LocalFileFactory{})
}

// Write writes storage.
func (l *LocalFile) Write(bs []byte) error {
	return ioutil.WriteFile(l.path, bs, 0644)
}

func (l *LocalFile) Read() ([]byte, error) {
	return ioutil.ReadFile(l.path)
}

// Exists check if the storage exists.
func (l *LocalFile) Exists() (bool, error) {
	if _, err := os.Stat(l.path); err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

// Create creates the storage.
func (l *LocalFile) Create() error {
	dir := path.Dir(l.path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}
	if exists, err := l.Exists(); err == nil && exists {
		return nil
	}
	f, err := os.Create(l.path)
	if err != nil {
		return err
	}
	f.Close()
	return nil
}

// Remove removes the storage.
func (l *LocalFile) Remove() error {
	return os.Remove(l.path)
}

// Close close the storage.
func (l *LocalFile) Close() error {
	return nil
}

// LocalFileLocation implements the local file location.
type LocalFileLocation struct {
	path string
}

// List lists the file in this location.
func (l *LocalFileLocation) List() ([]string, error) {
	entries, err := ioutil.ReadDir(l.path)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

// New new a sub storage from this location.
func (l *LocalFileLocation) New(sub string) (FStorage, error) {
	return &LocalFile{path: filepath.Join(l.path, sub)}, nil
}

// Create creates a sub storage in this location.
func (l *LocalFileLocation) Create(sub string) (FStorage, error) {
	fs, err := l.New(sub)
	if err != nil {
		return nil, err
	}
	if err = fs.Create(); err != nil {
		return nil, err
	}
	return fs, nil
}

// NewSubLocation new a sub location.
func (l *LocalFileLocation) NewSubLocation(sub ...string) (Location, error) {
	path := filepath.Join(l.path, filepath.Join(sub...))
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return nil, err
	}
	return &LocalFileLocation{path: path}, nil
}

//Close close this location
func (l *LocalFileLocation) Close() error {
	return nil
}

// LocalFileFactory creates a local file location.
type LocalFileFactory struct{}

// Create creates a location.
func (l *LocalFileFactory) Create(u *url.URL) (Location, error) {
	return &LocalFileLocation{path: u.Path}, nil
}
