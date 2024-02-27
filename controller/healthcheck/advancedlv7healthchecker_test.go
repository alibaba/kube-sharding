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

package healthcheck

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
)

func newWorkerNodeWithAdvLv7(name string) *carbonv1.WorkerNode {
	// plan := workernode.Spec.VersionPlan
	// finalPlan := workernode.Spec.FinalVersionPlan
	// porcessVersion := workernode.Status.Version
	// serviceMetas := workernode.Status.HealthCondition.Metas
	// signature := plan.Signature
	// customInfo := plan.CustomInfo

	var workerNode = carbonv1.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			Labels:      map[string]string{"foo": "bar", carbonv1.DefaultRollingsetUniqueLabelKey: "test1", carbonv1.LabelKeyClusterName: "hippo_c"},
		},
		Spec: carbonv1.WorkerNodeSpec{
			VersionPlan: carbonv1.VersionPlan{
				SignedVersionPlan: carbonv1.SignedVersionPlan{
					Signature: "Signature-1",
				},
				BroadcastPlan: carbonv1.BroadcastPlan{
					CustomInfo: "CustomInfo-1",
				},
			},
		},
		Status: carbonv1.WorkerNodeStatus{
			AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{Version: "a"},
			HealthCondition: carbonv1.HealthCondition{Version: "b",
				Metas: map[string]string{"info": "kkk"},
			},
		},
	}

	return &workerNode
}

func newWorkerNodeWithAdvLv7Base(name string, healthStatus carbonv1.HealthStatus, allocStatus carbonv1.AllocStatus, phase carbonv1.WorkerPhase, lostCount int32) *carbonv1.WorkerNode {
	// plan := workernode.Spec.VersionPlan
	// finalPlan := workernode.Spec.FinalVersionPlan
	// porcessVersion := workernode.Status.Version
	// serviceMetas := workernode.Status.HealthCondition.Metas
	// signature := plan.Signature
	// customInfo := plan.CustomInfo

	var workerNode = carbonv1.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			Labels:      map[string]string{"foo": "bar", carbonv1.DefaultRollingsetUniqueLabelKey: "test1", carbonv1.LabelKeyClusterName: "hippo_c"},
		},
		Spec: carbonv1.WorkerNodeSpec{
			VersionPlan: carbonv1.VersionPlan{
				SignedVersionPlan: carbonv1.SignedVersionPlan{
					Signature: "Signature-1",
				},
				BroadcastPlan: carbonv1.BroadcastPlan{
					CustomInfo: "CustomInfo-1",
				},
			},
		},
		Status: carbonv1.WorkerNodeStatus{
			AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
				Version: "testversion",
				IP:      "127.0.0.1",
				Phase:   carbonv1.Running,
			},

			HealthCondition: carbonv1.HealthCondition{Version: "b",
				Metas:     map[string]string{"info": "kkk"},
				Status:    healthStatus,
				LostCount: lostCount,
			},
			HealthStatus: healthStatus,
		},
	}

	return &workerNode
}

func newRollingSetWithFinal(custominfo string, compressedCustominfo string) *carbonv1.RollingSet {
	selector := map[string]string{"foo": "bar", carbonv1.DefaultRollingsetUniqueLabelKey: "test1"}
	rollingset := carbonv1.RollingSet{
		TypeMeta: metav1.TypeMeta{APIVersion: "carbon/v1", Kind: "Rollingset"},
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        "rollingset-m",
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			//ht.rollingSet.Labels[carbonv1.LabelKeyClusterName],
			//ht.rollingSet.Labels[carbonv1.LabelKeyAppName]
			Labels: map[string]string{carbonv1.LabelKeyClusterName: "1", carbonv1.LabelKeyAppName: "2"},
		},
		Spec: carbonv1.RollingSetSpec{
			Version: "version",
			VersionPlan: carbonv1.VersionPlan{
				BroadcastPlan: carbonv1.BroadcastPlan{
					CustomInfo:           custominfo,
					CompressedCustomInfo: compressedCustominfo,
				},
			},

			Selector:            &metav1.LabelSelector{MatchLabels: selector},
			HealthCheckerConfig: NewConfigWithType(carbonv1.AdvancedLv7Health),
		},
		Status: carbonv1.RollingSetStatus{},
	}
	return &rollingset
}

func Test_getQueryData(t *testing.T) {
	type args struct {
		workernode *carbonv1.WorkerNode
		rollingset *carbonv1.RollingSet
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test1",
			args: args{workernode: newWorkerNodeWithAdvLv7("test"), rollingset: newRollingSetWithFinal("CustomInfo-2", "")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getQueryData(tt.args.workernode, tt.args.rollingset)
			if got[HealthcheckPayloadSignature] != "Signature-1" {
				t.Errorf("getQueryData() = %v", got)
			}

			if got[HealthcheckPayloadCustominfo] != "CustomInfo-1" {
				t.Errorf("getQueryData() = %v", got)
			}

			if got[HealthcheckPayloadIdentifier] != "hippo_c:default:test" {
				t.Errorf("getQueryData() = %v", got)
			}
			if got[HealthcheckPayloadProcessVersion] != "a" {
				t.Errorf("getQueryData() = %v", got)
			}
		})
	}
}

func Test_getFinalCustomInfo(t *testing.T) {
	type args struct {
		workernode *carbonv1.WorkerNode
		rollingset *carbonv1.RollingSet
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test1",
			args: args{workernode: newWorkerNodeWithAdvLv7("test"), rollingset: newRollingSetWithFinal("CustomInfo-2", "")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getQueryData(tt.args.workernode, tt.args.rollingset)
			if got[HealthcheckPayloadSignature] != "Signature-1" {
				t.Errorf("getQueryData() = %v", got)
			}

			if got[HealthcheckPayloadCustominfo] != "CustomInfo-1" {
				t.Errorf("getQueryData() = %v", got)
			}

			if got[HealthcheckPayloadIdentifier] != "hippo_c:default:test" {
				t.Errorf("getQueryData() = %v", got)
			}
			if got[HealthcheckPayloadProcessVersion] != "a" {
				t.Errorf("getQueryData() = %v", got)
			}
			var versionPlan carbonv1.VersionPlan
			err := json.Unmarshal([]byte(got[HealthcheckPayloadGlobalCustominfo]), &versionPlan)
			if err != nil {
				t.Errorf("getQueryData() err %v", err)
			}
			if versionPlan.CustomInfo != "CustomInfo-2" {
				t.Errorf("getQueryData() = %v", got)
			}
			if versionPlan.DeepCopy().CompressedCustomInfo != "" {
				t.Errorf("getQueryData() = %v", got)
			}
		})
	}
}

func newWorkerNode2(name string) *carbonv1.WorkerNode {
	var service = carbonv1.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
			Labels:      map[string]string{"foo": "bar", carbonv1.DefaultRollingsetUniqueLabelKey: "test1"},
		},
		Spec: carbonv1.WorkerNodeSpec{
			//Selector: selector,
		},
	}
	return &service
}

func Test_parseHealthMetas(t *testing.T) {
	type args struct {
		body []byte
	}
	kv := make(map[string]string, 0)
	kv["a"] = "aaaa"
	kv["b"] = "vvvv"
	kv["c"] = "[1,2,3,4]"
	con := utils.ObjJSON(kv)
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "test1",
			args: args{body: []byte(con)},
			want: kv,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := parseHealthMetas(tt.args.body)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseHealthMetas() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_needHealthCheckAdvlv7(t *testing.T) {
	type args struct {
		worker *carbonv1.WorkerNode
	}
	tests := []struct {
		name  string
		args  args
		want  bool
		want1 *carbonv1.HealthCondition
	}{
		{
			name: "test1",
			args: args{worker: &carbonv1.WorkerNode{
				Spec: carbonv1.WorkerNodeSpec{
					Version: "aaa",
					VersionPlan: carbonv1.VersionPlan{
						BroadcastPlan: carbonv1.BroadcastPlan{
							UpdatingGracefully: utils.BoolPtr(true),
						},
					},
				},
				Status: carbonv1.WorkerNodeStatus{
					AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
						Version:      "bbb",
						Phase:        carbonv1.Running,
						ProcessReady: true,
					},
					AllocStatus:   carbonv1.WorkerAssigned,
					ServiceStatus: carbonv1.ServiceUnAvailable,
					HealthCondition: carbonv1.HealthCondition{
						Version: "bbb",
					},
				},
			}},
			want: false,
			want1: &carbonv1.HealthCondition{
				Version: "bbb",
			},
		},

		{
			name: "test1",
			args: args{worker: &carbonv1.WorkerNode{
				Spec: carbonv1.WorkerNodeSpec{
					Version: "aaa",
					VersionPlan: carbonv1.VersionPlan{
						BroadcastPlan: carbonv1.BroadcastPlan{
							UpdatingGracefully: utils.BoolPtr(true),
						},
					},
				},
				Status: carbonv1.WorkerNodeStatus{
					AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
						Version:      "",
						Phase:        carbonv1.Running,
						ProcessReady: true,
					},
					AllocStatus:   carbonv1.WorkerAssigned,
					ServiceStatus: carbonv1.ServiceAvailable,
					HealthCondition: carbonv1.HealthCondition{
						Version: "",
					},
				},
			}},
			want: true,
			want1: &carbonv1.HealthCondition{
				Version: "",
			},
		},
		{
			name: "test1",
			args: args{worker: &carbonv1.WorkerNode{
				Spec: carbonv1.WorkerNodeSpec{
					Version: "aaa",
					VersionPlan: carbonv1.VersionPlan{
						BroadcastPlan: carbonv1.BroadcastPlan{
							UpdatingGracefully: utils.BoolPtr(true),
						},
					},
				},
				Status: carbonv1.WorkerNodeStatus{
					AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
						Version:      "bbb",
						Phase:        carbonv1.Running,
						ProcessReady: true,
					},
					AllocStatus:   carbonv1.WorkerAssigned,
					ServiceStatus: carbonv1.ServiceAvailable,
					HealthCondition: carbonv1.HealthCondition{
						Version: "bbb",
					},
				},
			}},
			want: false,
			want1: &carbonv1.HealthCondition{
				Version: "bbb",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := needHealthCheckAdvlv7(tt.args.worker)
			if got != tt.want {
				t.Errorf("needHealthCheckAdvlv7() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("needHealthCheckAdvlv7() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
