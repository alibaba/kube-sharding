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

package transfer

import (
	"os"

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// GeneratePodSpec GeneratePodSpec
func GeneratePodSpec(hippoPodSpec *carbonv1.HippoPodSpec, version string, containersVersion map[string]string) (*corev1.PodSpec, *carbonv1.HippoPodSpecExtend, error) {
	var podSpec corev1.PodSpec
	var podSpecExtend carbonv1.HippoPodSpecExtend
	podSpec = hippoPodSpec.PodSpec
	podSpec.Containers = make([]corev1.Container, len(hippoPodSpec.Containers))
	podSpecExtend.Containers = make([]carbonv1.ContainerHippoExternedWithName, len(hippoPodSpec.Containers))
	podSpecExtend.InstanceID = version
	var cpu resource.Quantity
	var independentCPU = true //是不是独占完整cpu
	for i := range podSpec.Containers {
		podSpec.Containers[i] = hippoPodSpec.Containers[i].Container
		podSpecExtend.Containers[i].Name = hippoPodSpec.Containers[i].Name
		podSpecExtend.Containers[i].ContainerHippoExterned = hippoPodSpec.Containers[i].ContainerHippoExterned
		for j := range podSpec.Containers[i].Ports {
			if podSpec.Containers[i].Ports[j].Protocol == "" {
				podSpec.Containers[i].Ports[j].Protocol = corev1.ProtocolTCP
			}
		}
		if containerVersion, ok := containersVersion[hippoPodSpec.Containers[i].Name]; ok {
			podSpecExtend.Containers[i].InstanceID = containerVersion
		}
		if nil == podSpec.Containers[i].Resources.Requests {
			podSpec.Containers[i].Resources.Requests = podSpec.Containers[i].Resources.Limits
		}
		containerCPU, ok := podSpec.Containers[i].Resources.Limits["cpu"]
		if ok {
			cpu.Add(containerCPU)
		}
	}
	if cpu.MilliValue()/1000*1000 != cpu.MilliValue() {
		independentCPU = false
	}
	if hippoPodSpec.CpusetMode == "" {
		var priority int32
		if hippoPodSpec.Priority != nil {
			priority = *hippoPodSpec.Priority
		}
		if carbonv1.IsPodPriorityOnline(priority) {
			if independentCPU {
				hippoPodSpec.CpusetMode = string(carbonv1.CpusetReserved)
			} else {
				hippoPodSpec.CpusetMode = string(carbonv1.CpusetShare)
			}
		} else {
			hippoPodSpec.CpusetMode = string(carbonv1.CpusetNone)
		}
	}
	if hippoPodSpec.ContainerModel == "" {
		hippoPodSpec.ContainerModel = string(carbonv1.ContainerDocker)
	}
	podSpecExtend.HippoPodSpecExtendFields = hippoPodSpec.HippoPodSpecExtendFields
	podSpecExtend.Volumes = hippoPodSpec.HippoVolumes
	inMiniMode := os.Getenv("mini_mode")
	if "true" == inMiniMode {
		podSpecExtend.ContainerModel = "PROCESS"
	}

	return &podSpec, &podSpecExtend, nil
}
