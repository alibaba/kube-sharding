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
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"syscall"
	"time"
)

const (
	timeFormat = "2006-01-02-15-04-05"
)

// WriteHistoryRecord records any object to a list of history files.
func WriteHistoryRecord(path string, key string, suffix string, keepCnt int, totalCnt int, obj interface{}) error {
	// reset umask first, otherwise file mask is not expected
	mask := syscall.Umask(0)
	defer syscall.Umask(mask)
	if err := os.MkdirAll(path, 0766); err != nil {
		return err
	}
	namePattern := key + "_%s_" + suffix
	fileTime := time.Now().Format(timeFormat)
	fileName := path + "/" + fmt.Sprintf(namePattern, fileTime)
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(fileName, data, 0666); err != nil {
		return err
	}
	if err = removeKeyOldFiles(keepCnt, fmt.Sprintf("%s/"+namePattern, path, "*")); err != nil {
		return fmt.Errorf("remove key old files error: %v", err)
	}
	if err = removeOldFiles(totalCnt, path); err != nil {
		return fmt.Errorf("remove old files error: %v", err)
	}
	return nil
}

func removeKeyOldFiles(n int, pattern string) error {
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	if len(files) <= n {
		return nil
	}
	sort.Strings(files)
	for _, name := range files[:len(files)-n] {
		if err := os.Remove(name); err != nil {
			return err
		}
	}
	return nil
}

func removeOldFiles(n int, path string) error {
	if n <= 0 {
		return nil
	}
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	if len(files) <= n {
		return nil
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime().Before(files[j].ModTime())
	})
	for _, file := range files[:len(files)-n] {
		if err := os.Remove(path + "/" + file.Name()); err != nil {
			return err
		}
	}
	return nil
}
