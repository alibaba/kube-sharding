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
	"strings"

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	glog "k8s.io/klog"
)

const (
	pspWebHook = "kubernetes.io/psp"
)

var _ podStatusParser = &ackPodAllocator{}
var _ podSpecSyncer = &ackPodAllocator{}

type ackPodAllocator struct {
	basePodStatusParser
	basePodSpecSyncer
}

// -------- parser --------
func (a *ackPodAllocator) getCurrentResourceVersions(pod *corev1.Pod) (currentResourceVersion string, resourceMatched bool) {
	if pod == nil || len(pod.Annotations) == 0 {
		return "", false
	}
	resVer := pod.Annotations[carbonv1.AnnotationC2ResourceVersion]
	return resVer, resVer != ""
}

// -------- syncer --------
func (a *ackPodAllocator) computeProcessVersion(worker *carbonv1.WorkerNode, hippoPodSpec *carbonv1.HippoPodSpec, currentPod *corev1.Pod) (string, map[string]string, error) {
	return "", nil, nil
}

func (a *ackPodAllocator) computeResourceVersion(worker *carbonv1.WorkerNode, hippoPodSpec *carbonv1.HippoPodSpec, podSpecExtend *carbonv1.HippoPodSpecExtend) (string, error) {
	return worker.Spec.ResVersion, nil
}

func (a *ackPodAllocator) MergeWebHooks(targetPod *corev1.Pod, podSpec *corev1.PodSpec, currentPod *corev1.Pod) error {
	a.basePodSpecSyncer.MergeWebHooks(targetPod, podSpec, currentPod)
	if nil != currentPod && nil != currentPod.Annotations {
		if webhook, ok := currentPod.Annotations[pspWebHook]; ok {
			targetPod.Annotations[pspWebHook] = webhook
		}
	}
	return nil
}

func (a *ackPodAllocator) isPodReclaimed(pod *corev1.Pod) bool {
	if nil == pod {
		return false
	}
	return carbonv1.PodIsEvicted(pod.Status)
}

func (a *ackPodAllocator) preSync(worker *carbonv1.WorkerNode, hippoPodSpec *carbonv1.HippoPodSpec, labels map[string]string) error {
	hippoPodSpec.SchedulerName = carbonv1.SchedulerNameDefault
	if err := a.basePodSpecSyncer.preSync(worker, hippoPodSpec, labels); err != nil {
		glog.Errorf("preSync err:%v", err)
		return err
	}
	return a.preSyncResourceRequest(hippoPodSpec)
}

const (
	// AliyunEni Elastic Network Interface https://www.alibabacloud.com/help/zh/doc-detail/58496.htm
	AliyunEni = "aliyun/eni"
	// K8SStdResourceKeyDISK k8s standard disk resource key
	K8SStdResourceKeyDISK = "ephemeral-storage"
)

func (a *ackPodAllocator) preSyncResourceRequest(hippoPodSpec *carbonv1.HippoPodSpec) error {
	for i := range hippoPodSpec.Containers {
		var limits = corev1.ResourceList(map[corev1.ResourceName]resource.Quantity{})
		for k, v := range hippoPodSpec.Containers[i].Resources.Limits {
			switch k {
			case "cpu":
				limits[k] = v
			case "memory":
				limits[k] = v
			case carbonv1.ResourceENI:
				limits[AliyunEni] = v
			default:
				if strings.HasPrefix(string(k), carbonv1.ResourceDISK) {
					limits[K8SStdResourceKeyDISK] = v
				} else {
					limits[k] = v
				}
			}
		}
		hippoPodSpec.Containers[i].Resources.Limits = limits
		hippoPodSpec.Containers[i].Resources.Requests = limits
	}
	//delete carbonv1.ResourceIP "resource.hippo.io/ip" resource if hostNetworke=false
	if !hippoPodSpec.HostNetwork {
		for i := range hippoPodSpec.Containers {
			for k := range hippoPodSpec.Containers[i].Resources.Requests {
				if k == carbonv1.ResourceIP {
					delete(hippoPodSpec.Containers[i].Resources.Requests, k)
				}
			}
			for k := range hippoPodSpec.Containers[i].Resources.Limits {
				if k == carbonv1.ResourceIP {
					delete(hippoPodSpec.Containers[i].Resources.Limits, k)
				}
			}
		}
	}
	return nil
}
