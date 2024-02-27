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

package rollingset

import (
	"github.com/alibaba/kube-sharding/common/metric"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// RollingAdjustLatency Rolling Adjust延迟
	rollingAdjustLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "rollingset",
			Name:       "rolling_adjust_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)
	//释放状态的replica数量
	releasingReplicaCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "rollingset",
			Name:      "releasing_replica_gauge",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)
	//replica的版本个数
	versionReplicaCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "rollingset",
			Name:      "version_count_gauge",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)
	standbyReplica = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "rollingset",
			Name:      "standby_replica",
			Help:      "standby replica",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "workerMode", "scalerpool", "assigned"},
	)
	standbyCPU = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "rollingset",
			Name:      "standby_cpu",
			Help:      "standby cpu, 1core=100",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "workerMode", "scalerpool", "assigned"},
	)
	standbyGPU = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "rollingset",
			Name:      "standby_gpu",
			Help:      "standby gpu, 1card=100",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "workerMode", "scalerpool", "assigned"},
	)
	standbyMem = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "rollingset",
			Name:      "standby_mem",
			Help:      "standby mem",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "workerMode", "scalerpool", "assigned"},
	)
	fixedCyclicalCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "rollingset_fixed_cyclical_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "type"},
	)
)

func init() {
	prometheus.MustRegister(rollingAdjustLatency)
	prometheus.MustRegister(releasingReplicaCounter)
	prometheus.MustRegister(versionReplicaCounter)
	prometheus.MustRegister(standbyReplica)
	prometheus.MustRegister(standbyCPU)
	prometheus.MustRegister(standbyGPU)
	prometheus.MustRegister(standbyMem)
	prometheus.MustRegister(fixedCyclicalCounter)
}
