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
	"encoding/json"
	"fmt"

	lru "github.com/hashicorp/golang-lru"
)

// LoggerWriter is a function to write logs, e.g: glog.Infof
type LoggerWriter func(msg string)

// DiffLogger write logs/record files if object content is different.
type DiffLogger struct {
	kvs    *lru.Cache
	writer LoggerWriter

	keyKeepCount   int
	totalKeepCount int
}

// NewDiffLogger create a new diff logger.
func NewDiffLogger(size int, writer LoggerWriter, keyKeepCount, totalKeepCount int) (*DiffLogger, error) {
	cache, err := lru.New(size)
	if err != nil {
		return nil, err
	}
	return &DiffLogger{kvs: cache, writer: writer, keyKeepCount: keyKeepCount, totalKeepCount: totalKeepCount}, nil
}

// Log writes log if log message different.
func (l *DiffLogger) Log(key string, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	l.checkDo(key, msg, nil, func() error {
		l.writer(key + ": " + msg)
		return nil
	})
}

// WriteHistoryFile write a history record file for object.
func (l *DiffLogger) WriteHistoryFile(path string, key string, suffix string, obj interface{}, cmpFn func(prev, obj interface{}) (bool, error)) error {
	return l.checkDo(key, obj, cmpFn, func() error {
		return WriteHistoryRecord(path, key, suffix, l.keyKeepCount, l.totalKeepCount, obj)
	})
}

func (l *DiffLogger) checkDo(key string, obj interface{}, cmpFn func(prev, obj interface{}) (bool, error), fn func() error) error {
	prev, ok := l.kvs.Get(key)
	if ok {
		if cmpFn == nil {
			cmpFn = l.cmpValue
		}
		diff, err := cmpFn(prev, obj)
		if err != nil {
			return err
		}
		if !diff {
			return nil
		}
	}
	l.kvs.Add(key, obj)
	return fn()
}

func (l *DiffLogger) cmpValue(v1 interface{}, v2 interface{}) (bool, error) {
	b1, err := json.Marshal(v1)
	if err != nil {
		return false, err
	}
	b2, err := json.Marshal(v2)
	if err != nil {
		return false, err
	}
	return string(b1) != string(b2), nil
}
