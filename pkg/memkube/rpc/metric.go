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

package rpc

import (
	"github.com/alibaba/kube-sharding/pkg/ago-util/monitor"
)

// Metrics is a metric factory.
var Metrics = monitor.NewMetricFactory()

const (
	// MetricRPCReadTime read data in rpc time
	// labels: [cluster, namespace, resource]
	MetricRPCReadTime = "rpc_read_time"

	// MetricRPCUnmarshalTime unmarshal time in rpc
	// labels: [cluster, namespace, resource]
	MetricRPCUnmarshalTime = "rpc_unmarshal_time"

	// MetricListRemoteTime list remote data time in rpc
	// labels: [cluster, namespace, resource]
	MetricListRemoteTime = "rpc_list_remote_time"

	// MetricRPCWriteTime write data in rpc time
	// labels: [cluster, namespace, resource, method, success]
	MetricRPCWriteTime = "rpc_write_time"
)
