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

package rotate

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/natefinch/lumberjack.v2"
)

func TestLoggerFactory(t *testing.T) {
	assert := assert.New(t)
	defer CloseAll()
	assert.Nil(Load(`
	{
		"default": {
			"filename": "/tmp/default.log",
			"maxSize": 10,
			"maxbackups": 2
		},
		"stdout": {
			"filename": "stdout"
		},
		"stderr": {
			"filename": "stderr"
		}
	}
	`))
	wc := Get("default")
	logger := wc.(*lumberjack.Logger)
	assert.NotNil(logger)
	assert.Equal(2, logger.MaxBackups)
	assert.Equal("/tmp/default.log", logger.Filename)
	assert.Nil(Get("nonexist"))
	wc = Get("stdout")
	assert.Equal(os.Stdout, wc)
	wc = Get("stderr")
	assert.Equal(os.Stderr, wc)

	l := GetOrNop("noexist")
	_, ok := l.(nopWriteCloser)
	assert.True(ok)
}
