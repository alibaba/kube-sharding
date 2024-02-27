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
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// LoggerFactory manages a set of lumberjack.Logger
// See https://github.com/natefinch/lumberjack/blob/v2.0/lumberjack.go for config details.
type LoggerFactory struct {
	loggers map[string]io.WriteCloser
}

// LoadFile load logger configs
func (f *LoggerFactory) LoadFile(cfg string) error {
	bs, err := ioutil.ReadFile(cfg)
	if err != nil {
		return err
	}
	return f.Load(string(bs))
}

// Load loads from string
func (f *LoggerFactory) Load(cfg string) error {
	var loggers map[string]*lumberjack.Logger
	if err := json.Unmarshal([]byte(cfg), &loggers); err != nil {
		return err
	}
	for name, logger := range loggers {
		wc := f.create(logger)
		f.loggers[name] = wc
	}
	return nil
}

// Get gets a logger by the name
func (f *LoggerFactory) Get(name string) io.WriteCloser {
	return f.loggers[name]
}

// Must find the logger, panic if not found
func (f *LoggerFactory) Must(name string) io.WriteCloser {
	l, ok := f.loggers[name]
	if !ok {
		panic("not found logger:" + name)
	}
	return l
}

// GetOrNop gets a logger by the name, if not found, return a NOP WriteCloser
func (f *LoggerFactory) GetOrNop(name string) io.WriteCloser {
	l, ok := f.loggers[name]
	if !ok {
		return nopWriteCloser{}
	}
	return l
}

// CloseAll closes all loggers
func (f *LoggerFactory) CloseAll() {
	for _, logger := range f.loggers {
		logger.Close()
	}
	f.loggers = nil
}

func (f *LoggerFactory) add(name string, logger *lumberjack.Logger) {
	if _, ok := f.loggers[name]; ok {
		panic("duplicate logger name: " + name)
	}
	f.loggers[name] = f.create(logger)
}

func (f *LoggerFactory) create(logger *lumberjack.Logger) io.WriteCloser {
	if logger.Filename == "stdout" {
		return os.Stdout
	}
	if logger.Filename == "stderr" {
		return os.Stderr
	}
	return logger
}

// export apis
var (
	factory = &LoggerFactory{
		loggers: make(map[string]io.WriteCloser),
	}
	Load     = factory.Load
	LoadFile = factory.LoadFile
	Get      = factory.Get
	GetOrNop = factory.GetOrNop
	Must     = factory.Must
	CloseAll = factory.CloseAll
)

type nopWriteCloser struct {
}

func (wc nopWriteCloser) Write(p []byte) (n int, err error) {
	return
}

func (wc nopWriteCloser) Close() error {
	return nil
}
