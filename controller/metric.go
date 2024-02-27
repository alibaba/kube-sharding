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
	"time"

	"github.com/alibaba/kube-sharding/common/metric"
	"github.com/alibaba/kube-sharding/pkg/ago-util/monitor/metricgc"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	glog "k8s.io/klog"
)

var (
	// controllerSyncGauge controllersync一轮延迟
	controllerSyncSummary = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "controller",
			Name:       "controller_sync_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "controller", "success"},
	)

	// controllerQueueLatency 排队时间
	controllerQueueLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "controller",
			Name:       "controller_queue_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "controller"},
	)

	// controllerSyncCounter controller sync数量
	controllerSyncCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "controller_sync_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "controller", "success"},
	)

	// addqueue addqueue数量
	addQueueCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "add_queue_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "controller"},
	)

	// gcCounter
	gcCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "gc_counter",
			Help:      "no help can be found here",
		},
		[]string{"namespace", "name"},
	)

	handleSubObjCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "handle_sub_obj_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "controller"},
	)

	// queueLengthCounter 队列长度
	queueLengthGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "queue_length_gauge",
			Help:      "no help can be found here",
		},
		[]string{"controller"},
	)

	processCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "process_counter",
			Help:      "no help can be found here",
		},
		[]string{"controller"},
	)

	// rollingset个数指标
	rollingsetCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "rollingset_counter",
			Help:      "no help can be found here",
		},
		[]string{},
	)
	replicaCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "replica_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)
	activeReplicaCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "active_replica_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)
	standbyReplicaCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "standby_replica_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)
	workeModeMisMatchReplicaCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "workermode_mismatch_replica_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)
	unAssignedStandbyReplicaCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "unassigned_standby_replica_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)
	readyReplicaCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "ready_replica_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)
	assignedReplicaCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "assigned_replica_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)
	availableReplicaCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "available_replica_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)
	workerCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "worker_counter",
			Help:      "no help can be found here",
		},
		[]string{},
	)
	rollingSetWorkerCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "rollingset_worker_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)
	rollingSetSpotCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "rollingset_spot_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)
	rollingsetPodCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "rollingset_pod_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)
	podCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "pod_counter",
			Help:      "no help can be found here",
		},
		[]string{},
	)
	standbyGpuCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "standby_gpu",
			Help:      "standby gpu detail metric",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "host", "assigned", "gpu_model", "workMode", "scalerpool"},
	)
	// rollingset下释放中的replica个数
	rollingsetRepleasingReplica = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "rollingset_releasing_replica",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)
	// rollingset下释放中的worker个数
	rollingsetRepleasingWroker = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "rollingset_releasing_worker",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)
	//未完成状态的replica数量
	uncompleteReplicaCounter = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "rollingset",
			Name:      "uncomplete_replica_gauge",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)
	// rollingset下可用的replica个数比例
	rollingsetAvailableReplicaRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "rollingset_available_replica_rate",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)

	// rollingset下最新版本的replica个数比例
	rollingsetReadyReplicaRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "rollingset_ready_replica_rate",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)

	// rollingset下最新版本的replica个数比例
	rollingsetLatestReplicaRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "rollingset_latest_replica_rate",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)

	rollingsetUnssignedStandbyReplicaRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "unassigned_standby_replica_rate",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)

	rollingsetUnassignedWorker = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "rollingset_unassigned_worker",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "reason"},
	)
	// group指标

	// group个数指标
	groupCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "group_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "shardgroup", "group", "role", "status"},
	)

	//UpdateCrdLatency  crd UPDATE 到controller处理链路的延迟
	UpdateCrdLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "controller",
			Name:       "update_crd_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "kind", "controller"},
	)

	//CompleteLatency  从创建/更新到complete的时间
	CompleteLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "controller",
			Name:       "complete_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "kind", "controller"},
	)
	belowMinHealthGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "carbon",
			Subsystem: "controller",
			Name:      "rollingset_below_min_health",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role"},
	)
)

func init() {
	prometheus.MustRegister(controllerSyncSummary)
	prometheus.MustRegister(controllerQueueLatency)
	prometheus.MustRegister(controllerSyncCounter)
	prometheus.MustRegister(addQueueCounter)
	prometheus.MustRegister(gcCounter)
	prometheus.MustRegister(handleSubObjCounter)
	prometheus.MustRegister(queueLengthGauge)
	prometheus.MustRegister(rollingsetCounter)
	prometheus.MustRegister(replicaCounter)
	prometheus.MustRegister(activeReplicaCounter)
	prometheus.MustRegister(readyReplicaCounter)
	prometheus.MustRegister(assignedReplicaCounter)
	prometheus.MustRegister(availableReplicaCounter)
	prometheus.MustRegister(standbyReplicaCounter)
	prometheus.MustRegister(unAssignedStandbyReplicaCounter)
	prometheus.MustRegister(workeModeMisMatchReplicaCounter)
	prometheus.MustRegister(workerCounter)
	prometheus.MustRegister(rollingSetWorkerCounter)
	prometheus.MustRegister(podCounter)
	prometheus.MustRegister(standbyGpuCounter)
	prometheus.MustRegister(rollingsetPodCounter)
	prometheus.MustRegister(rollingsetUnassignedWorker)

	prometheus.MustRegister(rollingsetRepleasingReplica)
	prometheus.MustRegister(rollingsetRepleasingWroker)

	prometheus.MustRegister(rollingsetAvailableReplicaRate)
	prometheus.MustRegister(rollingsetReadyReplicaRate)
	prometheus.MustRegister(rollingsetLatestReplicaRate)
	prometheus.MustRegister(rollingsetUnssignedStandbyReplicaRate)
	prometheus.MustRegister(uncompleteReplicaCounter)

	prometheus.MustRegister(groupCounter)

	prometheus.MustRegister(UpdateCrdLatency)
	prometheus.MustRegister(CompleteLatency)
	prometheus.MustRegister(rollingSetSpotCounter)
	prometheus.MustRegister(belowMinHealthGauge)
	prometheus.MustRegister(processCounter)
}

func reportSyncMetric(labels map[string]string, key, agentName string, start time.Time, err error) {
	namespace, name, errSplit := cache.SplitMetaNamespaceKey(key)
	if nil != errSplit {
		return
	}
	var meta = &metav1.ObjectMeta{
		Namespace: namespace,
		Name:      name,
		Labels:    labels,
	}
	var success = "false"
	if nil == err {
		success = "true"
	} else if errors.IsNotFound(err) || errors.IsConflict(err) {
		success = "conflict"
	}
	if nil != labels {
		metricgc.WithLabelValuesSummary(controllerSyncSummary,
			carbonv1.GetSubObjectExtendScopes(meta, agentName, success)...,
		).Observe(float64(time.Since(start).Nanoseconds() / 1e6))
		controllerSyncCounter.WithLabelValues(
			carbonv1.GetSubObjectExtendScopes(meta, agentName, success)...,
		).Inc()
	}
}

func reportQueueTime(labels map[string]string, key, agentName string) {
	namespace, name, errSplit := cache.SplitMetaNamespaceKey(key)
	if nil != errSplit {
		return
	}
	if nil != labels {
		var meta = &metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    labels,
		}
		queueTime, err := time.Parse(time.RFC3339Nano, labels["queueTime"])
		if nil != err {
			glog.Errorf("phase time error %s,%v", labels["queueTime"], err)
		} else {
			metricgc.WithLabelValuesSummary(controllerQueueLatency,
				carbonv1.GetSubObjectExtendScopes(meta, agentName)...,
			).Observe(float64(time.Since(queueTime).Nanoseconds() / 1e6))
			if glog.V(4) {
				glog.Infof("queue latency :%s,%d", key, time.Since(queueTime).Nanoseconds()/1e6)
			}
		}
	}
}
