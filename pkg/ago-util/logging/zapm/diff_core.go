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

package zapm

import (
	"sort"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"go.uber.org/zap/zapcore"
)

// A DiffCore is a zapcore.Core object, which outputs log messages only if the result text is different.
// See diff_core_test.go for example.

const (
	// MsgUKey defines this message uniq key to make diff
	MsgUKey = "ukey"
)

// LogMux receives log messages
type LogMux interface {
	Write(lvl zapcore.Level, b []byte)
}

// Option options to set diffCore
type Option func(*diffCore)

// NewDiffCore creates a Core only output logs when different.
func NewDiffCore(enc zapcore.Encoder, ws zapcore.WriteSyncer, enab zapcore.LevelEnabler, cacheSize int, opts ...Option) zapcore.Core {
	cache, err := lru.New(cacheSize)
	if err != nil {
		panic("create lru cache failed:" + err.Error())
	}
	core := &diffCore{
		LevelEnabler: enab,
		enc:          enc,
		out:          ws,
		cache:        cache,
	}
	for _, opt := range opts {
		opt(core)
	}
	return core
}

// EnableGlogMux enable glog mux
func EnableGlogMux() Option {
	return func(c *diffCore) {
		c.mux = &glogMux{}
	}
}

type diffCore struct {
	zapcore.LevelEnabler
	enc   zapcore.Encoder
	out   zapcore.WriteSyncer
	cache *lru.Cache
	mux   LogMux
}

func (c *diffCore) With(fields []zapcore.Field) zapcore.Core {
	clone := c.clone()
	for i := range fields {
		fields[i].AddTo(clone.enc)
	}
	return clone
}

func (c *diffCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *diffCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	sort.SliceStable(fields, func(i, j int) bool {
		return fields[i].Key < fields[j].Key
	})
	buf, err := c.enc.EncodeEntry(ent, fields)
	if err != nil {
		return err
	}
	defer buf.Free()
	foundUkey := false
	for i := range fields {
		f := &fields[i]
		if f.Key == MsgUKey && f.Type == zapcore.StringType {
			if !c.isDiff(buf.Bytes(), f.String) {
				return nil
			}
			foundUkey = true
			break
		}
	}

	if foundUkey { // otherwise fallback default behaviour
		// append time
		var bs []byte
		bs = ent.Time.AppendFormat(bs, time.RFC3339Nano)
		bs = append(bs, 32) // space
		_, err = c.out.Write(bs)
		if err != nil {
			return err
		}
	}

	if c.mux != nil {
		c.mux.Write(ent.Level, buf.Bytes())
	}
	// the message body
	_, err = c.out.Write(buf.Bytes())
	if err != nil {
		return err
	}
	if ent.Level > zapcore.ErrorLevel {
		c.Sync()
	}
	return nil
}

func (c *diffCore) Sync() error {
	return c.out.Sync()
}

func (c *diffCore) clone() *diffCore {
	return &diffCore{
		LevelEnabler: c.LevelEnabler,
		enc:          c.enc.Clone(),
		out:          c.out,
		cache:        c.cache,
		mux:          c.mux,
	}
}

func (c *diffCore) isDiff(bs []byte, key string) bool {
	last, ok := c.cache.Get(key)
	str := string(bs)
	defer c.cache.Add(key, str)
	if ok {
		return last.(string) != str
	}
	return true
}
