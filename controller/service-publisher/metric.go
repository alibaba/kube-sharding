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

package publisher

import (
	"strconv"
	"time"

	"github.com/alibaba/kube-sharding/common/metric"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	addNodesHistogram = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "publisher",
			Name:      "add_nodes_latency",
			Help:      "no help can be found here",
		},
		[]string{"type", "service_name", "success"},
	)

	delNodesHistogram = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "publisher",
			Name:      "del_nodes_latency",
			Help:      "no help can be found here",
		},
		[]string{"type", "service_name", "success"},
	)

	updateNodesHistogram = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "publisher",
			Name:      "update_nodes_latency",
			Help:      "no help can be found here",
		},
		[]string{"type", "service_name", "success"},
	)

	syncLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "publisher",
			Name:      "sync_latency",
			Help:      "no help can be found here",
		},
		[]string{"type", "service_name", "success"},
	)

	getResourceLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "publisher",
			Name:      "get_resource_latency",
			Help:      "no help can be found here",
		},
		[]string{"type", "service_name", "success"},
	)

	publishLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "publisher",
			Name:      "publish_latency",
			Help:      "no help can be found here",
		},
		[]string{"type", "service_name", "success"},
	)

	updateWorkerLatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "publisher",
			Name:      "update_worker_latency",
			Help:      "no help can be found here",
		},
		[]string{"type", "service_name", "success"},
	)

	abnormalService = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "publisher",
			Name:      "abnormal_service",
			Help:      "no help can be found here",
		},
		[]string{"service_name"},
	)
)

func init() {
	prometheus.MustRegister(addNodesHistogram)
	prometheus.MustRegister(delNodesHistogram)
	prometheus.MustRegister(updateNodesHistogram)
	prometheus.MustRegister(syncLatency)
	prometheus.MustRegister(getResourceLatency)
	prometheus.MustRegister(publishLatency)
	prometheus.MustRegister(updateWorkerLatency)
	prometheus.MustRegister(abnormalService)
}

func recordSummaryMetric(ptype, key string, gauge *prometheus.GaugeVec, startTime time.Time, err error) {
	gauge.With(prometheus.Labels{
		"type":         ptype,
		"service_name": metric.EscapeLabelKey(key),
		"success":      strconv.FormatBool(err == nil),
	}).Set(float64(time.Now().Sub(startTime).Nanoseconds() / 1000000))
}
