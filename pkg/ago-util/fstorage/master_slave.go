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
	"sync"

	glog "k8s.io/klog"
)

// MasterSlaveStore master&slave store implementation.
type MasterSlaveStore struct {
	masterFStorge FStorage
	slaveFStorge  FStorage
	slaveWriteCh  chan []byte
	name          string
	loc           *MasterSlaveLocation
}

func (l *MasterSlaveStore) init() error {
	return nil
}

// Write writes storage.
func (l *MasterSlaveStore) Write(bs []byte) error {
	err := l.masterFStorge.Write(bs)
	if err != nil {
		return err
	}
	if l.slaveFStorge != nil {
		l.slaveWriteCh <- bs
	}
	return nil
}

func (l *MasterSlaveStore) Read() ([]byte, error) {
	return l.masterFStorge.Read()
}

// Exists check if the storage exists.
func (l *MasterSlaveStore) Exists() (bool, error) {
	return l.masterFStorge.Exists()
}

// Create creates the storage.
func (l *MasterSlaveStore) Create() error {
	err := l.masterFStorge.Create()
	if err != nil {
		return err
	}
	if l.slaveFStorge != nil {
		err = l.slaveFStorge.Create()
		if err != nil {
			return err
		}
		go l.processSlave()
		return nil
	}
	return nil
}

func (l *MasterSlaveStore) processSlave() {
	for {
		select {
		case data, ok := <-l.slaveWriteCh:
			if !ok {
				return
			}
			err := l.slaveFStorge.Write(data)
			if err != nil {
				glog.Errorf("Slave fstorge write err: %s, %v", l.name, err)
			}
		}
	}
}

// Remove removes the storage.
func (l *MasterSlaveStore) Remove() error {
	err := l.masterFStorge.Remove()
	if err != nil {
		return err
	}
	if l.slaveFStorge != nil {
		err = l.slaveFStorge.Remove()
	}
	if err == nil {
		// The file does not exist, stop the goroutine, and remove the cache in location
		l.cleanUp()
		l.loc.removeFile(l.name)
	}
	return err
}

// Close close the storage.
func (l *MasterSlaveStore) Close() error {
	// Do nothing
	return nil
}

// cleanUp channel
func (l *MasterSlaveStore) cleanUp() {
	if l.slaveWriteCh != nil {
		close(l.slaveWriteCh)
		l.slaveWriteCh = nil
	}
}

// MasterSlaveLocation implements the master&slave location.
type MasterSlaveLocation struct {
	master       Location
	slave        Location
	storageCache map[string]FStorage
	mutex        *sync.RWMutex
}

// List lists the file in this location.
func (m *MasterSlaveLocation) List() ([]string, error) {
	return m.master.List()
}

// New new a sub storage from this location.
func (m *MasterSlaveLocation) New(sub string) (FStorage, error) {
	mfs, err := m.master.New(sub)
	if err != nil {
		return nil, err
	}
	var sfs FStorage
	var slaveWriteCh chan []byte
	if m.slave != nil {
		sfs, err = m.slave.New(sub)
		if err != nil {
			return nil, err
		}
		slaveWriteCh = make(chan []byte, 1)
	}

	return &MasterSlaveStore{masterFStorge: mfs, slaveFStorge: sfs, slaveWriteCh: slaveWriteCh, name: sub, loc: m}, nil
}

// Create creates a sub storage in this locatio with cache
func (m *MasterSlaveLocation) Create(sub string) (FStorage, error) {
	var fstorage FStorage
	m.mutex.Lock()
	defer m.mutex.Unlock()
	fstorage, ok := m.storageCache[sub]
	if !ok {
		var err error
		fstorage, err = m.New(sub)
		if err != nil {
			return nil, err
		}
		if err = fstorage.Create(); err != nil {
			return nil, err
		}
		m.storageCache[sub] = fstorage
		glog.V(3).Infof("Add a storage cache: %s", sub)
	}
	return fstorage, nil
}

// NewSubLocation new a sub location.
func (m *MasterSlaveLocation) NewSubLocation(sub ...string) (Location, error) {
	master, err := m.master.NewSubLocation(sub...)
	if err != nil {
		return nil, err
	}
	var slave Location
	if m.slave != nil {
		slave, err = m.slave.NewSubLocation(sub...)
		if err != nil {
			return nil, err
		}
	}
	return &MasterSlaveLocation{master: master, slave: slave, storageCache: make(map[string]FStorage), mutex: new(sync.RWMutex)}, nil
}

//Close close this location
func (m *MasterSlaveLocation) Close() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	for _, fstorage := range m.storageCache {
		store := fstorage.(*MasterSlaveStore)
		store.cleanUp()
	}
	m.storageCache = nil
	m.master.Close()
	if m.slave != nil {
		m.slave.Close()
	}
	return nil
}

func (m *MasterSlaveLocation) removeFile(name string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.storageCache, name)
	glog.V(3).Infof("Remove a storage cache: %s", name)
}
