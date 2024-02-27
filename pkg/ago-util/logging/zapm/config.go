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
	"encoding/json"
	"errors"
	"io/ioutil"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Config configs for zap logger
type Config struct {
	Level         zap.AtomicLevel       `json:"level"`
	Sink          lumberjack.Logger     `json:"sink"`
	Encoding      string                `json:"encoding"`
	EncoderConfig zapcore.EncoderConfig `json:"encoderConfig"`
	// Whether to use diffCore
	DiffCore      bool `json:"diffCore"`
	DiffCacheSize int  `json:"diffCacheSize"`
	GlogMux       bool `json:"glogMux"`
}

// LoadConfigFile load config from file
func LoadConfigFile(f string) (*Config, error) {
	bs, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, err
	}
	return LoadConfig(bs)
}

// LoadConfig load config
func LoadConfig(bs []byte) (*Config, error) {
	config := &Config{
		// set default
		EncoderConfig: zap.NewProductionEncoderConfig(),
	}
	if err := json.Unmarshal(bs, config); err != nil {
		return nil, err
	}
	return config, nil
}

// Build constructs a logger from the Config and Options.
func (cfg *Config) Build(opts ...zap.Option) (*zap.Logger, error) {
	enc, err := cfg.buildEncoder()
	if err != nil {
		return nil, err
	}
	if cfg.Level == (zap.AtomicLevel{}) {
		return nil, errors.New("missing Level")
	}
	w := zapcore.AddSync(&cfg.Sink)
	var core zapcore.Core
	if cfg.DiffCore {
		var opts []Option
		if cfg.GlogMux {
			opts = append(opts, EnableGlogMux())
		}
		core = NewDiffCore(enc, w, cfg.Level, cfg.DiffCacheSize, opts...)
	} else {
		core = zapcore.NewCore(enc, w, cfg.Level)
	}
	log := zap.New(core)
	if len(opts) > 0 {
		log = log.WithOptions(opts...)
	}
	return log, nil
}

func (cfg *Config) buildEncoder() (zapcore.Encoder, error) {
	if cfg.DiffCore {
		// DiffCore requires TimeKey empty
		cfg.EncoderConfig.TimeKey = ""
	}
	if cfg.Encoding == "json" {
		return zapcore.NewJSONEncoder(cfg.EncoderConfig), nil
	}
	return nil, errors.New("invalid Encoding type")
}
