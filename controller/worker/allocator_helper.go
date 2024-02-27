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
	"fmt"
	"strconv"
	"strings"

	"github.com/alibaba/kube-sharding/common/features"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type basePodStatusParser struct{}

func (p *basePodStatusParser) getPodIP(pod *corev1.Pod) (hostIP, podIP string) {
	if nil != pod {
		hostIP = pod.Status.HostIP
		podIP = pod.Status.PodIP
	}
	return
}

func (p *basePodStatusParser) isPodReclaimed(pod *corev1.Pod) bool {
	return carbonv1.PodIsEvicted(pod.Status)
}

func (p *basePodStatusParser) getSlotID(pod *corev1.Pod) carbonv1.HippoSlotID {
	return carbonv1.HippoSlotID{}
}

func (p *basePodStatusParser) isPodProcessFailed(pod *corev1.Pod) bool {
	return false
}

func (p *basePodStatusParser) getPackageStatus(pod *corev1.Pod) string {
	return ""
}

// DEPRECATED
func (p *basePodStatusParser) getProcessScore(pod *corev1.Pod) int32 {
	return 0
}

func (p *basePodStatusParser) getCurrentResourceVersions(pod *corev1.Pod) (currentResourceVersion string, resourceMatched bool) {
	return "", true
}

func (p *basePodStatusParser) getCurrentProcessVersions(pod *corev1.Pod) (currentProcessVersion string, currentContainersProcessVersion map[string]string) {
	return "", nil
}

type basePodSpecSyncer struct{}

func (p *basePodSpecSyncer) computeResourceVersion(worker *carbonv1.WorkerNode, hippoPodSpec *carbonv1.HippoPodSpec, podSpecExtend *carbonv1.HippoPodSpecExtend) (string, error) {
	return "", nil
}

func (p *basePodSpecSyncer) computeProcessVersion(worker *carbonv1.WorkerNode, hippoPodSpec *carbonv1.HippoPodSpec, currentPod *corev1.Pod) (string, map[string]string, error) {
	return carbonv1.ComputeInstanceID(carbonv1.NeedRestartAfterResourceChange(&worker.Spec.VersionPlan), hippoPodSpec)
}

func (p *basePodSpecSyncer) updateResource(t *target, currentPod *corev1.Pod) (updateAll bool, err error) {
	return
}

func (p *basePodSpecSyncer) updateProcess(t *target) error {
	// 复制pod级别目标字段
	err := utils.JSONDeepCopy(t.podSpec, &t.targetPod.Spec)
	t.targetPod.Spec.Volumes = t.podSpec.Volumes
	t.targetPod.Spec.HostNetwork = t.podSpec.HostNetwork
	t.targetPod.Spec.Tolerations = t.podSpec.Tolerations
	t.targetPod.Spec.Affinity = t.podSpec.Affinity
	// 复制container级别目标字段
	t.targetPod.Spec.InitContainers = t.podSpec.InitContainers
	t.targetPod.Spec.Containers = t.podSpec.Containers
	return err
}

func (p *basePodSpecSyncer) preSync(worker *carbonv1.WorkerNode, hippoPodSpec *carbonv1.HippoPodSpec, labels map[string]string) error {
	var needIP bool
	for i := range hippoPodSpec.Containers {
		for j := range hippoPodSpec.Containers[i].Ports {
			if "" == hippoPodSpec.Containers[i].Ports[j].Protocol {
				hippoPodSpec.Containers[i].Ports[j].Protocol = corev1.ProtocolTCP
			}
		}
		for k := range hippoPodSpec.Containers[i].Resources.Limits {
			if k == carbonv1.ResourceIP {
				needIP = true
			}
		}
	}
	if hippoPodSpec.HostNetwork == false && labels[carbonv1.LabelKeyPodVersion] == carbonv1.PodVersion3 {
		needIP = true
	}
	if !needIP {
		hippoPodSpec.HostNetwork = true
	} else {
		hippoPodSpec.HostNetwork = false
	}
	// "app.hippo.io/exclusive-labels": "port-tcp-9091.port-tcp-9092",
	exclusive := carbonv1.GetExclusiveLableAnno(worker.Spec.Template)
	if exclusive == "" {
		exclusive = carbonv1.GetExclusiveLableAnno(worker)
	}
	// 增加端口互斥
	if exclusive != "" && !needIP {
		items := strings.Split(exclusive, ".")
		var ports []corev1.ContainerPort
		var affinityItems = []string{}
		for i := range items {
			item := items[i]
			port, isport := p.parsePort(item)
			if !isport {
				item = carbonv1.LabelValueHash(item, true)
				affinityItems = append(affinityItems, item)
				if labels != nil {
					labels[carbonv1.LabelKeyExlusive] = item
				}
			} else {
				ports = append(ports, corev1.ContainerPort{
					Name:          fmt.Sprintf("%s-%d", "port-tcp", port),
					HostPort:      int32(port),
					ContainerPort: int32(port),
				})
			}
		}

		if 0 != len(affinityItems) {
			if nil == hippoPodSpec.Affinity {
				hippoPodSpec.Affinity = &corev1.Affinity{}
			}
			if nil == hippoPodSpec.Affinity.PodAntiAffinity {
				hippoPodSpec.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
			}
			if nil == hippoPodSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
				hippoPodSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = []corev1.PodAffinityTerm{}
			}
			var podAffinityTerm = corev1.PodAffinityTerm{
				LabelSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      carbonv1.LabelKeyExlusive,
							Operator: metav1.LabelSelectorOpIn,
							Values:   affinityItems,
						},
					},
				},
				TopologyKey: "kubernetes.io/hostname",
			}
			hippoPodSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(
				hippoPodSpec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution, podAffinityTerm)
		}
		if 0 != len(ports) && 0 != len(hippoPodSpec.Containers) {
			hippoPodSpec.Containers[0].Ports = append(hippoPodSpec.Containers[0].Ports, ports...)
		}
	}
	if features.C2MutableFeatureGate.Enabled(features.UsePriorityClassName) && "" == hippoPodSpec.PriorityClassName && nil != hippoPodSpec.Priority {
		hippoPodSpec.PriorityClassName = fmt.Sprintf("%s-%d", HippoPriorityClassPrefix, *hippoPodSpec.Priority)
	}

	return nil
}

func (p *basePodSpecSyncer) parsePort(exclusive string) (port int, isport bool) {
	port, err := strconv.Atoi(exclusive)
	if nil == err {
		return port, true
	}
	slices := strings.Split(exclusive, "-")
	if 3 == len(slices) {
		var err error
		port, err = strconv.Atoi(slices[2])
		if nil != err {
			return
		}
		isport = true
		return
	}
	return
}

func (p *basePodSpecSyncer) postSync(t *target) error {
	return nil
}

func (p *basePodSpecSyncer) setProhibit(pod *corev1.Pod, prohibit string) {
	return
}

func (p *basePodSpecSyncer) MergeWebHooks(targetPod *corev1.Pod, podSpec *corev1.PodSpec, currentPod *corev1.Pod) error {
	return nil
}
