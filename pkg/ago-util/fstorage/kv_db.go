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
	"path/filepath"
	"strconv"
	"strings"

	"github.com/alibaba/kube-sharding/pkg/ago-util/fstorage/db"

	"github.com/abronan/valkeyrie/store"
	"github.com/abronan/valkeyrie/store/zookeeper"
	"github.com/go-zookeeper/zk"
	glog "k8s.io/klog"
)

// DefaultTimeout is the timeout to operate on remote kv storage.
const DefaultTimeout = 10 // seconds

func init() {
	zookeeper.Register()
	db.Register()
	// To remove these go module deps, i remove etcd & boltdb registers
	// If you want other kv stores, call valkeyrie/store register functions.
	// zookeeper
	Register("zfs", &KVStoreFactoryZfs{})
	// generic kv store based valkeyrie library
	Register("kv", &KVStoreFactory{})
}

type silientLogger struct{}

func (silientLogger) Printf(format string, a ...interface{}) {}

// SetZkSilentLogger disable zk logger.
func SetZkSilentLogger() {
	zk.DefaultLogger = silientLogger{}
}

// KVStore is a kv storage implementation.
type KVStore struct {
	key   string
	store storeapi
}

func (k *KVStore) Write(bs []byte) error {
	glog.V(4).Infof("write [%s] %d bytes", k.key, len(bs))
	return k.store.Put(k.key, bs, nil)
}

func (k *KVStore) Read() ([]byte, error) {
	pair, err := k.store.Get(k.key, nil)
	if err != nil {
		return nil, err
	}
	return pair.Value, nil
}

// Exists check if the storage exists.
func (k *KVStore) Exists() (bool, error) {
	return k.store.Exists(k.key, nil)
}

// Create creates the storage.
func (k *KVStore) Create() error {
	return nil
}

// Remove remove the storage.
func (k *KVStore) Remove() error {
	glog.Infof("remove key [%s]", k.key)
	return k.store.Delete(k.key)
}

// Close close the storage.
func (k *KVStore) Close() error {
	// For a key-value, no close
	return nil
}

// KVStoreLocation implements kv storage location.
type KVStoreLocation struct {
	path  string
	store storeapi
	root  bool
}

// List lists the storage(file) in this location.
func (k *KVStoreLocation) List() ([]string, error) {
	entries, err := k.store.List(k.path, nil)
	if err == store.ErrKeyNotFound {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	list := make([]string, 0)
	for _, pair := range entries {
		list = append(list, k.stripPrefix(pair.Key))
	}
	return list, nil
}

func (k *KVStoreLocation) stripPrefix(p string) string {
	secs := strings.Split(p, "/")
	return secs[len(secs)-1]
}

// New new a sub storage object.
func (k *KVStoreLocation) New(sub string) (FStorage, error) {
	return &KVStore{key: filepath.Join(k.path, sub), store: k.store}, nil
}

// Create creates the location.
func (k *KVStoreLocation) Create(sub string) (FStorage, error) {
	return k.New(sub)
}

// NewSubLocation new sub location by this location.
// All sub locations share the same store (usually connection) with root location.
// It's ok to close or not close the sub locaiton
func (k *KVStoreLocation) NewSubLocation(sub ...string) (Location, error) {
	path := filepath.Join(k.path, filepath.Join(sub...))
	return &KVStoreLocation{path: path, store: k.store}, nil
}

// Close close this location
func (k *KVStoreLocation) Close() error {
	if k.root {
		k.store.Close()
	}
	return nil
}

// KVStoreFactory creates kv store location.
type KVStoreFactory struct{}

func getValue(qs url.Values, k string) (string, bool) {
	if vs, ok := qs[k]; ok && len(vs) >= 1 {
		return vs[0], true
	}
	return "", false
}

func getIntValue(qs url.Values, k string, d int) int {
	if s, ok := getValue(qs, k); ok {
		if i, err := strconv.Atoi(s); err == nil {
			return i
		}
	}
	return d
}

type options struct {
	backend        string
	connectTimeout int
	maxBufferSize  int // in Bytes. for zookeeper, the buffer size in client to encode request.
	bucket         string
	poolLBType     int // see StorePoolLBTypeRR
	poolSize       int
}

func parseZfsOptions(qs url.Values) *options {
	opt := options{
		backend:        string(store.ZK),
		connectTimeout: DefaultTimeout,
	}
	if backend, ok := getValue(qs, "backend"); ok {
		opt.backend = backend
	}
	opt.connectTimeout = getIntValue(qs, "connectTimeout", opt.connectTimeout)
	// In Kb in url query string, default 100Mb
	opt.maxBufferSize = 1024 * getIntValue(qs, "maxBufferSize", 100*1024)
	opt.poolLBType = getIntValue(qs, "poolLBType", StorePoolLBTypeHash)
	opt.poolSize = getIntValue(qs, "poolSize", 1)
	return &opt
}

func doCreateKVStoreLocation(u *url.URL, backend string) (Location, error) {
	uStr := u.String()
	kvURL, err := (&KVURLParser{}).parseURL(uStr)
	if err != nil {
		return nil, err
	}
	opt := kvURL.Option
	endpoint := kvURL.Endpoint
	if opt == nil {
		return nil, fmt.Errorf("nil options, url:%s", uStr)
	}
	if len(endpoint) == 0 {
		return nil, fmt.Errorf("empty endpoint, url:%s", uStr)
	}
	if opt.backend == "" {
		opt.backend = string(store.ZK)
	}
	store, err := createStorePool(opt, endpoint)
	if err != nil {
		return nil, err
	}
	path := strings.TrimLeft(u.Path, "/")
	return &KVStoreLocation{path: path, store: store, root: true}, nil
}

// Create create a location.
func (k *KVStoreFactory) Create(u *url.URL) (Location, error) {
	return doCreateKVStoreLocation(u, "")
}

func doCreateZKKVStoreLocation(u *url.URL) (Location, error) {
	m, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, err
	}
	opt := parseZfsOptions(m)
	opt.backend = string(store.ZK)
	store, err := createStorePool(opt, u.Host)
	if err != nil {
		return nil, err
	}
	path := strings.TrimLeft(u.Path, "/")
	return &KVStoreLocation{path: path, store: store, root: true}, nil
}

// KVStoreFactoryZfs is a wrapper to create storage location by zk url.
type KVStoreFactoryZfs struct{}

// Create creates a zk location.
func (k *KVStoreFactoryZfs) Create(u *url.URL) (Location, error) {
	return doCreateZKKVStoreLocation(u)
}

// KVURL ...
type KVURL struct {
	Schema   string
	Endpoint string
	Option   *options
}

// KVURLParser ...
// Schema: kv://endpoint?_options_=k:v[,k:v]
type KVURLParser struct {
}

// OptionKey ...
const OptionKey = "__options__"

// Prefix ...
const Prefix = "kv://"

func (p *KVURLParser) parseURL(rawURL string) (*KVURL, error) {
	if len(rawURL) == 0 {
		return nil, fmt.Errorf("invalid kv url : %s", rawURL)
	}
	idx := strings.Index(rawURL, Prefix)
	if idx != 0 {
		return nil, fmt.Errorf("no kv:// prefix in %s", rawURL)
	}
	u := rawURL[len(Prefix):]
	params := p.parseQueryStr(u)
	opt := &options{}
	if optV, ok := params[OptionKey]; ok {
		if err := p.parseOptValue(optV, opt); err != nil {
			return nil, err
		}
		delete(params, OptionKey)
	}
	if idx := strings.Index(u, "?"); idx != -1 {
		u = u[:idx]
	}
	var q string
	for k, v := range params {
		q = q + fmt.Sprintf("%s=%s&", k, v)
	}
	if len(q) > 0 {
		u = u + "?" + q
	}
	u = strings.TrimSuffix(u, "&")
	return &KVURL{
		Schema:   "kv",
		Endpoint: u,
		Option:   opt,
	}, nil
}

func (p *KVURLParser) parseQueryStr(url string) map[string]string {
	params := make(map[string]string)
	if len(url) == 0 {
		return params
	}
	idx := strings.Index(url, "?")
	if idx == -1 {
		return params
	}
	kvPairs := strings.Split(url[idx+1:], "&")
	for _, kvPair := range kvPairs {
		kv := strings.Split(kvPair, "=")
		if len(kv) != 2 {
			continue
		}
		params[kv[0]] = kv[1]
	}
	return params
}

// k1,v1[,k2:v2]
func (p *KVURLParser) parseOptValue(optV string, opt *options) error {
	if len(optV) == 0 {
		// do nothing while empty str
		return nil
	}
	kvPairs := strings.Split(optV, ",")
	params := make(map[string]string)
	for _, kvPair := range kvPairs {
		kv := strings.Split(kvPair, ":")
		if len(kv) != 2 {
			return fmt.Errorf("illegal optValue : %s", optV)
		}
		params[kv[0]] = kv[1]
	}
	if backend, ok := params["backend"]; ok {
		opt.backend = backend
	}
	if bucket, ok := params["bucket"]; ok {
		opt.bucket = bucket
	}
	if connTimeoutStr, ok := params["connectTimeout"]; ok {
		if i, err := strconv.Atoi(connTimeoutStr); err == nil {
			opt.connectTimeout = i
		}
	}
	if maxBufferSizeStr, ok := params["maxBufferSize"]; ok {
		if i, err := strconv.Atoi(maxBufferSizeStr); err == nil {
			opt.maxBufferSize = i
		}
	}
	return nil
}
