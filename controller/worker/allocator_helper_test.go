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
	"reflect"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_basePodStatusParser_getPodIP(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name       string
		p          *basePodStatusParser
		args       args
		wantHostIP string
		wantPodIP  string
	}{
		{
			name: "get",
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						NodeName: "10.83.178.64-7707",
					},
					Status: corev1.PodStatus{},
				},
			},
			wantHostIP: "",
		},
		{
			name: "get",
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						NodeName:    "10.83.178.64-7707",
						HostNetwork: true,
					},
					Status: corev1.PodStatus{
						HostIP: "1.1.1.1",
						PodIP:  "1.1.1.2",
					},
				},
			},
			wantHostIP: "1.1.1.1",
			wantPodIP:  "1.1.1.2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &basePodStatusParser{}
			gotHostIP, gotPodIP := p.getPodIP(tt.args.pod)
			if gotHostIP != tt.wantHostIP {
				t.Errorf("basePodStatusParser.getPodIP() gotHostIP = %v, want %v", gotHostIP, tt.wantHostIP)
			}
			if gotPodIP != tt.wantPodIP {
				t.Errorf("basePodStatusParser.getPodIP() gotPodIP = %v, want %v", gotPodIP, tt.wantPodIP)
			}
		})
	}
}

func Test_basePodStatusParser_isPodReclaimed(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name string
		p    *basePodStatusParser
		args args
		want bool
	}{
		{
			name: "evicted",
			args: args{
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase:  corev1.PodFailed,
						Reason: "Evicted",
					},
				},
			},
			want: true,
		},
		{
			name: "not evicted",
			args: args{
				pod: &corev1.Pod{
					Status: corev1.PodStatus{
						Phase:  corev1.PodFailed,
						Reason: "none",
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &basePodStatusParser{}
			if got := p.isPodReclaimed(tt.args.pod); got != tt.want {
				t.Errorf("basePodStatusParser.isPodReclaimed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_basePodSpecSyncer_computeProcessVersion(t *testing.T) {
	type args struct {
		worker       *carbonv1.WorkerNode
		hippoPodSpec *carbonv1.HippoPodSpec
		currentPod   *corev1.Pod
	}
	res := resource.NewQuantity(int64(1), resource.DecimalExponent)
	res1 := resource.NewQuantity(int64(2), resource.DecimalExponent)
	tests := []struct {
		name    string
		p       *basePodSpecSyncer
		args    args
		want    string
		want1   map[string]string
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version: "1",
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								RestartAfterResourceChange: utils.BoolPtr(true),
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Version: "2",
						},
					},
				},
				hippoPodSpec: &carbonv1.HippoPodSpec{
					HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
						ContainerModel: "",
					},
					Containers: []carbonv1.HippoContainer{
						{
							Container: corev1.Container{
								Name:  "container",
								Image: "image",
								Resources: corev1.ResourceRequirements{
									Limits: map[corev1.ResourceName]resource.Quantity{
										carbonv1.ResourceIP: *res,
										"cpu":               *res,
									},
								},
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
						{
							Container: corev1.Container{
								Name:  "container1",
								Image: "image1",
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
					},
				},
			},
			want:    "5465b825",
			want1:   map[string]string{"container": "ac626c53", "container1": "9ed3e5e1"},
			wantErr: false,
		},
		{
			name: "resource change",
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version: "1",
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								RestartAfterResourceChange: utils.BoolPtr(true),
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Version: "2",
						},
					},
				},
				hippoPodSpec: &carbonv1.HippoPodSpec{
					HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
						ContainerModel: "",
					},
					Containers: []carbonv1.HippoContainer{
						{
							Container: corev1.Container{
								Name:  "container",
								Image: "image",
								Resources: corev1.ResourceRequirements{
									Limits: map[corev1.ResourceName]resource.Quantity{
										carbonv1.ResourceIP: *res,
										"cpu":               *res1,
									},
								},
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
						{
							Container: corev1.Container{
								Name:  "container1",
								Image: "image1",
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
					},
				},
			},
			want:    "5465b825",
			want1:   map[string]string{"container": "27061fb4", "container1": "8e425a86"},
			wantErr: false,
		},
		{
			name: "RestartAfterResourceChange",
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version: "1",
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								RestartAfterResourceChange: utils.BoolPtr(false),
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Version: "2",
						},
					},
				},
				hippoPodSpec: &carbonv1.HippoPodSpec{
					HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
						ContainerModel: "",
					},
					Containers: []carbonv1.HippoContainer{
						{
							Container: corev1.Container{
								Name:  "container",
								Image: "image",
								Resources: corev1.ResourceRequirements{
									Limits: map[corev1.ResourceName]resource.Quantity{
										carbonv1.ResourceIP: *res,
										"cpu":               *res1,
									},
								},
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
						{
							Container: corev1.Container{
								Name:  "container1",
								Image: "image1",
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
					},
				},
			},
			want:    "5465b825",
			want1:   map[string]string{"container": "341583ee", "container1": "3f36d131"},
			wantErr: false,
		},
		{
			name: "RestartAfterResourceChange",
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version: "1",
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								RestartAfterResourceChange: utils.BoolPtr(false),
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Version: "2",
						},
					},
				},
				hippoPodSpec: &carbonv1.HippoPodSpec{
					HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
						ContainerModel: "",
					},
					Containers: []carbonv1.HippoContainer{
						{
							Container: corev1.Container{
								Name:  "container",
								Image: "image",
								Resources: corev1.ResourceRequirements{
									Limits: map[corev1.ResourceName]resource.Quantity{
										carbonv1.ResourceIP: *res,
										"cpu":               *res1,
									},
								},
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
						{
							Container: corev1.Container{
								Name:  "container1",
								Image: "image1",
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
					},
				},
			},
			want:    "5465b825",
			want1:   map[string]string{"container": "341583ee", "container1": "3f36d131"},
			wantErr: false,
		},
		{
			name: "RestartAfterResourceChange disratio not exist",
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version: "1",
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								RestartAfterResourceChange: utils.BoolPtr(true),
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Version: "2",
						},
					},
				},
				hippoPodSpec: &carbonv1.HippoPodSpec{
					HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
						ContainerModel: "",
					},
					Containers: []carbonv1.HippoContainer{
						{
							Container: corev1.Container{
								Name:  "container",
								Image: "image",
								Resources: corev1.ResourceRequirements{
									Limits: map[corev1.ResourceName]resource.Quantity{
										carbonv1.ResourceIP: *res,
										"cpu":               *res,
									},
								},
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
						{
							Container: corev1.Container{
								Name:  "container1",
								Image: "image1",
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
					},
				},
			},
			want:    "5465b825",
			want1:   map[string]string{"container": "ac626c53", "container1": "9ed3e5e1"},
			wantErr: false,
		},
		{
			name: "RestartAfterResourceChange disratio exist",
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version: "1",
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								RestartAfterResourceChange: utils.BoolPtr(true),
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Version: "2",
						},
					},
				},
				hippoPodSpec: &carbonv1.HippoPodSpec{
					HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
						ContainerModel: "",
					},
					Containers: []carbonv1.HippoContainer{
						{
							Container: corev1.Container{
								Name:  "container",
								Image: "image",
								Resources: corev1.ResourceRequirements{
									Limits: map[corev1.ResourceName]resource.Quantity{
										carbonv1.ResourceIP:        *res,
										"cpu":                      *res,
										carbonv1.ResourceDiskRatio: *res,
									},
								},
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
						{
							Container: corev1.Container{
								Name:  "container1",
								Image: "image1",
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
					},
				},
			},
			want:    "5465b825",
			want1:   map[string]string{"container": "ac626c53", "container1": "9ed3e5e1"},
			wantErr: false,
		},
		{
			name: "RestartAfterResourceChange VM",
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version: "1",
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								RestartAfterResourceChange: utils.BoolPtr(false),
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Version: "2",
						},
					},
				},
				hippoPodSpec: &carbonv1.HippoPodSpec{
					HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
						ContainerModel: "VM",
					},
					Containers: []carbonv1.HippoContainer{
						{
							Container: corev1.Container{
								Name:  "container",
								Image: "image",
								Resources: corev1.ResourceRequirements{
									Limits: map[corev1.ResourceName]resource.Quantity{
										carbonv1.ResourceIP: *res,
										"cpu":               *res,
									},
								},
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
						{
							Container: corev1.Container{
								Name:  "container1",
								Image: "image1",
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
					},
				},
			},
			want:    "f7f6c3bc",
			want1:   map[string]string{"container": "341583ee", "container1": "3f36d131"},
			wantErr: false,
		},
		{
			name: "RestartAfterResourceChange VM",
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version: "1",
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								RestartAfterResourceChange: utils.BoolPtr(false),
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Version: "2",
						},
					},
				},
				hippoPodSpec: &carbonv1.HippoPodSpec{
					HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
						ContainerModel: "VM",
					},
					Containers: []carbonv1.HippoContainer{
						{
							Container: corev1.Container{
								Name:  "container",
								Image: "image",
								Resources: corev1.ResourceRequirements{
									Limits: map[corev1.ResourceName]resource.Quantity{
										carbonv1.ResourceIP: *res,
										"cpu":               *res1,
									},
								},
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
						{
							Container: corev1.Container{
								Name:  "container1",
								Image: "image1",
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
					},
				},
			},
			want:    "f7f6c3bc",
			want1:   map[string]string{"container": "341583ee", "container1": "3f36d131"},
			wantErr: false,
		},
		{
			name: "image change",
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						Version: "1",
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								RestartAfterResourceChange: utils.BoolPtr(true),
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Version: "2",
						},
					},
				},
				hippoPodSpec: &carbonv1.HippoPodSpec{
					HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
						ContainerModel: "",
					},
					Containers: []carbonv1.HippoContainer{
						{
							Container: corev1.Container{
								Name:  "container",
								Image: "image1",
								Resources: corev1.ResourceRequirements{
									Limits: map[corev1.ResourceName]resource.Quantity{
										carbonv1.ResourceIP: *res,
										"cpu":               *res,
									},
								},
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
						{
							Container: corev1.Container{
								Name:  "container1",
								Image: "image2",
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
					},
				},
			},
			want:    "5465b825",
			want1:   map[string]string{"container": "97a288e8", "container1": "5f14f654"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &basePodSpecSyncer{}
			got, got1, err := p.computeProcessVersion(tt.args.worker, tt.args.hippoPodSpec, tt.args.currentPod)
			if (err != nil) != tt.wantErr {
				t.Errorf("basePodSpecSyncer.computeProcessVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("basePodSpecSyncer.computeProcessVersion() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("basePodSpecSyncer.computeProcessVersion() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_basePodSpecSyncer_updateProcess(t *testing.T) {
	type args struct {
		t *target
	}
	res := resource.NewQuantity(int64(1), resource.DecimalExponent)
	tests := []struct {
		name    string
		p       *basePodSpecSyncer
		args    args
		wantErr bool
		want    *corev1.PodSpec
	}{
		{
			name: "update",
			args: args{
				t: &target{
					targetPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
					podSpec: &corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Resources: corev1.ResourceRequirements{
									Limits: map[corev1.ResourceName]resource.Quantity{
										"cpu": *res,
									},
								},
							},
						},
					},
					targetResourceVersion: "aaa",
				},
			},
			want: &corev1.PodSpec{
				NodeName: "node1",
				Priority: utils.Int32Ptr(1),
				Containers: []corev1.Container{
					{
						Resources: corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								"cpu": *res,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "update",
			args: args{
				t: &target{
					targetPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
					podSpec: &corev1.PodSpec{
						HostNetwork: true,
						Containers: []corev1.Container{
							{
								Resources: corev1.ResourceRequirements{
									Limits: map[corev1.ResourceName]resource.Quantity{
										carbonv1.ResourceIP: *res,
										"cpu":               *res,
									},
								},
							},
						},
					},
					targetResourceVersion: "aaa",
				},
			},
			want: &corev1.PodSpec{
				NodeName:    "node1",
				Priority:    utils.Int32Ptr(1),
				HostNetwork: true,
				Containers: []corev1.Container{
					{
						Resources: corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								carbonv1.ResourceIP: *res,
								"cpu":               *res,
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
			p := &basePodSpecSyncer{}
			err := p.updateProcess(tt.args.t)
			if (err != nil) != tt.wantErr {
				t.Errorf("basePodSpecSyncer.updateProcess() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(&tt.args.t.targetPod.Spec, tt.want) {
				t.Errorf("basePodSpecSyncer.updateProcess() got = %v, want %v", utils.ObjJSON(tt.args.t.targetPod.Spec), utils.ObjJSON(tt.want))
			}
		})
	}
}

func Test_basePodSpecSyncer_preSync(t *testing.T) {
	type args struct {
		worker       *carbonv1.WorkerNode
		hippoPodSpec *carbonv1.HippoPodSpec
		labels       map[string]string
	}
	res := resource.NewQuantity(int64(1), resource.DecimalExponent)
	tests := []struct {
		name       string
		p          *basePodSpecSyncer
		args       args
		wantErr    bool
		wantLabels map[string]string
		want       *carbonv1.HippoPodSpec
	}{
		{
			name: "without ip",
			args: args{
				labels: map[string]string{},
				worker: &carbonv1.WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							carbonv1.LabelKeyExlusive: "port-tcp-12745.port-tcp-12746.exclusive#tag",
						},
					},
				},
				hippoPodSpec: &carbonv1.HippoPodSpec{
					HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
						ContainerModel: "",
					},
					Containers: []carbonv1.HippoContainer{
						{
							Container: corev1.Container{
								Name:  "container",
								Image: "image",
								Resources: corev1.ResourceRequirements{
									Limits: map[corev1.ResourceName]resource.Quantity{
										"cpu": *res,
									},
								},
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
						{
							Container: corev1.Container{
								Name:  "container1",
								Image: "image1",
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
					},
				},
			},
			wantErr:    false,
			wantLabels: map[string]string{"app.hippo.io/exclusive-labels": "7d45d0f945b06903092f4bd128631441"},
			want: &carbonv1.HippoPodSpec{
				PodSpec: corev1.PodSpec{
					HostNetwork: true,
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
								{
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "app.hippo.io/exclusive-labels",
												Operator: metav1.LabelSelectorOpIn,
												Values: []string{
													"7d45d0f945b06903092f4bd128631441",
												},
											},
										},
									},
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
				},
				HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
					ContainerModel: "",
				},
				Containers: []carbonv1.HippoContainer{
					{
						Container: corev1.Container{
							Name:  "container",
							Image: "image",
							Resources: corev1.ResourceRequirements{
								Limits: map[corev1.ResourceName]resource.Quantity{
									"cpu": *res,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 12745,
									HostPort:      12745,
									Name:          "port-tcp-12745",
								},
								{
									ContainerPort: 12746,
									HostPort:      12746,
									Name:          "port-tcp-12746",
								},
							},
						},
						ContainerHippoExterned: carbonv1.ContainerHippoExterned{
							PreDeployImage: "preimage",
						},
					},
					{
						Container: corev1.Container{
							Name:  "container1",
							Image: "image1",
						},
						ContainerHippoExterned: carbonv1.ContainerHippoExterned{
							PreDeployImage: "preimage",
						},
					},
				},
			},
		},
		{
			name: "without ip",
			args: args{
				labels: map[string]string{},
				worker: &carbonv1.WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							carbonv1.LabelKeyExlusive: "port-tcp-12745.port-tcp-12746.exclusive-tag",
						},
					},
				},
				hippoPodSpec: &carbonv1.HippoPodSpec{
					HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
						ContainerModel: "",
					},
					Containers: []carbonv1.HippoContainer{
						{
							Container: corev1.Container{
								Name:  "container",
								Image: "image",
								Resources: corev1.ResourceRequirements{
									Limits: map[corev1.ResourceName]resource.Quantity{
										"cpu": *res,
									},
								},
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
						{
							Container: corev1.Container{
								Name:  "container1",
								Image: "image1",
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
					},
				},
			},
			wantErr:    false,
			wantLabels: map[string]string{"app.hippo.io/exclusive-labels": "exclusive-tag"},
			want: &carbonv1.HippoPodSpec{
				PodSpec: corev1.PodSpec{
					HostNetwork: true,
					Affinity: &corev1.Affinity{
						PodAntiAffinity: &corev1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
								{
									LabelSelector: &metav1.LabelSelector{
										MatchExpressions: []metav1.LabelSelectorRequirement{
											{
												Key:      "app.hippo.io/exclusive-labels",
												Operator: metav1.LabelSelectorOpIn,
												Values: []string{
													"exclusive-tag",
												},
											},
										},
									},
									TopologyKey: "kubernetes.io/hostname",
								},
							},
						},
					},
				},
				HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
					ContainerModel: "",
				},
				Containers: []carbonv1.HippoContainer{
					{
						Container: corev1.Container{
							Name:  "container",
							Image: "image",
							Resources: corev1.ResourceRequirements{
								Limits: map[corev1.ResourceName]resource.Quantity{
									"cpu": *res,
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 12745,
									HostPort:      12745,
									Name:          "port-tcp-12745",
								},
								{
									ContainerPort: 12746,
									HostPort:      12746,
									Name:          "port-tcp-12746",
								},
							},
						},
						ContainerHippoExterned: carbonv1.ContainerHippoExterned{
							PreDeployImage: "preimage",
						},
					},
					{
						Container: corev1.Container{
							Name:  "container1",
							Image: "image1",
						},
						ContainerHippoExterned: carbonv1.ContainerHippoExterned{
							PreDeployImage: "preimage",
						},
					},
				},
			},
		},
		{
			name: "with ip",
			args: args{
				worker: &carbonv1.WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							carbonv1.LabelKeyExlusive: "port-tcp-12745.port-tcp-12746.port-tcp-12747",
						},
					},
				},
				hippoPodSpec: &carbonv1.HippoPodSpec{
					HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
						ContainerModel: "",
					},
					Containers: []carbonv1.HippoContainer{
						{
							Container: corev1.Container{
								Name:  "container",
								Image: "image",
								Resources: corev1.ResourceRequirements{
									Limits: map[corev1.ResourceName]resource.Quantity{
										carbonv1.ResourceIP: *res,
										"cpu":               *res,
									},
								},
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
						{
							Container: corev1.Container{
								Name:  "container1",
								Image: "image1",
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
					},
				},
			},
			wantErr: false,
			want: &carbonv1.HippoPodSpec{
				PodSpec: corev1.PodSpec{
					HostNetwork: false,
				},
				HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
					ContainerModel: "",
				},
				Containers: []carbonv1.HippoContainer{
					{
						Container: corev1.Container{
							Name:  "container",
							Image: "image",
							Resources: corev1.ResourceRequirements{
								Limits: map[corev1.ResourceName]resource.Quantity{
									carbonv1.ResourceIP: *res,
									"cpu":               *res,
								},
							},
						},
						ContainerHippoExterned: carbonv1.ContainerHippoExterned{
							PreDeployImage: "preimage",
						},
					},
					{
						Container: corev1.Container{
							Name:  "container1",
							Image: "image1",
						},
						ContainerHippoExterned: carbonv1.ContainerHippoExterned{
							PreDeployImage: "preimage",
						},
					},
				},
			},
		},
		{
			name: "with ip",
			args: args{
				worker: &carbonv1.WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							carbonv1.LabelKeyExlusive: "port-tcp-12745.port-tcp-12746.port-tcp-12747",
						},
					},
				},
				hippoPodSpec: &carbonv1.HippoPodSpec{
					HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
						ContainerModel: "",
					},
					Containers: []carbonv1.HippoContainer{
						{
							Container: corev1.Container{
								Name:  "container",
								Image: "image",
								Resources: corev1.ResourceRequirements{
									Limits: map[corev1.ResourceName]resource.Quantity{
										"cpu": *res,
									},
								},
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
						{
							Container: corev1.Container{
								Name:  "container1",
								Image: "image1",
							},
							ContainerHippoExterned: carbonv1.ContainerHippoExterned{
								PreDeployImage: "preimage",
							},
						},
					},
				},
				labels: map[string]string{
					carbonv1.LabelKeyPodVersion: carbonv1.PodVersion3,
				},
			},
			wantErr: false,
			want: &carbonv1.HippoPodSpec{
				PodSpec: corev1.PodSpec{
					HostNetwork: false,
				},
				HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
					ContainerModel: "",
				},
				Containers: []carbonv1.HippoContainer{
					{
						Container: corev1.Container{
							Name:  "container",
							Image: "image",
							Resources: corev1.ResourceRequirements{
								Limits: map[corev1.ResourceName]resource.Quantity{
									"cpu": *res,
								},
							},
						},
						ContainerHippoExterned: carbonv1.ContainerHippoExterned{
							PreDeployImage: "preimage",
						},
					},
					{
						Container: corev1.Container{
							Name:  "container1",
							Image: "image1",
						},
						ContainerHippoExterned: carbonv1.ContainerHippoExterned{
							PreDeployImage: "preimage",
						},
					},
				},
			},
			wantLabels: map[string]string{
				carbonv1.LabelKeyPodVersion: carbonv1.PodVersion3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &basePodSpecSyncer{}
			err := p.preSync(tt.args.worker, tt.args.hippoPodSpec, tt.args.labels)
			if (err != nil) != tt.wantErr {
				t.Errorf("basePodSpecSyncer.preSync() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.hippoPodSpec, tt.want) {
				t.Errorf("basePodSpecSyncer.preSync() got = %v, want %v", utils.ObjJSON(tt.args.hippoPodSpec), utils.ObjJSON(tt.want))
			}
			if !reflect.DeepEqual(tt.args.labels, tt.wantLabels) {
				t.Errorf("basePodSpecSyncer.preSync() got = %v, want %v", utils.ObjJSON(tt.args.labels), utils.ObjJSON(tt.wantLabels))
			}
		})
	}
}
