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

package zk

import (
	"net/url"
	"strings"
	"time"

	"github.com/go-zookeeper/zk"
	glog "k8s.io/klog"
)

//InitZKConnector ...
func InitZKConnector(hosts []string, cb func(event zk.Event)) (*zk.Conn, error) {
	glog.Infof("connect zk :%v,%v", hosts, time.Second*60)
	conn, _, err := zk.Connect(hosts, time.Second*60)
	if err != nil {
		glog.Errorf("init zk connect error :%v,%v", hosts, err)
		return nil, err
	}

	return conn, err
}

//ParseZKPath ...
func ParseZKPath(path string) ([]string, string, error) {
	var host, subPath string
	if strings.HasPrefix(path, "zfs://") {
		path = path[6:]
	}
	index := strings.Index(path, "/")
	if -1 == index {
		index = len(path)
	}

	host = path[:index]
	subPath = path[index:]
	return strings.Split(host, ","), subPath, nil
}

//GenZKUrl ...
func GenZKUrl(hosts []string, path string) string {
	zkURL := url.URL{}
	zkURL.Host = strings.Join(hosts, ",")
	zkURL.Scheme = "zfs"
	zkURL.Path = path
	return zkURL.String()
}
