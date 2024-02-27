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

package worker

import (
	"time"

	"github.com/alibaba/kube-sharding/common/metric"
	"github.com/alibaba/kube-sharding/pkg/ago-util/monitor/metricgc"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	createWorkerCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "replica",
			Name:      "create_first_worker_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset"},
	)

	createBackupCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "replica",
			Name:      "create_backup_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset"},
	)

	swapWorkerCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "replica",
			Name:      "swap_worker_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset"},
	)

	createRRCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "replica",
			Name:      "create_rr_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)

	releaseBadCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "replica",
			Name:      "release_bad_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset"},
	)

	failoverLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "controller",
			Name:       "failover_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)

	assignLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "controller",
			Name:       "assign_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)

	podReadyLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "controller",
			Name:       "pod_ready_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)

	workerReadyLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "controller",
			Name:       "worker_ready_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)

	dataReadyLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "controller",
			Name:       "data_ready_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)
)

func init() {
	prometheus.MustRegister(createWorkerCounter)
	prometheus.MustRegister(createBackupCounter)
	prometheus.MustRegister(swapWorkerCounter)
	prometheus.MustRegister(releaseBadCounter)
	prometheus.MustRegister(createRRCounter)
	prometheus.MustRegister(failoverLatency)
	prometheus.MustRegister(assignLatency)
	prometheus.MustRegister(podReadyLatency)
	prometheus.MustRegister(workerReadyLatency)
	prometheus.MustRegister(dataReadyLatency)
}

func recordCounterMetric(counter *prometheus.CounterVec, worker *carbonv1.WorkerNode) {
	counter.WithLabelValues(
		carbonv1.GetSubObjectBaseScopes(worker)...,
	).Inc()
}

func recordLatencyMetric(summary *prometheus.SummaryVec, worker *carbonv1.WorkerNode) {
	metricgc.WithLabelValuesSummary(summary,
		carbonv1.GetSubObjectBaseScopes(worker)...,
	).Observe(float64(time.Since(worker.CreationTimestamp.Time).Nanoseconds() / 1e6))
}

func recordWorkerLatencyMetric(summary *prometheus.SummaryVec, worker *carbonv1.WorkerNode, ms int64) {
	metricgc.WithLabelValuesSummary(summary,
		carbonv1.GetSubObjectBaseScopes(worker)...,
	).Observe(float64(ms))
}

func reportWorkerLatencys(worker *carbonv1.WorkerNode) {
	if nil == worker {
		return
	}
	// 还未分配或者分配异常的worker不汇报latency
	if worker.Status.AssignedTime == 0 || carbonv1.IsWorkerHaveBeenUnassignException(worker) {
		return
	}
	// 过滤不合理数据
	if time.Now().Unix()-worker.Status.AssignedTime < 3600*24 {
		carbonv1.RecordPodReadyTime(worker)
		carbonv1.RecordWorkerReadyTime(worker)
	}
	assignedLatencyTime := (worker.Status.AssignedTime - worker.CreationTimestamp.Time.Unix()) * 1000
	recordWorkerLatencyMetric(assignLatency, worker, assignedLatencyTime)
	if 0 != worker.Status.PodReadyTime {
		podReadyLatencyTime := (worker.Status.PodReadyTime - worker.Status.AssignedTime) * 1000
		recordWorkerLatencyMetric(podReadyLatency, worker, podReadyLatencyTime)
	}
	if 0 != worker.Status.WorkerReadyTime {
		workerReadyLatencyTime := (worker.Status.WorkerReadyTime - worker.Status.AssignedTime) * 1000
		recordWorkerLatencyMetric(workerReadyLatency, worker, workerReadyLatencyTime)
		dataReadyLatencyTime := (worker.Status.WorkerReadyTime - worker.Status.PodReadyTime) * 1000
		recordWorkerLatencyMetric(dataReadyLatency, worker, dataReadyLatencyTime)
	}
}
