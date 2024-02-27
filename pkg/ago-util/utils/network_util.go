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
	"errors"
	"net"
	"os"
)

// GetLocalIP get ip by lookup dns
func GetLocalIP() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", err
	}
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return "", err
	}
	for _, ip := range ips {
		if ip.IsLoopback() {
			continue
		}
		if ip.To4() != nil {
			return ip.String(), nil
		}
	}
	return "", errors.New("no ip found")
}
