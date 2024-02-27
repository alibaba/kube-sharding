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

package carbonjob

import (
	"github.com/alibaba/kube-sharding/common/metric"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// SyncWorkerSpecFromCarbonJobLatency SyncWorkerSpecFromCarbonJobLatency
	SyncWorkerSpecFromCarbonJobLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "carbonjob",
			Name:       "sync_worker_spec_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{"namespace", "name", "success"},
	)
)

func init() {
	prometheus.MustRegister(SyncWorkerSpecFromCarbonJobLatency)
}
