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

package httph

import (
	"regexp"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/monitor/metricgc"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	summaryMaxAge            = time.Second * 30
	summaryAgeBuckets uint32 = 3
)

var _reg = regexp.MustCompile(`[^a-zA-Z0-9_]`)

var (
	encodeLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "proxy",
			Subsystem:  "proxy",
			Name:       "encode_latency",
			Help:       "no help can be found here",
			MaxAge:     summaryMaxAge,
			AgeBuckets: summaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "path", "method"},
	)

	transLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "proxy",
			Subsystem:  "proxy",
			Name:       "trans_latency",
			Help:       "no help can be found here",
			MaxAge:     summaryMaxAge,
			AgeBuckets: summaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "path", "method"},
	)

	panicCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "proxy",
			Subsystem: "proxy",
			Name:      "panic_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "method", "path"},
	)

	queryErrCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "proxy",
			Subsystem: "proxy",
			Name:      "query_err",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "method", "path"},
	)
)

func init() {
	prometheus.MustRegister(panicCounter)
	prometheus.MustRegister(queryErrCounter)
	prometheus.MustRegister(encodeLatency)
	prometheus.MustRegister(transLatency)
}

func reportPanicCounter(cluster, path, method string) {
	path = _reg.ReplaceAllString(path, "_")
	metricgc.WithLabelValuesCounter(panicCounter,
		cluster,
		path,
		method,
	).Inc()
}

func reportEncodeLatency(cluster, path, method string, latency int64) {
	path = _reg.ReplaceAllString(path, "_")
	metricgc.WithLabelValuesSummary(encodeLatency,
		cluster,
		path,
		method,
	).Observe(float64(latency) / 1e6)
}

func reportTransLatency(cluster, path, method string, latency int64) {
	path = _reg.ReplaceAllString(path, "_")
	metricgc.WithLabelValuesSummary(transLatency,
		cluster,
		path,
		method,
	).Observe(float64(latency) / 1e6)
}

func reportResErrCounter(cluster, path, method string) {
	path = _reg.ReplaceAllString(path, "_")
	metricgc.WithLabelValuesCounter(queryErrCounter,
		cluster,
		method,
		path,
	).Inc()
}
