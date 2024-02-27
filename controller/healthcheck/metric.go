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

package healthcheck

import (
	"github.com/alibaba/kube-sharding/common/metric"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// HealthCheckTaskCount task数
	HealthCheckTaskCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "healthcheck",
			Name:      "healthcheck_task_counter",
			Help:      "no help can be found here",
		},
		[]string{},
	)

	// HealthCheckTaskLatency task一轮耗时
	HealthCheckTaskLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "healthcheck",
			Name:       "healthcheck_task_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset"},
	)

	//Lv7HealthCheckHTTPLatency lv7 http 心跳延迟
	Lv7HealthCheckHTTPLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "healthcheck",
			Name:       "lv7_healthcheck_http_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	//Lv7HealthCheckHTTPPV lv7 http pv
	Lv7HealthCheckHTTPPV = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "healthcheck",
			Name:      "lv7_healthcheck_http_pv",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success", "code"},
	)

	// HealthCheckCounter 健康检查次数
	HealthCheckCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "healthcheck",
			Name:      "healthcheck_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)

	//SendTargetE2ELatency 全联路调度延迟
	SendTargetE2ELatency = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "healthcheck",
			Name:      "send_target_e2e_latency",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "group", "status"},
	)
)

func init() {
	prometheus.MustRegister(HealthCheckTaskCount)
	prometheus.MustRegister(HealthCheckTaskLatency)
	prometheus.MustRegister(HealthCheckCounter)
	prometheus.MustRegister(Lv7HealthCheckHTTPLatency)
	prometheus.MustRegister(Lv7HealthCheckHTTPPV)
	prometheus.MustRegister(SendTargetE2ELatency)
}
