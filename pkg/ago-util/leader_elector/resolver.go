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
	"errors"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-zookeeper/zk"
	glog "k8s.io/klog"
)

// fiber/carbon/hippo master resolver, resolve http-arpc endpoint from zookeeper

type zfsSt struct {
	hosts []string
	path  string
}

func parseZfs(zfs string) (*zfsSt, error) {
	u, err := url.Parse(zfs)
	if err != nil {
		return nil, err
	}
	return &zfsSt{
		hosts: strings.Split(u.Host, ","),
		path:  u.Path,
	}, nil
}

type masterResolver struct {
	zkconn *zk.Conn
}

const (
	electionSubPath = "LeaderElection"
)

// ResolveMaster resolves fiber/carbon master http arpc endpoint (host:port) from zookeeper.
// zfs://xxx/xxx/LeaderElection/$lock-file {endpoint-1\nendpoint-2}
func ResolveMaster(zfs string) (string, error) {
	r := &masterResolver{}
	defer r.Close()
	return r.Resolve(zfs)
}

func (r *masterResolver) Resolve(zfs string) (string, error) {
	zkInfo, err := parseZfs(zfs)
	if err != nil {
		glog.Errorf("parse zfs error: %s, %v", zfs, err)
		return "", err
	}
	r.zkconn, _, err = zk.Connect(zkInfo.hosts, time.Second*5)
	if err != nil {
		glog.Errorf("connect to zk error: %s, %v", zfs, err)
		return "", err
	}
	electPath := filepath.Join(zkInfo.path, electionSubPath)
	leaderSubPaths, _, err := r.zkconn.Children(electPath)
	if err != nil {
		glog.Errorf("list election children path failed: %s, %s, %v", zfs, electPath, err)
		return "", err
	}
	if len(leaderSubPaths) == 0 {
		return "", errors.New("no leader found in election path: " + zfs)
	}
	addrPath := filepath.Join(zkInfo.path, electionSubPath, leaderSubPaths[0])
	b, _, err := r.zkconn.Get(addrPath)
	if err != nil {
		glog.Errorf("get leader path data error: %s, %v", addrPath, err)
		return "", err
	}
	addrs := strings.Split(string(b), "\n")
	if len(addrs) < 2 {
		glog.Errorf("invalid addr data at path: %s, %s", addrPath, string(b))
		return "", err
	}
	return addrs[1], nil
}

func (r *masterResolver) Close() {
	if r.zkconn != nil {
		r.zkconn.Close()
		r.zkconn = nil
	}
}

// CacheResolver resolve endpoint with cache
type CacheResolver struct {
	zfs    string
	addr   string
	mutex  *sync.RWMutex
	implFn func(string) (string, error) // for test
}

// NewCacheResolver creates a cache resolver
func NewCacheResolver(zfs string) *CacheResolver {
	return &CacheResolver{zfs: zfs, implFn: ResolveMaster, mutex: new(sync.RWMutex)}
}

// Resolve resolves endpoint
func (r *CacheResolver) Resolve() (string, error) {
	if addr := r.loadCache(); addr != "" {
		return addr, nil
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()
	// double check
	if addr := r.addr; addr != "" {
		return addr, nil
	}
	addr, err := r.implFn(r.zfs)
	if err != nil {
		return "", err
	}
	r.addr = addr
	return addr, nil
}

func (r *CacheResolver) loadCache() string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.addr
}

// AfterCall reset cache address if failed.
func (r *CacheResolver) AfterCall(succ bool) {
	if !succ {
		r.mutex.Lock()
		defer r.mutex.Unlock()
		r.addr = ""
	}
}

// CacheResolverFactory manages a table of cached resolvers
type CacheResolverFactory struct {
	resolvers map[string]*CacheResolver
	rwLock    *sync.RWMutex
}

// NewCacheResolverFactory creates a new factory
func NewCacheResolverFactory() *CacheResolverFactory {
	return &CacheResolverFactory{
		resolvers: map[string]*CacheResolver{},
		rwLock:    new(sync.RWMutex),
	}
}

// Get returns a cache resolver
func (f *CacheResolverFactory) Get(zfs string) (*CacheResolver, error) {
	k := zfs
	f.rwLock.RLock()
	resolver, ok := f.resolvers[k]
	f.rwLock.RUnlock()
	if ok {
		return resolver, nil
	}
	resolver = NewCacheResolver(zfs)
	f.rwLock.Lock()
	f.resolvers[k] = resolver
	f.rwLock.Unlock()
	return resolver, nil
}
