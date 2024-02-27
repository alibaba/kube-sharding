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

package db

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/abronan/valkeyrie"
	"github.com/abronan/valkeyrie/store"
	db_util "github.com/alibaba/kube-sharding/pkg/ago-util/utils/db"
)

// DB relation-db storage
type DB struct {
	client    *db_util.DB
	tableName string
}

const (
	// KeyColName key column name
	KeyColName = "fs_key"
	// ValueColName value column name
	ValueColName = "fs_value"
)

// Register ...
func Register() {
	valkeyrie.AddStore("db", New)
}

var (
	// ErrDBMultipleEndpointsUnsupported ...
	ErrDBMultipleEndpointsUnsupported = errors.New("db supports one endpoint and should be a file path")
	// ErrDBNoEndpointGiven ...
	ErrDBNoEndpointGiven = errors.New("one endpoint at least should be given")
	// ErrDBBucketOptionMissing is thrown when boltBcuket config option is missing
	ErrDBBucketOptionMissing = errors.New("dbBucket config option missing")
)

var _ store.Store = &DB{}

// New ... endpoint format : [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
func New(endpoints []string, options *store.Config) (store.Store, error) {
	if len(endpoints) > 1 {
		return nil, ErrDBMultipleEndpointsUnsupported
	}
	if len(endpoints) == 0 {
		return nil, ErrDBNoEndpointGiven
	}
	if options == nil || len(options.Bucket) == 0 {
		return nil, ErrDBBucketOptionMissing
	}
	client := db_util.NewDB()
	// fmt dsn string
	e, err := supplyDefaultProtocol(endpoints[0])
	if err != nil {
		return nil, err
	}
	if err := client.OpenWithDSN(e); err != nil {
		return nil, err
	}
	return &DB{
		client:    client,
		tableName: options.Bucket,
	}, nil
}

// xyz@host:port/
func supplyDefaultProtocol(e string) (string, error) {
	idx0, idx1 := strings.Index(e, "@"), strings.Index(e, "/")
	if idx0 == -1 || idx1 == -1 {
		return e, fmt.Errorf("illegal endpoint %s", e)
	}
	host := e[idx0+1 : idx1]
	if strings.HasPrefix(host, "tcp(") && strings.HasSuffix(host, ")") {
		return e, nil
	}
	return strings.ReplaceAll(e, host, "tcp("+host+")"), nil

}

// Get ...
func (r *DB) Get(key string, opts *store.ReadOptions) (*store.KVPair, error) {
	sqlStr := r.newSelectWhereKVStatement()
	ret, err := r.client.QueryRow(r.newMapper(), sqlStr, key)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, store.ErrKeyNotFound
		}
		return nil, fmt.Errorf("get failed, err:%v, sql:%s", err, sqlStr)
	}
	if ret == nil {
		return nil, store.ErrKeyNotFound
	}
	kvPair := ret.(*store.KVPair)
	return kvPair, nil
}

// Put uk idx of `key` column required
func (r *DB) Put(key string, value []byte, opts *store.WriteOptions) error {
	sqlStr := r.newSelectWhereKVStatement()
	_, err := r.client.QueryRow(r.newMapper(), sqlStr, key)
	// insert if not exist
	if err != nil {
		if err == sql.ErrNoRows {
			iSQL := fmt.Sprintf("INSERT INTO %s(%s, %s) VALUES(?, ?)", r.tableName, KeyColName, ValueColName)
			err := r.client.Exec(iSQL, key, string(value))
			if err != nil {
				return fmt.Errorf("put failed in insert, err:%v, key:%s, sql:%s", err, key, iSQL)
			}
			return nil
		}
		return err
	}
	// update
	uSQL := fmt.Sprintf("UPDATE %s SET %s = ?  WHERE %s = ?", r.tableName, ValueColName, KeyColName)
	err = r.client.Exec(uSQL, string(value), key)
	if err != nil {
		return fmt.Errorf("put failed in update, err:%v, key:%s, sql:%s", err, key, uSQL)
	}
	return nil
}

// Delete ...
func (r *DB) Delete(key string) error {
	dSQL := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", r.tableName, KeyColName)
	return r.client.Exec(dSQL, key)
}

// Exists ...
func (r *DB) Exists(key string, opts *store.ReadOptions) (bool, error) {
	v, err := r.Get(key, opts)
	if err != nil {
		if err == store.ErrKeyNotFound {
			return false, nil
		}
		return false, err
	}
	return v != nil, nil
}

func (r *DB) newMapper() func(db_util.Row) (interface{}, error) {
	return func(row db_util.Row) (interface{}, error) {
		var k, v string
		err := row.Scan(&k, &v)
		if err != nil {
			return nil, err
		}
		return &store.KVPair{Key: k, Value: []byte(v)}, nil
	}
}

func (r *DB) newSelectWhereKVStatement() string {
	return fmt.Sprintf("SELECT %s, %s FROM %s WHERE %s = ?", KeyColName, ValueColName, r.tableName, KeyColName)
}

// List ...
func (r *DB) List(keyPrefix string, opts *store.ReadOptions) ([]*store.KVPair, error) {
	key := keyPrefix + "%"
	sqlStr := fmt.Sprintf("SELECT %s, %s FROM %s WHERE %s LIKE ?", KeyColName, ValueColName, r.tableName, KeyColName)
	ret, err := r.client.Query(r.newMapper(), sqlStr, key)
	if err != nil {
		return nil, fmt.Errorf("query failed, err:%v, keyPrefix:%s", err, keyPrefix)
	}
	retKV := make([]*store.KVPair, 0)
	for _, kv := range ret {
		retKV = append(retKV, (kv.(*store.KVPair)))
	}
	return retKV, nil
}

// DeleteTree ...
func (r *DB) DeleteTree(keyPrefix string) error {
	key := keyPrefix + "%"
	dSQL := fmt.Sprintf("DELETE FROM %s WHERE %s LIKE ?", r.tableName, KeyColName)
	return r.client.Exec(dSQL, key)
}

// Close ...
func (r *DB) Close() {
	r.client.Close()
}

// AtomicDelete ...
func (r *DB) AtomicDelete(key string, previous *store.KVPair) (bool, error) {
	return false, store.ErrCallNotSupported
}

// AtomicPut ...
func (r *DB) AtomicPut(key string, value []byte, previous *store.KVPair, options *store.WriteOptions) (bool, *store.KVPair, error) {
	return false, nil, store.ErrCallNotSupported
}

// NewLock ...
func (r *DB) NewLock(key string, options *store.LockOptions) (store.Locker, error) {
	return nil, store.ErrCallNotSupported
}

// Watch ...
func (r *DB) Watch(key string, stopCh <-chan struct{}, opts *store.ReadOptions) (<-chan *store.KVPair, error) {
	return nil, store.ErrCallNotSupported
}

// WatchTree ...
func (r *DB) WatchTree(directory string, stopCh <-chan struct{}, opts *store.ReadOptions) (<-chan []*store.KVPair, error) {
	return nil, store.ErrCallNotSupported
}
