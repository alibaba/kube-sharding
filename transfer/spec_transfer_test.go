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
	"fmt"
	"reflect"
	"testing"

	"github.com/alibaba/kube-sharding/common/features"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/carbon"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_mergeEnvs(t *testing.T) {
	type args struct {
		containerEnvs []corev1.EnvVar
		processEnvs   []corev1.EnvVar
	}
	tests := []struct {
		name string
		args args
		want []corev1.EnvVar
	}{
		{
			name: "merge",
			args: args{
				processEnvs: []corev1.EnvVar{
					{
						Name:  "api",
						Value: "test",
					},
					{
						Name:  "PATH",
						Value: "/sbin:/usr/local/bin:/usr/bin:/usr/local/sbin",
					},
					{
						Name:  "LD_LIBRARY_PATH",
						Value: "/sbin:/usr/local/bin:/usr/bin:/usr/local/sbin",
					},
				},
				containerEnvs: []corev1.EnvVar{
					{
						Name:  "api1",
						Value: "test1",
					},
					{
						Name:  "PATH",
						Value: "/usr/sbin:/usr/X11R6/bin:/opt/hippo/bin:/home/haibin.dhb/.local/bin:/home/haibin.dhb/bin:/opt/hippo/bin",
					},
					{
						Name:  "LD_LIBRARY_PATH",
						Value: "/usr/sbin:/usr/X11R6/bin:/opt/hippo/bin:/home/haibin.dhb/.local/bin:/home/haibin.dhb/bin:/opt/hippo/bin",
					},
				},
			},
			want: []corev1.EnvVar{
				{
					Name:  "api",
					Value: "test",
				},
				{
					Name:  "PATH",
					Value: "/sbin:/usr/local/bin:/usr/bin:/usr/local/sbin:/usr/sbin:/usr/X11R6/bin:/opt/hippo/bin:/home/haibin.dhb/.local/bin:/home/haibin.dhb/bin:/opt/hippo/bin",
				},
				{
					Name:  "LD_LIBRARY_PATH",
					Value: "/sbin:/usr/local/bin:/usr/bin:/usr/local/sbin:/usr/sbin:/usr/X11R6/bin:/opt/hippo/bin:/home/haibin.dhb/.local/bin:/home/haibin.dhb/bin:/opt/hippo/bin",
				},
				{
					Name:  "api1",
					Value: "test1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MergeEnvs(tt.args.containerEnvs, tt.args.processEnvs); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MergeEnvs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_transMajorPriority(t *testing.T) {
	type args struct {
		majorPriority int32
	}
	tests := []struct {
		name    string
		args    args
		want    *int32
		wantErr bool
	}{
		{
			name: "system",
			args: args{
				majorPriority: 0,
			},
			wantErr: false,
			want:    utils.Int32Ptr(200),
		},
		{
			name: "product",
			args: args{
				majorPriority: 32,
			},
			wantErr: false,
			want:    utils.Int32Ptr(100),
		},
		{
			name: "product",
			args: args{
				majorPriority: 2,
			},
			wantErr: false,
			want:    utils.Int32Ptr(190),
		},
		{
			name: "product",
			args: args{
				majorPriority: 1,
			},
			wantErr: false,
			want:    utils.Int32Ptr(195),
		},
		{
			name: "noproduct",
			args: args{
				majorPriority: 33,
			},
			wantErr: false,
			want:    utils.Int32Ptr(95),
		},
		{
			name: "noproduct",
			args: args{
				majorPriority: 40,
			},
			wantErr: false,
			want:    utils.Int32Ptr(72),
		},
		{
			name: "noproduct",
			args: args{
				majorPriority: 34,
			},
			wantErr: false,
			want:    utils.Int32Ptr(90),
		},
		{
			name: "noproduct",
			args: args{
				majorPriority: 64,
			},
			wantErr: false,
			want:    utils.Int32Ptr(0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transMajorPriority(tt.args.majorPriority)
			if (err != nil) != tt.wantErr {
				t.Errorf("transPriority() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if *got != *tt.want {
				t.Errorf("transPriority() = %v, want %v", *got, *tt.want)
			}
		})
	}
}

func Test_transContainerConfigs(t *testing.T) {
	type args struct {
		containerConfigs []string
	}
	var volumeType = corev1.HostPathDirectoryOrCreate
	var volumeTypeFile = corev1.HostPathFile
	var volumeTypeCharDev = corev1.HostPathCharDev
	var privileged = true
	size, _ := resource.ParseQuantity("20Gi")
	tests := []struct {
		name                         string
		args                         args
		wantContainerEnvs            []corev1.EnvVar
		wantContainerLabels          map[string]string
		wantVolumes                  []corev1.Volume
		wantVolumeMounts             []corev1.VolumeMount
		wantConfig                   *carbonv1.ContainerConfig
		wantDevice                   []carbonv1.Device
		wantCPUShareNum              int32
		containerModel               string
		wantDelayTime                int
		wantkataVolumes              []json.RawMessage
		wantpouchAnnotations         string
		wantneedHippoMounts          *bool
		wantrestartWithoutRemove     *bool
		wantErr                      bool
		wantDelayDeleteBackupSeconds int64
		wantWarmUpSeconds            int64
		wantSchedulerName            string
		wantServiceAccount           string
		wantWorkingDirs              map[string]string
		wantSecurityContext          *corev1.SecurityContext
		wantPodSecurityContext       *corev1.PodSecurityContext
		wantDependencyLevel          int
		wantShareProcessNamespace    bool
	}{
		{
			wantDelayTime: 10,
			wantContainerEnvs: []corev1.EnvVar{
				{
					Name:  "containerConfig_HIPPO_APP",
					Value: "wsearch",
				},
				{
					Name:  "containerConfig_spas_secretKey",
					Value: "9eTfzJZOo5x+chsooamvRw==",
				},
				{
					Name:  "containerConfig_tag_a",
					Value: "12",
				},
				{
					Name:  "NUMA_POLICY",
					Value: "MPOL_INTERLEAVE:65535",
				},
			},
			containerModel: "DOCKER",
			wantContainerLabels: map[string]string{
				"AA": "bb",
				"CC": "",
				"DD": "ee",
			},
			wantDevice: []carbonv1.Device{
				{
					PathOnHost:        "/dev/fuse",
					PathInContainer:   "/dev/fuse",
					CgroupPermissions: "mrw",
				},
				{
					PathOnHost:        "/dev/urandom",
					PathInContainer:   "/dev/urandom",
					CgroupPermissions: "mrw",
				},
			},
			wantVolumes: []corev1.Volume{
				{
					Name: "volume-0",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/home/admin/app_role",
							Type: &volumeType,
						},
					},
				},
				{
					Name: "volume-1",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/home/admin/app_role/logs",
							Type: &volumeType,
						},
					},
				},
				{
					Name: "podinfo-downward",
					VolumeSource: corev1.VolumeSource{
						DownwardAPI: &corev1.DownwardAPIVolumeSource{
							Items: []corev1.DownwardAPIVolumeFile{
								corev1.DownwardAPIVolumeFile{
									Path: "labels",
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.labels",
									},
								},
								corev1.DownwardAPIVolumeFile{
									Path: "annotations",
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.annotations",
									},
								},
							},
						},
					},
				},
				{
					Name: "test-virtio-volume",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "test-virtio-volume2",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "volume-m-0",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "volume-m-1",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "volume-m-2",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{
							Medium:    corev1.StorageMediumMemory,
							SizeLimit: &size,
						},
					},
				},
				{
					Name: "timezone-autotrans-0",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/usr/share/zoneinfo/Asia/Shanghai",
							Type: &volumeTypeFile,
						},
					},
				},
				{
					Name: "timezone-autotrans-1",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/usr/share/zoneinfo/Asia/Shanghai",
							Type: &volumeTypeFile,
						},
					},
				},
				{
					Name: "fuse-device",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/dev/fuse",
							Type: &volumeTypeCharDev,
						},
					},
				},
				{
					Name: "shared-data",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
			},
			wantVolumeMounts: []corev1.VolumeMount{
				{
					Name:      "volume-0",
					MountPath: "/home/admin/app_role",
				},
				{
					Name:      "volume-1",
					MountPath: "/home/admin/app_role/logs",
					ReadOnly:  true,
				},
				{
					Name:      "podinfo-downward",
					MountPath: "/etc/podinfo1",
				},
				{
					MountPath: "/dev/vdb",
					Name:      "test-virtio-volume",
				},
				{
					MountPath: "/dev/vdc",
					Name:      "test-virtio-volume2",
				},
				{
					Name:      "volume-m-0",
					MountPath: "/etc",
				},
				{
					Name:      "volume-m-1",
					MountPath: "/ebin",
				},
				{
					Name:      "volume-m-2",
					MountPath: "/dev/shm",
				},
				{
					Name:      "timezone-autotrans-0",
					MountPath: "/etc/localtime",
					ReadOnly:  true,
				},
				{
					Name:      "timezone-autotrans-1",
					MountPath: "/usr/share/zoneinfo/Asia/Shanghai",
					ReadOnly:  true,
				},
				{
					Name:      "fuse-device",
					MountPath: "/dev/fuse",
				},
				{
					Name:             "shared-data",
					MountPath:        "/mnt/shared",
					MountPropagation: (*corev1.MountPropagationMode)(utils.StringPtr("Bidirectional")),
				},
			},
			wantConfig: &carbonv1.ContainerConfig{
				CPUBvtWarpNs:   "CPU_BVT_WARP_NS",
				CPUllcCache:    "CPU_LLC_CACHE",
				NetPriority:    "NET_PRIORITY",
				NoMemcgReclaim: "NO_MEMCG_RECLAIM",
				MemWmarkRatio:  "MEM_WMARK_RATIO",
				MemForceEmpty:  "MEM_FORCE_EMPTY",
				MemExtraBytes:  "MEM_EXTRA_BYTES",
				MemExtraRatio:  "MEM_EXTRA_RATIO",
				Ulimits: []carbonv1.Ulimit{
					{
						Name: "core",
						Soft: -1,
						Hard: -1,
					},
					{
						Name: "memlock",
						Hard: 100,
						Soft: 200,
					},
				},
			},
			wantWorkingDirs: map[string]string{
				"process-name-a": "/home/admin/process/a",
				"process-name-b": "/home/admin/process/b",
			},
			wantCPUShareNum:              10,
			wantpouchAnnotations:         "{\"k\":\"v\"}",
			wantneedHippoMounts:          utils.BoolPtr(true),
			wantrestartWithoutRemove:     utils.BoolPtr(false),
			wantErr:                      false,
			name:                         "trans",
			wantDelayDeleteBackupSeconds: 300,
			wantWarmUpSeconds:            100,
			wantSchedulerName:            "ack",
			wantServiceAccount:           "c2-proxy-admin",
			wantkataVolumes: func() []json.RawMessage {
				var result []json.RawMessage
				json.Unmarshal([]byte(`[{"name": "test-virtio-volume", "mountPath": "/dev/vdb", "type": "virtioDisk", "capacity": "10G", "format": "raw"},{"name": "test-virtio-volume2", "type": "virtioDisk", "capacity": "20G", "format": "raw", "mountPath": "/dev/vdc"}]`), &result)
				return result
			}(),
			wantSecurityContext: func() *corev1.SecurityContext {
				return &corev1.SecurityContext{
					Capabilities: &corev1.Capabilities{
						Add: []corev1.Capability{
							corev1.Capability("sys_admin"),
							corev1.Capability("sys_rawio"),
						},
						Drop: []corev1.Capability{
							corev1.Capability("sys_admin"),
							corev1.Capability("sys_rawio"),
						},
					},
					Privileged: &privileged,
				}
			}(),
			wantPodSecurityContext: func() *corev1.PodSecurityContext {
				return &corev1.PodSecurityContext{
					Sysctls: []corev1.Sysctl{
						{
							Name:  "net.core.rmem_max",
							Value: "100",
						},
						{
							Name:  "vm.dirty_bytes",
							Value: "100",
						},
					},
				}
			}(),
			wantDependencyLevel:       1,
			wantShareProcessNamespace: true,
			args: args{
				containerConfigs: []string{
					"ENVS=(HIPPO_APP=wsearch,spas_secretKey=9eTfzJZOo5x+chsooamvRw==, tag_a = 12)",
					"LABELS=(AA=bb,CC,DD=ee)",
					"BIND_MOUNTS=(/home/admin/${appName}_${roleName}:/home/admin/${appName}_${roleName},/home/admin/${appName}_${roleName}/logs:/home/admin/${appName}_${roleName}/logs:ro)",
					"DOWNWARD_API=(metadata.labels,metadata.annotations:/etc/podinfo1)",
					"WORKING_DIRS=(process-name-a:/home/admin/process/a,process-name-b:/home/admin/process/b)",
					"KATA_VOLUME_MOUNTS=[{\"name\": \"test-virtio-volume\", \"mountPath\": \"/dev/vdb\", \"type\": \"virtioDisk\", \"capacity\": \"10G\", \"format\": \"raw\"},{\"name\": \"test-virtio-volume2\", \"type\": \"virtioDisk\", \"capacity\": \"20G\", \"format\": \"raw\", \"mountPath\": \"/dev/vdc\"}]",
					"POUCH_ANNOTATIONS={\"k\":\"v\"}",
					"NEED_HIPPO_MOUNT=true",
					"RESTART_WITHOUT_REMOVE=false",
					"VOLUME_MOUNTS=(/etc,/ebin,/dev/shm:Memory:20Gi)",
					"NO_MEM_LIMIT=NO_MEM_LIMIT",
					"CPU_RESERVED_TO_SHARE_NUM=10",
					"MEM_POLICY=MEM_INTERLEAVE",
					"NO_MEMCG_RECLAIM=NO_MEMCG_RECLAIM",
					"MEM_WMARK_RATIO=MEM_WMARK_RATIO",
					"MEM_FORCE_EMPTY=MEM_FORCE_EMPTY",
					"MEM_EXTRA_BYTES=MEM_EXTRA_BYTES",
					"MEM_EXTRA_RATIO=MEM_EXTRA_RATIO",
					"SERVICE_ACCOUNT=c2-proxy-admin",
					"OOM_CONTAINER_STOP=OOM_CONTAINER_STOP",
					"OOM_KILL_DISABLE=OOM_KILL_DISABLE",
					"NET_PRIORITY=NET_PRIORITY",
					"CPU_BVT_WARP_NS=CPU_BVT_WARP_NS",
					"CPU_LLC_CACHE=CPU_LLC_CACHE",
					"CONTAINER_MODEL=DOCKER",
					"TIMEZONE=/Asia/Shanghai",
					"PREDICT_DELAY_SECONDS=10",
					"DELAY_DELETE_BACKUP_SECONDS=300",
					"WARM_UP_SECONDS=100",
					"SCHEDULER_NAME=ack",
					"SYSCTLS=(net.core.rmem_max=100,vm.dirty_bytes=100)",
					"ULIMITS=(core:-1:-1,memlock:100:200)",
					"CAPADDS=(sys_admin,sys_rawio)",
					"CAPDROPS=(sys_admin,sys_rawio)",
					"DEVICES=/dev/fuse;/dev/urandom",
					"DEPENDENCY_LEVEL=1",
					"PRIVILEGED=true",
					"SHARE_PROCESS_NAMESPACE=true",
					"JSON_VOLUME_MOUNTS=[{\"mountPath\":\"/dev/fuse\",\"name\":\"fuse-device\"},{\"mountPath\":\"/mnt/shared\",\"name\":\"shared-data\",\"mountPropagation\":\"Bidirectional\"}]",
					"JSON_VOLUMES=[{\"hostPath\":{\"path\":\"/dev/fuse\",\"type\":\"CharDevice\"},\"name\":\"fuse-device\"},{\"name\":\"shared-data\",\"emptyDir\":{}}]",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cconf, err := transContainerConfigs("app", "role", tt.args.containerConfigs)
			if (err != nil) != tt.wantErr {
				t.Errorf("transContainerConfigs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(cconf.containerEnvs, tt.wantContainerEnvs) {
				t.Errorf("transContainerConfigs() gotContainerEnvs = %v, want %v", cconf.containerEnvs, tt.wantContainerEnvs)
			}
			if !reflect.DeepEqual(cconf.containerLabels, tt.wantContainerLabels) {
				t.Errorf("transContainerConfigs() gotContainerLabels = %v, want %v", cconf.containerLabels, tt.wantContainerLabels)
			}
			if !reflect.DeepEqual(cconf.volumes, tt.wantVolumes) {
				t.Errorf("transContainerConfigs() gotVolumes = %v, want %v", utils.ObjJSON(cconf.volumes), utils.ObjJSON(tt.wantVolumes))
			}
			if len(cconf.volumeMounts) != len(tt.wantVolumeMounts) {
				t.Errorf("transContainerConfigs() gotVolumeMounts = %v, want %v", cconf.volumeMounts, tt.wantVolumeMounts)
			}
			volumeMountsStr, err := json.Marshal(cconf.volumeMounts)
			if err != nil {
				t.Errorf("transContainerConfigs() can't Marshal gotVolumeMounts = %v, err %v", cconf.volumeMounts, err)
			}
			wantVolumeMountsStr, err := json.Marshal(tt.wantVolumeMounts)
			if err != nil {
				t.Errorf("transContainerConfigs() can't Marshal wantVolumeMounts = %v, err %v", tt.wantVolumeMounts, err)
			}
			if string(volumeMountsStr) != string(wantVolumeMountsStr) {
				t.Errorf("transContainerConfigs() gotVolumeMounts = %v, want %v", string(volumeMountsStr), string(wantVolumeMountsStr))
			}

			if !reflect.DeepEqual(cconf.config, tt.wantConfig) {
				t.Errorf("transContainerConfigs() gotConfig = %v, want %v", cconf.config, tt.wantConfig)
			}
			if cconf.cpuShareNum != tt.wantCPUShareNum {
				t.Errorf("transContainerConfigs() gotCpuShareNum = %v, want %v", cconf.cpuShareNum, tt.wantCPUShareNum)
			}
			if cconf.containerModel != tt.containerModel {
				t.Errorf("transContainerConfigs() gotContainerModel = %v, want %v", cconf.containerModel, tt.containerModel)
			}
			if int(cconf.predictDelayTime) != tt.wantDelayTime {
				t.Errorf("transContainerConfigs() delayTime = %v, want %v", cconf.predictDelayTime, tt.wantDelayTime)
			}
			if !reflect.DeepEqual(cconf.hippoVolumes, tt.wantkataVolumes) {
				t.Errorf("transContainerConfigs() gotKVolumes = %v, want %v", cconf.hippoVolumes, tt.wantkataVolumes)
			}
			if !reflect.DeepEqual(cconf.pouchAnnotations, tt.wantpouchAnnotations) {
				t.Errorf("transContainerConfigs() gotPouchAnnotation = %v, want %v", cconf.pouchAnnotations, tt.wantpouchAnnotations)
			}
			if !reflect.DeepEqual(cconf.needHippoMounts, tt.wantneedHippoMounts) {
				t.Errorf("transContainerConfigs() gotNeedHippoMounts = %v, want %v", cconf.needHippoMounts, tt.wantneedHippoMounts)
			}
			if !reflect.DeepEqual(cconf.restartWithoutRemove, tt.wantrestartWithoutRemove) {
				t.Errorf("transContainerConfigs() gotRestartWithoutRemove = %v, want %v", cconf.restartWithoutRemove, tt.wantrestartWithoutRemove)
			}
			if !reflect.DeepEqual(cconf.delayDeleteBackup, tt.wantDelayDeleteBackupSeconds) {
				t.Errorf("transContainerConfigs() gotDelayDeleteBackupSeconds= %v, want %v", cconf.delayDeleteBackup, tt.wantrestartWithoutRemove)
			}
			if !reflect.DeepEqual(cconf.warmUpSeconds, tt.wantWarmUpSeconds) {
				t.Errorf("transContainerConfigs() gotWarmUpSeconds= %v, want %v", cconf.warmUpSeconds, tt.wantWarmUpSeconds)
			}
			if !reflect.DeepEqual(cconf.schedulerName, tt.wantSchedulerName) {
				t.Errorf("transContainerConfigs() gotSchedulerName= %v, want %v", cconf.schedulerName, tt.wantSchedulerName)
			}
			if !reflect.DeepEqual(cconf.serviceAccount, tt.wantServiceAccount) {
				t.Errorf("transContainerConfigs() gotServiceAccount= %v, want %v", cconf.serviceAccount, tt.wantServiceAccount)
			}
			if !reflect.DeepEqual(cconf.workingDirs, tt.wantWorkingDirs) {
				t.Errorf("transContainerConfigs() gotWorkingDirs= %v, want %v", cconf.workingDirs, tt.wantWorkingDirs)
			}
			if !reflect.DeepEqual(cconf.securityContext, tt.wantSecurityContext) {
				t.Errorf("transContainerConfigs() gotSecrutiyContext= %v, want %v", cconf.securityContext, tt.wantSecurityContext)
			}
			if !reflect.DeepEqual(cconf.podSecurityContext, tt.wantPodSecurityContext) {
				t.Errorf("transContainerConfigs() gotPodSecurityContext= %v, want %v", cconf.podSecurityContext, tt.wantPodSecurityContext)
			}
			if !reflect.DeepEqual(cconf.dependencyLevel, tt.wantDependencyLevel) {
				t.Errorf("transContainerConfigs() gotDependencyLevel= %v, want %v", cconf.dependencyLevel, tt.wantDependencyLevel)
			}
			if !reflect.DeepEqual(*cconf.shareProcessNamespace, tt.wantShareProcessNamespace) {
				t.Errorf("transContainerConfigs() gotShareProcessNamespace= %v, want %v", *cconf.shareProcessNamespace, tt.wantShareProcessNamespace)
			}
			if !reflect.DeepEqual(*cconf.devices, tt.wantDevice) {
				t.Errorf("transContainerConfigs() gotDevice= %v, want %v", *cconf.devices, tt.wantDevice)
			}
		})
	}
}

func Test_getRecoverStrategy(t *testing.T) {
	type args struct {
		globalPlan typespec.GlobalPlan
	}
	tests := []struct {
		name string
		args args
		want carbonv1.RecoverStrategy
	}{
		{
			name: "nil property",
			args: args{
				globalPlan: typespec.GlobalPlan{},
			},
			want: carbonv1.DefaultRecoverStrategy,
		},
		{
			name: "smooth recover, nil value",
			args: args{
				globalPlan: typespec.GlobalPlan{
					Properties: func() *map[string]string {
						v := map[string]string{}
						return &v
					}(),
				},
			},
			want: carbonv1.DefaultRecoverStrategy,
		},
		{
			name: "smooth recover, wrong value",
			args: args{
				globalPlan: typespec.GlobalPlan{
					Properties: func() *map[string]string {
						v := map[string]string{
							propUseSmoothRecoverKey: "hehe",
						}
						return &v
					}(),
				},
			},
			want: carbonv1.DefaultRecoverStrategy,
		},
		{
			name: "smooth recover, true value",
			args: args{
				globalPlan: typespec.GlobalPlan{
					Properties: func() *map[string]string {
						v := map[string]string{
							propUseSmoothRecoverKey: "true",
						}
						return &v
					}(),
				},
			},
			want: carbonv1.DefaultRecoverStrategy,
		},
		{
			name: "smooth recover, false value",
			args: args{
				globalPlan: typespec.GlobalPlan{
					Properties: func() *map[string]string {
						v := map[string]string{
							propUseSmoothRecoverKey: "false",
						}
						return &v
					}(),
				},
			},
			want: carbonv1.DirectReleasedRecoverStrategy,
		},
		{
			name: "not recover, nil value",
			args: args{
				globalPlan: typespec.GlobalPlan{
					Properties: func() *map[string]string {
						v := map[string]string{}
						return &v
					}(),
				},
			},
			want: carbonv1.DefaultRecoverStrategy,
		},
		{
			name: "not recover, wrong value",
			args: args{
				globalPlan: typespec.GlobalPlan{
					Properties: func() *map[string]string {
						v := map[string]string{
							propRecoverKey: "hehe",
						}
						return &v
					}(),
				},
			},
			want: carbonv1.DefaultRecoverStrategy,
		},
		{
			name: "default recover",
			args: args{
				globalPlan: typespec.GlobalPlan{
					Properties: func() *map[string]string {
						v := map[string]string{
							propRecoverKey: carbonv1.DefaultRecoverStrategy,
						}
						return &v
					}(),
				},
			},
			want: carbonv1.DefaultRecoverStrategy,
		},
		{
			name: "not recover",
			args: args{
				globalPlan: typespec.GlobalPlan{
					Properties: func() *map[string]string {
						v := map[string]string{
							propRecoverKey: carbonv1.NotRecoverStrategy,
						}
						return &v
					}(),
				},
			},
			want: carbonv1.NotRecoverStrategy,
		},
		{
			name: "recover overwrite",
			args: args{
				globalPlan: typespec.GlobalPlan{
					Properties: func() *map[string]string {
						v := map[string]string{
							propRecoverKey:          carbonv1.NotRecoverStrategy,
							propUseSmoothRecoverKey: "true",
						}
						return &v
					}(),
				},
			},
			want: carbonv1.NotRecoverStrategy,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getRecoverStrategy(tt.args.globalPlan); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getRecoverStrategy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getTerminationGracePeriodSeconds(t *testing.T) {
	type args struct {
		containers []carbonv1.HippoContainer
	}
	containers := make([]carbonv1.HippoContainer, 0, 0)
	container1 := carbonv1.HippoContainer{}
	container1.Configs = &carbonv1.ContainerConfig{}
	container1.Configs.StopGracePeriod = 90

	container2 := carbonv1.HippoContainer{}

	container3 := carbonv1.HippoContainer{}
	container3.Configs = &carbonv1.ContainerConfig{}
	container3.Configs.StopGracePeriod = 120

	containers = append(containers, container1)
	containers = append(containers, container2)
	containers = append(containers, container3)
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "test1",
			args: args{containers: containers},
			want: int64(120),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getTerminationGracePeriodSeconds(tt.args.containers); got != tt.want {
				t.Errorf("getTerminationGracePeriodSeconds() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getTerminationGracePeriodSeconds2(t *testing.T) {
	type args struct {
		containers []carbonv1.HippoContainer
	}
	containers := make([]carbonv1.HippoContainer, 0, 0)
	container1 := carbonv1.HippoContainer{}
	container1.Configs = &carbonv1.ContainerConfig{}
	container1.Configs.StopGracePeriod = 90

	container2 := carbonv1.HippoContainer{}

	container3 := carbonv1.HippoContainer{}
	container3.Configs = &carbonv1.ContainerConfig{}
	container3.Configs.StopGracePeriod = 0

	containers = append(containers, container1)
	containers = append(containers, container2)
	containers = append(containers, container3)
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "test1",
			args: args{containers: containers},
			want: int64(100),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getTerminationGracePeriodSeconds(tt.args.containers); got != tt.want {
				t.Errorf("getTerminationGracePeriodSeconds() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_fillTemplateMetaTags(t *testing.T) {
	type args struct {
		hippoPodTemplate *carbonv1.HippoPodTemplate
		metaTags         *map[string]string
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		wantLabels map[string]string
	}{
		{
			name: "normal",
			args: args{
				hippoPodTemplate: &carbonv1.HippoPodTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
						Labels:      map[string]string{},
					},
				},
				metaTags: &map[string]string{
					"k": "v",
				},
			},
			wantErr: false,
			wantLabels: map[string]string{
				"k": "v",
			},
		},
		{
			name: "not valid",
			args: args{
				hippoPodTemplate: &carbonv1.HippoPodTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
						Labels:      map[string]string{},
					},
				},
				metaTags: &map[string]string{
					"--.0": "p0.9~",
				},
			},
			wantLabels: map[string]string{},
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := fillTemplateByMetaTags(tt.args.hippoPodTemplate, tt.args.metaTags, nil); (err != nil) != tt.wantErr {
				t.Errorf("tranMetaTags() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.hippoPodTemplate.ObjectMeta.Labels, tt.wantLabels) {
				t.Errorf("tranMetaTags() got labels %v, want labels %v", tt.args.hippoPodTemplate.ObjectMeta.Labels, tt.wantLabels)
			}
		})
	}
}

func Test_isSysRole(t *testing.T) {
	type args struct {
		role *typespec.RolePlan
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "not-sys-role",
			args: args{
				role: &typespec.RolePlan{
					Version: typespec.VersionedPlan{
						ResourcePlan: typespec.ResourcePlan{
							Priority: &carbon.CarbonPriority{
								MajorPriority: func() *int32 { var p1 int32 = 1; return &p1 }(),
							},
							Queue: utils.StringPtr("system"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "not-sys-role",
			args: args{
				role: &typespec.RolePlan{
					Version: typespec.VersionedPlan{
						ResourcePlan: typespec.ResourcePlan{
							Priority: &carbon.CarbonPriority{
								MajorPriority: func() *int32 { var p1 int32; return &p1 }(),
							},
							Queue: utils.StringPtr("system1"),
						},
					},
				},
			},
			want: false,
		},
		{
			name: "sys-role",
			args: args{
				role: &typespec.RolePlan{
					Version: typespec.VersionedPlan{
						ResourcePlan: typespec.ResourcePlan{
							Priority: &carbon.CarbonPriority{
								MajorPriority: func() *int32 { var p1 int32; return &p1 }(),
							},
							Queue: utils.StringPtr("system"),
						},
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSysRole(tt.args.role); got != tt.want {
				t.Errorf("isSysRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_checkRollingsetValidation(t *testing.T) {
	type args struct {
		rollingset *carbonv1.RollingSet
		template   *carbonv1.HippoPodTemplate
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "validation",
			args: args{
				rollingset: &carbonv1.RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"sigma.ali/instance-group": "group",
						},
					},
				},
				template: &carbonv1.HippoPodTemplate{
					Spec: carbonv1.HippoPodSpec{
						Containers: []carbonv1.HippoContainer{
							{
								Container: corev1.Container{
									Env: []corev1.EnvVar{
										{
											Name: "a",
										},
										{
											Name: "b",
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkRollingsetValidation(tt.args.rollingset, tt.args.template); (err != nil) != tt.wantErr {
				t.Errorf("checkRollingsetValidation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_transMetaTags(t *testing.T) {
	type args struct {
		metaTags *map[string]string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		want1   map[string]string
		wantErr bool
	}{
		{
			name: "simple",
			args: args{
				&map[string]string{
					"k1": "v1",
				},
			},
			want: map[string]string{
				"k1": "v1",
			},
			want1:   map[string]string{},
			wantErr: false,
		},
		{
			name: "with annotations",
			args: args{
				&map[string]string{
					"k1":            "v1",
					"annotation:k1": "_v1",
					"annotation":    "a1",
				},
			},
			want: map[string]string{
				"k1":         "v1",
				"annotation": "a1",
			},
			want1: map[string]string{
				"k1": "_v1",
			},
			wantErr: false,
		},
		{
			name: "portrait annotations",
			args: args{
				&map[string]string{
					"app.hippo.io/appShortName":  "app1",
					"app.hippo.io/roleShortName": "role1",
				},
			},
			want: map[string]string{},
			want1: map[string]string{
				"app.hippo.io/appShortName":  "app1",
				"app.hippo.io/roleShortName": "role1",
			},
			wantErr: false,
		},
		{
			name: "label key error",
			args: args{
				&map[string]string{
					"_invalid_k": "v",
				},
			},
			want:    nil,
			want1:   nil,
			wantErr: true,
		},
		{
			name: "label value error",
			args: args{
				&map[string]string{
					"k": "_v",
				},
			},
			want:    nil,
			want1:   nil,
			wantErr: true,
		},
		{
			name: "annotation key error",
			args: args{
				&map[string]string{
					"annotation:_invalid_k": "v",
				},
			},
			want:    nil,
			want1:   nil,
			wantErr: true,
		},
		{
			name: "valid serverless label",
			args: args{
				&map[string]string{
					carbonv1.LabelServerlessAppName:       "app1",
					carbonv1.LabelServerlessInstanceGroup: "app1.group1",
				},
			},
			want: map[string]string{
				carbonv1.LabelServerlessAppName:       "app1",
				carbonv1.LabelServerlessInstanceGroup: "app1.group1",
			},
			want1:   map[string]string{},
			wantErr: false,
		},
		{
			name: "invalid serverless label",
			args: args{
				&map[string]string{
					carbonv1.LabelServerlessAppName:       "@app1",
					carbonv1.LabelServerlessInstanceGroup: "@app1.group1",
				},
			},
			want: map[string]string{
				carbonv1.LabelServerlessAppName:       carbonv1.LabelValueHash("@app1", true),
				carbonv1.LabelServerlessInstanceGroup: carbonv1.LabelValueHash("@app1.group1", true),
			},
			want1: map[string]string{
				carbonv1.LabelServerlessAppName:       "@app1",
				carbonv1.LabelServerlessInstanceGroup: "@app1.group1",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := transMetaTags(tt.args.metaTags)
			if (err != nil) != tt.wantErr {
				t.Errorf("transMetaTags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("transMetaTags() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("transMetaTags() got1 = %v, want1 %v", got1, tt.want1)
			}
		})
	}
}

func TestCreateTemplateLabels(t *testing.T) {
	plan := &typespec.RolePlan{
		Global: typespec.GlobalPlan{},
		Version: typespec.VersionedPlan{
			LaunchPlan: typespec.LaunchPlan{
				ProcessInfos: []typespec.ProcessInfo{
					{
						Name: utils.StringPtr("p1"),
						Cmd:  utils.StringPtr("sleep"),
					},
				},
			},
		},
	}
	app := "app1"
	groupID := "g1"
	roleID := "r1"
	// AOP usage
	pod, _, err := CreateTemplate(app, groupID, roleID, plan, SchTypeNode)
	assert.Nil(t, err)
	assert.Equal(t, app, carbonv1.GetAppName(pod))
	assert.Equal(t, groupID+"."+roleID, carbonv1.GetRoleName(pod))

	// BS usage
	plan.Global.Properties = &map[string]string{
		propIsMasterFrameworkModeKey: "true",
	}
	pod, _, err = CreateTemplate(app, groupID, roleID, plan, SchTypeNode)
	assert.Nil(t, err)
	assert.Equal(t, roleID, carbonv1.GetRoleName(pod))

	// Other
	pod, _, err = CreateTemplate(app, groupID, roleID, plan, SchTypeRole)
	assert.Nil(t, err)
	assert.Equal(t, groupID+"."+roleID, carbonv1.GetRoleName(pod))

	pod, _, err = CreateTemplate(app, groupID, roleID, plan, SchTypeRole)
	assert.Nil(t, err)
	assert.Equal(t, groupID+"."+roleID, carbonv1.GetRoleName(pod))
}

func Test_createWorkerNodeName(t *testing.T) {
	type args struct {
		appName        string
		groupID        string
		rollingsetName string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				appName:        "appname",
				groupID:        "groupid",
				rollingsetName: "rollingsetname",
			},
			want:    "appname.groupid.rollingsetname",
			wantErr: false,
		},
		{
			name: "test",
			args: args{
				appName:        "appname",
				groupID:        "groupid..groupid",
				rollingsetName: "rollingsetname",
			},
			want:    "0b0b2e7ec79164247992a1f081c6936d",
			wantErr: false,
		},
		{
			name: "test",
			args: args{
				appName:        "appnameappnameappnameappnameappnameappnameappname",
				groupID:        "groupidgroupidgroupidgroupid",
				rollingsetName: "rollingsetname",
			},
			want:    "appnameappnameappnameappnameappnameappnameappname.groupidgroupidgroupidgroupid.rollingsetname",
			wantErr: false,
		},
		{
			name: "test",
			args: args{
				appName:        "appname",
				groupID:        "groupid",
				rollingsetName: "groupid",
			},
			want:    "appname.groupid",
			wantErr: false,
		},
		{
			name: "too long ",
			args: args{
				appName:        "appnametoolongappnametoolongappnametoolongappnametoolongappnametoolongappnametoolongappnametoolongappnametoolongappnametoolongappnametoolongappnametoolongappnametoolongappnametoolong",
				groupID:        "groupid",
				rollingsetName: "groupid",
			},
			want:    "5cf9ca4.groupid",
			wantErr: false,
		},
		{
			name: "too long ",
			args: args{
				appName:        "appname",
				groupID:        "groupidtoolonggroupidtoolonggroupidtoolonggroupidtoolonggroupidtoolonggroupidtoolonggroupidtoolonggroupidtoolonggroupidtoolonggroupidtoolonggroupidtoolonggroupidtoolonggroupidtoolong",
				rollingsetName: "groupidtoolonggroupidtoolonggroupidtoolonggroupidtoolonggroupidtoolonggroupidtoolonggroupidtoolonggroupidtoolonggroupidtoolonggroupidtoolonggroupidtoolonggroupidtoolonggroupidtoolong",
			},
			want:    "appname.fafae54a3b2a7c79be593b83202f49c4",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createWorkerNodeName(tt.args.appName, tt.args.groupID, tt.args.rollingsetName)
			if (err != nil) != tt.wantErr {
				t.Errorf("createWorkerNodeName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("createWorkerNodeName() = :%v, want :%v", got, tt.want)
			}
		})
	}
}

func TestInjectSystemdToContainer(t *testing.T) {
	type args struct {
		hippoPodTemplate *carbonv1.HippoPodTemplate
		openInject       bool
	}
	tests := []struct {
		name              string
		args              args
		wantAnno          map[string]string
		wantContainerName []string
	}{
		{
			name: "open",
			args: args{
				hippoPodTemplate: &carbonv1.HippoPodTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
						Labels:      map[string]string{},
					},
					Spec: carbonv1.HippoPodSpec{
						Containers: []carbonv1.HippoContainer{
							{
								Container: corev1.Container{
									Name: "test",
								},
							},
						},
					},
				},
				openInject: true,
			},
			wantAnno: map[string]string{
				carbonv1.AnnotationInjectSystemd: carbonv1.MainContainer,
			},
			wantContainerName: []string{carbonv1.MainContainer},
		},
		{
			name: "close",
			args: args{
				hippoPodTemplate: &carbonv1.HippoPodTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
						Labels:      map[string]string{},
					},
					Spec: carbonv1.HippoPodSpec{
						Containers: []carbonv1.HippoContainer{
							{
								Container: corev1.Container{
									Name: "test",
								},
							},
						},
					},
				},
				openInject: false,
			},
			wantAnno:          map[string]string{},
			wantContainerName: []string{"test"},
		},
		{
			name: "no container",
			args: args{
				hippoPodTemplate: &carbonv1.HippoPodTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
						Labels:      map[string]string{},
					},
					Spec: carbonv1.HippoPodSpec{},
				},
				openInject: true,
			},
			wantAnno:          map[string]string{},
			wantContainerName: []string{},
		},
		{
			name: "two container",
			args: args{
				hippoPodTemplate: &carbonv1.HippoPodTemplate{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
						Labels:      map[string]string{},
					},
					Spec: carbonv1.HippoPodSpec{
						Containers: []carbonv1.HippoContainer{
							{
								Container: corev1.Container{
									Name: "test",
								},
							},
							{
								Container: corev1.Container{
									Name: "test2",
								},
							},
						},
					},
				},
				openInject: true,
			},
			wantAnno:          map[string]string{},
			wantContainerName: []string{"test", "test2"},
		},
	}
	defer func() {
		features.C2MutableFeatureGate.Set("InjectSystemdToContainer=false")
	}()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features.C2MutableFeatureGate.Set(fmt.Sprintf("InjectSystemdToContainer=%v", tt.args.openInject))
			injectSystemdToContainer(tt.args.hippoPodTemplate)
			if !reflect.DeepEqual(tt.args.hippoPodTemplate.ObjectMeta.Annotations, tt.wantAnno) {
				t.Errorf("injectSystemdToContainer() got anno %v, want anno %v", tt.args.hippoPodTemplate.ObjectMeta.Labels, tt.wantAnno)
			}
			for i, container := range tt.args.hippoPodTemplate.Spec.Containers {
				if tt.wantContainerName[i] != container.Name {
					t.Errorf("injectSystemdToContainer() got ContainerName %v, want Name %v", container.Name, tt.wantContainerName[i])
				}
			}
		})
	}
}
