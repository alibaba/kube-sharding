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
	"sync"

	"go.uber.org/zap"
)

var (
	// Logger is the default logger
	Logger *zap.Logger
	once   sync.Once
)

// InitDefault init default logger
func InitDefault(f string, opts ...zap.Option) error {
	cfg, err := LoadConfigFile(f)
	if err != nil {
		return err
	}
	once.Do(func() {
		Logger, err = cfg.Build(opts...)
	})
	return err
}
