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
	"fmt"
	"net/url"
	"sync"
)

// original author: qiyan.zm@alibaba-inc.com for fiber-box project

// FStorage abstracs a file-like storage.
type FStorage interface {
	Write([]byte) error
	Read() ([]byte, error)
	Exists() (bool, error)
	Create() error
	Remove() error
	Close() error
}

// Location abstracs storage location, like a file direcotry.
type Location interface {
	List() ([]string, error)
	New(sub string) (FStorage, error)
	Create(sub string) (FStorage, error)
	NewSubLocation(sub ...string) (Location, error)
	Close() error
}

// LocationFactory creates a location.
type LocationFactory interface {
	Create(u *url.URL) (Location, error)
}

var factories map[string]LocationFactory

func init() {
	factories = make(map[string]LocationFactory)
}

// Register regists a new location implementation.
func Register(name string, factory LocationFactory) error {
	if _, ok := factories[name]; ok {
		return fmt.Errorf("storage location factory exists: %s", name)
	}
	factories[name] = factory
	return nil
}

// Create creates a location object by file url.
func Create(urlStr string) (Location, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	name := u.Scheme
	fac, ok := factories[name]
	if !ok {
		return nil, fmt.Errorf("no storage factory for: %s", name)
	}
	return fac.Create(u)
}

// CreateWithBackup creates a location object by file url.
func CreateWithBackup(urlStr, backupURLStr string) (Location, error) {
	master, err := Create(urlStr)
	if err != nil {
		return nil, err
	}
	var slave Location
	if backupURLStr != "" && urlStr != backupURLStr {
		slave, err = Create(backupURLStr)
		if err != nil {
			return nil, err
		}
	}
	return &MasterSlaveLocation{master: master, slave: slave, storageCache: make(map[string]FStorage), mutex: new(sync.RWMutex)}, nil
}

// MakeCompressive wraps a storage compressive.
func MakeCompressive(fs FStorage) FStorage {
	return &CompressStorage{fs: fs}
}

// MakeCompressiveNotify same as MakeCompressive but with a notifier which notified the compressed data.
func MakeCompressiveNotify(fs FStorage, cb func([]byte)) FStorage {
	return &CompressStorage{fs: fs, cb: cb}
}

// JoinPath joins a sub directory(location) to url.
func JoinPath(urlStr string, sub ...string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}
	for _, key := range sub {
		u.Path += "/" + key
	}

	return u.String()
}
