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

package controller

import (
	"github.com/alibaba/kube-sharding/common/metric"
	"github.com/alibaba/kube-sharding/pkg/ago-util/monitor"
	"github.com/alibaba/kube-sharding/pkg/ago-util/monitor/metricgc"
	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	persistWriteLatencySummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "memkube",
			Name:       mem.MetricPersistWriteTime,
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"namespace", "batch_name", "kind"},
	)

	persistMarshalLatencySummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "memkube",
			Name:       mem.MetricPersistMarshalTime,
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"namespace", "batch_name", "kind"},
	)

	persistFileSizeSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "memkube",
			Name:       mem.MetricPersistFileSize,
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"namespace", "batch_name", "kind"},
	)

	persistCountInBatchSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "memkube",
			Name:       mem.MetricPersistCountInBatch,
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"namespace", "batch_name", "kind"},
	)

	storeRunLatencySummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "memkube",
			Name:       mem.MetricStoreRunTime,
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"name", "namespace", "kind"},
	)

	watchSendLatencySummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "memkube",
			Name:       mem.MetricWatchSendTime,
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.9: 0.01, 0.99: 0.001},
		},
		[]string{"kind", "source"},
	)
)

func init() {
	mem.Metrics.Register(mem.MetricPersistWriteTime, monitor.CurryMetric{Func: func(lbls ...string) monitor.GenericMetric {
		return metricgc.WithLabelValuesSummary(persistWriteLatencySummary, lbls...)
	}})
	prometheus.MustRegister(persistWriteLatencySummary)

	mem.Metrics.Register(mem.MetricPersistMarshalTime, monitor.CurryMetric{Func: func(lbls ...string) monitor.GenericMetric {
		return metricgc.WithLabelValuesSummary(persistMarshalLatencySummary, lbls...)
	}})
	prometheus.MustRegister(persistMarshalLatencySummary)

	mem.Metrics.Register(mem.MetricPersistFileSize, monitor.CurryMetric{Func: func(lbls ...string) monitor.GenericMetric {
		return metricgc.WithLabelValuesSummary(persistFileSizeSummary, lbls...)
	}})
	prometheus.MustRegister(persistFileSizeSummary)

	mem.Metrics.Register(mem.MetricPersistCountInBatch, monitor.CurryMetric{Func: func(lbls ...string) monitor.GenericMetric {
		return metricgc.WithLabelValuesSummary(persistCountInBatchSummary, lbls...)
	}})
	prometheus.MustRegister(persistCountInBatchSummary)

	mem.Metrics.Register(mem.MetricStoreRunTime, monitor.CurryMetric{Func: func(lbls ...string) monitor.GenericMetric {
		return metricgc.WithLabelValuesSummary(storeRunLatencySummary, lbls...)
	}})
	prometheus.MustRegister(storeRunLatencySummary)

	mem.Metrics.Register(mem.MetricWatchSendTime, monitor.CurryMetric{Func: func(lbls ...string) monitor.GenericMetric {
		return metricgc.WithLabelValuesSummary(watchSendLatencySummary, lbls...)
	}})
	prometheus.MustRegister(watchSendLatencySummary)
}
