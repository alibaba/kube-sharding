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

package common

import (
	"os"
	"strconv"
	"time"

	glog "k8s.io/klog"
)

const (
	batcherTimeoutKey string = "batcherTimeoutSecond"
	staticAssetsKey   string = "staticAssets"
)

var (
	batcherTimeoutValue time.Duration = 0
	staticAssets        string        = "/home/admin/c2/assets/static"
)

// GetBatcherTimeout get batcher time out from env / args
func GetBatcherTimeout(defaultValue time.Duration) time.Duration {
	if batcherTimeoutValue != 0 {
		return batcherTimeoutValue
	}
	return defaultValue
}

func getIntEnvArg(key string) (int, bool, error) {
	strV := os.Getenv(key)
	if strV != "" {
		intV, err := strconv.Atoi(strV)
		return intV, true, err
	}
	return 0, false, nil
}

func initBatcherTimeout() {
	timeout, exist, err := getIntEnvArg(batcherTimeoutKey)
	if nil != err {
		glog.Errorf("init batcher timeout failed, err: %s", err)
		return
	}
	if exist {
		batcherTimeoutValue = time.Duration(timeout) * time.Second
	}
}

func getStrEnvArg(key, defaultVal string) string {
	strV := os.Getenv(key)
	if strV != "" {
		return strV
	}
	return defaultVal
}

func initStaticAssets() {
	staticAssets = getStrEnvArg(staticAssetsKey, staticAssets)
}

func init() {
	initBatcherTimeout()
	initStaticAssets()
}
