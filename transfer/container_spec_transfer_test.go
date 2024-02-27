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
	"reflect"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func Test_transPodContainer(t *testing.T) {
	type args struct {
		podDescStr      string
		ress            *corev1.ResourceRequirements
		volumeMounts    []corev1.VolumeMount
		containerEnvs   []corev1.EnvVar
		containerConfig *carbonv1.ContainerConfig
	}
	pod1 := podDesc{
		StopGracePeriod: 3,
		InstanceID:      "ignore-inst-id",
		InstanceName:    "tf-worker",
		Image:           "reg:latest",
		HostConfig: hostConfig{
			Ulimits: []carbonv1.Ulimit{
				{Name: "core", Hard: -1, Soft: -1},
			},
			IpcMode: "host",
		},
		RestartCountLimit: 10,
		Entrypoint:        []string{"sleep", "30"},
		IsDaemon:          true,
		Volumes:           podVolumeDesc{"/etc/": {}},
	}
	pod1VolumeName := "volume-m-0"
	ress1 := &corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU: resource.Quantity{Format: "1m"},
		},
	}
	con1 := carbonv1.HippoContainer{
		Container: corev1.Container{
			Name:         pod1.InstanceName,
			Image:        pod1.Image,
			Command:      pod1.Entrypoint,
			Resources:    *ress1.DeepCopy(),
			VolumeMounts: []corev1.VolumeMount{{Name: pod1VolumeName, MountPath: "/etc/"}},
		},
		ContainerHippoExterned: carbonv1.ContainerHippoExterned{
			Configs: &carbonv1.ContainerConfig{
				Ulimits:           pod1.HostConfig.Ulimits,
				RestartCountLimit: pod1.RestartCountLimit,
				StopGracePeriod:   pod1.StopGracePeriod,
			},
			Alias: pod1.InstanceName,
		},
	}
	podWithEnvs := podDesc{
		HostConfig: hostConfig{
			Ulimits: []carbonv1.Ulimit{
				{Name: "core", Hard: -1, Soft: -1},
			},
		},
		Entrypoint: []string{"sleep", "30"},
		Env:        []string{"a=b", "c=d=e"},
	}
	conEnv := carbonv1.HippoContainer{
		Container: corev1.Container{
			Command: podWithEnvs.Entrypoint,
			Env: []corev1.EnvVar{
				{Name: "a", Value: "b"},
				{Name: "c", Value: "d=e"},
				{Name: "k1", Value: "v1"},
			},
		},
		ContainerHippoExterned: carbonv1.ContainerHippoExterned{
			Configs: &carbonv1.ContainerConfig{
				Ulimits: podWithEnvs.HostConfig.Ulimits,
			},
		},
	}
	podWithChild := podDesc{
		Entrypoint: []string{"sleep", "60"},
		User:       "admin", // non-root user will be omit
		IsDaemon:   true,
		PodExtendedContainers: []podDesc{
			{
				Cmd:   []string{"/bin/sleep", "100"},
				Image: "reg:v1",
			},
		},
	}
	consList := []carbonv1.HippoContainer{
		{
			Container: corev1.Container{
				Command: podWithChild.Entrypoint,
			},
			ContainerHippoExterned: carbonv1.ContainerHippoExterned{
				Configs: &carbonv1.ContainerConfig{},
			},
		},
		{
			Container: corev1.Container{
				Command: podWithChild.PodExtendedContainers[0].Cmd,
				Image:   podWithChild.PodExtendedContainers[0].Image,
			},
			ContainerHippoExterned: carbonv1.ContainerHippoExterned{
				Configs: &carbonv1.ContainerConfig{},
			},
		},
	}

	podWithContainerConfig := podDesc{
		Entrypoint: []string{"sleep", "60"},
		User:       "root",
		IsDaemon:   true,
		HostConfig: hostConfig{
			CapAdd: []corev1.Capability{
				"SYSADMIN",
				"SYS_RAWIO",
			},
			CapDrop: []corev1.Capability{
				"MKNOD",
			},
		},
		PodExtendedContainers: []podDesc{
			{
				Cmd:   []string{"/bin/sleep", "100"},
				Image: "reg:v1",
			},
		},
	}
	consListWithContainerConifg := []carbonv1.HippoContainer{
		{
			Container: corev1.Container{
				Command: podWithContainerConfig.Entrypoint,
				SecurityContext: &corev1.SecurityContext{
					RunAsUser:  utils.Int64Ptr(0),
					RunAsGroup: utils.Int64Ptr(0),
					Capabilities: &corev1.Capabilities{
						Add: []corev1.Capability{
							"SYSADMIN",
							"SYS_RAWIO",
						},
						Drop: []corev1.Capability{
							"MKNOD",
						},
					},
				},
			},
			ContainerHippoExterned: carbonv1.ContainerHippoExterned{
				Configs: &carbonv1.ContainerConfig{
					CPUBvtWarpNs: "55",
				},
			},
		},
		{
			Container: corev1.Container{
				Command: podWithContainerConfig.PodExtendedContainers[0].Cmd,
				Image:   podWithContainerConfig.PodExtendedContainers[0].Image,
			},
			ContainerHippoExterned: carbonv1.ContainerHippoExterned{
				Configs: &carbonv1.ContainerConfig{
					CPUBvtWarpNs: "55",
				},
			},
		},
	}

	tests := []struct {
		name    string
		args    args
		want    []carbonv1.HippoContainer
		want1   corev1.RestartPolicy
		want2   bool
		wantErr bool

		wantVolumes []corev1.Volume
	}{
		{
			name:    "normal case",
			args:    args{utils.ObjJSON(&pod1), ress1, nil, nil, nil},
			want:    []carbonv1.HippoContainer{con1},
			want1:   corev1.RestartPolicyAlways,
			want2:   true,
			wantErr: false,
			wantVolumes: []corev1.Volume{{
				Name: pod1VolumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			}},
		},
		{
			name:        "pod with envs",
			args:        args{utils.ObjJSON(&podWithEnvs), nil, nil, []corev1.EnvVar{{Name: "k1", Value: "v1"}}, nil},
			want:        []carbonv1.HippoContainer{conEnv},
			want1:       corev1.RestartPolicyOnFailure,
			want2:       false,
			wantErr:     false,
			wantVolumes: []corev1.Volume{},
		},
		{
			name:        "pod with children",
			args:        args{utils.ObjJSON(&podWithChild), nil, nil, nil, nil},
			want:        consList,
			want1:       corev1.RestartPolicyAlways,
			want2:       false,
			wantErr:     false,
			wantVolumes: []corev1.Volume{},
		},
		{
			name:        "pod with containerConfig",
			args:        args{utils.ObjJSON(&podWithContainerConfig), nil, nil, nil, &carbonv1.ContainerConfig{CPUBvtWarpNs: "55"}},
			want:        consListWithContainerConifg,
			want1:       corev1.RestartPolicyAlways,
			want2:       false,
			wantErr:     false,
			wantVolumes: []corev1.Volume{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotVolumes, got1, got2, err := transPodContainer(tt.args.podDescStr, tt.args.ress, tt.args.volumeMounts, tt.args.containerEnvs, tt.args.containerConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("transPodContainer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Replace Json cmp because DeepEqual failed.
			if utils.ObjJSON(got) != utils.ObjJSON(tt.want) {
				t.Errorf("container jsonized content not equal: got: %s, want: %s", utils.ObjJSON(got), utils.ObjJSON(tt.want))
				return
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("transPodContainer() got1 = %v, want %v", got1, tt.want1)
			}
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("transPodContainer() got2 = %v, want %v", got2, tt.want2)
			}
			if !reflect.DeepEqual(gotVolumes, tt.wantVolumes) {
				t.Errorf("transPodContainer() gotVolumes = %v, want %v", gotVolumes, tt.wantVolumes)
			}
		})
	}
}

func Test_getLifeCycleFromProcessInfo(t *testing.T) {
	type args struct {
		process typespec.ProcessInfo
	}
	tests := []struct {
		name string
		args args
		want *corev1.Lifecycle
	}{
		{
			name: "nil",
			args: args{
				process: typespec.ProcessInfo{
					OtherInfos: &[][]string{
						{"aaa", "bbb"},
					},
				},
			},
			want: nil,
		},
		{
			name: "preUpdateInfo",
			args: args{
				process: typespec.ProcessInfo{
					OtherInfos: &[][]string{
						{"preStop", "/bin/sh", "sleep 10"},
						{"postStart", "/bin/sh", "sleep 100"},
					},
				},
			},
			want: &corev1.Lifecycle{
				PreStop: &corev1.LifecycleHandler{
					Exec: &corev1.ExecAction{
						Command: []string{"/bin/sh", "sleep 10"},
					},
				},
				PostStart: &corev1.LifecycleHandler{
					Exec: &corev1.ExecAction{
						Command: []string{"/bin/sh", "sleep 100"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getLifeCycleFromProcessInfo(tt.args.process); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getLifeCycleFromProcessInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getImageFromProcessInfo(t *testing.T) {
	type args struct {
		process typespec.ProcessInfo
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "nil",
			args: args{
				process: typespec.ProcessInfo{
					OtherInfos: &[][]string{
						{"aaa", "bbb"},
					},
				},
			},
			want: "",
		},
		{
			name: "imgage",
			args: args{
				process: typespec.ProcessInfo{
					OtherInfos: &[][]string{
						{"image", "reg:imgage"},
					},
				},
			},
			want: "reg:imgage",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getImageFromProcessInfo(tt.args.process); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getLifeCycleFromProcessInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseProbe(t *testing.T) {
	type args struct {
		s string
	}
	getDefault := func() *corev1.Probe {
		return &corev1.Probe{
			InitialDelaySeconds: 600,
			TimeoutSeconds:      5,
			PeriodSeconds:       5,
			SuccessThreshold:    1,
			FailureThreshold:    3,
		}
	}
	tests := []struct {
		name string
		args args
		want *corev1.Probe
	}{
		{
			name: "command only",
			args: args{
				s: "/home/admin/health.sh",
			},
			want: func() *corev1.Probe {
				p := getDefault()
				p.Exec = &corev1.ExecAction{
					Command: []string{"/home/admin/health.sh"},
				}
				return p
			}(),
		},
		{
			name: "invalid command",
			args: args{
				s: "/home/admin/health.sh;",
			},
			want: nil,
		},
		{
			name: "with configs",
			args: args{
				s: " command=/home/admin/health.sh; initialDelaySeconds= 500 ; timeoutSeconds=3",
			},
			want: func() *corev1.Probe {
				p := getDefault()
				p.Exec = &corev1.ExecAction{
					Command: []string{"/home/admin/health.sh"},
				}
				p.InitialDelaySeconds = 500
				p.TimeoutSeconds = 3
				return p
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseProbe(tt.args.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseProbe() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getProbeFromProcessInfo(t *testing.T) {
	type args struct {
		process   typespec.ProcessInfo
		probeType string
	}
	tests := []struct {
		name string
		args args
		want *corev1.Probe
	}{
		{
			name: "normal",
			args: args{
				process: typespec.ProcessInfo{
					OtherInfos: &[][]string{
						{
							"livenessProbe",
							"/home/admin/health.sh",
						},
					},
				},
				probeType: "livenessProbe",
			},
			want: func() *corev1.Probe {
				return &corev1.Probe{
					InitialDelaySeconds: 600,
					TimeoutSeconds:      5,
					PeriodSeconds:       5,
					SuccessThreshold:    1,
					FailureThreshold:    3,
					ProbeHandler: corev1.ProbeHandler{
						Exec: &corev1.ExecAction{
							Command: []string{"/home/admin/health.sh"},
						},
					},
				}
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getProbeFromProcessInfo(tt.args.process, tt.args.probeType); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getProbeFromProcessInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}
