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

package elector

import (
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/abronan/valkeyrie"
	"github.com/abronan/valkeyrie/store"
	"github.com/abronan/valkeyrie/store/zookeeper"
)

// leader election based on distributed lock

func init() {
	zookeeper.Register()
}

// Elector is the elector api.
type Elector interface {
	GetLeader() (string, error)
	IsLeader() bool
	Elect(stopChan chan struct{}) (<-chan struct{}, error)
	Close() error
}

// NopElector is a test nop election implementation.
type NopElector struct {
}

// IsLeader check if it's the leader.
func (n NopElector) IsLeader() bool {
	return true
}

// GetLeader get the leader spec.
func (n NopElector) GetLeader() (string, error) {
	return "", nil
}

// Elect try to elect the leader.
func (n NopElector) Elect(stopChan chan struct{}) (<-chan struct{}, error) {
	return make(chan struct{}), nil
}

// Close close.
func (n NopElector) Close() error {
	return nil
}

// electorImpl the real election implementation.
type electorImpl struct {
	store    store.Store
	lock     store.Locker
	key      string
	isLeader int32
}

// ElectOptions is the election options.
type ElectOptions struct {
	Backend     string
	Hosts       []string
	Key         string
	Val         string
	ConnTimeout int
	LostTimeout int
}

// NewElector creates a new elector.
func NewElector(opts *ElectOptions) (Elector, error) {
	store0, err := valkeyrie.NewStore(store.Backend(opts.Backend), opts.Hosts,
		&store.Config{
			ConnectionTimeout: time.Duration(opts.ConnTimeout) * time.Second,
		})
	if err != nil {
		return nil, err
	}
	lock, err := store0.NewLock(opts.Key, &store.LockOptions{Value: []byte(opts.Val), TTL: time.Duration(opts.LostTimeout) * time.Second})
	if err != nil {
		return nil, err
	}
	return &electorImpl{
		store: store0,
		lock:  lock,
		key:   opts.Key,
	}, nil
}

func (elect *electorImpl) IsLeader() bool {
	return atomic.LoadInt32(&elect.isLeader) > 0
}

func (elect *electorImpl) GetLeader() (string, error) {
	pair, err := elect.store.Get(elect.key, nil)
	if err != nil {
		return "", err
	}
	return string(pair.Value), nil
}

func (elect *electorImpl) Elect(stopChan chan struct{}) (<-chan struct{}, error) {
	lostCh, err := elect.lock.Lock(stopChan)
	if err != nil {
		return lostCh, err
	}
	atomic.StoreInt32(&elect.isLeader, 1)
	return lostCh, err
}

func (elect *electorImpl) Close() error {
	elect.lock.Unlock()
	elect.store.Close()
	return nil
}

// ParseElectOptions parse election options from a url query string.
func ParseElectOptions(urlStr string) (*ElectOptions, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	backend := u.Scheme
	if backend == "zfs" {
		backend = string(store.ZK)
	}
	hosts := strings.Split(u.Host, ",")
	return &ElectOptions{Backend: backend, Hosts: hosts, Key: strings.TrimLeft(u.Path, "/")}, nil
}
