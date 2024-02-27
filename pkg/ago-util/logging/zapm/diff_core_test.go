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
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type girl struct {
	Size int
	Name string
}

// a stringer object
func (g girl) String() string {
	return fmt.Sprintf("size: %d, name: %s", g.Size, g.Name)
}

type boy struct {
	Age int
}

// an ObjectMarshaler object
func (b boy) ObjectMarshaler(enc zapcore.ObjectEncoder) error {
	enc.AddInt("age", b.Age)
	return nil
}

func TestDiffCoreSample(t *testing.T) {
	assert := assert.New(t)
	sink, _, err := zap.Open("stdout")
	assert.Nil(err)
	conf := zap.NewProductionEncoderConfig()
	// don't append time, otherwise the message always diff
	conf.TimeKey = ""
	core := NewDiffCore(
		zapcore.NewJSONEncoder(conf),
		sink,
		zap.InfoLevel,
		100,
		EnableGlogMux(),
	)
	logger := zap.New(core, zap.AddCaller())
	defer logger.Sync()
	logger = logger.With(zap.String("namespace", "ns1"))
	// test diff logging
	for i := 0; i < 100; i++ {
		time.Sleep(time.Millisecond * 2)
		logger.Info("sync pod",
			zap.String(MsgUKey, "sync"),
			zap.String("id", "123"),
		)
	}
	logger.Info("update pod", zap.String(MsgUKey, "update"), zap.Int64("index", 10))
	// test fallback, default behavioud
	logger.Info("update pod", zap.Int64("index", 11))
	// test fmt.Stringer object
	logger.Info("create girl", zap.Any("girl", girl{Size: 80}))
	// test ObjectMarshaler object
	logger.Info("create boy", zap.Any("boy", boy{Age: 8}))
}

type fakeWriteSyncer struct {
	idx   int
	lines []string
}

func (ws *fakeWriteSyncer) Write(p []byte) (n int, err error) {
	ws.idx++
	if (ws.idx)%2 != 0 { // filter timestamp
		return
	}
	ws.lines = append(ws.lines, string(p))
	return len(p), nil
}

func (ws *fakeWriteSyncer) Sync() error {
	return nil
}

func TestDiffCoreOrder(t *testing.T) {
	assert := assert.New(t)

	ws := &fakeWriteSyncer{}
	conf := zap.NewProductionEncoderConfig()
	conf.TimeKey = ""
	core := NewDiffCore(
		zapcore.NewJSONEncoder(conf),
		ws,
		zap.InfoLevel,
		100,
	)
	logger := zap.New(core, zap.AddCaller())
	defer logger.Sync()
	kvs := map[string]string{
		"k1": "v1",
		"k2": "v2",
		"k3": "v3",
	}
	for i := 0; i < 10000; i++ {
		fs := []zapcore.Field{
			zap.String(MsgUKey, "testkey"),
		}
		for k, v := range kvs {
			fs = append(fs, zap.String(k, v))
		}
		// fs order diff, by golang map
		logger.Info("testmsg", fs...)
	}
	assert.Equal(1, len(ws.lines))
}
