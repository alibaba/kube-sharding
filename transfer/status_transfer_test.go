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
	"time"

	"github.com/alibaba/kube-sharding/common/features"

	"github.com/alibaba/kube-sharding/common"
	"github.com/alibaba/kube-sharding/common/config"
	"github.com/golang/mock/gomock"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/carbon"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/hippo"
	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestTransRoleSlotStatus(t *testing.T) {
	type args struct {
		worker     *carbonv1.WorkerNode
		restarting bool
	}
	var daemonPod carbonv1.HippoPodTemplate
	daemonPod.Spec.RestartPolicy = corev1.RestartPolicyAlways
	tests := []struct {
		name    string
		args    args
		want    *carbon.SlotStatus
		wantErr bool
	}{
		{
			name: "TerminateDaemon",
			args: args{worker: &carbonv1.WorkerNode{
				Spec: carbonv1.WorkerNodeSpec{VersionPlan: carbonv1.VersionPlan{
					SignedVersionPlan: carbonv1.SignedVersionPlan{
						Template: &daemonPod,
					},
				}},
				Status: carbonv1.WorkerNodeStatus{AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{Phase: carbonv1.Terminated}}}},
			want: &carbon.SlotStatus{
				Status: func() *carbon.SlotType {
					slotType := carbon.SlotType_ST_PROC_FAILED
					return &slotType
				}(),
			},
		},
		{
			name: "TestTerminate",
			args: args{worker: &carbonv1.WorkerNode{Status: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{Phase: carbonv1.Terminated}}}},
			want: &carbon.SlotStatus{
				Status: func() *carbon.SlotType {
					slotType := carbon.SlotType_ST_PROC_TERMINATED
					return &slotType
				}(),
			},
		},
		{
			name: "TestFailed",
			args: args{worker: &carbonv1.WorkerNode{Status: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{Phase: carbonv1.Failed}}}},
			want: &carbon.SlotStatus{
				Status: func() *carbon.SlotType {
					slotType := carbon.SlotType_ST_PROC_FAILED
					return &slotType
				}(),
			},
		},
		{
			name: "TestUnknown",
			args: args{worker: &carbonv1.WorkerNode{Status: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{Phase: carbonv1.Unknown}}}},
			want: &carbon.SlotStatus{
				Status: func() *carbon.SlotType {
					slotType := carbon.SlotType_ST_UNKNOWN
					return &slotType
				}(),
			},
		},
		{
			name: "TestRunning",
			args: args{worker: &carbonv1.WorkerNode{Status: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{Phase: carbonv1.Running}}}},
			want: &carbon.SlotStatus{
				Status: func() *carbon.SlotType {
					slotType := carbon.SlotType_ST_PROC_RUNNING
					return &slotType
				}(),
			},
		},
		{
			name: "TestRunning",
			args: args{worker: &carbonv1.WorkerNode{Status: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{Phase: carbonv1.Running}}},
				restarting: true,
			},
			want: &carbon.SlotStatus{
				Status: func() *carbon.SlotType {
					slotType := carbon.SlotType_ST_PROC_RESTARTING
					return &slotType
				}(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TransRoleSlotStatus(tt.args.worker, tt.args.restarting)
			if (err != nil) != tt.wantErr {
				t.Errorf("TransRoleSlotStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TransRoleSlotStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_transPackageStatus(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want *hippo.PackageStatus_Status
	}{
		{
			name: "pod phase unknown",
			args: args{
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodUnknown,
					},
				},
			},
			want: func() *hippo.PackageStatus_Status {
				status := hippo.PackageStatus_IS_UNKNOWN
				return &status
			}(),
		},
		{
			name: "pod phase Pending allocated",
			args: args{
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase:  corev1.PodPending,
						HostIP: "1.1.1.1",
					},
				},
			},
			want: func() *hippo.PackageStatus_Status {
				status := hippo.PackageStatus_IS_INSTALLING
				return &status
			}(),
		},
		{
			name: "pod phase Pending not allocated",
			args: args{
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase: corev1.PodPending,
					},
				},
			},
			want: func() *hippo.PackageStatus_Status {
				status := hippo.PackageStatus_IS_UNKNOWN
				return &status
			}(),
		},
		{
			name: "pod phase Running",
			args: args{
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase:  corev1.PodRunning,
						HostIP: "1.1.1.1",
					},
				},
			},
			want: func() *hippo.PackageStatus_Status {
				status := hippo.PackageStatus_IS_INSTALLED
				return &status
			}(),
		},
		{
			name: "pod phase Succeeded",
			args: args{
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase:  corev1.PodSucceeded,
						HostIP: "1.1.1.1",
					},
				},
			},
			want: func() *hippo.PackageStatus_Status {
				status := hippo.PackageStatus_IS_INSTALLED
				return &status
			}(),
		},
		{
			name: "pod phase Failed",
			args: args{
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase:  corev1.PodFailed,
						HostIP: "1.1.1.1",
					},
				},
			},
			want: func() *hippo.PackageStatus_Status {
				status := hippo.PackageStatus_IS_INSTALLED
				return &status
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transPackageStatus(tt.args.pod); !reflect.DeepEqual(got.Status, tt.want) {
				t.Errorf("transPackageStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_transSlaveStatus(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want *hippo.SlaveStatus_Status
	}{
		{
			name: "host not allocate",
			args: args{
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						HostIP: "",
					},
				},
			},
			want: func() *hippo.SlaveStatus_Status {
				status := hippo.SlaveStatus_UNKNOWN
				return &status
			}(),
		},
		{
			name: "host allocate",
			args: args{
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						HostIP: "1.1.1.1",
					},
				},
			},
			want: func() *hippo.SlaveStatus_Status {
				status := hippo.SlaveStatus_ALIVE
				return &status
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transSlaveStatus(tt.args.pod); !reflect.DeepEqual(got.Status, tt.want) {
				t.Errorf("transSlaveStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTransSlotInfo(t *testing.T) {
	roleName := "roleName"
	workerNode := carbonv1.WorkerNode{
		Status: carbonv1.WorkerNodeStatus{
			AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
				Reclaim: true,
			},
			SlotID: carbonv1.HippoSlotID{
				SlotID:       172507,
				SlaveAddress: "11.22.24.235:7007",
			},
		},
	}
	workerNode.Labels = map[string]string{}
	workerNode.Labels[carbonv1.LabelKeyRoleName] = roleName
	timeNow := time.Now()
	pod := corev1.Pod{
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyOnFailure,
		},
		Status: corev1.PodStatus{
			Phase:  corev1.PodRunning,
			HostIP: "1.1.1.1",
			ContainerStatuses: []v1.ContainerStatus{
				{
					Name: "pause",
				},
				{
					Name:         "c2",
					RestartCount: 0,
					State: v1.ContainerState{
						Terminated: nil,
						Running: &corev1.ContainerStateRunning{
							StartedAt: metav1.NewTime(timeNow),
						},
					},
					Ready: true,
				},
				{
					Name:         "c2-restarting",
					RestartCount: 1,
					State: v1.ContainerState{
						Terminated: nil,
						Running: &corev1.ContainerStateRunning{
							StartedAt: metav1.NewTime(timeNow),
						},
					},
					Ready: false,
				},
				{
					Name:         "c2-terminated",
					RestartCount: 10,
					State: v1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode:  -1,
							StartedAt: metav1.NewTime(timeNow),
						},
					},
				},
				{
					Name:         "c2-unknown",
					RestartCount: 10,
					State: v1.ContainerState{
						Terminated: nil,
						Running:    nil,
					},
				},
			},
		},
	}
	pod.Annotations = map[string]string{}
	pod.Annotations[carbonv1.AnnotationKeySpecExtend] = "{\"containers\":[{\"name\":\"c2\",\"instanceId\":\"a45da933\"}],\"cpusetMode\":\"RESERVED\",\"containerModel\":\"DOCKER\",\"instanceId\":\"5465b825\"}"
	pod.CreationTimestamp.Time = time.Unix(1585553142, 0)
	slotInfo := &carbon.SlotInfo{
		Role:       utils.StringPtr(roleName),
		Reclaiming: utils.BoolPtr(true),
		SlotId: &hippo.SlotId{
			SlaveAddress: utils.StringPtr("11.22.24.235:7007"),
			Id:           utils.Int32Ptr(172507),
			DeclareTime:  utils.Int64Ptr(1585553142000000),
		},
		ProcessStatus: ([]*hippo.ProcessStatus{
			{
				IsDaemon: utils.BoolPtr(false),
				Status: func() *hippo.ProcessStatus_Status {
					a := hippo.ProcessStatus_PS_RUNNING
					return &a
				}(),
				ProcessName:  utils.StringPtr("c2"),
				StartTime:    utils.Int64Ptr(timeNow.UnixNano() / 1000),
				RestartCount: utils.Int32Ptr(0),
			},
			{
				IsDaemon: utils.BoolPtr(false),
				Status: func() *hippo.ProcessStatus_Status {
					a := hippo.ProcessStatus_PS_RESTARTING
					return &a
				}(),
				ProcessName:  utils.StringPtr("c2-restarting"),
				StartTime:    utils.Int64Ptr(timeNow.UnixNano() / 1000),
				RestartCount: utils.Int32Ptr(1),
			},
			{
				IsDaemon: utils.BoolPtr(false),
				Status: func() *hippo.ProcessStatus_Status {
					a := hippo.ProcessStatus_PS_TERMINATED
					return &a
				}(),
				ProcessName:  utils.StringPtr("c2-terminated"),
				StartTime:    utils.Int64Ptr(timeNow.UnixNano() / 1000),
				RestartCount: utils.Int32Ptr(10),
				ExitCode:     utils.Int32Ptr(-1),
			},
			{
				IsDaemon: utils.BoolPtr(false),
				Status: func() *hippo.ProcessStatus_Status {
					a := hippo.ProcessStatus_PS_UNKNOWN
					return &a
				}(),
				ProcessName:  utils.StringPtr("c2-unknown"),
				RestartCount: utils.Int32Ptr(10),
			},
		}),
		PackageStatus: &hippo.PackageStatus{
			Status: func() *hippo.PackageStatus_Status {
				a := hippo.PackageStatus_IS_INSTALLED
				return &a
			}(),
		},
		SlaveStatus: &hippo.SlaveStatus{
			Status: func() *hippo.SlaveStatus_Status {
				a := hippo.SlaveStatus_ALIVE
				return &a
			}(),
		},
	}
	type args struct {
		worker *carbonv1.WorkerNode
		pod    *corev1.Pod
	}
	tests := []struct {
		name           string
		args           args
		want           *carbon.SlotInfo
		wantRestarting bool
		wantErr        bool
	}{
		{
			name: "normal test",
			args: args{
				worker: &workerNode,
				pod:    &pod,
			},
			want:           slotInfo,
			wantRestarting: true,
			wantErr:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotRestarting, err := TransSlotInfo(tt.args.worker, tt.args.pod)
			if (err != nil) != tt.wantErr {
				t.Errorf("TransSlotInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TransSlotInfo() = \n%s, want \n%s", utils.ObjJSON(got), utils.ObjJSON(tt.want))
			}
			if !reflect.DeepEqual(gotRestarting, tt.wantRestarting) {
				t.Errorf("TransSlotInfo() = %t, want %t", gotRestarting, tt.wantRestarting)
			}
		})
	}
}

func TestTransHealthInfo(t *testing.T) {
	type args struct {
		worker *carbonv1.WorkerNode
	}
	tests := []struct {
		name string
		args args

		wantErr        bool
		wantCustominfo string
	}{
		{
			name: "test1",
			args: args{worker: &carbonv1.WorkerNode{Status: carbonv1.WorkerNodeStatus{HealthCondition: carbonv1.HealthCondition{
				Metas: map[string]string{"CustomInfo": "11111"},
			}}}},

			wantCustominfo: "11111",
		},

		{
			name: "test2",
			args: args{worker: &carbonv1.WorkerNode{Status: carbonv1.WorkerNodeStatus{HealthCondition: carbonv1.HealthCondition{
				Metas:           map[string]string{"CustomInfo": "11111"},
				CompressedMetas: "invaild",
			}}}},

			wantCustominfo: "11111",
		},
		{
			name: "test3",
			args: args{worker: &carbonv1.WorkerNode{Status: carbonv1.WorkerNodeStatus{HealthCondition: carbonv1.HealthCondition{
				CompressedMetas: "H4sIAAAAAAAA/6pWci4tLsnP9cxLy1eyUkoGAaVaAAAAAP//",
			}}}},

			wantCustominfo: "ccccc",
		},
		{
			name: "test4",
			args: args{worker: &carbonv1.WorkerNode{Status: carbonv1.WorkerNodeStatus{HealthCondition: carbonv1.HealthCondition{
				Metas:           map[string]string{"CustomInfo": "11111"},
				CompressedMetas: "H4sIAAAAAAAA/6pWci4tLsnP9cxLy1eyUkoGAaVaAAAAAP//",
			}}}},

			wantCustominfo: "11111",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TransHealthInfo(tt.args.worker)
			if (err != nil) != tt.wantErr {
				t.Errorf("TransHealthInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got.Metas["CustomInfo"] != tt.wantCustominfo {
				t.Errorf("TransHealthInfo() CustomInfo = %v, want %v", got, tt.wantCustominfo)
			}
		})
	}
}

func TestTransRoleStatusSimple(t *testing.T) {
	assert := assert.New(t)
	rs := &carbonv1.RollingSet{
		Spec: carbonv1.RollingSetSpec{
			SchedulePlan: rollalgorithm.SchedulePlan{
				Replicas:           utils.Int32Ptr(11),
				LatestVersionRatio: utils.Int32Ptr(79),
			},
			Version: "123",
		},
	}
	gid := "g1"
	roleStatus, err := TransRoleStatus(gid, rs, []ReplicaNodes{})
	assert.Nil(err)
	assert.Equal(int32(79), *roleStatus.GlobalPlan.LatestVersionRatio)
	assert.Equal(int32(11), *roleStatus.GlobalPlan.Count)
	assert.Equal("123", *roleStatus.LatestVersion)
}

func initMockConfiger(ctrl *gomock.Controller, configStr string) {
	mockConfiger := config.NewMockConfiger(ctrl)
	common.SetGlobalConfiger(mockConfiger, "")
	mockConfiger.EXPECT().GetString(common.ConfProxyCopyLabels).Return(configStr, nil).AnyTimes()
}

func TestTransWorkerNodeStatusLabels(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	type args struct {
		worker          *carbonv1.WorkerNode
		pod             *corev1.Pod
		globalConfig    string
		enablePodStatus bool
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
	}{
		{
			name: "with deletion-cost",
			args: args{
				worker: &carbonv1.WorkerNode{
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							EntityName:   "pod1",
							DeletionCost: 11,
						},
					},
				},
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels:      map[string]string{},
						Annotations: map[string]string{},
					},
				},
			},
			want: map[string]string{
				"namespace":                   "",
				"app.c2.io/pod-deletion-cost": "11",
				"app.c2.io/pod-name":          "pod1",
			},
			wantErr: false,
		}, {
			name: "without deletion-cost",
			args: args{
				worker: &carbonv1.WorkerNode{
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							EntityName: "pod1",
						},
					},
				},
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							carbonv1.LabelKeyPodSN:       "sn-123",
							carbonv1.LabelKeyPreInactive: "false",
						},
						Annotations: map[string]string{},
					},
				},
			},
			want: map[string]string{
				"namespace":                  "",
				"app.c2.io/pod-name":         "pod1",
				carbonv1.LabelKeyPodSN:       "sn-123",
				carbonv1.LabelKeyPreInactive: "false",
			},
			wantErr: false,
		},
		{
			name: "nil pod",
			args: args{
				worker: &carbonv1.WorkerNode{
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{},
					},
				},
				pod: nil,
			},
			want: map[string]string{
				"namespace":          "",
				"app.c2.io/pod-name": "",
			},
			wantErr: false,
		}, {
			name: "test global config",
			args: args{
				globalConfig: "{\"workerLabels\":{\"app.c2.io\":[\"app.hippo.io/pod-version\"]},\"workerAnnotations\":{\"_direct\":[\"app.c2.io/c2-declared-keys\"]},\"podLabels\":{\"app.c2.io\":[\"rs-version-hash\"]},\"podAnnotations\":{\"app.c2.io\":[\"kruise.io/sidecarset-injected-list\"]}}",
				worker: &carbonv1.WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app.hippo.io/pod-version": "v3.0",
						},
						Annotations: map[string]string{
							"app.c2.io/c2-declared-keys": "volumeKeys:xxx",
						},
					},
				},
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"rs-version-hash": "rollingset",
						},
						Annotations: map[string]string{
							"kruise.io/sidecarset-injected-list": "staragent-sidecarset",
						},
					},
				},
			},
			want: map[string]string{
				"app.c2.io/pod-version":              "v3.0",
				"app.c2.io/c2-declared-keys":         "volumeKeys:xxx",
				"app.c2.io/rs-version-hash":          "rollingset",
				"app.c2.io/sidecarset-injected-list": "staragent-sidecarset",
				"namespace":                          "",
				"app.c2.io/pod-name":                 "",
			},
			wantErr: false,
		},
		{
			name: "test global config missing",
			args: args{
				globalConfig: "{\"workerAnnotations\":{\"_direct\":[\"app.c2.io/c2-declared-keys\"]},\"podLabels\":{\"app.c2.io\":[\"rs-version-hash\"]},\"podAnnotations\":{\"app.c2.io\":[\"kruise.io/sidecarset-injected-list\"]}}",
				worker: &carbonv1.WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app.hippo.io/pod-version": "v3.0",
						},
					},
				},
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"rs-version-hash": "",
						},
						Annotations: map[string]string{
							"kruise.io/sidecarset-injected-list": "staragent-sidecarset",
						},
					},
				},
			},
			want: map[string]string{
				"app.c2.io/sidecarset-injected-list": "staragent-sidecarset",
				"namespace":                          "",
				"app.c2.io/pod-name":                 "",
			},
			wantErr: false,
		},
		{
			name: "with container status",
			args: args{
				worker: &carbonv1.WorkerNode{
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							EntityName:   "pod1",
							DeletionCost: 10001,
						},
					},
				},
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels:      map[string]string{},
						Annotations: map[string]string{},
					},
					Status: v1.PodStatus{
						ContainerStatuses: []v1.ContainerStatus{{
							Name:         "main",
							RestartCount: 2,
							Started:      utils.BoolPtr(false),
							Ready:        false,
						}},
					},
				},
				enablePodStatus: true,
			},
			want: map[string]string{
				"namespace":                    "",
				"app.c2.io/pod-deletion-cost":  "10001",
				"app.c2.io/pod-name":           "pod1",
				"app.c2.io/container-statuses": "[{\"name\":\"main\",\"ready\":false,\"restartCount\":2,\"started\":false}]",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.enablePodStatus {
				features.C2MutableFeatureGate.Set("EnableTransContainers=true")
				defer features.C2MutableFeatureGate.Set("EnableTransContainers=false")
			}
			if tt.args.globalConfig != "" {
				initMockConfiger(ctl, tt.args.globalConfig)
			}
			got, err := TransWorkerNodeStatusLabels(tt.args.worker, tt.args.pod)
			if (err != nil) != tt.wantErr {
				t.Errorf("TransWorkerNodeStatusLabels() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TransWorkerNodeStatusLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}
