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

package util

import (
	"strconv"
	"time"

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"

	"github.com/alibaba/kube-sharding/common/metric"
	"github.com/alibaba/kube-sharding/pkg/ago-util/monitor/metricgc"
	"github.com/prometheus/client_golang/prometheus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	glog "k8s.io/klog"
)

// CallType is a type for ProvidorType
type CallType string

const (
	// CallTypeCreateReplica is call type
	CallTypeCreateReplica CallType = "CreateReplica"
	// CallTypeUpdateReplica is call type
	CallTypeUpdateReplica CallType = "UpdateReplica"
	// CallTypeReleaseReplica is call type
	CallTypeReleaseReplica CallType = "ReleaseReplica"
	// CallTypeDeleteReplica is call type
	CallTypeDeleteReplica CallType = "DeleteReplica"

	// CallTypeCreateWorker is call type
	CallTypeCreateWorker CallType = "CreateWorker"
	// CallTypeUpdateWorker is call type
	CallTypeUpdateWorker CallType = "UpdateWorker"
	// CallTypeDeleteWorker is call type
	CallTypeDeleteWorker CallType = "DeleteWorker"

	// CallTypeCreatePod is call type
	CallTypeCreatePod CallType = "CreatePod"
	// CallTypeUpdatePod is call type
	CallTypeUpdatePod CallType = "UpdatePod"
	// CallTypePatchPod is call type
	CallTypePatchPod CallType = "PatchPod"
	// CallTypeDeletePod is call type
	CallTypeDeletePod CallType = "DeletePod"

	// CallTypeUpdateRollingset is call type
	CallTypeUpdateRollingset CallType = "UpdateRollingset"

	// CallTypeListReplica is call type
	CallTypeListReplica CallType = "ListReplica"

	// CallTypeListRollingSet is call type
	CallTypeListRollingSet CallType = "ListRollingSet"

	// CallTypeListWorker is call type
	CallTypeListWorker CallType = "ListWorker"
	// CallTypeCreateRollingSet is call type
	CallTypeCreateRollingSet = "CreateRollingSet"

	// CallTypeUpdateShardGroup is call type
	CallTypeUpdateShardGroup = "UpdateShardGroup"

	// CallTypeUpdateShardGroupStatus is call type
	CallTypeUpdateShardGroupStatus = "UpdateShardGroupStatus"

	// CallTypeDeleteShardGroup is call type
	CallTypeDeleteShardGroup = "DeleteShardGroup"

	// CallTypeCreateCarbonJob CreateCarbonJob
	CallTypeCreateCarbonJob = "CreateCarbonJob"

	// CallTypeUpdateCarbonJob UpdateCarbonJob
	CallTypeUpdateCarbonJob = "UpdateCarbonJob"

	// CallTypeUpdateCarbonJobStatus UpdateCarbonJobStatus
	CallTypeUpdateCarbonJobStatus = "UpdateCarbonJobStatus"

	// CallTypeDeleteCarbonJob DeleteCarbonJob
	CallTypeDeleteCarbonJob = "DeleteCarbonJob"

	// CallTypeCreateWorkerNodeEviction CreateWorkerNodeEviction
	CallTypeCreateWorkerNodeEviction = "CreateWorkerNodeEviction"

	// CallTypeUpdateWorkerNodeEviction UpdateWorkerNodeEviction
	CallTypeUpdateWorkerNodeEviction = "UpdateWorkerNodeEviction"

	// CallTypeUpdateWorkerNodeEvictionStatus UpdateWorkerNodeEvictionStatus
	CallTypeUpdateWorkerNodeEvictionStatus = "UpdateWorkerNodeEvictionStatus"

	// CallTypeDeleteWorkerNodeEviction DeleteWorkerNodeEviction
	CallTypeDeleteWorkerNodeEviction = "DeleteWorkerNodeEviction"
)

// RecordAPICall 记录调用api service的延迟和次数，为减少日志仅写操作调用接口
func RecordAPICall(callType CallType, start time.Time, object *metav1.ObjectMeta, requestID string, err error) {
	elapsed := time.Since(start)
	costMiss := float64(elapsed.Nanoseconds() / 1000000)
	switch callType {
	case CallTypeCreateReplica:
		metricgc.WithLabelValuesSummary(CreateReplicaLatency,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Observe(costMiss)
		metricgc.WithLabelValuesCounter(CreateReplicaCounter,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Inc()
		log(callType, costMiss, object.Name, requestID)
	case CallTypeUpdateReplica:
		metricgc.WithLabelValuesSummary(UpdateReplicaLatency,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Observe(costMiss)
		metricgc.WithLabelValuesCounter(UpdateReplicaCounter,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Inc()
		log(callType, costMiss, object.Name, requestID)
	case CallTypeDeleteReplica:
		metricgc.WithLabelValuesSummary(DeleteReplicaLatency,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Observe(costMiss)
		metricgc.WithLabelValuesCounter(DeleteReplicaCounter,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Inc()
		log(callType, costMiss, object.Name, requestID)
	case CallTypeCreateWorker:
		metricgc.WithLabelValuesCounter(CreateWorkerCounter,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Inc()
		metricgc.WithLabelValuesSummary(CreateWorkerLatency, carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Observe(costMiss)
		log(callType, costMiss, object.Name, requestID)

	case CallTypeUpdateWorker:
		metricgc.WithLabelValuesCounter(UpdateWorkerCounter,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Inc()
		metricgc.WithLabelValuesSummary(UpdateWorkerLatency,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Observe(costMiss)
		log(callType, costMiss, object.Name, requestID)
	case CallTypeDeleteWorker:
		metricgc.WithLabelValuesCounter(DeleteWorkerCounter,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Inc()
		metricgc.WithLabelValuesSummary(DeleteWorkerLatency,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Observe(costMiss)
		log(callType, costMiss, object.Name, requestID)

	case CallTypeCreatePod:
		metricgc.WithLabelValuesCounter(createPodCounter,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Inc()
		metricgc.WithLabelValuesSummary(createPodLatency,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Observe(costMiss)
		log(callType, costMiss, object.Name, requestID)
	case CallTypeUpdatePod:
		metricgc.WithLabelValuesCounter(updatePodCounter,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Inc()
		metricgc.WithLabelValuesSummary(updatePodLatency,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Observe(costMiss)
		log(callType, costMiss, object.Name, requestID)
	case CallTypeDeletePod:
		metricgc.WithLabelValuesCounter(deletePodCounter,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Inc()
		metricgc.WithLabelValuesSummary(deletePodLatency,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Observe(costMiss)
		log(callType, costMiss, object.Name, requestID)

	case CallTypeUpdateRollingset:
		metricgc.WithLabelValuesSummary(UpdateRollingsetLatency,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Observe(costMiss)
		metricgc.WithLabelValuesCounter(UpdateRollingsetCounter,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Inc()
		log(callType, costMiss, object.Name, requestID)
	case CallTypeListReplica:
		metricgc.WithLabelValuesSummary(ListReplicaLatency,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Observe(costMiss)
		metricgc.WithLabelValuesCounter(ListReplicaCounter,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Inc()
		log(callType, costMiss, object.Name, requestID)
	case CallTypeListWorker:
		metricgc.WithLabelValuesSummary(ListWorkerLatency,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Observe(costMiss)
		metricgc.WithLabelValuesCounter(ListWorkerCounter,
			carbonv1.GetSubObjectExtendScopes(object, strconv.FormatBool(err == nil))...,
		).Inc()
		log(callType, costMiss, object.Name, requestID)
	case CallTypeUpdateShardGroup:
		metricgc.WithLabelValuesSummary(UpdateGroupLatency,
			object.Labels[carbonv1.LabelKeyClusterName],
			object.Labels[carbonv1.LabelKeyAppName],
			object.Labels[carbonv1.DefaultRollingsetUniqueLabelKey],
			strconv.FormatBool(err == nil)).Observe(costMiss)
		metricgc.WithLabelValuesCounter(UpdateGroupCounter,
			object.Labels[carbonv1.LabelKeyClusterName],
			object.Labels[carbonv1.LabelKeyAppName],
			object.Labels[carbonv1.DefaultRollingsetUniqueLabelKey],
			strconv.FormatBool(err == nil)).Inc()
		log(callType, costMiss, object.Name, requestID)
	case CallTypeUpdateShardGroupStatus:
		metricgc.WithLabelValuesSummary(UpdateGroupLatency,
			object.Labels[carbonv1.LabelKeyClusterName],
			object.Labels[carbonv1.LabelKeyAppName],
			object.Labels[carbonv1.DefaultShardGroupUniqueLabelKey],
			strconv.FormatBool(err == nil)).Observe(costMiss)
		metricgc.WithLabelValuesCounter(UpdateGroupCounter,
			object.Labels[carbonv1.LabelKeyClusterName],
			object.Labels[carbonv1.LabelKeyAppName],
			object.Labels[carbonv1.DefaultShardGroupUniqueLabelKey],
			strconv.FormatBool(err == nil)).Inc()
		log(callType, costMiss, object.Name, requestID)
	default:
		log(callType, costMiss, object.Name, requestID)
	}
}

func log(callType CallType, costMiss float64, objectName, requestID string) {
	glog.Infof("call api service `%v` requestID: %s, objectName: %s, elapsed: %v ms", callType, requestID, objectName, costMiss)
}

var (
	// CreateWorkerLatency 调用创建worker api延迟
	CreateWorkerLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "kube_api_client",
			Name:       "create_worker_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)
	// UpdateWorkerLatency 调用更新worker api延迟
	UpdateWorkerLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "kube_api_client",
			Name:       "update_worker_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	// DeleteWorkerLatency 调用删除worker api延迟
	DeleteWorkerLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "kube_api_client",
			Name:       "delete_worker_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	// CreateWorkerCounter 调用创建worker api次数
	CreateWorkerCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "kube_api_client",
			Name:      "create_worker_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	// UpdateWorkerCounter 调用更新worker api次数
	UpdateWorkerCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "kube_api_client",
			Name:      "update_worker_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	// DeleteWorkerCounter 调用删除worker api次数
	DeleteWorkerCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "kube_api_client",
			Name:      "delete_worker_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	// CreateReplicaLatency 调用创建replica api延迟
	CreateReplicaLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "kube_api_client",
			Name:       "create_replica_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)
	// UpdateReplicaLatency 调用更新replica api延迟
	UpdateReplicaLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "kube_api_client",
			Name:       "update_replica_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	// DeleteReplicaLatency 调用删除replica api延迟
	DeleteReplicaLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "kube_api_client",
			Name:       "delete_replica_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	// CreateReplicaCounter 调用创建replica api次数
	CreateReplicaCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "kube_api_client",
			Name:      "create_replica_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)
	// UpdateReplicaCounter 调用更新replica api次数
	UpdateReplicaCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "kube_api_client",
			Name:      "update_replica_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	// DeleteReplicaCounter 调用删除replica api次数
	DeleteReplicaCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "kube_api_client",
			Name:      "delete_replica_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	// UpdateRollingsetLatency 更新rs延迟
	UpdateRollingsetLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "kube_api_client",
			Name:       "update_rollingset_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)
	// UpdateRollingsetCounter 调用更新Rollingset api次数
	UpdateRollingsetCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "kube_api_client",
			Name:      "update_rollingset_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	// ListWorkerLatency 调用ListWorker api延迟
	ListWorkerLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "kube_api_client",
			Name:       "list_worker_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	// ListWorkerCounter 调用ListWorker api次数
	ListWorkerCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "kube_api_client",
			Name:      "list_worker_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)
	// ListReplicaLatency 调用ListReplica api延迟
	ListReplicaLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "kube_api_client",
			Name:       "list_replica_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	// ListReplicaCounter 调用ListReplica api次数
	ListReplicaCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "kube_api_client",
			Name:      "list_replica_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	createPodCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "kube_api_client",
			Name:      "create_pod_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	createPodLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "kube_api_client",
			Name:       "create_pod_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	deletePodCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "kube_api_client",
			Name:      "delete_pod_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	deletePodLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "kube_api_client",
			Name:       "delete_pod_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	updatePodCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "kube_api_client",
			Name:      "update_pod_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	updatePodLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "kube_api_client",
			Name:       "update_pod_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "rollingset", "group", "role", "success"},
	)

	// UpdateGroupLatency 更新group延迟
	UpdateGroupLatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "carbon",
			Subsystem:  "kube_api_client",
			Name:       "update_group_latency",
			Help:       "no help can be found here",
			MaxAge:     metric.SummaryMaxAge,
			AgeBuckets: metric.SummaryAgeBuckets,
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		},
		[]string{"cluster", "application", "group", "success"},
	)
	// UpdateGroupCounter 调用更新group api次数
	UpdateGroupCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "carbon",
			Subsystem: "kube_api_client",
			Name:      "update_group_counter",
			Help:      "no help can be found here",
		},
		[]string{"cluster", "application", "group", "success"},
	)
)

func init() {
	prometheus.MustRegister(CreateWorkerLatency)
	prometheus.MustRegister(UpdateWorkerLatency)
	prometheus.MustRegister(DeleteWorkerLatency)
	prometheus.MustRegister(CreateWorkerCounter)
	prometheus.MustRegister(UpdateWorkerCounter)
	prometheus.MustRegister(DeleteWorkerCounter)

	prometheus.MustRegister(createPodLatency)
	prometheus.MustRegister(updatePodLatency)
	prometheus.MustRegister(deletePodLatency)
	prometheus.MustRegister(createPodCounter)
	prometheus.MustRegister(updatePodCounter)
	prometheus.MustRegister(deletePodCounter)

	prometheus.MustRegister(CreateReplicaLatency)
	prometheus.MustRegister(UpdateReplicaLatency)
	prometheus.MustRegister(DeleteReplicaLatency)
	prometheus.MustRegister(CreateReplicaCounter)
	prometheus.MustRegister(UpdateReplicaCounter)
	prometheus.MustRegister(DeleteReplicaCounter)

	prometheus.MustRegister(UpdateRollingsetLatency)
	prometheus.MustRegister(UpdateRollingsetCounter)

	prometheus.MustRegister(ListWorkerLatency)
	prometheus.MustRegister(ListWorkerCounter)

	prometheus.MustRegister(ListReplicaLatency)
	prometheus.MustRegister(ListReplicaCounter)

	prometheus.MustRegister(UpdateGroupCounter)
	prometheus.MustRegister(UpdateGroupLatency)
}
