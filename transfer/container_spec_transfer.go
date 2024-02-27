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
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec"
	corev1 "k8s.io/api/core/v1"

	"github.com/alibaba/kube-sharding/common"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
)

func transContainers(
	role *typespec.RolePlan,
	ress *corev1.ResourceRequirements,
	image string,
	containerEnvs []corev1.EnvVar,
	volumeMounts []corev1.VolumeMount,
	labels map[string]string,
	containerConfig *carbonv1.ContainerConfig,
	workingDirs map[string]string,
	securityContext *corev1.SecurityContext,
	devices *[]carbonv1.Device,
) (
	[]carbonv1.HippoContainer,
	[]corev1.Volume,
	corev1.RestartPolicy,
	bool,
	error,
) {
	if role.Version.LaunchPlan.PodDesc != "" {
		return transPodContainer(role.Version.LaunchPlan.PodDesc, ress, volumeMounts, containerEnvs, containerConfig)
	}
	return transT4Containers(role, ress, image, containerEnvs, volumeMounts, labels, containerConfig, workingDirs, securityContext, devices)
}

type hostConfig struct {
	Ulimits []carbonv1.Ulimit   `json:"Ulimits,omitempty"`
	CapAdd  []corev1.Capability `json:"CapAdd,omitempty"`
	CapDrop []corev1.Capability `json:"CapDrop,omitempty"`
	IpcMode string              `json:"IpcMode,omitempty"`
}

type podVolumeDesc map[string]struct{}

// See http://gitlab.alibaba-inc.com/hippo/hippo_pe_conf/blob/HIPPO_RU66_7U/sys_conf_ru66/sys_pod.json
// Some known fields are not supported: StopSignal
type podDesc struct {
	StopGracePeriod   int        `json:"StopGracePeriod"`
	InstanceID        string     `json:"InstanceId"`
	InstanceName      string     `json:"InstanceName"`
	Image             string     `json:"Image"`
	User              string     `json:"User"`
	Cmd               []string   `json:"Cmd,omitempty"`
	HostConfig        hostConfig `json:"HostConfig"`
	RestartCountLimit int        `json:"RestartCountLimit"`
	Entrypoint        []string   `json:"Entrypoint,omitempty"`
	Env               []string   `json:"Env,omitempty"`
	IsDaemon          bool       `json:"IsDaemon"`

	Volumes podVolumeDesc     `json:"volumes"` // map to empty dir ([path]{})
	Labels  map[string]string `json:"Labels,omitempty"`

	PodExtendedContainers []podDesc `json:"PodExtendedContainers"`
}

func transPodVolumes(desc podVolumeDesc) ([]corev1.Volume, []corev1.VolumeMount) {
	paths := make([]string, 0)
	for path := range desc {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return doTransPodVolumes(paths)
}

func doTransPodVolumes(paths []string) ([]corev1.Volume, []corev1.VolumeMount) {
	volumes := make([]corev1.Volume, 0, len(paths))
	mounts := make([]corev1.VolumeMount, 0, len(paths))
	for i, volumesInfo := range paths {
		path := volumesInfo
		var medium corev1.StorageMedium
		var size resource.Quantity
		volumesInfos := strings.Split(volumesInfo, ":")
		if len(volumesInfos) == 3 {
			path = volumesInfos[0]
			medium = corev1.StorageMedium(volumesInfos[1])
			size, _ = resource.ParseQuantity(volumesInfos[2])
		}
		name := fmt.Sprintf("volume-m-%d", i) // NOTE: must be different from BIND_MOUNTS names
		volume := corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}
		if string(medium) != "" {
			volume.VolumeSource.EmptyDir.Medium = medium
			volume.VolumeSource.EmptyDir.SizeLimit = &size
		}
		volumes = append(volumes, volume)
		mounts = append(mounts, corev1.VolumeMount{Name: name, MountPath: path})
	}
	return volumes, mounts
}

func transPodDesc2Container(podDesc *podDesc, volumeMounts []corev1.VolumeMount,
	containerEnvs []corev1.EnvVar, containerConfig *carbonv1.ContainerConfig) (carbonv1.HippoContainer, []corev1.Volume, corev1.RestartPolicy, error) {
	restartPolicy := corev1.RestartPolicyAlways
	cmd := podDesc.Entrypoint
	if cmd == nil || len(cmd) == 0 {
		cmd = podDesc.Cmd
	}
	parseEnvs := func(kvs []string) []corev1.EnvVar {
		envs := make([]corev1.EnvVar, 0, len(kvs))
		for _, kv := range kvs {
			vec := strings.Split(kv, "=")
			if len(vec) >= 2 {
				envs = append(envs, corev1.EnvVar{Name: vec[0], Value: strings.Join(vec[1:], "=")})
			}
		}
		return envs
	}
	volumes, podMounts := transPodVolumes(podDesc.Volumes)
	var container = corev1.Container{
		Name:         EscapeName(podDesc.InstanceName),
		Image:        podDesc.Image,
		Command:      cmd,
		Env:          append(parseEnvs(podDesc.Env), containerEnvs...),
		VolumeMounts: append(volumeMounts, podMounts...),
	}
	if podDesc.User == "root:root" || podDesc.User == "root" {
		if container.SecurityContext == nil {
			container.SecurityContext = &corev1.SecurityContext{}
		}
		container.SecurityContext.RunAsUser = utils.Int64Ptr(0)
		container.SecurityContext.RunAsGroup = utils.Int64Ptr(0)
	}
	if len(podDesc.HostConfig.CapAdd) > 0 || len(podDesc.HostConfig.CapDrop) > 0 {
		if container.SecurityContext == nil {
			container.SecurityContext = &corev1.SecurityContext{}
		}
		container.SecurityContext.Capabilities = &corev1.Capabilities{
			Add:  podDesc.HostConfig.CapAdd,
			Drop: podDesc.HostConfig.CapDrop,
		}
	}
	if nil == containerConfig {
		containerConfig = &carbonv1.ContainerConfig{}
	}
	containerConfig.Ulimits = podDesc.HostConfig.Ulimits
	containerConfig.RestartCountLimit = podDesc.RestartCountLimit
	containerConfig.StopGracePeriod = podDesc.StopGracePeriod
	var extendCon = carbonv1.ContainerHippoExterned{
		Labels:  podDesc.Labels,
		Configs: containerConfig,
		Alias:   podDesc.InstanceName, // NOTE: it's used as working directory
	}
	if !podDesc.IsDaemon {
		restartPolicy = corev1.RestartPolicyOnFailure
	}
	correctContainerProc(&container)
	var hippoContariner = carbonv1.HippoContainer{
		Container:              container,
		ContainerHippoExterned: extendCon,
	}
	return hippoContariner, volumes, restartPolicy, nil
}

func transPodContainer(podDescStr string, ress *corev1.ResourceRequirements,
	volumeMounts []corev1.VolumeMount, containerEnvs []corev1.EnvVar,
	containerConfig *carbonv1.ContainerConfig) (
	[]carbonv1.HippoContainer, []corev1.Volume, corev1.RestartPolicy, bool, error) {
	var podDesc podDesc
	var hostIPC = false
	if err := json.Unmarshal([]byte(podDescStr), &podDesc); err != nil {
		return nil, nil, corev1.RestartPolicyOnFailure, hostIPC, fmt.Errorf("unmarshal podDesc error: %v", err)
	}
	var cons []carbonv1.HippoContainer
	con0, podVolumes, restartPolicy, err := transPodDesc2Container(&podDesc, volumeMounts, containerEnvs, containerConfig)
	if err != nil {
		return nil, nil, restartPolicy, hostIPC, err
	}
	if ress != nil {
		con0.Resources = *ress.DeepCopy()
	}
	hostIPC = podDesc.HostConfig.IpcMode == "host"
	cons = append(cons, con0)
	for _, subDesc := range podDesc.PodExtendedContainers {
		// only use main container's Volumes
		con, _, _, err := transPodDesc2Container(&subDesc, volumeMounts, containerEnvs, containerConfig)
		if err != nil {
			return nil, nil, restartPolicy, hostIPC, fmt.Errorf("trans sub podDesc error: %v", err)
		}
		cons = append(cons, con)
	}
	return cons, podVolumes, restartPolicy, hostIPC, nil
}

func transT4Containers(
	role *typespec.RolePlan,
	ress *corev1.ResourceRequirements,
	image string,
	containerEnvs []corev1.EnvVar,
	volumeMounts []corev1.VolumeMount,
	labels map[string]string,
	containerConfig *carbonv1.ContainerConfig,
	workingDirs map[string]string,
	securityContext *corev1.SecurityContext,
	devices *[]carbonv1.Device,
) (
	[]carbonv1.HippoContainer,
	[]corev1.Volume,
	corev1.RestartPolicy,
	bool,
	error,
) {
	return transT4ProcessInfos(role.Version.LaunchPlan.ProcessInfos, ress, image, containerEnvs, volumeMounts, labels, containerConfig, workingDirs, securityContext, devices)
}

func transT4ProcessInfos(
	processInfos []typespec.ProcessInfo,
	ress *corev1.ResourceRequirements,
	image string,
	containerEnvs []corev1.EnvVar,
	volumeMounts []corev1.VolumeMount,
	labels map[string]string,
	containerConfig *carbonv1.ContainerConfig,
	workingDirs map[string]string,
	securityContext *corev1.SecurityContext,
	devices *[]carbonv1.Device,
) (
	[]carbonv1.HippoContainer,
	[]corev1.Volume,
	corev1.RestartPolicy,
	bool,
	error,
) {
	var hippoContainers = make([]carbonv1.HippoContainer, 0, len(processInfos))
	var restartPolicy corev1.RestartPolicy = corev1.RestartPolicyAlways
	processResource, err := getLimitsFromProcessInfo(processInfos)
	if err != nil {
		return nil, nil, restartPolicy, false, err
	}
	ress, err = getRemainRequirements(ress, processResource)
	if err != nil {
		return nil, nil, restartPolicy, false, err
	}

	for i, process := range processInfos {
		envs, err := GetEnvs(&process)
		if nil != err {
			return nil, nil, restartPolicy, false, err
		}
		envs = MergeEnvs(containerEnvs, envs)
		containerName := process.GetName()
		if "" == containerName {
			err := fmt.Errorf("nil process name")
			return hippoContainers, nil, restartPolicy, false, err
		}
		escapeName := EscapeNamespace(containerName)
		var alias string
		if escapeName != containerName {
			alias = containerName
		}
		containerImage := getImageFromProcessInfo(process)
		if "" != containerImage {
			image = containerImage
		}
		var container = corev1.Container{
			Name:            escapeName,
			Image:           image,
			Command:         process.GetCmd(),
			Args:            process.GetArgs(),
			Env:             envs,
			VolumeMounts:    volumeMounts,
			SecurityContext: securityContext,
		}
		container.WorkingDir = workingDirs[process.GetName()]
		container.Lifecycle = getLifeCycleFromProcessInfo(process)
		container.LivenessProbe = getProbeFromProcessInfo(process, "livenessProbe")
		if nil != ress {
			requirements := getResourceRequirements(ress, processResource, i)
			container.Resources = *requirements
		}
		if !process.GetIsDaemon() {
			restartPolicy = corev1.RestartPolicyOnFailure
		}
		var extend = carbonv1.ContainerHippoExterned{}
		if 0 != process.GetRestartCountLimit() || 0 != process.GetStopTimeout() || nil != containerConfig {
			if nil == containerConfig {
				containerConfig = &carbonv1.ContainerConfig{}
			} else {
				containerConfig = containerConfig.DeepCopy()
			}
			containerConfig.RestartCountLimit = process.GetRestartCountLimit()
			containerConfig.StopGracePeriod = process.GetStopTimeout()
			extend.Configs = containerConfig
			if devices != nil {
				extend.Devices = *devices
			}
		}
		if nil != labels && 0 != len(labels) {
			extend.Labels = labels
		}
		extend.Alias = alias
		correctContainerProc(&container)
		var hippoContariner = carbonv1.HippoContainer{
			Container:              container,
			ContainerHippoExterned: extend,
		}
		hippoContainers = append(hippoContainers, hippoContariner)
	}
	return hippoContainers, nil, restartPolicy, false, nil
}

const (
	postStart    = "postStart"
	preStop      = "preStop"
	processImage = "image"
	resources    = "resources"
)

func getRemainRequirements(ress *corev1.ResourceRequirements, processResources []*corev1.ResourceList) (*corev1.ResourceRequirements, error) {
	if ress == nil {
		return ress, nil
	}
	copyRequirements := ress.DeepCopy()
	for _, processResource := range processResources {
		if processResource == nil {
			continue
		}
		for k, v := range *processResource {
			quantity, ok := copyRequirements.Limits[k]
			if ok {
				quantity.SetMilli(quantity.MilliValue() - v.MilliValue())
				if quantity.MilliValue() < 0 {
					return ress, errors.New(fmt.Sprintf("fixed remain requirements %s resources is less than zero", k))
				}
				copyRequirements.Limits[k] = quantity
			}
		}
	}
	return copyRequirements, nil
}

func getResourceRequirements(ress *corev1.ResourceRequirements, processResource []*corev1.ResourceList, processIndex int) *corev1.ResourceRequirements {
	// 这个不会为nil
	specificResources := processResource[processIndex]
	if specificResources != nil {
		var resourcess corev1.ResourceRequirements
		if processIndex == 0 {
			resourcess = *ress.DeepCopy()
		}
		resourcess.Limits = *specificResources
		return &resourcess
	}
	// 这个是剩余没指定资源的长度
	var remainLen = 0
	// 这个是剩余没指定资源最后一个
	var remainLastIndex = 0
	for index, list := range processResource {
		if list == nil {
			remainLastIndex = index
			remainLen++
		}
	}

	var resourcess corev1.ResourceRequirements
	if processIndex == 0 {
		resourcess = *ress.DeepCopy()
	} else {
		resourcess.Limits = map[corev1.ResourceName]k8sresource.Quantity{"cpu": ress.Limits["cpu"]}
		resourcess.Limits["memory"] = ress.Limits["memory"]
		if ress.Requests != nil {
			resourcess.Requests = map[corev1.ResourceName]k8sresource.Quantity{"cpu": ress.Requests["cpu"]}
			resourcess.Requests["memory"] = ress.Requests["memory"]
		}
	}
	if remainLen > 1 {
		splitResource(resourcess.Limits, processIndex, remainLastIndex, remainLen)
		if resourcess.Requests != nil {
			splitResource(resourcess.Requests, processIndex, remainLastIndex, remainLen)
		}
	}

	return &resourcess
}

func splitResource(resourceList corev1.ResourceList, processIndex, remainLastIndex, remainLen int) {
	for k, v := range resourceList {
		if k == "cpu" || k == "memory" {
			total := v.MilliValue()
			if processIndex != remainLastIndex {
				value := total / int64(remainLen)
				v.SetMilli(value)
			} else {
				value := total / int64(remainLen)
				value = total - int64(remainLen-1)*value
				v.SetMilli(value)
			}
			resourceList[k] = v
		}
	}
}

func getLimitsFromProcessInfo(processInfos []typespec.ProcessInfo) ([]*corev1.ResourceList, error) {
	resourceList := make([]*corev1.ResourceList, 0, len(processInfos))
	for _, process := range processInfos {
		otherInfos := process.OtherInfos
		if nil == otherInfos || len(*otherInfos) == 0 {
			resourceList = append(resourceList, nil)
			continue
		}
		var resourcesInfoStr = ""
		for _, otherInfo := range *otherInfos {
			if len(otherInfo) == 2 && resources == otherInfo[0] {
				resourcesInfoStr = otherInfo[1]
				break
			}
		}
		if resourcesInfoStr == "" {
			resourceList = append(resourceList, nil)
			continue
		}
		limitsInfo := make(corev1.ResourceList)
		split := strings.Split(resourcesInfoStr, ";")
		for _, val := range split {
			limit := strings.Split(val, ":")
			if len(limit) != 2 {
				continue
			}
			parseInt, err := strconv.ParseInt(limit[1], 10, 32)
			if err != nil {
				return nil, err
			}
			if limit[0] == "mem" {
				quantity, err := k8sresource.ParseQuantity(fmt.Sprintf("%dMi", parseInt))
				if err != nil {
					return nil, err
				}
				limitsInfo[carbonv1.ResourceMEM] = quantity
			}
			if limit[0] == "cpu" {
				var quantity k8sresource.Quantity
				if err = quantity.UnmarshalJSON([]byte(fmt.Sprintf("\"%dm\"", parseInt*10))); nil != err {
					err = fmt.Errorf("ParseQuantity error %v", err)
					return nil, err
				}
				limitsInfo[carbonv1.ResourceCPU] = quantity
			}
		}
		_, cpuExist := limitsInfo[carbonv1.ResourceCPU]
		_, memExist := limitsInfo[carbonv1.ResourceMEM]
		if !cpuExist || !memExist {
			return nil, errors.New("resources info error：" + resourcesInfoStr)
		}
		resourceList = append(resourceList, &limitsInfo)
	}
	return resourceList, nil
}

func getLifeCycleFromProcessInfo(process typespec.ProcessInfo) *corev1.Lifecycle {
	otherInfos := process.OtherInfos
	var lifecycle *corev1.Lifecycle
	if nil != otherInfos {
		for _, otherInfo := range *otherInfos {
			if len(otherInfo) > 1 && preStop == otherInfo[0] {
				if nil == lifecycle {
					lifecycle = &corev1.Lifecycle{}
				}
				lifecycle.PreStop = &corev1.LifecycleHandler{
					Exec: &corev1.ExecAction{
						Command: otherInfo[1:],
					},
				}
			}
			if len(otherInfo) > 1 && postStart == otherInfo[0] {
				if nil == lifecycle {
					lifecycle = &corev1.Lifecycle{}
				}
				lifecycle.PostStart = &corev1.LifecycleHandler{
					Exec: &corev1.ExecAction{
						Command: otherInfo[1:],
					},
				}
			}
		}
	}
	return lifecycle
}

func getProbeFromProcessInfo(process typespec.ProcessInfo, probeType string) *corev1.Probe {
	otherInfos := process.OtherInfos
	if nil == otherInfos {
		return nil
	}
	for _, otherInfo := range *otherInfos {
		if len(otherInfo) > 1 && probeType == otherInfo[0] {
			return parseProbe(otherInfo[1])
		}
	}
	return nil
}

// some-command or
// command=some-command; confKey=xx; confKey=xx
func parseProbe(s string) *corev1.Probe {
	probe := &corev1.Probe{
		InitialDelaySeconds: 600,
		TimeoutSeconds:      5,
		PeriodSeconds:       5,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}
	v := strings.Split(s, ";")
	if len(v) == 1 {
		probe.Exec = &corev1.ExecAction{
			Command: []string{v[0]},
		}
		return probe
	}
	addrs := map[string]*int32{
		"initialDelaySeconds": &probe.InitialDelaySeconds,
		"timeoutSeconds":      &probe.TimeoutSeconds,
		"periodSeconds":       &probe.PeriodSeconds,
		"successThreshold":    &probe.SuccessThreshold,
		"failureThreshold":    &probe.FailureThreshold,
	}
	for _, kvstr := range v {
		kv := strings.Split(kvstr, "=")
		k := strings.TrimSpace(kv[0])
		if k == "command" {
			probe.Exec = &corev1.ExecAction{
				Command: []string{kv[1]},
			}
			continue
		}
		if ptr, ok := addrs[k]; ok && len(kv) == 2 {
			if i, err := strconv.Atoi(strings.TrimSpace(kv[1])); err == nil {
				*ptr = int32(i)
			}
		}
	}
	if probe.Exec == nil {
		return nil
	}
	return probe
}

func getImageFromProcessInfo(process typespec.ProcessInfo) string {
	otherInfos := process.OtherInfos
	if nil != otherInfos {
		for _, otherInfo := range *otherInfos {
			if len(otherInfo) > 1 && processImage == otherInfo[0] {
				return otherInfo[1]
			}
		}
	}
	return ""
}

func correctContainerProc(container *corev1.Container) error {
	if nil == container {
		return nil
	}
	for i := range container.Command {
		container.Command[i] = common.CorrecteShellExpr(container.Command[i])
	}
	for i := range container.Args {
		container.Args[i] = common.CorrecteShellExpr(container.Args[i])
	}
	for i := range container.Env {
		container.Env[i].Value = common.CorrecteShellExpr(container.Env[i].Value)
	}
	return nil
}
