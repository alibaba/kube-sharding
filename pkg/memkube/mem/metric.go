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

package mem

import (
	"github.com/alibaba/kube-sharding/pkg/ago-util/monitor"
)

// Metrics is a metric factory.
var Metrics = monitor.NewMetricFactory()

const (
	// MetricPersistMarshalTime marshal time in persist
	// labels: [namespace, batch_name, kind]
	MetricPersistMarshalTime = "persist_marshal_time"
	// MetricPersistWriteTime write marshaled data in store time
	// labels: [namespace, batch_name, kind]
	MetricPersistWriteTime = "persist_write_time"
	// MetricPersistFileSize the final file size write to storage
	// labels: [namespace, batch_name, kind]
	MetricPersistFileSize = "persist_file_size"
	// MetricPersistCountInBatch the items count in 1 batch
	// labels: [namespace, batch_name, kind]
	MetricPersistCountInBatch = "persist_count_in_batch"
	// MetricStoreRunTime the cost time each store.run
	// labels: [name, namespace, kind]
	MetricStoreRunTime = "store_run_time"
	// MetricWatchSendTime send watch event time
	// labels: [kind, source]
	MetricWatchSendTime = "watch_send_time"
)
