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

package rollingset

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/alibaba/kube-sharding/common/features"

	mapset "github.com/deckarep/golang-set"

	"github.com/alibaba/kube-sharding/controller/util"
	mock "github.com/alibaba/kube-sharding/controller/util/mock"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/rollalgorithm"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/uuid"
	k8scontroller "k8s.io/kubernetes/pkg/controller"
)

func newRollingSet(version string) *carbonv1.RollingSet {
	selector := map[string]string{"foo": "bar"}
	metaStr := `{"metadata":{"name":"nginx-deployment","labels":{"app": "nginx"}}}`
	var template carbonv1.HippoPodTemplate
	json.Unmarshal([]byte(metaStr), &template)
	rollingset := carbonv1.RollingSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        "rollingset-m",
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
		Spec: carbonv1.RollingSetSpec{
			Version: version,
			SchedulePlan: rollalgorithm.SchedulePlan{
				Strategy: apps.DeploymentStrategy{
					Type: apps.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &apps.RollingUpdateDeployment{
						MaxUnavailable: func() *intstr.IntOrString { i := intstr.FromInt(2); return &i }(),
						MaxSurge:       func() *intstr.IntOrString { i := intstr.FromInt(2); return &i }(),
					},
				},
				Replicas:           func() *int32 { i := int32(10); return &i }(),
				LatestVersionRatio: func() *int32 { i := int32(rollalgorithm.DefaultLatestVersionRatio); return &i }(),
			},

			VersionPlan: carbonv1.VersionPlan{
				BroadcastPlan: carbonv1.BroadcastPlan{
					WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
						MinReadySeconds:      1024,
						ResourceMatchTimeout: 2000,
						WorkerReadyTimeout:   2014,
					},
				},
				SignedVersionPlan: carbonv1.SignedVersionPlan{
					Template:          &template,
					ShardGroupVersion: "latest-group",
				},
			},
			Selector: &metav1.LabelSelector{MatchLabels: selector},
		},
		Status: carbonv1.RollingSetStatus{},
	}
	return &rollingset
}

func newFieldRollingSetBySchedule(name string, replicas int, maxSurge int, maxUnavailable int, latestVersionRatio int) *carbonv1.RollingSet {
	selector := map[string]string{"foo": "bar"}
	rollingset := carbonv1.RollingSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
		Spec: carbonv1.RollingSetSpec{
			Version: "version-new",
			SchedulePlan: rollalgorithm.SchedulePlan{
				Strategy: apps.DeploymentStrategy{
					Type: apps.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &apps.RollingUpdateDeployment{
						MaxUnavailable: func() *intstr.IntOrString { i := intstr.FromInt(maxUnavailable); return &i }(),
						MaxSurge:       func() *intstr.IntOrString { i := intstr.FromInt(maxSurge); return &i }(),
					},
				},
				Replicas:           func() *int32 { i := int32(replicas); return &i }(),
				LatestVersionRatio: func() *int32 { i := int32(latestVersionRatio); return &i }(),
			},
			Selector: &metav1.LabelSelector{MatchLabels: selector},
		},
		Status: carbonv1.RollingSetStatus{},
	}
	return &rollingset
}

func newFieldRollingSetBySchedule2(name string, replicas int, maxSurge int, maxUnavailable int) *carbonv1.RollingSet {
	selector := map[string]string{"foo": "bar"}
	rollingset := carbonv1.RollingSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
		Spec: carbonv1.RollingSetSpec{
			Version: "version-new",
			SchedulePlan: rollalgorithm.SchedulePlan{
				Strategy: apps.DeploymentStrategy{
					Type: apps.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &apps.RollingUpdateDeployment{
						MaxUnavailable: func() *intstr.IntOrString { i := intstr.FromInt(maxUnavailable); return &i }(),
						MaxSurge:       func() *intstr.IntOrString { i := intstr.FromInt(maxSurge); return &i }(),
					},
				},
				Replicas: func() *int32 { i := int32(replicas); return &i }(),
			},
			Selector: &metav1.LabelSelector{MatchLabels: selector},
		},
		Status: carbonv1.RollingSetStatus{},
	}
	return &rollingset
}

func newReplicas(rollingset *carbonv1.RollingSet) []*carbonv1.Replica {
	replicas := make([]*carbonv1.Replica, 10)
	for i := 0; i < 10; i++ {
		version := rollingset.Spec.Version
		if i%4 == 0 {
			version = "version-old"
		}
		replica := &carbonv1.Replica{
			WorkerNode: carbonv1.WorkerNode{
				TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:            "replica-" + strconv.Itoa(i),
					UID:             uuid.NewUUID(),
					Namespace:       metav1.NamespaceDefault,
					Labels:          rollingset.Spec.Selector.MatchLabels,
					OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(rollingset, controllerKind)},
				},
				Spec: carbonv1.WorkerNodeSpec{
					Version:  version,
					Selector: rollingset.Spec.Selector,
				},
				Status: carbonv1.WorkerNodeStatus{
					Complete: true,
				},
			},
		}
		replicas[i] = replica
	}
	return replicas
}

func newReplicasWithVerisonPlan(complete bool) []*carbonv1.Replica {
	replicas := make([]*carbonv1.Replica, 10)
	for i := 0; i < 10; i++ {
		index := i % 4
		version := "version-" + strconv.Itoa(index)
		replica := &carbonv1.Replica{
			WorkerNode: carbonv1.WorkerNode{
				TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "replica-" + strconv.Itoa(i),
					UID:       uuid.NewUUID(),
					Namespace: metav1.NamespaceDefault,
				},
				Spec: carbonv1.WorkerNodeSpec{
					Version: version,
					VersionPlan: carbonv1.VersionPlan{
						BroadcastPlan: carbonv1.BroadcastPlan{
							WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
								MinReadySeconds:      int32(index * 1000),
								ResourceMatchTimeout: 2000,
								WorkerReadyTimeout:   int64(index * 3000),
							},
						},
						SignedVersionPlan: carbonv1.SignedVersionPlan{
							ShardGroupVersion: "group-v-" + strconv.Itoa(index),
						},
					},
				},
				Status: carbonv1.WorkerNodeStatus{
					AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{Version: version},
					Complete:              complete,
					AllocStatus:           "UNASSIGNED",
					ServiceStatus: func() carbonv1.ServiceStatus {
						if complete {
							return carbonv1.ServiceAvailable
						}
						return carbonv1.ServiceUnAvailable
					}(),
				},
			},
		}
		replicas[i] = replica
	}
	return replicas
}

func newReplicasOneVersion(version string, complete bool) []*carbonv1.Replica {
	replicas := make([]*carbonv1.Replica, 10)
	hash, _ := utils.SignatureShort(strconv.FormatInt(time.Now().UnixNano(), 10))
	for i := 0; i < 10; i++ {
		index := i % 4
		replica := &carbonv1.Replica{
			WorkerNode: carbonv1.WorkerNode{
				TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      hash + "replica-" + strconv.Itoa(i),
					UID:       uuid.NewUUID(),
					Namespace: metav1.NamespaceDefault,
				},
				Spec: carbonv1.WorkerNodeSpec{
					Version: version,
					VersionPlan: carbonv1.VersionPlan{
						BroadcastPlan: carbonv1.BroadcastPlan{
							WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
								MinReadySeconds:      int32(index * 1000),
								ResourceMatchTimeout: 2000,
								WorkerReadyTimeout:   int64(index * 3000),
							},
						},
						SignedVersionPlan: carbonv1.SignedVersionPlan{
							ShardGroupVersion: "group-v-" + strconv.Itoa(index),
						},
					},
				},
				Status: carbonv1.WorkerNodeStatus{
					Complete:     complete,
					ServiceReady: complete,
					AllocStatus:  "ASSIGNED",
					AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
						Version: version,
					},
					ServiceStatus: func() carbonv1.ServiceStatus {
						if complete {
							return carbonv1.ServiceAvailable
						}
						return carbonv1.ServiceUnAvailable
					}(),
				},
			},
		}
		replicas[i] = replica
	}
	return replicas
}

func newReplicasOneVersionWithMode(version string, complete bool, dependency bool, count int, targetWorkerMode, currentWorkerMode carbonv1.WorkerModeType, resourcePool string) []*carbonv1.Replica {
	replicas := make([]*carbonv1.Replica, count)
	for i := 0; i < count; i++ {
		index := i % 4
		replica := &carbonv1.Replica{
			WorkerNode: carbonv1.WorkerNode{
				TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "replica-" + strconv.Itoa(i),
					UID:       uuid.NewUUID(),
					Namespace: metav1.NamespaceDefault,
				},
				Spec: carbonv1.WorkerNodeSpec{
					WorkerMode:      targetWorkerMode,
					Version:         version,
					DependencyReady: utils.BoolPtr(dependency),
					VersionPlan: carbonv1.VersionPlan{
						BroadcastPlan: carbonv1.BroadcastPlan{
							WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
								MinReadySeconds:      int32(index * 1000),
								ResourceMatchTimeout: 2000,
								WorkerReadyTimeout:   int64(index * 3000),
							},
						},
						SignedVersionPlan: carbonv1.SignedVersionPlan{
							ShardGroupVersion: "group-v-" + strconv.Itoa(index),
						},
					},
				},
				Status: carbonv1.WorkerNodeStatus{
					Complete:     complete,
					ServiceReady: complete,
					AllocStatus:  "ASSIGNED",
					AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
						Version:      version,
						WorkerMode:   currentWorkerMode,
						ResourcePool: resourcePool,
					},
					ServiceStatus: func() carbonv1.ServiceStatus {
						if complete {
							return carbonv1.ServiceAvailable
						}
						return carbonv1.ServiceUnAvailable
					}(),
				},
			},
		}
		replicas[i] = replica
	}
	return replicas
}

func newReplicasOneVersionWithCout(version string, count int, complete bool) *[]*carbonv1.Replica {
	replicas := make([]*carbonv1.Replica, count)
	for i := 0; i < count; i++ {
		index := i % 4
		replica := &carbonv1.Replica{
			WorkerNode: carbonv1.WorkerNode{
				TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "replica-" + strconv.Itoa(i),
					UID:       uuid.NewUUID(),
					Namespace: metav1.NamespaceDefault,
				},
				Spec: carbonv1.WorkerNodeSpec{
					Version: version,
					VersionPlan: carbonv1.VersionPlan{
						BroadcastPlan: carbonv1.BroadcastPlan{
							WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
								MinReadySeconds:      int32(index * 1000),
								ResourceMatchTimeout: 2000,
								WorkerReadyTimeout:   int64(index * 3000),
							},
						},
						SignedVersionPlan: carbonv1.SignedVersionPlan{
							ShardGroupVersion: "group-v-" + strconv.Itoa(index),
						},
					},
				},
				Status: carbonv1.WorkerNodeStatus{
					AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
						Version: version,
					},
					Complete:    complete,
					AllocStatus: "ASSIGNED",
					ServiceStatus: func() carbonv1.ServiceStatus {
						if complete {
							return carbonv1.ServiceAvailable
						}
						return carbonv1.ServiceUnAvailable
					}(),
				},
			},
		}
		replicas[i] = replica
	}
	return &replicas
}

func newReplicasCollectLatestRedundantReplicas(version string, count int, releasing bool) []*carbonv1.Replica {
	replicas := make([]*carbonv1.Replica, count)
	for i := 0; i < count; i++ {

		replica := &carbonv1.Replica{
			WorkerNode: carbonv1.WorkerNode{
				TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "replica-" + strconv.Itoa(i),
					UID:       uuid.NewUUID(),
					Namespace: metav1.NamespaceDefault,
				},
				Spec: carbonv1.WorkerNodeSpec{
					Version: version,
					VersionPlan: carbonv1.VersionPlan{
						BroadcastPlan: carbonv1.BroadcastPlan{
							WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
								MinReadySeconds:      100,
								ResourceMatchTimeout: 2000,
								WorkerReadyTimeout:   300,
							},
						},
					},
					ToDelete: releasing,
				},
				Status: carbonv1.WorkerNodeStatus{
					AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{Version: version},
					AllocStatus:           carbonv1.WorkerAssigned,
					Complete:              true,
					Score:                 int64(i % 3),
					ServiceStatus:         carbonv1.ServiceAvailable,
				},
			},
		}
		replicas[i] = replica
	}
	return replicas
}

func newReplicasCollectLatestRedundantReplicasNotComplete(version string, count int, releasing bool) []*carbonv1.Replica {
	replicas := make([]*carbonv1.Replica, count)
	for i := 0; i < count; i++ {

		replica := &carbonv1.Replica{
			WorkerNode: carbonv1.WorkerNode{
				TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "replica-" + strconv.Itoa(i),
					UID:       uuid.NewUUID(),
					Namespace: metav1.NamespaceDefault,
				},
				Spec: carbonv1.WorkerNodeSpec{
					Version: version,
					VersionPlan: carbonv1.VersionPlan{
						BroadcastPlan: carbonv1.BroadcastPlan{
							WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
								MinReadySeconds:      100,
								ResourceMatchTimeout: 2000,
								WorkerReadyTimeout:   300,
							},
						},
					},
					ToDelete: releasing,
				},
				Status: carbonv1.WorkerNodeStatus{
					AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{Version: version},
					AllocStatus:           carbonv1.WorkerUnAssigned,
					//	Complete:    false,
					Score:    int64(i % 3),
					Complete: false,
				},
			},
		}
		replicas[i] = replica
	}
	return replicas
}

// 部分可用
func newReplicasWithVerisonPlanPartComplete() []*carbonv1.Replica {
	replicas := make([]*carbonv1.Replica, 10)
	for i := 0; i < 10; i++ {
		index := i % 2
		complete := true
		if i%4 == 0 {
			complete = false
		}
		version := "version-" + strconv.Itoa(index)
		replica := &carbonv1.Replica{
			WorkerNode: carbonv1.WorkerNode{
				TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "replica-" + strconv.Itoa(i),
					UID:       uuid.NewUUID(),
					Namespace: metav1.NamespaceDefault,
				},
				Spec: carbonv1.WorkerNodeSpec{
					Version: version,
					VersionPlan: carbonv1.VersionPlan{
						BroadcastPlan: carbonv1.BroadcastPlan{
							WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
								MinReadySeconds:      int32(index * 1000),
								ResourceMatchTimeout: 2000,
								WorkerReadyTimeout:   int64(index * 3000),
							},
						},
					},
				},
				Status: carbonv1.WorkerNodeStatus{
					AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{Version: version},
					Complete:              complete,
					ServiceStatus: func() carbonv1.ServiceStatus {
						if complete {
							return carbonv1.ServiceAvailable
						}
						return carbonv1.ServiceUnAvailable

					}(),
				},
			},
		}
		replicas[i] = replica
	}
	return replicas
}

func newFieldRollingSetByScheduleDeletion(name string, replicas int, maxSurge int, maxUnavailable int, latestVersionRatio int) *carbonv1.RollingSet {
	selector := map[string]string{"foo": "bar"}
	rollingset := carbonv1.RollingSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:               uuid.NewUUID(),
			Name:              name,
			Namespace:         metav1.NamespaceDefault,
			Annotations:       make(map[string]string),
			DeletionTimestamp: func() *metav1.Time { time := metav1.Now(); return &time }(),
			Finalizers:        []string{"smoothDeletion"},
		},
		Spec: carbonv1.RollingSetSpec{
			Version: "version-new",
			SchedulePlan: rollalgorithm.SchedulePlan{
				Strategy: apps.DeploymentStrategy{
					Type: apps.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &apps.RollingUpdateDeployment{
						MaxUnavailable: func() *intstr.IntOrString { i := intstr.FromInt(maxUnavailable); return &i }(),
						MaxSurge:       func() *intstr.IntOrString { i := intstr.FromInt(maxSurge); return &i }(),
					},
				},
				Replicas:           func() *int32 { i := int32(replicas); return &i }(),
				LatestVersionRatio: func() *int32 { i := int32(latestVersionRatio); return &i }(),
			},
			Selector: &metav1.LabelSelector{MatchLabels: selector},
		},
		Status: carbonv1.RollingSetStatus{},
	}

	return &rollingset
}

// 新的0个版本， 老的四个版本
func TestInPlaceScheduler_initVersionArgs_01(t *testing.T) {
	type fields struct {
		rollingSet      *carbonv1.RollingSet
		replicas        []*carbonv1.Replica
		resourceManager util.ResourceManager
	}
	type args struct {
		rollingsetArgs *rollingsetArgs
	}
	rollingsetArgs := rollingsetArgs{}
	rollingsetArgs.ShardScheduleParams = &rollalgorithm.ShardScheduleParams{}
	rollingsetArgs.LatestVersion = "version-new"
	replicaSet := newReplicasWithVerisonPlan(true)
	rollingsetArgs._replicaSet = replicaSet

	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "test1",
			fields: fields{
				rollingSet: newRollingSet("version-new"),
			},
			args: args{rollingsetArgs: &rollingsetArgs},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceScheduler{
				rollingSet:      tt.fields.rollingSet,
				replicas:        tt.fields.replicas,
				resourceManager: &tt.fields.resourceManager,
			}
			s.initVersionArgs(tt.args.rollingsetArgs)

			if len(tt.args.rollingsetArgs.versionReplicaMap) != 5 {
				t.Errorf("TestInPlaceScheduler_initVersionArgs versionRepliaMap.size got = %v, want %v", len(tt.args.rollingsetArgs.versionReplicaMap), 4)
			}

			if len(tt.args.rollingsetArgs.versionPlanMap) != 5 {
				t.Errorf("TestInPlaceScheduler_initVersionArgs versionPlanMap.size got = %v, want %v", len(tt.args.rollingsetArgs.versionReplicaMap), 4)
			}

			if len(tt.args.rollingsetArgs.versionReplicaMap["version-0"]) != 3 {
				t.Errorf("TestInPlaceScheduler_initVersionArgs versionRepliaMap[0].size got = %v, want %v", len(tt.args.rollingsetArgs.versionReplicaMap["version-0"]), 3)
			}
			if len(tt.args.rollingsetArgs.versionReplicaMap["version-3"]) != 2 {
				t.Errorf("TestInPlaceScheduler_initVersionArgs versionRepliaMap[3].size got = %v, want %v", len(tt.args.rollingsetArgs.versionReplicaMap["version-3"]), 2)
			}

			if len(tt.args.rollingsetArgs.versionReplicaMap["version-new"]) != 0 {
				t.Errorf("TestInPlaceScheduler_initVersionArgs versionRepliaMap[new].size got = %v, want %v", len(tt.args.rollingsetArgs.versionReplicaMap["version-new"]), 0)
			}

			if len(tt.args.rollingsetArgs.versionPlanMap) != 5 {
				t.Errorf("TestInPlaceScheduler_initVersionArgs versionRepliaMap[new].size got = %v, want %v", len(tt.args.rollingsetArgs.versionReplicaMap["version-new"]), 0)
			}

			if tt.args.rollingsetArgs.versionPlanMap["version-new"] == nil {
				t.Errorf("TestInPlaceScheduler_initVersionArgs versionRepliaMap[new].size got = %v, want %v", len(tt.args.rollingsetArgs.versionReplicaMap["version-new"]), 0)
			}

			if tt.args.rollingsetArgs.versionPlanMap["version-new"].VersionPlan.MinReadySeconds != 1024 {
				t.Errorf("TestInPlaceScheduler_initVersionArgs versionRepliaMap[new].size got = %v, want %v", len(tt.args.rollingsetArgs.versionReplicaMap["version-new"]), 0)
			}

			if tt.args.rollingsetArgs.versionPlanMap["version-0"].VersionPlan.MinReadySeconds != 0 {
				t.Errorf("TestInPlaceScheduler_initVersionArgs versionRepliaMap[new].size got = %v, want %v", len(tt.args.rollingsetArgs.versionReplicaMap["version-new"]), 0)
			}

		})
	}
}

// 新+老的四个版本
func TestInPlaceScheduler_initVersionArgs_02(t *testing.T) {
	type fields struct {
		rollingSet      *carbonv1.RollingSet
		replicas        []*carbonv1.Replica
		resourceManager util.ResourceManager
	}
	type args struct {
		rollingsetArgs *rollingsetArgs
	}
	rollingsetArgs := rollingsetArgs{}
	rollingsetArgs.ShardScheduleParams = &rollalgorithm.ShardScheduleParams{}
	rollingsetArgs.LatestVersion = "version-0"
	replicaSet := newReplicasWithVerisonPlan(true)
	rollingsetArgs._replicaSet = replicaSet

	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "test1",
			fields: fields{
				rollingSet: newRollingSet("version-0"),
			},
			args: args{rollingsetArgs: &rollingsetArgs},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceScheduler{
				rollingSet:      tt.fields.rollingSet,
				replicas:        tt.fields.replicas,
				resourceManager: &tt.fields.resourceManager,
			}
			s.initVersionArgs(tt.args.rollingsetArgs)
			fmt.Println(tt.args.rollingsetArgs.versionReplicaMap)
			if len(tt.args.rollingsetArgs.versionReplicaMap) != 4 {
				t.Errorf("TestInPlaceScheduler_initVersionArgs versionRepliaMap.size got = %v, want %v", len(tt.args.rollingsetArgs.versionReplicaMap), 4)
			}

			if len(tt.args.rollingsetArgs.versionPlanMap) != 4 {
				t.Errorf("TestInPlaceScheduler_initVersionArgs versionPlanMap.size got = %v, want %v", len(tt.args.rollingsetArgs.versionReplicaMap), 4)
			}

			if len(tt.args.rollingsetArgs.versionReplicaMap["version-0"]) != 3 {
				t.Errorf("TestInPlaceScheduler_initVersionArgs versionRepliaMap[0].size got = %v, want %v", len(tt.args.rollingsetArgs.versionReplicaMap["version-0"]), 3)
			}
			if len(tt.args.rollingsetArgs.versionReplicaMap["version-3"]) != 2 {
				t.Errorf("TestInPlaceScheduler_initVersionArgs versionRepliaMap[3].size got = %v, want %v", len(tt.args.rollingsetArgs.versionReplicaMap["version-3"]), 2)
			}

			if len(tt.args.rollingsetArgs.versionPlanMap) != 4 {
				t.Errorf("TestInPlaceScheduler_initVersionArgs versionRepliaMap[new].size got = %v, want %v", len(tt.args.rollingsetArgs.versionReplicaMap["version-new"]), 0)
			}

			if tt.args.rollingsetArgs.versionPlanMap["version-0"] == nil {
				t.Errorf("TestInPlaceScheduler_initVersionArgs versionRepliaMap[new].size got = %v, want %v", len(tt.args.rollingsetArgs.versionReplicaMap["version-new"]), 0)
			}

			if tt.args.rollingsetArgs.versionPlanMap["version-0"].VersionPlan.MinReadySeconds != 1024 {
				t.Errorf("TestInPlaceScheduler_initVersionArgs versionRepliaMap[new].size got = %v, want %v", len(tt.args.rollingsetArgs.versionReplicaMap["version-new"]), 0)
			}

			if tt.args.rollingsetArgs.versionPlanMap["version-1"].VersionPlan.MinReadySeconds != 1000 {
				t.Errorf("TestInPlaceScheduler_initVersionArgs versionRepliaMap[new].size got = %v, want %v", len(tt.args.rollingsetArgs.versionReplicaMap["version-new"]), 0)
			}
		})
	}
}

func TestInPlaceScheduler_checkArgs(t *testing.T) {
	type fields struct {
		rollingSet      *carbonv1.RollingSet
		replicas        []*carbonv1.Replica
		resourceManager util.ResourceManager
	}
	rollingset := newRollingSet("version-new")
	tests := []struct {
		name    string
		fields  fields
		want    rollingsetArgs
		wantErr bool
	}{
		{
			name:    "test1",
			fields:  fields{rollingSet: rollingset, replicas: newReplicas(rollingset)},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceScheduler{
				rollingSet:      tt.fields.rollingSet,
				replicas:        tt.fields.replicas,
				resourceManager: &tt.fields.resourceManager,
			}
			_, err := s.checkArgs()
			if (err != nil) != tt.wantErr {
				t.Errorf("InPlaceScheduler.checkArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func newRollingsetArgs(versionCount int, latestVersionRatio int) *rollingsetArgs {
	rollingsetArgs := rollingsetArgs{}
	rollingsetArgs.versionPlanMap = make(map[string]*carbonv1.VersionPlanWithGangPlan, versionCount)
	for i := 0; i < versionCount; i++ {
		rollingsetArgs.versionPlanMap["plan-"+strconv.Itoa(i)] = &carbonv1.VersionPlanWithGangPlan{VersionPlan: &carbonv1.VersionPlan{}}
	}
	return &rollingsetArgs
}

func newRollingsetArgsCollectLatestRedundantReplicas(versionCount int, curLatestReplicas int, targetLatestReplicas int, latestHoldCount int) *rollingsetArgs {
	rollingsetArgs := rollingsetArgs{}
	rollingsetArgs.versionPlanMap = make(map[string]*carbonv1.VersionPlanWithGangPlan, versionCount)
	for i := 0; i < versionCount; i++ {
		rollingsetArgs.versionPlanMap["version-"+strconv.Itoa(i)] = &carbonv1.VersionPlanWithGangPlan{VersionPlan: &carbonv1.VersionPlan{}}
	}
	versionHoldingMap := make(map[string]int32, 0)
	versionHoldingMap["version-new"] = int32(latestHoldCount)
	versionHoldingMap["version-1"] = int32(10 - latestHoldCount)
	versionRepliaMap := make(map[string][]*carbonv1.Replica, 0)
	versionRepliaMap["version-new"] = newReplicasCollectLatestRedundantReplicas("version-new", 5, false)
	versionRepliaMap["version-1"] = newReplicasCollectLatestRedundantReplicas("version-1", 5, false)
	rollingsetArgs.versionReplicaMap = versionRepliaMap
	return &rollingsetArgs
}

type testCount struct {
	count int32
}

// 指针
func TestInPlaceScheduler_test1(t *testing.T) {
	redundantNodes := make([]*testCount, 0)
	testCount := testCount{count: 1}
	redundantNodes = append(redundantNodes, &testCount)
	t.Logf("main value = %v", redundantNodes)
	testC(&redundantNodes, t)
	t.Logf("main value = %v", redundantNodes)

}

func testC(redundantNodes *[]*testCount, t *testing.T) {
	testCount := testCount{count: 2}
	*redundantNodes = append(*redundantNodes, &testCount)
	t.Logf("testC value = %v", redundantNodes)
}

func newRollingsetArgsReplaceAssignedReplicas(desired int, targetOldReplicas int, curOldReplicas int, statusReplicas int, maxSurge int) *rollingsetArgs {
	rollingsetArgs := rollingsetArgs{}

	rollingsetArgs.activeReplicas = newReplicasWithAssigend("ASSIGNED", false)
	rollingsetArgs.versionPlanMap = map[string]*carbonv1.VersionPlanWithGangPlan{}
	rollingsetArgs.versionPlanMap["version-"+strconv.Itoa(0)] = &carbonv1.VersionPlanWithGangPlan{VersionPlan: &carbonv1.VersionPlan{}}
	rollingsetArgs.versionPlanMap["version-"+strconv.Itoa(1)] = &carbonv1.VersionPlanWithGangPlan{VersionPlan: &carbonv1.VersionPlan{}}
	rollingsetArgs.versionPlanMap["version-"+strconv.Itoa(2)] = &carbonv1.VersionPlanWithGangPlan{VersionPlan: &carbonv1.VersionPlan{}}

	return &rollingsetArgs
}

func newRollingsetArgsReplaceNotAssignedReplicas(allocStatus string, desired int, targetOldReplicas int, curOldReplicas int, statusReplicas int, maxSurge int) *rollingsetArgs {
	rollingsetArgs := rollingsetArgs{}

	rollingsetArgs.activeReplicas = newReplicasWithAssigend(allocStatus, false)
	rollingsetArgs.versionPlanMap = map[string]*carbonv1.VersionPlanWithGangPlan{}
	rollingsetArgs.versionPlanMap["version-"+strconv.Itoa(0)] = &carbonv1.VersionPlanWithGangPlan{VersionPlan: &carbonv1.VersionPlan{}}
	rollingsetArgs.versionPlanMap["version-"+strconv.Itoa(1)] = &carbonv1.VersionPlanWithGangPlan{VersionPlan: &carbonv1.VersionPlan{}}
	rollingsetArgs.versionPlanMap["version-"+strconv.Itoa(2)] = &carbonv1.VersionPlanWithGangPlan{VersionPlan: &carbonv1.VersionPlan{}}
	rollingsetArgs.supplements = mapset.NewThreadUnsafeSet()
	return &rollingsetArgs
}

func newReplicasWithAssigend(allocStatus string, released bool) []*carbonv1.Replica {
	replicas := make([]*carbonv1.Replica, 10)
	if released {
		allocStatus = string(carbonv1.WorkerReleased)
	}
	var status carbonv1.AllocStatus = carbonv1.AllocStatus(allocStatus)
	for i := 0; i < 10; i++ {
		index := i % 3
		version := "version-" + strconv.Itoa(index)
		replica := &carbonv1.Replica{
			WorkerNode: carbonv1.WorkerNode{
				TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "replica-" + strconv.Itoa(i),
					UID:       uuid.NewUUID(),
					Namespace: metav1.NamespaceDefault,
				},
				Spec: carbonv1.WorkerNodeSpec{
					Version: version,
					VersionPlan: carbonv1.VersionPlan{
						BroadcastPlan: carbonv1.BroadcastPlan{
							WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
								MinReadySeconds:      int32(index * 1000),
								ResourceMatchTimeout: 2000,
								WorkerReadyTimeout:   int64(index * 3000),
							},
						},
					},
				},
				Status: carbonv1.WorkerNodeStatus{
					Complete:    true,
					AllocStatus: status, //"UNASSIGNED",
				},
			},
		}
		replicas[i] = replica
	}
	return replicas
}

func newFullRedundantsAssignedAndUn() *[]*carbonv1.Replica {
	redundants := newReplicasAssignedAndUn("version-1", 10, false)
	return &redundants
}

func newMaxRedundantsAssignedAndUn(count int) *[]*carbonv1.Replica {
	redundants := newReplicasAssignedAndUn("version-1", count, false)
	return &redundants
}

func newReplicasAssignedAndUn(version string, count int, releasing bool) []*carbonv1.Replica {
	replicas := make([]*carbonv1.Replica, count)
	for i := 0; i < count; i++ {
		status := "ASSIGNED"
		if i%2 == 0 {
			status = "UNASSIGNED"
		}
		replica := &carbonv1.Replica{
			WorkerNode: carbonv1.WorkerNode{
				TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "replica-" + strconv.Itoa(i),
					UID:       uuid.NewUUID(),
					Namespace: metav1.NamespaceDefault,
				},
				Spec: carbonv1.WorkerNodeSpec{
					Version: version,
					VersionPlan: carbonv1.VersionPlan{
						BroadcastPlan: carbonv1.BroadcastPlan{
							WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
								MinReadySeconds:      100,
								ResourceMatchTimeout: 2000,
								WorkerReadyTimeout:   300,
							},
						},
					},
					ToDelete: releasing,
				},
				Status: carbonv1.WorkerNodeStatus{
					Complete:    true,
					Score:       int64(i % 3),
					AllocStatus: carbonv1.AllocStatus(status),
				},
			},
		}
		replicas[i] = replica
	}
	return replicas
}

func newEmptyRedundants() *[]*carbonv1.Replica {
	redundants := make([]*carbonv1.Replica, 0)
	return &redundants
}

func TestInPlaceScheduler_releaseReplicaReplicas(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockResourceManager := mock.NewMockResourceManager(ctl)
	//	mockResourceManager.EXPECT().ReleaseReplica(gomock.Any(), gomock.Any()).Return(nil).MaxTimes(10)

	mockResourceManager.EXPECT().BatchReleaseReplica(gomock.Any(), gomock.Any()).Return(1, nil).MaxTimes(10)

	type fields struct {
		rollingSet      *carbonv1.RollingSet
		replicas        []*carbonv1.Replica
		resourceManager util.ResourceManager
	}
	type args struct {
		redundants *[]*carbonv1.Replica
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		len    int
	}{
		{
			name: "test1",
			fields: fields{resourceManager: mockResourceManager,
				rollingSet: newRollingset("version-1", 1000)},
			args: args{redundants: newFullRedundantsAssignedAndUn()},
			len:  10,
		},
		{
			name:   "test2",
			fields: fields{resourceManager: mockResourceManager},
			args:   args{redundants: newEmptyRedundants()},
			len:    0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceScheduler{
				rollingSet:      tt.fields.rollingSet,
				replicas:        tt.fields.replicas,
				resourceManager: &tt.fields.resourceManager,
			}
			s.releaseReplicas(tt.args.redundants)
			if len(*tt.args.redundants) != tt.len {
				t.Errorf("InPlaceScheduler.releaseReplicaReplicas() = %v, want %v", tt.args.redundants, tt.len)
			}
			for _, replica := range *tt.args.redundants {
				if replica.Spec.ToDelete != false {
					t.Errorf("InPlaceScheduler.releaseReplicaReplicas() = %v, want %v", replica.Spec.ToDelete, true)

				}
			}
		})

	}
}

func TestInPlaceScheduler_releaseReplicaReplicas_max(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockResourceManager := mock.NewMockResourceManager(ctl)
	//	mockResourceManager.EXPECT().ReleaseReplica(gomock.Any(), gomock.Any()).Return(nil).MaxTimes(10)

	mockResourceManager.EXPECT().BatchReleaseReplica(gomock.Any(), gomock.Any()).Return(1, nil).MaxTimes(10)

	type fields struct {
		rollingSet      *carbonv1.RollingSet
		replicas        []*carbonv1.Replica
		resourceManager util.ResourceManager
	}
	type args struct {
		redundants *[]*carbonv1.Replica
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		len    int
	}{
		{
			name: "test1",
			fields: fields{resourceManager: mockResourceManager,
				rollingSet: newRollingset("version-1", 1000)},
			args: args{redundants: newMaxRedundantsAssignedAndUn(5000)},
			len:  4000,
		},
		{
			name: "test2",
			fields: fields{resourceManager: mockResourceManager,
				rollingSet: newRollingset("version-1", 1000)},
			args: args{redundants: newMaxRedundantsAssignedAndUn(4000)},
			len:  4000,
		},
		{
			name: "test3",
			fields: fields{resourceManager: mockResourceManager,
				rollingSet: newRollingset("version-1", 1000)},
			args: args{redundants: newMaxRedundantsAssignedAndUn(3999)},
			len:  3999,
		},
		{
			name: "test4",
			fields: fields{resourceManager: mockResourceManager,
				rollingSet: newRollingset("version-1", 1000)},
			args: args{redundants: newMaxRedundantsAssignedAndUn(4001)},
			len:  4000,
		},
		{
			name: "test5",
			fields: fields{resourceManager: mockResourceManager,
				rollingSet: newRollingset("version-1", 1000)},
			args: args{redundants: newMaxRedundantsAssignedAndUn(10001)},
			len:  4000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceScheduler{
				rollingSet:      tt.fields.rollingSet,
				replicas:        tt.fields.replicas,
				resourceManager: &tt.fields.resourceManager,
			}
			s.releaseReplicas(tt.args.redundants)
			if len(*tt.args.redundants) != tt.len {
				t.Errorf("InPlaceScheduler.releaseReplicaReplicas() = %v, want %v", tt.args.redundants, tt.len)
			}
			for _, replica := range *tt.args.redundants {
				if replica.Spec.ToDelete != false {
					t.Errorf("InPlaceScheduler.releaseReplicaReplicas() = %v, want %v", replica.Spec.ToDelete, true)

				}
			}
		})

	}
}

func newRollingsetArgsRemoveReleasedReplicaNodes(released bool) *rollingsetArgs {
	rollingsetArgs := rollingsetArgs{}

	rollingsetArgs._replicaSet = newReplicasWithAssigend("ASSIGNED", false)
	rollingsetArgs._replicaSet = append(rollingsetArgs._replicaSet, newReplicasWithAssigend("ASSIGNED", released)...)

	return &rollingsetArgs
}

func TestInPlaceScheduler_removeReleasedReplicaNodes(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockResourceManager := mock.NewMockResourceManager(ctl)
	//	mockResourceManager.EXPECT().RemoveReplica(gomock.Any(), gomock.Any()).Return(nil).MaxTimes(10)
	mockResourceManager.EXPECT().BatchRemoveReplica(gomock.Any(), gomock.Any()).Return(1, nil).MaxTimes(10)

	type fields struct {
		rollingSet      *carbonv1.RollingSet
		replicas        []*carbonv1.Replica
		resourceManager util.ResourceManager
	}
	type args struct {
		rollingsetArgs *rollingsetArgs
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		len    int
	}{
		{
			name:   "test1",
			fields: fields{resourceManager: mockResourceManager, rollingSet: newRollingSet("")},
			args:   args{rollingsetArgs: newRollingsetArgsRemoveReleasedReplicaNodes(false)},
			len:    20,
		},
		{
			name:   "test2",
			fields: fields{resourceManager: mockResourceManager, rollingSet: newRollingSet("")},
			args:   args{rollingsetArgs: newRollingsetArgsRemoveReleasedReplicaNodes(true)},
			len:    10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceScheduler{
				rollingSet:      tt.fields.rollingSet,
				replicas:        tt.fields.replicas,
				resourceManager: &tt.fields.resourceManager,
			}
			s.removeReleasedReplicaNodes(tt.args.rollingsetArgs)

			if len(tt.args.rollingsetArgs._replicaSet) != tt.len {
				t.Errorf("InPlaceScheduler.releaseReplicaReplicas() = %v, want %v", tt.args.rollingsetArgs._replicaSet, tt.len)
			}
			for _, replica := range tt.args.rollingsetArgs._replicaSet {
				if carbonv1.IsReplicaReleased(replica) {
					t.Errorf("InPlaceScheduler.releaseReplicaReplicas() = %v, want %v", replica.Status.AllocStatus, true)

				}
			}

		})
	}
}

func newReplicasWithStatus(count int) *[]*carbonv1.Replica {
	replicas := make([]*carbonv1.Replica, 10)
	for i := 0; i < 10; i++ {

		version := "version-" + strconv.Itoa(i)
		if i < count {
			replica := &carbonv1.Replica{
				WorkerNode: carbonv1.WorkerNode{
					TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "replica-" + strconv.Itoa(i),
						UID:       uuid.NewUUID(),
						Namespace: metav1.NamespaceDefault,
					},
					Spec: carbonv1.WorkerNodeSpec{
						Version:     version,
						VersionPlan: *testNewVersionPlan("share", "docker", "64m", "regiest", false),
					},
				},
			}
			replicas[i] = replica
		} else {
			replica := &carbonv1.Replica{
				WorkerNode: carbonv1.WorkerNode{
					TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "replica-" + strconv.Itoa(i),
						UID:       uuid.NewUUID(),
						Namespace: metav1.NamespaceDefault,
					},
					Spec: carbonv1.WorkerNodeSpec{
						Version:     version,
						VersionPlan: *testNewVersionPlan("share", "docker", "64m", "regiest", false),
					},
					Status: carbonv1.WorkerNodeStatus{
						Complete:    true,
						AllocStatus: "UNASSIGNED",
					},
				},
			}
			replicas[i] = replica
		}
	}

	return &replicas
}

func TestInPlaceScheduler_syncReplicas(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockResourceManager := mock.NewMockResourceManager(ctl)
	// mockResourceManager.EXPECT().UpdateReplica(gomock.Any(), gomock.Any()).Return(nil).MaxTimes(4)
	// mockResourceManager.EXPECT().CreateReplica(gomock.Any(), gomock.Any()).Return(nil, nil).MaxTimes(20)
	mockResourceManager.EXPECT().BatchCreateReplica(gomock.Any(), gomock.Any()).Return(1, nil).MaxTimes(10)
	mockResourceManager.EXPECT().BatchUpdateReplica(gomock.Any(), gomock.Any()).Return(1, nil).MaxTimes(10)

	type fields struct {
		rollingSet      *carbonv1.RollingSet
		replicas        []*carbonv1.Replica
		resourceManager util.ResourceManager
	}
	type args struct {
		rollingsetArgs *rollingsetArgs
		//supplementReplicaSet *[]*carbonv1.Replica
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "test1",
			fields: fields{resourceManager: mockResourceManager, rollingSet: newRollingSet("")},
			args:   args{rollingsetArgs: &rollingsetArgs{supplementReplicaSet: *newReplicasWithStatus(10), versionPlanMap: make(map[string]*carbonv1.VersionPlanWithGangPlan, 0)}},
		},

		{
			name:   "test1",
			fields: fields{resourceManager: mockResourceManager, rollingSet: newRollingSet("")},
			args:   args{rollingsetArgs: &rollingsetArgs{supplementReplicaSet: *newReplicasWithStatus(5), versionPlanMap: make(map[string]*carbonv1.VersionPlanWithGangPlan, 0)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceScheduler{
				rollingSet:      tt.fields.rollingSet,
				replicas:        tt.fields.replicas,
				resourceManager: &tt.fields.resourceManager,
				c:               &Controller{},
			}
			s.syncReplicas(tt.args.rollingsetArgs)
		})
	}
}

func TestInPlaceScheduler_syncReplicas_error(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockResourceManager := mock.NewMockResourceManager(ctl)
	//err := &errors.StatusError{metav1.StatusReasonAlreadyExists{Reason: StatusReasonAlreadyExists}}
	err := &errors.StatusError{}

	// mockResourceManager.EXPECT().UpdateReplica(gomock.Any(), gomock.Any()).Return(nil).MaxTimes(4)
	// mockResourceManager.EXPECT().CreateReplica(gomock.Any(), gomock.Any()).Return(nil, (err)).MaxTimes(20)
	mockResourceManager.EXPECT().BatchCreateReplica(gomock.Any(), gomock.Any()).Return(1, err).MaxTimes(10)
	mockResourceManager.EXPECT().BatchUpdateReplica(gomock.Any(), gomock.Any()).Return(1, nil).MaxTimes(10)

	type fields struct {
		rollingSet      *carbonv1.RollingSet
		replicas        []*carbonv1.Replica
		resourceManager util.ResourceManager
	}
	type args struct {
		rollingsetArgs *rollingsetArgs
		//	supplementReplicaSet *[]*carbonv1.Replica
	}
	// rollingsetArgs := &rollingsetArgs{supplementReplicaSet: *newReplicasWithStatus(10)}
	// rollingsetArgs.supplementReplicaSet = *newReplicasWithStatus(10)
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "test1",
			fields: fields{resourceManager: mockResourceManager, rollingSet: newRollingSet("")},
			args:   args{rollingsetArgs: &rollingsetArgs{supplementReplicaSet: *newReplicasWithStatus(10), versionPlanMap: make(map[string]*carbonv1.VersionPlanWithGangPlan, 0)}},
		},

		{
			name:   "test1",
			fields: fields{resourceManager: mockResourceManager, rollingSet: newRollingSet("")},
			args:   args{rollingsetArgs: &rollingsetArgs{supplementReplicaSet: *newReplicasWithStatus(5), versionPlanMap: make(map[string]*carbonv1.VersionPlanWithGangPlan, 0)}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceScheduler{
				rollingSet:      tt.fields.rollingSet,
				replicas:        tt.fields.replicas,
				resourceManager: &tt.fields.resourceManager,
				c:               &Controller{},
			}
			s.syncReplicas(tt.args.rollingsetArgs)
		})
	}
}

func newReplicasSyncRollingsetStatus(count int) *[]*carbonv1.Replica {
	replicas := make([]*carbonv1.Replica, 100)
	for i := 0; i < count; i++ {

		version := "version-" + strconv.Itoa(i)

		replica := &carbonv1.Replica{
			WorkerNode: carbonv1.WorkerNode{
				TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "replica-" + strconv.Itoa(i),
					UID:       uuid.NewUUID(),
					Namespace: metav1.NamespaceDefault,
				},
				Spec: carbonv1.WorkerNodeSpec{
					Version: version,
					VersionPlan: carbonv1.VersionPlan{
						BroadcastPlan: carbonv1.BroadcastPlan{
							WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
								MinReadySeconds:      190,
								ResourceMatchTimeout: 2000,
								WorkerReadyTimeout:   20999,
							},
						},
					},
				},
				Status: carbonv1.WorkerNodeStatus{
					AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{Version: version},
					Complete:              true,
					ServiceStatus:         carbonv1.ServiceAvailable,
				},
			},
		}
		replicas[i] = replica
	}

	for i := count; i < 100; i++ {

		version := "version-" + strconv.Itoa(i)

		replica := &carbonv1.Replica{
			WorkerNode: carbonv1.WorkerNode{
				TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "replica-" + strconv.Itoa(i),
					UID:       uuid.NewUUID(),
					Namespace: metav1.NamespaceDefault,
				},
				Spec: carbonv1.WorkerNodeSpec{
					Version: version,
					VersionPlan: carbonv1.VersionPlan{
						BroadcastPlan: carbonv1.BroadcastPlan{
							WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
								MinReadySeconds:      190,
								ResourceMatchTimeout: 2000,
								WorkerReadyTimeout:   20999,
							},
						},
					},
				},
				Status: carbonv1.WorkerNodeStatus{
					Complete:      false,
					ServiceStatus: carbonv1.ServiceAvailable,
				},
			},
		}
		replicas[i] = replica
	}
	return &replicas
}
func TestInPlaceScheduler_syncRollingsetStatus(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockResourceManager := mock.NewMockResourceManager(ctl)
	mockResourceManager.EXPECT().UpdateRollingSetStatus(gomock.Any()).Return(nil).MaxTimes(10)

	type fields struct {
		rollingSet       *carbonv1.RollingSet
		rollingSetStatus *carbonv1.RollingSetStatus
		replicas         []*carbonv1.Replica
		resourceManager  util.ResourceManager
	}
	type args struct {
		replicaSet          *[]*carbonv1.Replica
		complete            bool
		serviceReady        bool
		uncompleteStatusMap map[string]carbonv1.ProcessStep
	}
	tests := []struct {
		name                string
		fields              fields
		args                args
		wantErr             bool
		replicas            int32
		updatedReplicas     int32
		readyReplicas       int32
		availableReplicas   int32
		unavailableReplicas int32
		complete            bool
		//	observedGeneration  int64
	}{
		{
			name: "test1",
			fields: fields{
				rollingSet:       newRollingset("version-new", 10),
				resourceManager:  mockResourceManager,
				rollingSetStatus: newRollingset("version-new", 10).Status.DeepCopy(),
			},
			args:                args{replicaSet: newReplicasSyncRollingsetStatus(10), complete: true},
			replicas:            100,
			readyReplicas:       10,
			availableReplicas:   100,
			unavailableReplicas: 0,
			//	observedGeneration:  1000,
			updatedReplicas: 0,
			complete:        true,
		},
		{
			name: "test2",
			fields: fields{
				rollingSet:       newRollingset("version-2", 1000),
				resourceManager:  mockResourceManager,
				rollingSetStatus: newRollingset("version-new", 10).Status.DeepCopy(),
			},
			args:                args{replicaSet: newReplicasSyncRollingsetStatus(40), complete: false},
			replicas:            100,
			readyReplicas:       40,
			availableReplicas:   100,
			unavailableReplicas: 0,
			//	observedGeneration:  1000,
			updatedReplicas: 1,
			complete:        false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceScheduler{
				rollingSet:       tt.fields.rollingSet,
				replicas:         tt.fields.replicas,
				rollingSetStatus: tt.fields.rollingSetStatus,
				resourceManager:  &tt.fields.resourceManager,
				c:                &Controller{},
			}
			if err := s.syncRollingsetStatus(tt.args.replicaSet, tt.args.complete, tt.args.serviceReady, carbonv1.SpotStatus{}, tt.args.uncompleteStatusMap); (err != nil) != tt.wantErr {
				t.Errorf("InPlaceScheduler.syncRollingsetStatus() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.fields.rollingSet.Status.Replicas != tt.replicas {
				t.Errorf("InPlaceScheduler.syncRollingsetStatus() error = %v, wantErr %v", tt.fields.rollingSet.Status.Replicas, tt.replicas)

			}
			if tt.fields.rollingSet.Status.ReadyReplicas != tt.readyReplicas {
				t.Errorf("InPlaceScheduler.syncRollingsetStatus() error = %v, wantErr %v", tt.fields.rollingSet.Status.ReadyReplicas, tt.readyReplicas)
			}
			if tt.fields.rollingSet.Status.AvailableReplicas != tt.availableReplicas {
				t.Errorf("InPlaceScheduler.syncRollingsetStatus() error = %v, wantErr %v", tt.fields.rollingSet.Status.AvailableReplicas, tt.availableReplicas)
			}
			if tt.fields.rollingSet.Status.UnavailableReplicas != tt.unavailableReplicas {
				t.Errorf("InPlaceScheduler.syncRollingsetStatus() error = %v, wantErr %v", tt.fields.rollingSet.Status.UnavailableReplicas, tt.unavailableReplicas)
			}
			if tt.fields.rollingSet.Status.UpdatedReplicas != tt.updatedReplicas {
				t.Errorf("InPlaceScheduler.syncRollingsetStatus() error = %v, wantErr %v", tt.fields.rollingSet.Status.UpdatedReplicas, tt.updatedReplicas)
			}
			if tt.fields.rollingSet.Status.Complete != tt.complete {
				t.Errorf("InPlaceScheduler.syncRollingsetStatus() error = %v, wantErr %v", tt.fields.rollingSet.Status.Complete, tt.complete)
			}
		})
	}
}

func newRollingSetInitargs(version string, replicas int) *carbonv1.RollingSet {
	selector := map[string]string{"foo": "bar"}
	rollingset := carbonv1.RollingSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        "rollingset-m",
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
		Spec: carbonv1.RollingSetSpec{
			Version: version,
			SchedulePlan: rollalgorithm.SchedulePlan{
				Strategy: apps.DeploymentStrategy{
					Type: apps.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &apps.RollingUpdateDeployment{
						MaxUnavailable: func() *intstr.IntOrString { i := intstr.FromInt(2); return &i }(),
						MaxSurge:       func() *intstr.IntOrString { i := intstr.FromInt(2); return &i }(),
					},
				},
				Replicas:           func() *int32 { i := int32(replicas); return &i }(),
				LatestVersionRatio: func() *int32 { i := int32(rollalgorithm.DefaultLatestVersionRatio); return &i }(),
			},

			VersionPlan: *testNewVersionPlan("share", "docker", "64m", "regiest", false),

			Selector: &metav1.LabelSelector{MatchLabels: selector},
		},
		Status: carbonv1.RollingSetStatus{},
	}
	return &rollingset
}

func newReplicasBase(version string, count int) *[]*carbonv1.Replica {
	replicas := make([]*carbonv1.Replica, count)
	for i := 0; i < count; i++ {
		replica := &carbonv1.Replica{
			WorkerNode: carbonv1.WorkerNode{
				TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "replica-" + strconv.Itoa(i),
					UID:       uuid.NewUUID(),
					Namespace: metav1.NamespaceDefault,
				},
				Spec: carbonv1.WorkerNodeSpec{
					Version: version,
					VersionPlan: carbonv1.VersionPlan{
						BroadcastPlan: carbonv1.BroadcastPlan{
							WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
								MinReadySeconds:      190,
								ResourceMatchTimeout: 2000,
								WorkerReadyTimeout:   20999,
							},
						},
					},
				},
				Status: carbonv1.WorkerNodeStatus{
					Complete: true,
				},
			},
		}
		replicas[i] = replica
	}
	return &replicas
}

func Test_getScheduleId(t *testing.T) {
	type args struct {
		rsName string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test",
			args: args{rsName: "test"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getScheduleID(tt.args.rsName)
			if got == "" {
				t.Errorf("id: %s", got)
			}
		})
	}
}

func TestInPlaceScheduler_collectResourceChangeReplicas(t *testing.T) {
	type fields struct {
		scheduleID      string
		rollingSet      *carbonv1.RollingSet
		replicas        []*carbonv1.Replica
		resourceManager *util.ResourceManager
		generalLabels   map[string]string
	}
	type args struct {
		replicas []*carbonv1.Replica
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantCopy    []*carbonv1.Replica
		wantChanged []*carbonv1.Replica
	}{
		{
			name: "test",
			fields: fields{
				rollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{ResVersion: "1"},
				},
			},
			args: args{
				replicas: []*carbonv1.Replica{
					{
						WorkerNode: carbonv1.WorkerNode{
							ObjectMeta: metav1.ObjectMeta{
								Name: "1",
							},
							Spec: carbonv1.WorkerNodeSpec{ResVersion: "1"},
						},
					},
					{
						WorkerNode: carbonv1.WorkerNode{
							ObjectMeta: metav1.ObjectMeta{
								Name: "2",
							},
							Spec: carbonv1.WorkerNodeSpec{ResVersion: "2"},
						},
					},
				},
			},
			wantCopy: []*carbonv1.Replica{
				{
					WorkerNode: carbonv1.WorkerNode{
						ObjectMeta: metav1.ObjectMeta{
							Name: "1",
						},
						Spec: carbonv1.WorkerNodeSpec{ResVersion: "1"},
					},
				},
				{
					WorkerNode: carbonv1.WorkerNode{
						ObjectMeta: metav1.ObjectMeta{
							Name: "2",
						},
						Spec: carbonv1.WorkerNodeSpec{ResVersion: "2"},
					},
				},
			},
			wantChanged: []*carbonv1.Replica{
				{
					WorkerNode: carbonv1.WorkerNode{
						ObjectMeta: metav1.ObjectMeta{
							Name: "2",
						},
						Spec: carbonv1.WorkerNodeSpec{ResVersion: "2"},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceScheduler{
				scheduleID:      tt.fields.scheduleID,
				rollingSet:      tt.fields.rollingSet,
				replicas:        tt.fields.replicas,
				resourceManager: tt.fields.resourceManager,
				generalLabels:   tt.fields.generalLabels,
				c:               &Controller{},
			}
			gotCopy, gotChanged := s.collectResourceChangeReplicas(tt.args.replicas)
			if !reflect.DeepEqual(gotCopy, tt.wantCopy) {
				t.Errorf("InPlaceScheduler.collectResourceChangeReplicas() gotCopy = %v, want %v", gotCopy, tt.wantCopy)
			}
			if !reflect.DeepEqual(gotChanged, tt.wantChanged) {
				t.Errorf("InPlaceScheduler.collectResourceChangeReplicas() gotChanged = %v, want %v", gotChanged, tt.wantChanged)
			}
		})
	}
}

func TestInPlaceScheduler_initResVersionMap(t *testing.T) {
	type fields struct {
		scheduleID      string
		rollingSet      *carbonv1.RollingSet
		replicas        []*carbonv1.Replica
		resourceManager *util.ResourceManager
		generalLabels   map[string]string
	}
	type args struct {
		rollingsetArgs *rollingsetArgs
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string][]*carbonv1.Replica
		want1  map[string]*carbonv1.VersionPlan
	}{
		{
			name: "test",
			fields: fields{
				rollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{ResVersion: "1"},
				},
			},
			args: args{
				rollingsetArgs: &rollingsetArgs{
					_resChangedReplicaSet: []*carbonv1.Replica{
						{
							WorkerNode: carbonv1.WorkerNode{
								ObjectMeta: metav1.ObjectMeta{
									Name: "1",
								},
								Spec: carbonv1.WorkerNodeSpec{
									ResVersion: "1",
									VersionPlan: carbonv1.VersionPlan{
										SignedVersionPlan: carbonv1.SignedVersionPlan{
											Template: &carbonv1.HippoPodTemplate{},
										},
									},
								},
							},
						},
						{
							WorkerNode: carbonv1.WorkerNode{
								ObjectMeta: metav1.ObjectMeta{
									Name: "3",
								},
								Spec: carbonv1.WorkerNodeSpec{
									ResVersion: "1",
									VersionPlan: carbonv1.VersionPlan{
										SignedVersionPlan: carbonv1.SignedVersionPlan{
											Template: &carbonv1.HippoPodTemplate{},
										},
									},
								},
							},
						},
						{
							WorkerNode: carbonv1.WorkerNode{
								ObjectMeta: metav1.ObjectMeta{
									Name: "2",
								},
								Spec: carbonv1.WorkerNodeSpec{
									ResVersion: "2",
									VersionPlan: carbonv1.VersionPlan{
										SignedVersionPlan: carbonv1.SignedVersionPlan{
											Template: &carbonv1.HippoPodTemplate{
												ObjectMeta: metav1.ObjectMeta{
													Labels: map[string]string{"b": "b"},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: map[string][]*carbonv1.Replica{
				"1": {
					{
						WorkerNode: carbonv1.WorkerNode{
							ObjectMeta: metav1.ObjectMeta{
								Name: "1",
							},
							Spec: carbonv1.WorkerNodeSpec{
								ResVersion: "1",
								VersionPlan: carbonv1.VersionPlan{
									SignedVersionPlan: carbonv1.SignedVersionPlan{
										Template: &carbonv1.HippoPodTemplate{},
									},
								},
							},
						},
					},
					{
						WorkerNode: carbonv1.WorkerNode{
							ObjectMeta: metav1.ObjectMeta{
								Name: "3",
							},
							Spec: carbonv1.WorkerNodeSpec{
								ResVersion: "1",
								VersionPlan: carbonv1.VersionPlan{
									SignedVersionPlan: carbonv1.SignedVersionPlan{
										Template: &carbonv1.HippoPodTemplate{},
									},
								},
							},
						},
					},
				},
				"2": {
					{
						WorkerNode: carbonv1.WorkerNode{
							ObjectMeta: metav1.ObjectMeta{
								Name: "2",
							},
							Spec: carbonv1.WorkerNodeSpec{
								ResVersion: "2",
								VersionPlan: carbonv1.VersionPlan{
									SignedVersionPlan: carbonv1.SignedVersionPlan{
										Template: &carbonv1.HippoPodTemplate{
											ObjectMeta: metav1.ObjectMeta{
												Labels: map[string]string{"b": "b"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want1: map[string]*carbonv1.VersionPlan{
				"1": {
					SignedVersionPlan: carbonv1.SignedVersionPlan{
						Template: &carbonv1.HippoPodTemplate{},
					},
				},
				"2": {
					SignedVersionPlan: carbonv1.SignedVersionPlan{
						Template: &carbonv1.HippoPodTemplate{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{"b": "b"},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceScheduler{
				scheduleID:      tt.fields.scheduleID,
				rollingSet:      tt.fields.rollingSet,
				replicas:        tt.fields.replicas,
				resourceManager: tt.fields.resourceManager,
				generalLabels:   tt.fields.generalLabels,
			}
			s.initResVersionMap(tt.args.rollingsetArgs)
			if !reflect.DeepEqual(tt.args.rollingsetArgs.resVersionReplicaMap, tt.want) {
				t.Errorf("rollingsetArgs.resVersionRepliaMap gotCopy = %v, want %v", tt.args.rollingsetArgs.resVersionReplicaMap, tt.want)
			}
			if !reflect.DeepEqual(tt.args.rollingsetArgs.resVersionPlanMap, tt.want1) {
				t.Errorf("rollingsetArgs.resVersionPlanMap gotCopy = %v, want %v", tt.args.rollingsetArgs.resVersionReplicaMap, tt.want)
			}
		})
	}
}

func testNewVersionPlan(cpumode, containernode, cpus, image string, ip bool) *carbonv1.VersionPlan {
	var hippoPodSpec = &carbonv1.HippoPodSpec{
		HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
			CpusetMode:     cpumode,
			ContainerModel: containernode,
		},
		Containers: []carbonv1.HippoContainer{
			{
				Container: corev1.Container{
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							"cpu": resource.MustParse(cpus),
						},
					},
					Image: image,
				},
			},
		},
	}
	if ip {
		hippoPodSpec.Containers[0].Container.Resources.Limits[carbonv1.ResourceIP] = resource.MustParse("1")
	}
	var template = carbonv1.HippoPodTemplate{
		Spec: *hippoPodSpec,
	}
	var versionPlan = carbonv1.VersionPlan{
		SignedVersionPlan: carbonv1.SignedVersionPlan{
			Template: &template,
		},
	}
	return &versionPlan
}

func TestInPlaceScheduler_updateReplicaResource(t *testing.T) {
	type fields struct {
		scheduleID      string
		rollingSet      *carbonv1.RollingSet
		replicas        []*carbonv1.Replica
		resourceManager *util.ResourceManager
		generalLabels   map[string]string
	}
	type args struct {
		rollingsetArgs *rollingsetArgs
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string][]*carbonv1.Replica
		wantErr bool
	}{
		{
			name: "update",
			fields: fields{
				rollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						ResVersion:  "1",
						VersionPlan: *testNewVersionPlan("share", "docker", "64m", "regiest", false),
					},
				},
			},
			args: args{
				rollingsetArgs: &rollingsetArgs{
					resVersionReplicaMap: map[string][]*carbonv1.Replica{
						"1": {
							{
								WorkerNode: carbonv1.WorkerNode{
									ObjectMeta: metav1.ObjectMeta{
										Name: "1",
									},
									Spec: carbonv1.WorkerNodeSpec{
										ResVersion:  "1",
										VersionPlan: *testNewVersionPlan("share", "docker", "64m", "regiest", false),
									},
								},
							},
							{
								WorkerNode: carbonv1.WorkerNode{
									ObjectMeta: metav1.ObjectMeta{
										Name: "3",
									},
									Spec: carbonv1.WorkerNodeSpec{
										ResVersion:  "1",
										VersionPlan: *testNewVersionPlan("share", "docker", "64m", "regiest", false),
									},
								},
							},
						},
						"2": {
							{
								WorkerNode: carbonv1.WorkerNode{
									ObjectMeta: metav1.ObjectMeta{
										Name: "2",
										Labels: map[string]string{
											carbonv1.DefaultReplicaUniqueLabelKey:    "2",
											carbonv1.DefaultWorkernodeUniqueLabelKey: "2-a",
										},
									},
									Spec: carbonv1.WorkerNodeSpec{ResVersion: "2",
										VersionPlan: *testNewVersionPlan("reserved", "docker", "64m", "regiest", false),
									},
								},
							},
						},
					},
					resVersionPlanMap: map[string]*carbonv1.VersionPlan{
						"1": testNewVersionPlan("share", "docker", "64m", "regiest", false),
						"2": testNewVersionPlan("reserved", "docker", "64m", "regiest", false),
					},
				},
			},
			wantErr: false,
			want: map[string][]*carbonv1.Replica{
				"1": {
					{
						WorkerNode: carbonv1.WorkerNode{
							ObjectMeta: metav1.ObjectMeta{
								Name: "1",
							},
							Spec: carbonv1.WorkerNodeSpec{
								ResVersion:  "1",
								VersionPlan: *testNewVersionPlan("share", "docker", "64m", "regiest", false),
							},
						},
					},
					{
						WorkerNode: carbonv1.WorkerNode{
							ObjectMeta: metav1.ObjectMeta{
								Name: "3",
							},
							Spec: carbonv1.WorkerNodeSpec{
								ResVersion:  "1",
								VersionPlan: *testNewVersionPlan("share", "docker", "64m", "regiest", false),
							},
						},
					},
				},
				"2": {
					{
						WorkerNode: carbonv1.WorkerNode{
							ObjectMeta: metav1.ObjectMeta{
								Name: "2",
								Labels: map[string]string{
									"replica-version-hash":                   "2",
									carbonv1.DefaultWorkernodeUniqueLabelKey: "2-a",
								},
							},
							Spec: carbonv1.WorkerNodeSpec{
								ResVersion:  "1",
								VersionPlan: *testNewVersionPlan("share", "docker", "64m", "regiest", false),
							},
						},
					},
				},
			},
		},
		{
			name: "not-update",
			fields: fields{
				rollingSet: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						ResVersion:  "1",
						VersionPlan: *testNewVersionPlan("share", "docker", "64m", "regiest", false),
					},
				},
			},
			args: args{
				rollingsetArgs: &rollingsetArgs{
					resVersionReplicaMap: map[string][]*carbonv1.Replica{
						"1": {
							{
								WorkerNode: carbonv1.WorkerNode{
									ObjectMeta: metav1.ObjectMeta{
										Name: "1",
									},
									Spec: carbonv1.WorkerNodeSpec{
										ResVersion:  "1",
										VersionPlan: *testNewVersionPlan("share", "docker", "64m", "regiest", false),
									},
								},
							},
							{
								WorkerNode: carbonv1.WorkerNode{
									ObjectMeta: metav1.ObjectMeta{
										Name: "3",
									},
									Spec: carbonv1.WorkerNodeSpec{
										ResVersion:  "1",
										VersionPlan: *testNewVersionPlan("share", "docker", "64m", "regiest", false),
									},
								},
							},
						},
						"2": {
							{
								WorkerNode: carbonv1.WorkerNode{
									ObjectMeta: metav1.ObjectMeta{
										Name: "2",
										Labels: map[string]string{
											carbonv1.DefaultReplicaUniqueLabelKey:    "2",
											carbonv1.DefaultWorkernodeUniqueLabelKey: "2-a",
										},
									},
									Spec: carbonv1.WorkerNodeSpec{ResVersion: "2",
										VersionPlan: *testNewVersionPlan("reserved", "docker", "64m", "regiest", true),
									},
								},
							},
						},
					},
					resVersionPlanMap: map[string]*carbonv1.VersionPlan{
						"1": testNewVersionPlan("share", "docker", "64m", "regiest", false),
						"2": testNewVersionPlan("reserved", "docker", "64m", "regiest", true),
					},
				},
			},
			wantErr: false,
			want: map[string][]*carbonv1.Replica{
				"1": {
					{
						WorkerNode: carbonv1.WorkerNode{
							ObjectMeta: metav1.ObjectMeta{
								Name: "1",
							},
							Spec: carbonv1.WorkerNodeSpec{
								ResVersion:  "1",
								VersionPlan: *testNewVersionPlan("share", "docker", "64m", "regiest", false),
							},
						},
					},
					{
						WorkerNode: carbonv1.WorkerNode{
							ObjectMeta: metav1.ObjectMeta{
								Name: "3",
							},
							Spec: carbonv1.WorkerNodeSpec{
								ResVersion:  "1",
								VersionPlan: *testNewVersionPlan("share", "docker", "64m", "regiest", false),
							},
						},
					},
				},
				"2": {
					{
						WorkerNode: carbonv1.WorkerNode{
							ObjectMeta: metav1.ObjectMeta{
								Name: "2",
								Labels: map[string]string{
									carbonv1.DefaultReplicaUniqueLabelKey:    "2",
									carbonv1.DefaultWorkernodeUniqueLabelKey: "2-a",
								},
							},
							Spec: carbonv1.WorkerNodeSpec{ResVersion: "2",
								VersionPlan: *testNewVersionPlan("reserved", "docker", "64m", "regiest", true),
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceScheduler{
				scheduleID:      tt.fields.scheduleID,
				rollingSet:      tt.fields.rollingSet,
				replicas:        tt.fields.replicas,
				resourceManager: tt.fields.resourceManager,
				generalLabels:   tt.fields.generalLabels,
				c:               &Controller{},
			}
			if err := s.updateReplicaResource(tt.args.rollingsetArgs); (err != nil) != tt.wantErr {
				t.Errorf("InPlaceScheduler.updateReplicaResource() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.rollingsetArgs.resVersionReplicaMap, tt.want) {
				t.Errorf("rollingsetArgs.resVersionRepliaMap gotCopy = %v, want %v", utils.ObjJSON(tt.args.rollingsetArgs.resVersionReplicaMap), utils.ObjJSON(tt.want))
			}
		})
	}
}

func testNewHippoPodSpec(cpumode, containernode, cpus, image string, ip bool) *carbonv1.HippoPodSpec {
	var hippoPodSpec = &carbonv1.HippoPodSpec{
		HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
			CpusetMode:     cpumode,
			ContainerModel: containernode,
		},
		Containers: []carbonv1.HippoContainer{
			{
				Container: corev1.Container{
					Name: "test",
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							"cpu": resource.MustParse(cpus),
						},
					},
					Image: image,
				},
			},
		},
	}
	if ip {
		hippoPodSpec.Containers[0].Container.Resources.Limits[carbonv1.ResourceIP] = resource.MustParse("1")
	}
	return hippoPodSpec
}

func Test_couldRollingByPass(t *testing.T) {
	type args struct {
		hippoPodSpec    *carbonv1.HippoPodSpec
		oldHippoPodSpec *carbonv1.HippoPodSpec
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "ipchanged",
			args: args{
				hippoPodSpec:    testNewHippoPodSpec("share", "docker", "64m", "regiest", false),
				oldHippoPodSpec: testNewHippoPodSpec("share", "docker", "64m", "regiest", true),
			},
			want: false,
		},
		{
			name: "ipnotchanged",
			args: args{
				hippoPodSpec:    testNewHippoPodSpec("share", "docker", "64m", "regiest", false),
				oldHippoPodSpec: testNewHippoPodSpec("share", "docker", "64m", "regiest", false),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := couldRollingByPass(tt.args.hippoPodSpec, tt.args.oldHippoPodSpec); got != tt.want {
				t.Errorf("couldRollingByPass() = %v, want %v", got, tt.want)

			}
		})
	}
}

func Test_appendReplicas(t *testing.T) {
	target := make([]*carbonv1.Replica, 0)
	target = append(target, &carbonv1.Replica{})
	replicas := make([]*carbonv1.Replica, 0)
	replicas = append(replicas, &carbonv1.Replica{})
	type args struct {
		target   []*carbonv1.Replica
		replicas []*carbonv1.Replica
	}
	tests := []struct {
		name string
		args args
		len  int
	}{
		{
			name: "test1",
			args: args{target: target, replicas: replicas},
			len:  2,
		},
		{
			name: "test2",
			args: args{target: nil, replicas: replicas},
			len:  1,
		},
		{
			name: "test2",
			args: args{target: target, replicas: nil},
			len:  1,
		},
		{
			name: "test2",
			args: args{target: nil, replicas: nil},
			len:  0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendReplicas(tt.args.target, tt.args.replicas)
			if len(got) != tt.len {
				t.Errorf("appendReplicas() = %v, want %v", got, tt.len)
			}
		})
	}
}

func newRollingsetArgsWithFinalPlan() *rollingsetArgs {
	rollingsetArgs := rollingsetArgs{}
	rollingsetArgs.ShardScheduleParams = &rollalgorithm.ShardScheduleParams{}
	rollingsetArgs.LatestVersion = "latest-version"
	rollingsetArgs.latestVersionPlan = &carbonv1.VersionPlanWithGangPlan{VersionPlan: &carbonv1.VersionPlan{}}
	replica1 := newReplicaWithFinalPlan("name1", "latest-version")
	replica2 := newReplicaWithFinalPlan("name2", "version1")
	replica3 := newReplicaWithFinalPlan("name3", "version1")
	replica4 := newReplicaWithFinalPlan("name4", "version2")
	replica5 := newReplicaWithFinalPlan("name5", "version2")
	replica6 := newReplicaWithFinalPlan("name6", "latest-version")
	replica7 := newReplicaWithFinalPlan("name7", "latest-version")
	replica8 := newReplicaWithFinalPlan("name8", "latest-version")
	replica9 := newReplicaWithFinalPlan("name9", "version2")
	replica9.Spec.ToDelete = true
	replica10 := newReplicaWithFinalPlan("name10", "version2")
	replica10.Status.AllocStatus = carbonv1.WorkerReleased
	replica11 := newReplicaWithFinalPlan("name11", "latest-version")
	// rollingsetArgs._resChangedReplicaSet=
	rollingsetArgs._resChangedReplicaSet = append(rollingsetArgs._resChangedReplicaSet, replica1)
	rollingsetArgs._resChangedReplicaSet = append(rollingsetArgs._resChangedReplicaSet, replica6)
	rollingsetArgs.supplementReplicaSet = append(rollingsetArgs.supplementReplicaSet, replica8)

	rollingsetArgs._replicaSet = append(rollingsetArgs._replicaSet, replica1) //u
	rollingsetArgs._replicaSet = append(rollingsetArgs._replicaSet, replica2)
	rollingsetArgs._replicaSet = append(rollingsetArgs._replicaSet, replica3)
	rollingsetArgs._replicaSet = append(rollingsetArgs._replicaSet, replica4)
	rollingsetArgs._replicaSet = append(rollingsetArgs._replicaSet, replica5)

	rollingsetArgs._replicaSet = append(rollingsetArgs._replicaSet, replica6) //u o
	rollingsetArgs._replicaSet = append(rollingsetArgs._replicaSet, replica7) //o
	rollingsetArgs._replicaSet = append(rollingsetArgs._replicaSet, replica8) //u o

	rollingsetArgs._replicaSet = append(rollingsetArgs._replicaSet, replica9)  // r
	rollingsetArgs._replicaSet = append(rollingsetArgs._replicaSet, replica10) // r
	rollingsetArgs._replicaSet = append(rollingsetArgs._replicaSet, replica11) //u

	return &rollingsetArgs
}

func newReplicaWithFinalPlan(name, finalVersion string) *carbonv1.Replica {

	replica := &carbonv1.Replica{
		WorkerNode: carbonv1.WorkerNode{
			TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				UID:       uuid.NewUUID(),
				Namespace: metav1.NamespaceDefault,
			},
			Spec: carbonv1.WorkerNodeSpec{
				Version: name,
				VersionPlan: carbonv1.VersionPlan{
					BroadcastPlan: carbonv1.BroadcastPlan{
						WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
							MinReadySeconds:      190,
							ResourceMatchTimeout: 2000,
							WorkerReadyTimeout:   20999,
						},
					},
				},
			},
			Status: carbonv1.WorkerNodeStatus{
				Complete: true,
				Score:    10,
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					Version: name,
				},
				ServiceStatus: carbonv1.ServiceAvailable,
			},
		},
	}
	return replica
}

func newRollingSetWithHoldMatrix(version string) *carbonv1.RollingSet {
	selector := map[string]string{"foo": "bar"}
	metaStr := `{"metadata":{"name":"nginx-deployment","labels":{"app": "nginx"}}}`
	var template carbonv1.HippoPodTemplate
	json.Unmarshal([]byte(metaStr), &template)
	rollingset := carbonv1.RollingSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        "rollingset-m",
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
		Spec: carbonv1.RollingSetSpec{
			Version: version,
			SchedulePlan: rollalgorithm.SchedulePlan{
				Strategy: apps.DeploymentStrategy{
					Type: apps.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &apps.RollingUpdateDeployment{
						MaxUnavailable: func() *intstr.IntOrString { i := intstr.FromInt(2); return &i }(),
						MaxSurge:       func() *intstr.IntOrString { i := intstr.FromInt(2); return &i }(),
					},
				},
				Replicas:           func() *int32 { i := int32(10); return &i }(),
				LatestVersionRatio: func() *int32 { i := int32(rollalgorithm.DefaultLatestVersionRatio); return &i }(),
				VersionHoldMatrix:  map[string]int32{"g-version1": 4},
			},

			VersionPlan: carbonv1.VersionPlan{
				SignedVersionPlan: carbonv1.SignedVersionPlan{
					Template: &template,
				},
				BroadcastPlan: carbonv1.BroadcastPlan{
					WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
						MinReadySeconds:      1024,
						ResourceMatchTimeout: 2000,
						WorkerReadyTimeout:   2014,
					},
				},
			},
			Selector: &metav1.LabelSelector{MatchLabels: selector},
		},
		Status: carbonv1.RollingSetStatus{},
	}
	return &rollingset
}

func TestInPlaceScheduler_getCompleteStatus(t *testing.T) {

	versionPlanMap1 := make(map[string]*carbonv1.VersionPlanWithGangPlan, 1)
	for i := 0; i < 1; i++ {
		versionPlanMap1["plan-"+strconv.Itoa(i)] = &carbonv1.VersionPlanWithGangPlan{VersionPlan: &carbonv1.VersionPlan{}}
	}

	versionPlanMap2 := make(map[string]*carbonv1.VersionPlanWithGangPlan, 2)
	for i := 0; i < 2; i++ {
		versionPlanMap2["plan-"+strconv.Itoa(i)] = &carbonv1.VersionPlanWithGangPlan{VersionPlan: &carbonv1.VersionPlan{}}
	}

	type fields struct {
		rsKey           string
		scheduleID      string
		rollingSet      *carbonv1.RollingSet
		replicas        []*carbonv1.Replica
		resourceManager *util.ResourceManager
		generalLabels   map[string]string
		expectations    *k8scontroller.UIDTrackingControllerExpectations
	}
	type args struct {
		rollingsetArgs *rollingsetArgs
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "test1",
			args: args{rollingsetArgs: &rollingsetArgs{}},
			want: false,
		},
		{
			name: "test2",
			args: args{rollingsetArgs: &rollingsetArgs{ShardScheduleParams: &rollalgorithm.ShardScheduleParams{
				ShardRuntimeArgs:  rollalgorithm.ShardRuntimeArgs{LatestVersion: "version-0"},
				ShardInternalArgs: rollalgorithm.ShardInternalArgs{Desired: 10}},
				_replicaSet: newReplicasOneVersion("version-0", true), versionPlanMap: versionPlanMap1}},
			want: true,
		},
		{
			name: "test3",
			args: args{rollingsetArgs: &rollingsetArgs{ShardScheduleParams: &rollalgorithm.ShardScheduleParams{
				ShardRuntimeArgs:  rollalgorithm.ShardRuntimeArgs{LatestVersion: "version-0"},
				ShardInternalArgs: rollalgorithm.ShardInternalArgs{Desired: 12}},
				_replicaSet: newReplicasOneVersion("version-0", true), versionPlanMap: versionPlanMap1}},
			want: false,
		},

		{
			name: "test4",
			args: args{rollingsetArgs: &rollingsetArgs{ShardScheduleParams: &rollalgorithm.ShardScheduleParams{
				ShardRuntimeArgs:  rollalgorithm.ShardRuntimeArgs{LatestVersion: "version-0"},
				ShardInternalArgs: rollalgorithm.ShardInternalArgs{Desired: 10}},
				_replicaSet: newReplicasOneVersion("version-0", true), versionPlanMap: versionPlanMap2}},
			want: false,
		},
		{
			name: "test5",
			args: args{rollingsetArgs: &rollingsetArgs{ShardScheduleParams: &rollalgorithm.ShardScheduleParams{
				ShardRuntimeArgs:  rollalgorithm.ShardRuntimeArgs{LatestVersion: "version-1"},
				ShardInternalArgs: rollalgorithm.ShardInternalArgs{Desired: 10}},
				_replicaSet: newReplicasOneVersion("version-0", true), versionPlanMap: versionPlanMap1}},
			want: false,
		},
		{
			name: "test6",
			args: args{rollingsetArgs: &rollingsetArgs{ShardScheduleParams: &rollalgorithm.ShardScheduleParams{
				ShardRuntimeArgs:  rollalgorithm.ShardRuntimeArgs{LatestVersion: "version-0"},
				ShardInternalArgs: rollalgorithm.ShardInternalArgs{Desired: 10}},
				_replicaSet: newReplicasWithVerisonPlan(true), versionPlanMap: versionPlanMap2}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceScheduler{
				rsKey:           tt.fields.rsKey,
				scheduleID:      tt.fields.scheduleID,
				rollingSet:      tt.fields.rollingSet,
				replicas:        tt.fields.replicas,
				resourceManager: tt.fields.resourceManager,
				generalLabels:   tt.fields.generalLabels,
				expectations:    tt.fields.expectations,
			}
			if got := s.getCompleteStatus(tt.args.rollingsetArgs); got != tt.want {
				t.Errorf("InPlaceScheduler.getCompleteStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInPlaceScheduler_getServiceReadyStatus(t *testing.T) {

	versionPlanMap1 := make(map[string]*carbonv1.VersionPlanWithGangPlan, 1)
	for i := 0; i < 1; i++ {
		versionPlanMap1["plan-"+strconv.Itoa(i)] = &carbonv1.VersionPlanWithGangPlan{VersionPlan: &carbonv1.VersionPlan{}}
	}

	versionPlanMap2 := make(map[string]*carbonv1.VersionPlanWithGangPlan, 2)
	for i := 0; i < 2; i++ {
		versionPlanMap2["plan-"+strconv.Itoa(i)] = &carbonv1.VersionPlanWithGangPlan{VersionPlan: &carbonv1.VersionPlan{}}
	}

	type fields struct {
		rsKey           string
		scheduleID      string
		rollingSet      *carbonv1.RollingSet
		replicas        []*carbonv1.Replica
		resourceManager *util.ResourceManager
		generalLabels   map[string]string
		expectations    *k8scontroller.UIDTrackingControllerExpectations
	}
	type args struct {
		rollingsetArgs *rollingsetArgs
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "test1",
			args: args{rollingsetArgs: &rollingsetArgs{}},
			want: false,
		},
		{
			name: "test2",
			args: args{rollingsetArgs: &rollingsetArgs{ShardScheduleParams: &rollalgorithm.ShardScheduleParams{
				ShardRuntimeArgs:  rollalgorithm.ShardRuntimeArgs{LatestVersion: "version-0"},
				ShardInternalArgs: rollalgorithm.ShardInternalArgs{Desired: 10}},
				_replicaSet: newReplicasOneVersion("version-0", true), versionPlanMap: versionPlanMap1}},
			want: true,
		},
		{
			name: "test3",
			args: args{rollingsetArgs: &rollingsetArgs{ShardScheduleParams: &rollalgorithm.ShardScheduleParams{
				ShardRuntimeArgs:  rollalgorithm.ShardRuntimeArgs{LatestVersion: "version-0"},
				ShardInternalArgs: rollalgorithm.ShardInternalArgs{Desired: 12}},
				_replicaSet: newReplicasOneVersion("version-0", true), versionPlanMap: versionPlanMap1}},
			want: false,
		},

		{
			name: "test4",
			args: args{rollingsetArgs: &rollingsetArgs{ShardScheduleParams: &rollalgorithm.ShardScheduleParams{
				ShardRuntimeArgs:  rollalgorithm.ShardRuntimeArgs{LatestVersion: "version-0"},
				ShardInternalArgs: rollalgorithm.ShardInternalArgs{Desired: 10}},
				_replicaSet: newReplicasOneVersion("version-0", true), versionPlanMap: versionPlanMap2}},
			want: false,
		},
		{
			name: "test5",
			args: args{rollingsetArgs: &rollingsetArgs{ShardScheduleParams: &rollalgorithm.ShardScheduleParams{
				ShardRuntimeArgs:  rollalgorithm.ShardRuntimeArgs{LatestVersion: "version-1"},
				ShardInternalArgs: rollalgorithm.ShardInternalArgs{Desired: 10}},
				_replicaSet: newReplicasOneVersion("version-0", true), versionPlanMap: versionPlanMap1}},
			want: false,
		},
		{
			name: "test6",
			args: args{rollingsetArgs: &rollingsetArgs{ShardScheduleParams: &rollalgorithm.ShardScheduleParams{
				ShardRuntimeArgs:  rollalgorithm.ShardRuntimeArgs{LatestVersion: "version-0"},
				ShardInternalArgs: rollalgorithm.ShardInternalArgs{Desired: 10}},
				_replicaSet: newReplicasWithVerisonPlan(true), versionPlanMap: versionPlanMap2}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceScheduler{
				rsKey:           tt.fields.rsKey,
				scheduleID:      tt.fields.scheduleID,
				rollingSet:      tt.fields.rollingSet,
				replicas:        tt.fields.replicas,
				resourceManager: tt.fields.resourceManager,
				generalLabels:   tt.fields.generalLabels,
				expectations:    tt.fields.expectations,
				c:               &Controller{},
			}
			if got := s.getServiceReadyStatus(tt.args.rollingsetArgs); got != tt.want {
				t.Errorf("InPlaceScheduler.getServiceReadyStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func newMockShardScheduler(t *testing.T, target map[string]int32) rollalgorithm.ShardScheduler {
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	scheduler := rollalgorithm.NewMockShardScheduler(ctl)
	scheduler.EXPECT().Schedule(gomock.Any(), gomock.Any()).Return(target, nil, nil).MinTimes(0)
	return scheduler
}

func TestInPlaceScheduler_calStandbyCountsByRatio(t *testing.T) {
	type args struct {
		standbys int
		target   map[carbonv1.WorkerModeType]int
		ratio    map[carbonv1.WorkerModeType]int
	}
	tests := []struct {
		name string
		args args
		want map[carbonv1.WorkerModeType]int
	}{
		{
			name: "hot & warm",
			args: args{
				standbys: 7,
				ratio: map[carbonv1.WorkerModeType]int{
					carbonv1.WorkerModeTypeCold: 20,
					carbonv1.WorkerModeTypeWarm: 30,
					carbonv1.WorkerModeTypeHot:  50,
				},
				target: map[carbonv1.WorkerModeType]int{},
			},
			want: map[carbonv1.WorkerModeType]int{
				carbonv1.WorkerModeTypeCold:     0,
				carbonv1.WorkerModeTypeWarm:     3,
				carbonv1.WorkerModeTypeHot:      4,
				carbonv1.WorkerModeTypeInactive: 0,
			},
		},
		{
			name: "all",
			args: args{
				standbys: 7,
				ratio: map[carbonv1.WorkerModeType]int{
					carbonv1.WorkerModeTypeCold: 20,
					carbonv1.WorkerModeTypeWarm: 30,
					carbonv1.WorkerModeTypeHot:  40,
				},
				target: map[carbonv1.WorkerModeType]int{},
			},
			want: map[carbonv1.WorkerModeType]int{
				carbonv1.WorkerModeTypeCold:     1,
				carbonv1.WorkerModeTypeWarm:     3,
				carbonv1.WorkerModeTypeHot:      3,
				carbonv1.WorkerModeTypeInactive: 0,
			},
		},
		{
			name: "hot only",
			args: args{
				standbys: 7,
				ratio: map[carbonv1.WorkerModeType]int{
					carbonv1.WorkerModeTypeCold: 0,
					carbonv1.WorkerModeTypeWarm: 10,
					carbonv1.WorkerModeTypeHot:  90,
				},
				target: map[carbonv1.WorkerModeType]int{},
			},
			want: map[carbonv1.WorkerModeType]int{
				carbonv1.WorkerModeTypeCold:     0,
				carbonv1.WorkerModeTypeWarm:     0,
				carbonv1.WorkerModeTypeHot:      7,
				carbonv1.WorkerModeTypeInactive: 0,
			},
		},
		{
			name: "inactive only",
			args: args{
				standbys: 7,
				ratio: map[carbonv1.WorkerModeType]int{
					carbonv1.WorkerModeTypeCold:     0,
					carbonv1.WorkerModeTypeWarm:     0,
					carbonv1.WorkerModeTypeHot:      0,
					carbonv1.WorkerModeTypeInactive: 100,
				},
				target: map[carbonv1.WorkerModeType]int{},
			},
			want: map[carbonv1.WorkerModeType]int{
				carbonv1.WorkerModeTypeCold:     0,
				carbonv1.WorkerModeTypeWarm:     0,
				carbonv1.WorkerModeTypeHot:      0,
				carbonv1.WorkerModeTypeInactive: 7,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceScheduler{}
			s.calStandbyCountsByRatio(tt.args.standbys, tt.args.ratio, tt.args.target)
			if !reflect.DeepEqual(tt.args.target, tt.want) {
				t.Errorf("calStandbyCountsByRatio want:%v actual:%v", tt.want, tt.args.target)
			}
		})
	}
}

func TestInPlaceScheduler_supplementSpotReplicas(t *testing.T) {
	type fields struct {
		rollingSet *carbonv1.RollingSet
	}
	type args struct {
		rollingsetArgs *rollingsetArgs
		redundants     []*carbonv1.Replica
		scheduleTimes  int
	}
	tests := []struct {
		name             string
		fields           fields
		args             args
		wantCurSpotNames []string
	}{
		{
			name:   "normal",
			fields: fields{rollingSet: newRollingSet("version-0")},
			args: args{
				rollingsetArgs: &rollingsetArgs{
					activeReplicas:  newSpotReplicasBase(10, false, true, false, "current-", nil),
					standByReplicas: newSpotReplicasBase(5, false, false, false, "spot-", nil),
					spotTarget:      5,
					supplements:     mapset.NewThreadUnsafeSet(),
				},
				redundants:    []*carbonv1.Replica{},
				scheduleTimes: 3,
			},
			wantCurSpotNames: []string{"spot-replica-0", "spot-replica-1", "spot-replica-2", "spot-replica-3", "spot-replica-4"},
		},
		{
			name:   "supply from candidates spot",
			fields: fields{rollingSet: newRollingSet("version-0")},
			args: args{
				rollingsetArgs: &rollingsetArgs{
					activeReplicas: newSpotReplicasBase(10, false, true, false, "current-", nil),
					standByReplicas: func() []*carbonv1.Replica {
						current := newSpotReplicasBase(5, false, true, false, "current-", nil)
						return newSpotReplicasBase(5, false, false, false, "spot-", current)
					}(),
					spotTarget:  5,
					supplements: mapset.NewThreadUnsafeSet(),
				},
				redundants:    []*carbonv1.Replica{},
				scheduleTimes: 3,
			},
			wantCurSpotNames: []string{"spot-replica-0", "spot-replica-1", "spot-replica-2", "spot-replica-3", "spot-replica-4"},
		},
		{
			name:   "release spot without redundants",
			fields: fields{rollingSet: newRollingSet("version-0")},
			args: args{
				rollingsetArgs: &rollingsetArgs{
					activeReplicas: newSpotReplicasBase(10, false, true, false, "current-", nil),
					standByReplicas: func() []*carbonv1.Replica {
						current := newSpotReplicasBase(5, false, true, false, "current-", nil)
						return newSpotReplicasBase(5, true, false, false, "spot-", current)
					}(),
					spotTarget:  0,
					supplements: mapset.NewThreadUnsafeSet(),
				},
				redundants:    []*carbonv1.Replica{},
				scheduleTimes: 3,
			},
			wantCurSpotNames: []string{"spot-replica-0", "spot-replica-1", "spot-replica-2", "spot-replica-3", "spot-replica-4"},
		},
		{
			name:   "release spot with redundants",
			fields: fields{rollingSet: newRollingSet("version-0")},
			args: args{
				rollingsetArgs: &rollingsetArgs{
					activeReplicas: newSpotReplicasBase(10, false, true, false, "current-", nil),
					standByReplicas: func() []*carbonv1.Replica {
						current := newSpotReplicasBase(5, false, true, false, "current-", nil)
						return newSpotReplicasBase(5, true, false, false, "spot-", current)
					}(),
					spotTarget: 0,
				},
				redundants: func() []*carbonv1.Replica {
					redundants := newSpotReplicasBase(10, false, true, false, "redundants-", nil)
					return redundants
				}(),
				scheduleTimes: 3,
			},
			wantCurSpotNames: []string{},
		},
		{
			name:   "invalid spot",
			fields: fields{rollingSet: newRollingSet("version-0")},
			args: args{
				rollingsetArgs: &rollingsetArgs{
					activeReplicas: newSpotReplicasBase(10, false, true, false, "current-", nil),
					standByReplicas: func() []*carbonv1.Replica {
						invalid := newSpotReplicasBase(5, false, true, false, "invalid-", nil)
						invalidSpot := newSpotReplicasBase(5, true, false, false, "spot-", invalid)
						return appendReplicas(invalidSpot, newSpotReplicasBase(5, false, false, false, "creating-", invalid))
					}(),
					spotTarget: 5,
				},
				redundants:    []*carbonv1.Replica{},
				scheduleTimes: 3,
			},
			wantCurSpotNames: []string{"creating-replica-0", "creating-replica-1", "creating-replica-2", "creating-replica-3", "creating-replica-4"},
		},
		{
			name:   "candidates has backup",
			fields: fields{rollingSet: newRollingSet("version-0")},
			args: args{
				rollingsetArgs: &rollingsetArgs{
					activeReplicas: newSpotReplicasBase(10, false, true, true, "current-", nil),
					standByReplicas: func() []*carbonv1.Replica {
						return newSpotReplicasBase(5, false, false, false, "creating-", nil)
					}(),
					spotTarget: 5,
				},
				redundants:    []*carbonv1.Replica{},
				scheduleTimes: 3,
			},
			wantCurSpotNames: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceScheduler{
				rollingSet: tt.fields.rollingSet,
			}
			tt.args.rollingsetArgs.supplements = mapset.NewThreadUnsafeSet()
			for i := 0; i < tt.args.scheduleTimes; i++ {
				tt.args.rollingsetArgs.curSpotNames = []string{}
				s.syncSpotReplicas(tt.args.rollingsetArgs, tt.args.redundants)
			}
			if len(tt.args.rollingsetArgs.curSpotNames) != len(tt.wantCurSpotNames) {
				t.Errorf("calc SpotNames error = %v, want %v", tt.args.rollingsetArgs.curSpotNames, tt.wantCurSpotNames)
			}
		})
	}
}

func newSpotReplicasBase(count int, spot, allocated, backup bool, namePrefix string, backupOf []*carbonv1.Replica) []*carbonv1.Replica {
	replicas := make([]*carbonv1.Replica, count)
	for i := 0; i < count; i++ {
		entityName := ""
		entityUid := ""
		resourceVersion := ""
		allocStatus := carbonv1.WorkerUnAssigned
		if allocated {
			entityUid = namePrefix + "entityUid-" + strconv.Itoa(i)
			entityName = namePrefix + "entityName-" + strconv.Itoa(i)
			resourceVersion = "resourceVersion-" + strconv.Itoa(i)
			allocStatus = carbonv1.WorkerAssigned
		}
		backupOfPod := carbonv1.BackupOfPod{}
		if i < len(backupOf) {
			backupOfPod = carbonv1.BackupOfPod{
				Name: backupOf[i].Status.Name,
				Uid:  backupOf[i].Status.EntityUid,
			}
		}
		replica := &carbonv1.Replica{
			WorkerNode: carbonv1.WorkerNode{
				TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:            namePrefix + "replica-" + strconv.Itoa(i),
					UID:             uuid.NewUUID(),
					Namespace:       metav1.NamespaceDefault,
					ResourceVersion: resourceVersion,
				},
				Spec: carbonv1.WorkerNodeSpec{
					Version: "version-01",
					VersionPlan: carbonv1.VersionPlan{
						BroadcastPlan: carbonv1.BroadcastPlan{
							WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
								MinReadySeconds:      190,
								ResourceMatchTimeout: 2000,
								WorkerReadyTimeout:   20999,
							},
						},
					},
					IsSpot:      spot,
					BackupOfPod: backupOfPod,
				},
				Status: carbonv1.WorkerNodeStatus{
					Complete:    true,
					AllocStatus: allocStatus,
					AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
						EntityAlloced: allocated,
						EntityUid:     entityUid,
						EntityName:    entityName,
					},
				},
			},
		}
		if backup {
			replica.Backup.Spec.BackupOfPod.Uid = replica.Status.AllocatorSyncedStatus.EntityUid
			replica.Backup.Spec.BackupOfPod.Name = replica.Status.AllocatorSyncedStatus.EntityName
		}
		replicas[i] = replica
	}
	return replicas
}

func newStandbyReplica(workerMode carbonv1.WorkerModeType, standbyHours []int64) *carbonv1.Replica {
	replica := &carbonv1.Replica{
		WorkerNode: carbonv1.WorkerNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: "asa",
			},
			Spec: carbonv1.WorkerNodeSpec{
				WorkerMode: workerMode,
			},
			Status: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					StandbyHours: utils.ObjJSON(standbyHours),
					DeletionCost: int64(1),
				},
			},
		},
	}
	return replica
}

func Test_ActiveStandbyStatusChangeLog(t *testing.T) {

	tests := []struct {
		name       string
		nowHour    int64
		replica    *carbonv1.Replica
		targetMode carbonv1.WorkerModeType
		want       bool
	}{
		{
			name:       "active->standby",
			nowHour:    23,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeActive, []int64{0, 1, 2}),
			targetMode: carbonv1.WorkerModeTypeWarm,
			want:       true,
		},
		{
			name:       "active->standby2",
			nowHour:    11,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeActive, []int64{0, 1, 2, 12, 13}),
			targetMode: carbonv1.WorkerModeTypeWarm,
			want:       true,
		},
		{
			name:       "active->standby3",
			nowHour:    0,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeActive, []int64{1, 2, 12, 13}),
			targetMode: carbonv1.WorkerModeTypeWarm,
			want:       true,
		},
		{
			name:       "active->standby4",
			nowHour:    0,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeActive, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}),
			targetMode: carbonv1.WorkerModeTypeWarm,
			want:       true,
		},
		{
			name:       "active->standby5",
			nowHour:    11,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeActive, []int64{12}),
			targetMode: carbonv1.WorkerModeTypeWarm,
			want:       true,
		},
		{
			name:       "active->standby6",
			nowHour:    0,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeActive, []int64{1}),
			targetMode: carbonv1.WorkerModeTypeWarm,
			want:       true,
		},
		{
			name:       "active->standby7",
			nowHour:    0,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeActive, []int64{0}),
			targetMode: carbonv1.WorkerModeTypeWarm,
			want:       false,
		},
		{
			name:       "active->standby8",
			nowHour:    23,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeActive, []int64{0}),
			targetMode: carbonv1.WorkerModeTypeWarm,
			want:       true,
		},
		{
			name:       "active->standby9",
			nowHour:    23,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeActive, []int64{22}),
			targetMode: carbonv1.WorkerModeTypeWarm,
			want:       false,
		},
		{
			name:       "active->active",
			nowHour:    0,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeActive, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}),
			targetMode: carbonv1.WorkerModeTypeActive,
			want:       false,
		},
		{
			name:       "standby->standby",
			nowHour:    0,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeWarm, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}),
			targetMode: carbonv1.WorkerModeTypeWarm,
			want:       false,
		},
		{
			name:       "standby->active",
			nowHour:    0,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeWarm, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}),
			targetMode: carbonv1.WorkerModeTypeActive,
			want:       true,
		},
		{
			name:       "standby->active2",
			nowHour:    11,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeWarm, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}),
			targetMode: carbonv1.WorkerModeTypeActive,
			want:       true,
		},
		{
			name:       "standby->active3",
			nowHour:    11,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeWarm, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}),
			targetMode: carbonv1.WorkerModeTypeActive,
			want:       false,
		},
		{
			name:       "standby->active4",
			nowHour:    11,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeWarm, []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23}),
			targetMode: carbonv1.WorkerModeTypeActive,
			want:       false,
		},
		{
			name:       "standby->active5",
			nowHour:    23,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeWarm, []int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22}),
			targetMode: carbonv1.WorkerModeTypeActive,
			want:       true,
		},
		{
			name:       "standby->active6",
			nowHour:    22,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeWarm, []int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21}),
			targetMode: carbonv1.WorkerModeTypeActive,
			want:       true,
		},
		{
			name:       "standby->active7",
			nowHour:    11,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeWarm, []int64{10}),
			targetMode: carbonv1.WorkerModeTypeActive,
			want:       true,
		},
		{
			name:       "standby->active8",
			nowHour:    11,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeWarm, []int64{}),
			targetMode: carbonv1.WorkerModeTypeActive,
			want:       false,
		},
		{
			name:       "standby->active9",
			nowHour:    11,
			replica:    newStandbyReplica(carbonv1.WorkerModeTypeWarm, nil),
			targetMode: carbonv1.WorkerModeTypeActive,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transformation := activeStandbyStatusChangeLog(tt.replica, tt.targetMode, tt.nowHour)
			assert.True(t, transformation == tt.want)
		})
	}
}

func TestInPlaceScheduler_checkRSValidate(t *testing.T) {
	type fields struct {
		rsKey              string
		scheduleID         string
		rollingSet         *carbonv1.RollingSet
		groupRollingSet    *carbonv1.RollingSet
		rollingSetStatus   *carbonv1.RollingSetStatus
		replicas           []*carbonv1.Replica
		resourceManager    *util.ResourceManager
		generalLabels      map[string]string
		generalAnnotations map[string]string
		expectations       *k8scontroller.UIDTrackingControllerExpectations
		scheduler          rollalgorithm.ShardScheduler
		c                  *Controller
	}
	type args struct {
		rs    *carbonv1.RollingSet
		vmaps map[string]*rollalgorithm.VersionStatus
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				rs: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								ShardGroupVersion: "10-adfal",
							},
						},
					},
				},
				vmaps: map[string]*rollalgorithm.VersionStatus{
					"5-adfd":  nil,
					"6-sadfd": nil,
					"7-gdsfd": nil,
				},
			},
			wantErr: false,
		},
		{
			name: "error",
			args: args{
				rs: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								ShardGroupVersion: "10-adfal",
							},
						},
					},
				},
				vmaps: map[string]*rollalgorithm.VersionStatus{
					"5-adfd":   nil,
					"6-sadfd":  nil,
					"17-gdsfd": nil,
				},
			},
			wantErr: true,
		},
		{
			name: "normal",
			args: args{
				rs: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						Version: "10-adfal",
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{},
						},
					},
				},
				vmaps: map[string]*rollalgorithm.VersionStatus{
					"5-adfd":  nil,
					"6-sadfd": nil,
					"7-gdsfd": nil,
				},
			},
			wantErr: false,
		},
		{
			name: "error",
			args: args{
				rs: &carbonv1.RollingSet{
					Spec: carbonv1.RollingSetSpec{
						Version: "10-adfal",
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{},
						},
					},
				},
				vmaps: map[string]*rollalgorithm.VersionStatus{
					"5-adfd":   nil,
					"6-sadfd":  nil,
					"17-gdsfd": nil,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceScheduler{
				rsKey:              tt.fields.rsKey,
				scheduleID:         tt.fields.scheduleID,
				rollingSet:         tt.fields.rollingSet,
				groupRollingSet:    tt.fields.groupRollingSet,
				rollingSetStatus:   tt.fields.rollingSetStatus,
				replicas:           tt.fields.replicas,
				resourceManager:    tt.fields.resourceManager,
				generalLabels:      tt.fields.generalLabels,
				generalAnnotations: tt.fields.generalAnnotations,
				expectations:       tt.fields.expectations,
				c:                  tt.fields.c,
			}
			if err := s.validateRsVersion(tt.args.rs, tt.args.vmaps, nil); (err != nil) != tt.wantErr {
				t.Errorf("InPlaceScheduler.checkRSValidate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func newResourcePoolReplica(cost int64, pool, hours string) *carbonv1.Replica {
	return &carbonv1.Replica{
		WorkerNode: carbonv1.WorkerNode{
			Spec: carbonv1.WorkerNodeSpec{
				DeletionCost: cost,
			},
			Status: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					DeletionCost: cost,
					ResourcePool: pool,
					StandbyHours: hours,
				},
			},
		},
	}
}

func Test_calcStandbyHours(t *testing.T) {
	type args struct {
		fixedReplicas    []*carbonv1.Replica
		cyclicalReplicas []*carbonv1.Replica
		predictInfo      map[int64]int64
	}
	tests := []struct {
		name             string
		args             args
		wantStandbyHours map[int64][]int64
	}{
		{
			name: "normal",
			args: args{
				fixedReplicas: []*carbonv1.Replica{
					newResourcePoolReplica(9998, "Fixed", ""),
					newResourcePoolReplica(9996, "Fixed", ""),
					newResourcePoolReplica(9997, "Fixed", ""),
					newResourcePoolReplica(10001, "Cyclical", ""),
				},
				cyclicalReplicas: []*carbonv1.Replica{
					newResourcePoolReplica(-100, "Fixed", "[2,3,4]"),
					newResourcePoolReplica(4998, "Cyclical", ""),
				},
				predictInfo: map[int64]int64{
					2: 2,
					3: 2,
					4: 4,
					5: 6,
				},
			},
			wantStandbyHours: map[int64][]int64{
				10001: {},
				9998:  {},
				9997:  {2, 3},
				9996:  {2, 3},
				4998:  {2, 3, 4},
				-100:  {},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InPlaceScheduler{}
			r := rollingsetArgs{
				predictInfo:      tt.args.predictInfo,
				fixedReplicas:    tt.args.fixedReplicas,
				cyclicalReplicas: tt.args.cyclicalReplicas,
				supplements:      mapset.NewThreadUnsafeSet(),
				fixedCount:       int64(len(tt.args.fixedReplicas)),
			}
			s.calcStandbyHours(&r)
			for _, replica := range carbonv1.MergeReplica(r.fixedReplicas, r.cyclicalReplicas) {
				cost := carbonv1.GetDeletionCost(replica)
				want := tt.wantStandbyHours[cost]
				if want == nil {
					t.Errorf("calcStandbyHours error cost:%d want nil", cost)
				}
				if len(want) == 0 && replica.Spec.StandbyHours == "" {
					continue
				}
				var hours []int64
				err := json.Unmarshal([]byte(replica.Spec.StandbyHours), &hours)
				assert.Nil(t, err)
				if !carbonv1.SliceEquals(want, hours) {
					t.Errorf("calcStandbyHours error, want:%v, got:%v", want, hours)
				}
			}
		})
	}
}

func newFieldRollingSetBySchedule3(version string, replicas int, maxSurge int, maxUnavailable int, latestVersionRatio int, holdMatrix map[string]float64, paddedRatio int) *carbonv1.RollingSet {
	hold := map[string]int32{}
	for key, val := range holdMatrix {
		hold[key] = int32(math.Floor(float64(val)*float64(replicas)-0.5)/100 + 0.7)
	}
	selector := map[string]string{"foo": "bar"}
	metaStr := `{"metadata":{"name":"nginx-deployment","labels":{"app": "nginx"}}}`
	var template carbonv1.HippoPodTemplate
	json.Unmarshal([]byte(metaStr), &template)
	rollingset := carbonv1.RollingSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        "rollingset-m",
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
		Spec: carbonv1.RollingSetSpec{
			Version: version,
			SchedulePlan: rollalgorithm.SchedulePlan{
				Strategy: apps.DeploymentStrategy{
					Type: apps.RollingUpdateDeploymentStrategyType,
					RollingUpdate: &apps.RollingUpdateDeployment{
						MaxUnavailable: func() *intstr.IntOrString { i := intstr.FromInt(maxUnavailable); return &i }(),
						MaxSurge:       func() *intstr.IntOrString { i := intstr.FromInt(maxSurge); return &i }(),
					},
				},
				Replicas:                 func() *int32 { i := int32(replicas); return &i }(),
				LatestVersionRatio:       func() *int32 { i := int32(latestVersionRatio); return &i }(),
				VersionHoldMatrix:        hold,
				VersionHoldMatrixPercent: holdMatrix,
				PaddedLatestVersionRatio: func() *int32 { i := int32(paddedRatio); return &i }(),
			},

			VersionPlan: carbonv1.VersionPlan{
				SignedVersionPlan: carbonv1.SignedVersionPlan{
					Template:          &template,
					ShardGroupVersion: version + "-" + version + version,
				},
			},
			Selector: &metav1.LabelSelector{MatchLabels: selector},
		},
		Status: carbonv1.RollingSetStatus{},
	}
	return &rollingset
}

func newReplicasOneVersionWithMode1(version string, status string, dependency bool, count int) []*carbonv1.Replica {
	replicas := make([]*carbonv1.Replica, count)
	for i := 0; i < count; i++ {
		replica := &carbonv1.Replica{
			WorkerNode: carbonv1.WorkerNode{
				TypeMeta: metav1.TypeMeta{Kind: "ReplicaSet"},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "replica-" + version + "-" + status + strconv.Itoa(i),
					UID:       uuid.NewUUID(),
					Namespace: metav1.NamespaceDefault,
				},
				Spec: carbonv1.WorkerNodeSpec{
					WorkerMode:      carbonv1.WorkerModeTypeActive,
					Version:         version,
					DependencyReady: utils.BoolPtr(dependency),
					VersionPlan: carbonv1.VersionPlan{
						BroadcastPlan: carbonv1.BroadcastPlan{
							WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
								MinReadySeconds:      0,
								ResourceMatchTimeout: 2000,
								WorkerReadyTimeout:   2014,
								ProcessMatchTimeout:  600,
							},
						},
						SignedVersionPlan: carbonv1.SignedVersionPlan{
							ShardGroupVersion: version + "-" + version + version,
						},
					},
				},
				Status: carbonv1.WorkerNodeStatus{
					AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
						Version:    version,
						WorkerMode: carbonv1.WorkerModeTypeActive,
					},
				},
			},
		}
		switch status {
		case "ready":
			setReplicaReady(replica)
		case "assigned":
			setReplicaAssigned(replica)
		case "release":
			setReplicaRelease(replica)
		case "hot":
			setReplicaStandby(replica, carbonv1.WorkerModeTypeHot)
		case "warm":
			setReplicaStandby(replica, carbonv1.WorkerModeTypeWarm)
		case "cold":
			setReplicaStandby(replica, carbonv1.WorkerModeTypeCold)
		case "inactive":
			setReplicaStandby(replica, carbonv1.WorkerModeTypeInactive)
		}
		replicas[i] = replica
	}
	return replicas
}

func setReplicaReady(replica *carbonv1.Replica) {
	replica.Status.ServiceReady = true
	replica.Status.AllocStatus = carbonv1.WorkerAssigned
	replica.Status.Phase = carbonv1.Running
	replica.Status.ServiceStatus = carbonv1.ServiceAvailable
	replica.Status.HealthStatus = carbonv1.HealthAlive
	replica.Status.ProcessReady = true
	replica.Status.ProcessMatch = true
	replica.Status.HealthCondition.Version = replica.Spec.Version
	replica.Status.WorkerReady = true
}

func setReplicaAssigned(replica *carbonv1.Replica) {
	replica.Status.AllocStatus = carbonv1.WorkerAssigned
}

func setReplicaRelease(replica *carbonv1.Replica) {
	replica.Spec.ToDelete = true
}

func setReplicaStandby(replica *carbonv1.Replica, mode carbonv1.WorkerModeType) {
	replica.Spec.WorkerMode = mode
	replica.Status.WorkerMode = mode
	setReplicaAssigned(replica)
}

func TestInPlaceScheduler_adjust(t *testing.T) {
	type fields struct {
		rsKey              string
		scheduleID         string
		rollingSet         *carbonv1.RollingSet
		groupRollingSet    *carbonv1.RollingSet
		rollingSetStatus   *carbonv1.RollingSetStatus
		replicas           []*carbonv1.Replica
		resourceManager    *util.ResourceManager
		generalLabels      map[string]string
		generalAnnotations map[string]string
		expectations       *k8scontroller.UIDTrackingControllerExpectations
		nodescheduler      rollalgorithm.RollingUpdater
		c                  *Controller
	}
	getInPlaceScheduler := func(f fields) *InPlaceScheduler {
		s := InPlaceScheduler{
			rsKey:              f.rsKey,
			scheduleID:         f.scheduleID,
			rollingSet:         f.rollingSet.DeepCopy(),
			groupRollingSet:    f.groupRollingSet,
			rollingSetStatus:   f.rollingSetStatus,
			replicas:           make([]*carbonv1.Replica, 0),
			resourceManager:    f.resourceManager,
			generalLabels:      f.generalLabels,
			generalAnnotations: f.generalAnnotations,
			expectations:       f.expectations,
			nodescheduler:      f.nodescheduler,
			c:                  f.c,
		}
		for _, replica := range f.replicas {
			s.replicas = append(s.replicas, replica.DeepCopy())
		}
		return &s
	}
	mergeReplicas := func(list [][]*carbonv1.Replica) []*carbonv1.Replica {
		replicas := make([]*carbonv1.Replica, 0)
		for _, rs := range list {
			replicas = append(replicas, rs...)
		}
		return replicas
	}
	getNames := func(replicas []*carbonv1.Replica) (name []string) {
		for _, replica := range replicas {
			name = append(name, replica.Name)
		}
		sort.Strings(name)
		return
	}
	tests := []struct {
		name          string
		fields        fields
		want          int
		wantName      []string
		want1         int
		wantErr       bool
		wantSpot      int64
		wantBroadcast *int64
	}{
		{
			name: "451to453-1",
			fields: fields{
				rollingSet: newFieldRollingSetBySchedule3("453", 10, 2, 2, 100, nil, 0),
				replicas: mergeReplicas([][]*carbonv1.Replica{
					newReplicasOneVersionWithMode1("451", "ready", true, 10),
				}),
				nodescheduler: rollalgorithm.NewDefaultNodeSetUpdater(nil),
			},
			want:  0,
			want1: 2,
		},
		{
			name: "451to453-2",
			fields: fields{
				rollingSet: newFieldRollingSetBySchedule3("453", 10, 2, 2, 100, nil, 0),
				replicas: mergeReplicas([][]*carbonv1.Replica{
					newReplicasOneVersionWithMode1("451", "ready", true, 8),
					newReplicasOneVersionWithMode1("453", "ready", true, 2),
				}),
				nodescheduler: rollalgorithm.NewDefaultNodeSetUpdater(nil),
			},
			want:  0,
			want1: 2,
		},
		{
			name: "451to453-3",
			fields: fields{
				rollingSet: newFieldRollingSetBySchedule3("453", 10, 2, 2, 100, nil, 0),
				replicas: mergeReplicas([][]*carbonv1.Replica{
					newReplicasOneVersionWithMode1("451", "ready", true, 6),
					newReplicasOneVersionWithMode1("453", "ready", true, 4),
				}),
				nodescheduler: rollalgorithm.NewDefaultNodeSetUpdater(nil),
			},
			want:  0,
			want1: 2,
		},
		{
			name: "target version back",
			fields: fields{
				rollingSet: newFieldRollingSetBySchedule3("453", 10, 2, 2, 100, nil, 0),
				replicas: mergeReplicas([][]*carbonv1.Replica{
					newReplicasOneVersionWithMode1("451", "ready", true, 6),
					newReplicasOneVersionWithMode1("455", "ready", true, 4),
				}),
				nodescheduler: rollalgorithm.NewDefaultNodeSetUpdater(nil),
			},
			want:    0,
			want1:   0,
			wantErr: true,
		},
		{
			name: "reduce1",
			fields: fields{
				rollingSet: newFieldRollingSetBySchedule3("451", 4, 2, 2, 100, nil, 0),
				replicas: mergeReplicas([][]*carbonv1.Replica{
					newReplicasOneVersionWithMode1("451", "ready", true, 4),
					newReplicasOneVersionWithMode1("451", "empty", true, 2),
				}),
				nodescheduler: rollalgorithm.NewDefaultNodeSetUpdater(nil),
			},
			want:     2,
			wantName: []string{"replica-451-empty0", "replica-451-empty1"},
			want1:    0,
		},
		{
			name: "reduce2",
			fields: fields{
				rollingSet: newFieldRollingSetBySchedule3("451", 4, 2, 2, 100, nil, 0),
				replicas: mergeReplicas([][]*carbonv1.Replica{
					newReplicasOneVersionWithMode1("451", "ready", true, 3),
					newReplicasOneVersionWithMode1("451", "assigned", true, 1),
					newReplicasOneVersionWithMode1("451", "empty", true, 1),
					newReplicasOneVersionWithMode1("451", "release", true, 1),
				}),
				nodescheduler: rollalgorithm.NewDefaultNodeSetUpdater(nil),
			},
			want:     1,
			wantName: []string{"replica-451-empty0"},
			want1:    0,
		},
		{
			name: "expand1",
			fields: fields{
				rollingSet: newFieldRollingSetBySchedule3("451", 4, 2, 2, 100, nil, 0),
				replicas: mergeReplicas([][]*carbonv1.Replica{
					newReplicasOneVersionWithMode1("451", "ready", true, 3),
				}),
				nodescheduler: rollalgorithm.NewDefaultNodeSetUpdater(nil),
			},
			want:  0,
			want1: 1,
		},
		{
			name: "expand2",
			fields: fields{
				rollingSet: newFieldRollingSetBySchedule3("451", 4, 2, 2, 100, nil, 0),
				replicas: mergeReplicas([][]*carbonv1.Replica{
					newReplicasOneVersionWithMode1("451", "ready", true, 3),
					newReplicasOneVersionWithMode1("451", "hot", true, 1),
				}),
				nodescheduler: rollalgorithm.NewDefaultNodeSetUpdater(nil),
			},
			want:  0,
			want1: 1,
		},
		{
			name: "expand_standby1",
			fields: fields{
				rollingSet: newFieldRollingSetBySchedule3("451", 6, 2, 2, 100, nil, 0),
				replicas: mergeReplicas([][]*carbonv1.Replica{
					newReplicasOneVersionWithMode1("451", "ready", true, 2),
					newReplicasOneVersionWithMode1("451", "hot", true, 1),
					newReplicasOneVersionWithMode1("451", "warm", true, 1),
					newReplicasOneVersionWithMode1("451", "cold", true, 1),
				}),
				nodescheduler: rollalgorithm.NewDefaultNodeSetUpdater(nil),
			},
			want:  0,
			want1: 4,
		},
		{
			name: "replace1",
			fields: fields{
				rollingSet: newFieldRollingSetBySchedule3("451", 4, 2, 2, 100, nil, 0),
				replicas: mergeReplicas([][]*carbonv1.Replica{
					newReplicasOneVersionWithMode1("451", "ready", true, 3),
					newReplicasOneVersionWithMode1("451", "empty", true, 1),
					newReplicasOneVersionWithMode1("451", "hot", true, 1),
				}),
				nodescheduler: rollalgorithm.NewDefaultNodeSetUpdater(nil),
			},
			want:     1,
			wantName: []string{"replica-451-empty0"},
			want1:    1,
		},
		{
			name: "group1",
			fields: fields{
				rollingSet: newFieldRollingSetBySchedule3("458", 6, 1, 1, 25, map[string]float64{
					"453-453453": 50,
					"455-455455": 16,
					"457-457457": 0,
					"458-458458": 25,
				}, 34),
				replicas: mergeReplicas([][]*carbonv1.Replica{
					newReplicasOneVersionWithMode1("453", "ready", true, 3),
					newReplicasOneVersionWithMode1("455", "ready", true, 1),
					newReplicasOneVersionWithMode1("457", "assigned", true, 1),
					newReplicasOneVersionWithMode1("458", "assigned", true, 1),
				}),
				nodescheduler: rollalgorithm.NewDefaultNodeSetUpdater(nil),
			},
			want:  0,
			want1: 1,
		},
		{
			name: "adjust collect",
			fields: fields{
				rollingSet: newRollingSet("version-1234"),
				replicas: mergeReplicas([][]*carbonv1.Replica{
					newReplicasOneVersion("version-0", true),
					newReplicasOneVersion("version-1", true),
					newReplicasOneVersion("version-2", true),
					newReplicasOneVersion("version-3", true),
				}),
				nodescheduler: rollalgorithm.NewNodeSetUpdater(newMockShardScheduler(t,
					map[string]int32{
						"version-0": 1,
						"version-1": 9,
						"version-2": 5,
						"version-3": 7,
					}), nil),
			},
			want:     18,
			want1:    0,
			wantSpot: 12,
		},
		{
			name: "adjust collect and supplement",
			fields: fields{
				rollingSet: newRollingSet("version-1234"),
				replicas: mergeReplicas([][]*carbonv1.Replica{
					newReplicasOneVersion("version-0", true),
					newReplicasOneVersion("version-1", true),
					newReplicasOneVersion("version-2", true),
					newReplicasOneVersion("version-3", true),
				}),
				nodescheduler: rollalgorithm.NewNodeSetUpdater(newMockShardScheduler(t,
					map[string]int32{
						"version-0": 1,
						"version-1": 19,
						"version-2": 5,
						"version-3": 15,
					}), nil),
			},
			want:     0,
			want1:    14,
			wantSpot: 30,
		},
		{
			name: "adjust supplement",
			fields: fields{
				rollingSet: newRollingSet("version-1234"),
				replicas: mergeReplicas([][]*carbonv1.Replica{
					newReplicasOneVersion("version-0", true),
					newReplicasOneVersion("version-1", true),
					newReplicasOneVersion("version-2", true),
					newReplicasOneVersion("version-3", true),
				}),
				nodescheduler: rollalgorithm.NewNodeSetUpdater(newMockShardScheduler(t,
					map[string]int32{
						"version-0": 11,
						"version-1": 19,
						"version-2": 15,
						"version-3": 15,
					}), nil),
			},
			want:     0,
			want1:    20,
			wantSpot: 50,
		},
		{
			name: "spot target",
			fields: fields{
				rollingSet: func() *carbonv1.RollingSet {
					rs := newRollingSet("version-1234")
					rs.Spec.Replicas = utils.Int32Ptr(10)
					rs.Spec.ScaleSchedulePlan = &carbonv1.ScaleSchedulePlan{
						Replicas: utils.Int32Ptr(15),
					}
					return rs
				}(),
				replicas: mergeReplicas([][]*carbonv1.Replica{
					newReplicasOneVersion("version-0", true),
					newReplicasOneVersion("version-1", true),
				}),
				nodescheduler: rollalgorithm.NewNodeSetUpdater(newMockShardScheduler(t,
					map[string]int32{
						"version-0": 10,
						"version-1": 11,
					}), nil),
			},
			want:     0,
			want1:    1,
			wantSpot: 6,
		},
		{
			name: "broadcast sync nonVersionedKeys",
			fields: fields{
				rollingSet: func() *carbonv1.RollingSet {
					rs := newFieldRollingSetBySchedule3("453", 10, 2, 2, 100, nil, 0)
					rs.Spec.BroadcastPlan.ResourceMatchTimeout = 2000
					rs.Spec.BroadcastPlan.WorkerReadyTimeout = 2014
					rs.Spec.BroadcastPlan.MinReadySeconds = 0
					rs.Labels = map[string]string{"serverless.io/app-stage": "xxx"}
					return rs
				}(),
				replicas: mergeReplicas([][]*carbonv1.Replica{
					newReplicasOneVersionWithMode1("453", "ready", true, 4),
				}),
				nodescheduler: rollalgorithm.NewDefaultNodeSetUpdater(nil),
			},
			want:          0,
			want1:         6,
			wantBroadcast: utils.Int64Ptr(4),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features.C2MutableFeatureGate.Set("OpenRollingSpotPod=true")
			defer func() {
				features.C2MutableFeatureGate.Set("OpenRollingSpotPod=false")
			}()
			s := getInPlaceScheduler(tt.fields)
			args, _ := s.initArgs()
			args.fixedCount = 0
			got := s.adjust(&args)
			if len(got) != tt.want {
				t.Errorf("InPlaceScheduler.sync() = %v, want %v", got, tt.want)
			}
			if name := getNames(got); tt.wantName != nil && !reflect.DeepEqual(name, tt.wantName) {
				if len(name) != 0 || len(tt.wantName) != 0 {
					t.Errorf("InPlaceScheduler.rollingUpdate() = %v, want = %v", name, tt.want)
					return
				}
			}
			if ls := len(args.supplementReplicaSet); ls != tt.want1 {
				t.Errorf("InPlaceScheduler.rollingUpdate() supplement= %v, want = %v", ls, tt.want1)
				return
			}
			if args.spotTarget != tt.wantSpot {
				t.Errorf("InPlaceScheduler.adjust() = %v, want %v", args.spotTarget, tt.wantSpot)
			}
			if tt.wantBroadcast != nil && len(args.broadcastReplicaSet) != int(*(tt.wantBroadcast)) {
				t.Errorf("InPlaceScheduler.wantBroadcast.adjust() = %v, want %v", len(args.broadcastReplicaSet), tt.wantBroadcast)
			}
		})
	}
}
