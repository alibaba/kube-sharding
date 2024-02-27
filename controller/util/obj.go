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
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	clientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/uuid"
	glog "k8s.io/klog"
	"k8s.io/kubernetes/pkg/controller"

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
)

var errNilLister = errors.New("lister couldn't be nil")

// GetWorkerServicesSync get the services ref the worker
func GetWorkerServicesSync(carbonclientset clientset.Interface, worker *carbonv1.WorkerNode) ([]*carbonv1.ServicePublisher, error) {
	allServices, err := carbonclientset.CarbonV1().ServicePublishers(worker.Namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var services []*carbonv1.ServicePublisher
	for i := range allServices.Items {
		service := allServices.Items[i]
		if service.DeletionTimestamp != nil {
			continue
		}
		if service.Spec.Selector == nil {
			// services with nil selectors match nothing, not everything.
			continue
		}
		selector := labels.Set(service.Spec.Selector).AsSelectorPreValidated()
		if selector.Matches(labels.Set(worker.Labels)) {
			services = append(services, &service)
		}
	}

	return services, nil
}

// GetWorkerServices get the services ref the worker
func GetWorkerServices(s listers.ServicePublisherLister, worker *carbonv1.WorkerNode) ([]*carbonv1.ServicePublisher, error) {
	allServices, err := s.ServicePublishers(worker.Namespace).List(labels.Everything())
	if err != nil {
		return nil, err
	}

	var services []*carbonv1.ServicePublisher
	for i := range allServices {
		service := allServices[i]
		if service.DeletionTimestamp != nil {
			continue
		}
		if service.Spec.Selector == nil {
			// services with nil selectors match nothing, not everything.
			continue
		}
		selector := labels.Set(service.Spec.Selector).AsSelectorPreValidated()
		if selector.Matches(labels.Set(worker.Labels)) {
			services = append(services, service)
		}
	}

	return services, nil
}

// GetWorkerServiceMemberships get the name of services which ref the worker
func GetWorkerServiceMemberships(s listers.ServicePublisherLister, worker *carbonv1.WorkerNode) (sets.String, error) {
	set := sets.String{}
	services, err := GetWorkerServices(s, worker)
	if err != nil {
		// don't log this error because this function makes pointless
		// errors when no services match.
		return set, nil
	}
	for i := range services {
		key, err := controller.KeyFunc(services[i])
		if err != nil {
			return nil, err
		}
		set.Insert(key)
	}
	return set, nil
}

// IsObjDelete means is the obj has been delete
func IsObjDelete(obj interface{}) bool {
	if object, ok := obj.(metav1.Object); ok {
		return object.GetDeletionTimestamp() != nil
	}
	return false
}

// GenerateObjName generate name for obj
func GenerateObjName(prefix string, version string, superVersion string) string {
	version = utils.AppendString(version, superVersion)
	rand.Seed(time.Now().Unix())
	suffix, err := utils.SignatureShort(version, string(uuid.NewUUID()), rand.Float64())
	if nil != err {
		glog.Error(err)
		suffix = string(uuid.NewUUID())[0:8]
	}
	return utils.AppendString(prefix, "-", suffix)
}

// MergeLabels merge labels
func MergeLabels(labels, labels2 map[string]string) map[string]string {
	// Clone.
	newLabels := map[string]string{}
	if labels != nil {
		for key, value := range labels {
			newLabels[key] = value
		}
	}
	if labels2 != nil {
		for key, value := range labels2 {
			newLabels[key] = value
		}
	}
	return newLabels
}

// UpdateCrdTime 更新crd对象变更的时间点，  在调用apiserver之前 //concurrent map writes 问题无法解决
func UpdateCrdTime(meta *metav1.ObjectMeta, now time.Time) {
	if meta.Annotations == nil {
		meta.Annotations = make(map[string]string)
	}
	timeString := now.Format(time.RFC3339Nano)
	meta.Annotations["updateCrdTime"] = timeString
}

// CalcCrdChangeLatency 计算从变更crd到controller收到变更的延迟 //concurrent map writes 问题无法解决
func CalcCrdChangeLatency(lastUpdateTime time.Time) float64 {
	if lastUpdateTime.IsZero() {
		return 0
	}

	elapsed := time.Since(lastUpdateTime)
	costMiss := float64(elapsed.Nanoseconds() / 1e6)
	if 0 == costMiss {
		costMiss = 1
	}
	// 过滤掉明显不合理的，可能是重启造成
	if costMiss > 1000*60*60 {
		costMiss = 0
	}
	return costMiss
}

func updateVersionKey(version string) string {
	if "" == version {
		version = "default"
	}
	return "updateVersionTime" + "-" + version
}

// UpdateVersionTime 更新crd对象变更的时间点，  在调用apiserver之前
func UpdateVersionTime(meta *metav1.ObjectMeta, version string, now time.Time) {
	if meta.Annotations == nil {
		meta.Annotations = make(map[string]string)
	}
	timeString := now.Format(time.RFC3339Nano)
	key := updateVersionKey(version)
	_, ok := meta.Annotations[key]
	if !ok {
		meta.Annotations[key] = timeString
	}
	for k := range meta.Annotations {
		if strings.HasPrefix(k, "updateVersionTime") && k != key {
			delete(meta.Annotations, k)
		}
	}
}

// CalcCompleteLatency 计算从变更crd到controller收到变更的延迟
func CalcCompleteLatency(annotations map[string]string, version string) float64 {
	if annotations == nil {
		return 0
	}
	updateCrdTimestampString := annotations[updateVersionKey(version)]
	if updateCrdTimestampString == "" {
		return 0
	}
	updateCrdTime, err := time.Parse(time.RFC3339Nano, updateCrdTimestampString)
	if err != nil {
		return 0
	}
	elapsed := time.Since(updateCrdTime)
	costMiss := float64(elapsed.Nanoseconds() / 1e6)

	return costMiss
}

//ReplicaKey 生成replica key
func ReplicaKey(replica *carbonv1.Replica) string {
	return fmt.Sprintf("%v/%v", replica.Namespace, replica.Name)
}

//RollingSetKey 生成rollingset key
func RollingSetKey(r *carbonv1.RollingSet) string {
	return fmt.Sprintf("%v/%v", r.Namespace, r.Name)
}

func getReplicaKeys(replicas []*carbonv1.Replica) []string {
	replicaKeys := make([]string, 0, len(replicas))
	for _, replica := range replicas {
		replicaKeys = append(replicaKeys, ReplicaKey(replica))
	}
	return replicaKeys
}

// NewWorker NewWorker
func NewWorker(rsName, gangPartName, gangID string, rs *carbonv1.RollingSet, owners []metav1.OwnerReference, label, anno map[string]string) *carbonv1.WorkerNode {
	var newReplicaName string
	if gangID != "" {
		newReplicaName = gangID
	} else {
		newReplicaName = GenerateObjName(rsName, rs.ResourceVersion, rs.Spec.Version)
	}
	var gangName string
	if gangPartName != "" {
		gangName = newReplicaName
		newReplicaName = newReplicaName + "." + gangPartName
	}
	workerName := newReplicaName + "-a"
	// Add podTemplateHash label to selector.
	labels := carbonv1.CreateWorkerLabels(label, newReplicaName, workerName)
	newRSSelector := carbonv1.CloneSelector(rs.Spec.Selector)
	carbonv1.CreateWorkerSelector(newRSSelector, newReplicaName, workerName)
	worker := &carbonv1.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			// Make the name deterministic, to ensure idempotence
			Name:            workerName,
			Namespace:       rs.Namespace,
			OwnerReferences: owners,
			Labels:          labels,
			Annotations:     carbonv1.CloneMap(anno),
		},
		Spec: carbonv1.WorkerNodeSpec{
			ResVersion: rs.Spec.ResVersion,
			Selector:   newRSSelector,
			//version, versionplan在外面统一设置
			VersionPlan: carbonv1.VersionPlan{
				BroadcastPlan: carbonv1.BroadcastPlan{
					WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
						BrokenRecoverQuotaConfig: rs.Spec.BrokenRecoverQuotaConfig,
					},
				},
			},
		},
	}
	if gangPartName != "" {
		carbonv1.SetGangPartName(worker, gangPartName)
		carbonv1.AddGangUniqueLabelHashKey(worker.Labels, gangName)
	}
	carbonv1.FixNewCreateLazyV3PodVersion(worker)
	return worker
}
