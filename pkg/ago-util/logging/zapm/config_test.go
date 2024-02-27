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
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestConfigBuild(t *testing.T) {
	assert := assert.New(t)
	bs := []byte(`
	{
		"level": "info",
		"sink": {
			"filename": "zap.log",
			"maxSize": 10
		},
		"encoding": "json",
		"diffCore": true,
		"diffCacheSize": 100,
		"glogMux": true
	}
	`)
	config, err := LoadConfig(bs)
	assert.Nil(err)
	logger, err := config.Build()
	assert.Nil(err)
	logger.Info("hello zap", zap.String(MsgUKey, "test"), zap.String("name", "alice"))
}
