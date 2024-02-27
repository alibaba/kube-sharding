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

package proxy

import (
	"regexp"
	"strconv"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/monitor"
	"github.com/alibaba/kube-sharding/pkg/ago-util/monitor/metricgc"
	"github.com/alibaba/kube-sharding/pkg/memkube/rpc"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	summaryMaxAge            = time.Second * 30
	summaryAgeBuckets uint32 = 3
)

var _reg = regexp.MustCompile(`[^a-zA-Z0-9_]`)

var (
	queryCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "proxy",
			Subsystem: "proxy",
			Name:      "query_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "path", "method", "code"},
	)

	// queryLatency 排队时间
	queryLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "proxy",
			Subsystem:  "proxy",
			Name:       "query_latency",
			Help:       "no help can be found here",
			MaxAge:     summaryMaxAge,
			AgeBuckets: summaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "path", "method", "code"},
	)

	rpcReadLatencySummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "memkube",
			Name:       rpc.MetricRPCReadTime,
			Help:       "no help can be found here",
			MaxAge:     summaryMaxAge,
			AgeBuckets: summaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "namespace", "resource"},
	)

	rpcUnmarshalLatencySummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "memkube",
			Name:       rpc.MetricRPCUnmarshalTime,
			Help:       "no help can be found here",
			MaxAge:     summaryMaxAge,
			AgeBuckets: summaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "namespace", "resource"},
	)

	rpcListRemoteLatencySummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "memkube",
			Name:       rpc.MetricListRemoteTime,
			Help:       "no help can be found here",
			MaxAge:     summaryMaxAge,
			AgeBuckets: summaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "namespace", "resource"},
	)
)

func init() {
	prometheus.MustRegister(queryCounter)
	prometheus.MustRegister(queryLatency)

	rpc.Metrics.Register(rpc.MetricRPCReadTime, monitor.CurryMetric{Func: func(lbls ...string) monitor.GenericMetric {
		return rpcReadLatencySummary.WithLabelValues(lbls...)
	}})
	prometheus.MustRegister(rpcReadLatencySummary)

	rpc.Metrics.Register(rpc.MetricRPCUnmarshalTime, monitor.CurryMetric{Func: func(lbls ...string) monitor.GenericMetric {
		return rpcUnmarshalLatencySummary.WithLabelValues(lbls...)
	}})
	prometheus.MustRegister(rpcUnmarshalLatencySummary)

	rpc.Metrics.Register(rpc.MetricListRemoteTime, monitor.CurryMetric{Func: func(lbls ...string) monitor.GenericMetric {
		return rpcListRemoteLatencySummary.WithLabelValues(lbls...)
	}})
	prometheus.MustRegister(rpcListRemoteLatencySummary)

}

func reportQueryCounter(cluster, path, method string, code int) {
	path = _reg.ReplaceAllString(path, "_")
	queryCounter.WithLabelValues(
		cluster,
		path,
		method,
		strconv.Itoa(code),
	).Inc()
}

func reportQueryLatency(cluster, path, method string, code int, latency int64) {
	path = _reg.ReplaceAllString(path, "_")
	metricgc.WithLabelValuesSummary(queryLatency,
		cluster,
		path,
		method,
		strconv.Itoa(code),
	).Observe(float64(latency) / 1e6)
}
