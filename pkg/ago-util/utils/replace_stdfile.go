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
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

var _rfile *os.File

// 10 MB, in kb
var maxSize = int64(10 * 1024)

// SetMaxStdFileSize SetMaxStdFileSize
func SetMaxStdFileSize(size int64) { maxSize = size }

func replaceStdfile(name string, std *os.File) error {
	file, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	if err = syscall.Dup2(int(file.Fd()), int(std.Fd())); err != nil {
		return err
	}
	// hold ref by global variable to avoid GC
	_rfile = file
	return nil
}

func appendHeader(f *os.File) {
	program := filepath.Base(os.Args[0])
	t := time.Now()
	s := fmt.Sprintf("\n%s [%s] replace stderr by PID %d.\n", t.Format(time.RFC3339), program, os.Getpid())
	f.WriteString(s)
}

func disableReplace() bool {
	return os.Getenv("DISABLE_REPLACE_STDFILE") == "true"
}

func exceedMaxFileSize(name string, maxSize int64) bool {
	fileInfo, err := os.Stat(name)
	if err != nil {
		return false
	}
	size := fileInfo.Size()
	return size/1024 >= maxSize
}

func cleanExceedMaxSizeFile(name string, maxSize int64) bool {
	if !exceedMaxFileSize(name, maxSize) {
		return false
	}
	if err := _rfile.Truncate(0); err != nil {
		return false
	}
	_, err := _rfile.Seek(0, 0)
	return err == nil
}

func stdFileDaemon(name string, maxSize int64) {
	go func() {
		for {
			select {
			case <-time.After(time.Second * 30):
				cleanExceedMaxSizeFile(name, maxSize)
			}
		}
	}()
}

// ReplaceStderr replaces os.Stderr by a custom file. Usually to write panics
func ReplaceStderr(name string, writeHeader bool) error {
	if disableReplace() {
		return nil
	}
	if err := replaceStdfile(name, os.Stderr); err != nil {
		return err
	}
	if writeHeader {
		appendHeader(os.Stderr)
	}
	stdFileDaemon(name, maxSize)
	return nil
}
