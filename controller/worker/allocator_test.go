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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"k8s.io/kubernetes/pkg/util/labels"

	"github.com/alibaba/kube-sharding/pkg/memkube/testutils"

	"github.com/alibaba/kube-sharding/common/features"
	"github.com/alibaba/kube-sharding/controller/util"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	gomock "github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
)

func newPod(name, ip string, phase corev1.PodPhase, statusExtend string) *corev1.Pod {
	var podSpec = corev1.PodSpec{
		NodeName: "node1",
		Priority: utils.Int32Ptr(1),
	}
	var pod = corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			//UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: map[string]string{},
			Labels:      map[string]string{},
		},
		Spec: podSpec,
		Status: corev1.PodStatus{
			Phase: phase,
			PodIP: ip,
			Conditions: []corev1.PodCondition{
				{
					Type:    carbonv1.ConditionKeyScheduleStatusExtend,
					Message: statusExtend,
				},
			},
		},
	}
	return &pod
}

func newPodWithOutUID(name, ip string, phase corev1.PodPhase, statusExtend string) *corev1.Pod {
	var pod = newPod(name, ip, phase, statusExtend)
	pod.UID = ""
	return pod
}

func newPodWithDeclaredKeys(name, ip string, phase corev1.PodPhase, statusExtend string) *corev1.Pod {
	pod := newPodWithOutUID(name, ip, phase, statusExtend)
	pod.Annotations[carbonv1.AnnotationC2DeclaredKeys] = `{}`
	return pod
}

func newPodWithStatusExtend(name, ip string, phase corev1.PodPhase, score int32, requirementID string, match bool) *corev1.Pod {
	status := newScheduleStatusExtend(score, requirementID, match)
	b, _ := json.Marshal(status)
	return newPod(name, ip, phase, string(b))
}

func newTemplate(containerMode string, priority int32, cpu int) *carbonv1.HippoPodTemplate {
	var podSpec = corev1.PodSpec{
		NodeName: "node1",
		Priority: utils.Int32Ptr(priority),
	}
	var hippoPodSpec = carbonv1.HippoPodSpec{
		PodSpec: podSpec,
		HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
			ContainerModel: containerMode,
		},
		Containers: []carbonv1.HippoContainer{
			{
				Container: corev1.Container{
					Name:  "container",
					Image: "image",
				},
				ContainerHippoExterned: carbonv1.ContainerHippoExterned{
					PreDeployImage: "preimage",
				},
			},
		},
	}
	if 0 != cpu {
		res := resource.NewMilliQuantity(int64(cpu), resource.DecimalSI)
		hippoPodSpec.Containers[0].Resources = corev1.ResourceRequirements{
			Limits: map[corev1.ResourceName]resource.Quantity{
				"cpu": *res,
			},
		}
	}
	var hippoPodTemplate carbonv1.HippoPodTemplate
	hippoPodTemplate.Spec = hippoPodSpec
	return &hippoPodTemplate
}

func newTemplateWithSchName(schedulerName string) *carbonv1.HippoPodTemplate {
	var podSpec = corev1.PodSpec{
		NodeName:      "node1",
		SchedulerName: schedulerName,
		Priority:      utils.Int32Ptr(1),
	}
	var hippoPodSpec = carbonv1.HippoPodSpec{
		PodSpec: podSpec,
		Containers: []carbonv1.HippoContainer{
			{
				Container: corev1.Container{
					Name:  "container",
					Image: "image",
				},
				ContainerHippoExterned: carbonv1.ContainerHippoExterned{
					PreDeployImage: "preimage",
				},
			},
		},
	}
	var hippoPodTemplate carbonv1.HippoPodTemplate
	hippoPodTemplate.Spec = hippoPodSpec
	return &hippoPodTemplate
}

func newWorker(name, ip string, releasing bool, containerMode string, selector map[string]string, priority int32, cpu int) *carbonv1.WorkerNode {
	var worker = carbonv1.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			UID:         uuid.NewUUID(),
			Name:        name,
			Namespace:   metav1.NamespaceDefault,
			Annotations: make(map[string]string),
		},
		Spec: carbonv1.WorkerNodeSpec{
			Version:    "1",
			ResVersion: "2",
			Releasing:  releasing,
			VersionPlan: carbonv1.VersionPlan{
				SignedVersionPlan: carbonv1.SignedVersionPlan{
					Template: newTemplate(containerMode, priority, cpu),
				},
			},
		},
		Status: carbonv1.WorkerNodeStatus{
			AllocStatus:   carbonv1.WorkerAssigned,
			ServiceStatus: carbonv1.ServicePartAvailable,
			HealthStatus:  carbonv1.HealthAlive,
			AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
				Phase: carbonv1.Running,
				IP:    ip,
			},
			ToRelease: releasing,
		},
	}
	if releasing {
		worker.Status.AllocStatus = carbonv1.WorkerReleasing
	}
	return &worker
}

func newScheduleStatusExtend(score int32, requirementID string, match bool) *carbonv1.ScheduleStatusExtend {
	return &carbonv1.ScheduleStatusExtend{
		HealthScore:      score,
		RequirementID:    requirementID,
		RequirementMatch: match,
	}
}

func testnewPodStatusExtend(instanceID, podName, podInstanceID string) *carbonv1.PodStatusExtend {
	return &carbonv1.PodStatusExtend{
		InstanceID: instanceID,
		Containers: []carbonv1.ContainerStatusHippoExterned{
			{
				Name:       podName,
				InstanceID: podInstanceID,
			},
		},
	}
}

func testsetPodstatusextend(pod *corev1.Pod, instanceID, podName, podInstanceID string) {
	status := testnewPodStatusExtend(instanceID, podName, podInstanceID)
	b, _ := json.Marshal(status)
	pod.Status.Conditions = append(pod.Status.Conditions, corev1.PodCondition{
		Type:    carbonv1.ConditionKeyPodStatusExtend,
		Message: string(b),
	})
}

func testnewHippoPodSpecExtend(instanceID, podName, podInstanceID string) carbonv1.HippoPodSpecExtend {
	return carbonv1.HippoPodSpecExtend{
		InstanceID: instanceID,
		Containers: []carbonv1.ContainerHippoExternedWithName{
			{
				Name:       podName,
				InstanceID: podInstanceID,
			},
		},
	}
}

func Test_podAllocator_init(t *testing.T) {
	type fields struct {
		schedulerType string
		target        target
		current       current
		worker        *carbonv1.WorkerNode
		parser        podStatusParser
		syncer        podSpecSyncer
		executor      executor
	}

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	parser := NewMockpodStatusParser(ctl)
	syncer := NewMockpodSpecSyncer(ctl)

	parser.EXPECT().getCurrentResourceVersions(gomock.Any()).Return("", false).MaxTimes(1)
	parser.EXPECT().getCurrentProcessVersions(gomock.Any()).Return("", nil).MaxTimes(1)
	syncer.EXPECT().computeProcessVersion(gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil, nil).MaxTimes(1)
	syncer.EXPECT().computeResourceVersion(gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).MaxTimes(1)
	syncer.EXPECT().preSync(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).MaxTimes(1)

	parserNilCullrent := NewMockpodStatusParser(ctl)
	syncerNilCullrent := NewMockpodSpecSyncer(ctl)
	syncerNilCullrent.EXPECT().computeProcessVersion(gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil, nil).MaxTimes(1)
	syncerNilCullrent.EXPECT().computeResourceVersion(gomock.Any(), gomock.Any(), gomock.Any()).Return("", nil).MaxTimes(1)
	syncerNilCullrent.EXPECT().preSync(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).MaxTimes(1)

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "normal",
			fields: fields{
				target: target{
					worker: newWorker("test", "1", true, "", nil, 1, 0),
				},
				current: current{
					currentPod: newPod("test", "1", corev1.PodPending, ""),
				},
				parser: parser,
				syncer: syncer,
			},
			wantErr: false,
		},
		{
			name: "nil current",
			fields: fields{
				target: target{
					worker: newWorker("test", "1", true, "", nil, 1, 0),
				},
				current: current{},
				parser:  parserNilCullrent,
				syncer:  syncerNilCullrent,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &podAllocator{
				schedulerType: tt.fields.schedulerType,
				target:        tt.fields.target,
				current:       tt.fields.current,

				parser:   tt.fields.parser,
				syncer:   tt.fields.syncer,
				executor: tt.fields.executor,
			}
			if err := a.init(); (err != nil) != tt.wantErr {
				t.Errorf("podAllocator.init() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_podAllocator_allocate(t *testing.T) {
	type fields struct {
		schedulerType string
		target        target
		current       current
		worker        *carbonv1.WorkerNode
		parser        podStatusParser
		syncer        podSpecSyncer
		executor      executor
		modifyWorker  func(node *carbonv1.WorkerNode)
		writeLabels   map[string]string
	}
	ctl := gomock.NewController(t)
	defer ctl.Finish()

	syncer := NewMockpodSpecSyncer(ctl)
	executor := NewMockexecutor(ctl)
	syncer.EXPECT().postSync(gomock.Any()).Return(nil).MinTimes(1)
	executor.EXPECT().createPod(gomock.Any()).Return(newPodWithOutUID("test", "1", corev1.PodPending, ""), nil).MaxTimes(4)

	var targetPod = func() *corev1.Pod {
		pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
		pod.Labels = map[string]string{
			"alibabacloud.com/owner":    "hippo",
			"alibabacloud.com/platform": "c2",
			"app.hippo.io/pod-version":  "v3.0",
		}
		pod.Annotations = map[string]string{
			"app.c2.io/c2-declared-keys":     `{"lableKeys":["alibabacloud.com/owner","alibabacloud.com/platform","app.hippo.io/pod-version"],"annotationKeys":["app.c2.io/c2-declared-env-keys","app.c2.io/pod-resource-version","app.c2.io/pod-version"]}`,
			"app.c2.io/pod-resource-version": "",
			"app.c2.io/pod-version":          "",
			"app.c2.io/c2-declared-env-keys": "[]",
		}
		pod.Spec.HostNetwork = false
		return pod
	}()
	syncer1 := NewMockpodSpecSyncer(ctl)
	executor1 := NewMockexecutor(ctl)
	syncer1.EXPECT().postSync(gomock.Any()).Return(nil).MinTimes(1)
	executor1.EXPECT().createPod(targetPod).Return(newPodWithOutUID("test", "1", corev1.PodPending, ""), nil).MaxTimes(1)

	executorWithError := NewMockexecutor(ctl)
	executorWithError.EXPECT().createPod(gomock.Any()).Return(nil, errors.NewAlreadyExists(carbonv1.Resource("carbon.taobao.com/v1"), "workernode")).MaxTimes(1)

	tests := []struct {
		name    string
		fields  fields
		want    *corev1.Pod
		wantErr bool
	}{
		{
			name: "normal",
			fields: fields{
				target: target{
					worker:    newWorker("test", "1", true, "", nil, 1, 0),
					targetPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
				},
				current: current{
					currentPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
				},
				syncer:   syncer,
				executor: executor,
			},
			want: func() *corev1.Pod {
				pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
				pod.Annotations = map[string]string{
					"app.c2.io/c2-declared-env-keys": "[]",
					"app.c2.io/c2-declared-keys":     "{\"annotationKeys\":[\"app.c2.io/c2-declared-env-keys\",\"app.c2.io/pod-resource-version\",\"app.c2.io/pod-version\"]}",
					"app.c2.io/pod-resource-version": "",
					"app.c2.io/pod-version":          "",
				}
				return pod
			}(),
			wantErr: false,
		},
		{
			name: "simplify",
			fields: fields{
				target: target{
					worker: newWorker("test", "1", true, "", nil, 1, 0),
					targetPod: func() *corev1.Pod {
						pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
						pod.Labels = map[string]string{
							"alibabacloud.com/owner":    "hippo",
							"alibabacloud.com/platform": "c2",
							"app.hippo.io/pod-version":  "v3.0",
						}
						pod.Annotations = map[string]string{
							"alibabacloud.com/update-spec":     "test",
							"alibabacloud.com/resource-update": "test1",
						}
						pod.Spec.HostNetwork = false
						pod.Spec.SchedulerName = "default-scheduler"

						return pod
					}(),
				},
				current: current{
					currentPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
				},
				syncer:   syncer,
				executor: executor,
			},
			want: func() *corev1.Pod {
				pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
				pod.Labels = map[string]string{
					"alibabacloud.com/owner":    "hippo",
					"alibabacloud.com/platform": "c2",
					"app.hippo.io/pod-version":  "v3.0",
				}
				pod.Annotations = map[string]string{
					"alibabacloud.com/resource-update": "test1",
					"alibabacloud.com/update-spec":     "test",
					"app.c2.io/c2-declared-env-keys":   "[]",
					"app.c2.io/c2-declared-keys":       "{\"lableKeys\":[\"alibabacloud.com/owner\",\"alibabacloud.com/platform\",\"app.hippo.io/pod-version\"],\"annotationKeys\":[\"alibabacloud.com/resource-update\",\"alibabacloud.com/update-spec\",\"app.c2.io/c2-declared-env-keys\",\"app.c2.io/pod-resource-version\",\"app.c2.io/pod-version\"],\"schedulerName\":\"default-scheduler\"}",
					"app.c2.io/pod-resource-version":   "",
					"app.c2.io/pod-version":            "",
				}
				pod.Spec.HostNetwork = false
				pod.Spec.SchedulerName = "default-scheduler"

				return pod
			}(),
			wantErr: false,
		},
		{
			name: "add finalizer",
			fields: fields{
				target: target{
					worker: newWorker("test", "1", true, "", nil, 1, 0),
					targetPod: func() *corev1.Pod {
						pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
						pod.Labels = map[string]string{
							"alibabacloud.com/owner":    "hippo",
							"alibabacloud.com/platform": "c2",
							"app.hippo.io/pod-version":  "v3.0",
						}
						pod.Spec.HostNetwork = false

						return pod
					}(),
				},
				current: current{
					currentPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
				},
				syncer:   syncer1,
				executor: executor1,
			},
			want: func() *corev1.Pod {
				pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
				pod.Labels = map[string]string{
					"alibabacloud.com/owner":    "hippo",
					"alibabacloud.com/platform": "c2",
					"app.hippo.io/pod-version":  "v3.0",
				}
				pod.Annotations = map[string]string{
					"app.c2.io/c2-declared-env-keys": "[]",
					"app.c2.io/c2-declared-keys":     "{\"lableKeys\":[\"alibabacloud.com/owner\",\"alibabacloud.com/platform\",\"app.hippo.io/pod-version\"],\"annotationKeys\":[\"app.c2.io/c2-declared-env-keys\",\"app.c2.io/pod-resource-version\",\"app.c2.io/pod-version\"]}",
					"app.c2.io/pod-resource-version": "",
					"app.c2.io/pod-version":          "",
				}
				pod.Spec.HostNetwork = false

				return pod
			}(),
			wantErr: false,
		},
		{
			name: "return err",
			fields: fields{
				target: target{
					worker:    newWorker("test", "1", true, "", nil, 1, 0),
					targetPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
				},
				current:  current{},
				syncer:   syncer,
				executor: executorWithError,
			},
			wantErr: true,
		},
		{
			name: "update standby info",
			fields: fields{
				target: target{
					worker:    newWorker("test", "1", true, "", nil, 1, 0),
					targetPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
				},
				modifyWorker: func(w *carbonv1.WorkerNode) {
					w.Spec.DeletionCost = 100
					w.Spec.StandbyHours = "[1,2,5,23]"
				},
				current: current{
					currentPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
				},
				syncer:   syncer,
				executor: executor,
			},
			want: func() *corev1.Pod {
				pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
				pod.Annotations = map[string]string{
					"app.c2.io/c2-declared-env-keys":                      "[]",
					"app.c2.io/c2-declared-keys":                          "{\"annotationKeys\":[\"app.c2.io/c2-declared-env-keys\",\"app.c2.io/pod-resource-version\",\"app.c2.io/pod-version\"]}",
					"app.c2.io/pod-resource-version":                      "",
					"app.c2.io/pod-version":                               "",
					"controller.kubernetes.io/original-pod-deletion-cost": "100",
					"alibabacloud.com/standby-stable-status":              "{\"standbyHours\":[1,2,5,23]}",
				}
				pod.Labels = map[string]string{}
				return pod
			}(),
			wantErr: false,
		},
		{
			name: "write labels",
			fields: fields{
				target: target{
					worker:    newWorker("test", "1", true, "", nil, 1, 0),
					targetPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
				},
				modifyWorker: func(w *carbonv1.WorkerNode) {
					w.Spec.DeletionCost = 100
					w.Spec.StandbyHours = "[1,2,5,23]"
				},
				current: current{
					currentPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
				},
				syncer:      syncer,
				executor:    executor,
				writeLabels: map[string]string{"key": "value"},
			},
			want: func() *corev1.Pod {
				pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
				pod.Annotations = map[string]string{
					"app.c2.io/c2-declared-env-keys":                      "[]",
					"app.c2.io/c2-declared-keys":                          "{\"annotationKeys\":[\"app.c2.io/c2-declared-env-keys\",\"app.c2.io/pod-resource-version\",\"app.c2.io/pod-version\"]}",
					"app.c2.io/pod-resource-version":                      "",
					"app.c2.io/pod-version":                               "",
					"controller.kubernetes.io/original-pod-deletion-cost": "100",
					"alibabacloud.com/standby-stable-status":              "{\"standbyHours\":[1,2,5,23]}",
				}
				pod.Labels = map[string]string{
					"key": "value",
				}
				return pod
			}(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		features.C2MutableFeatureGate.Set("SimplifyUpdateField=true")
		t.Run(tt.name, func(t *testing.T) {
			if tt.fields.modifyWorker != nil {
				tt.fields.modifyWorker(tt.fields.target.worker)
			}
			a := &podAllocator{
				schedulerType: tt.fields.schedulerType,
				target:        tt.fields.target,
				current:       tt.fields.current,

				parser:   tt.fields.parser,
				syncer:   tt.fields.syncer,
				executor: tt.fields.executor,
				c:        &Controller{writeLabels: tt.fields.writeLabels},
			}
			a.pSpecMerger = &util.PodSpecMerger{}
			err := a.allocate()
			if (err != nil) != tt.wantErr {
				t.Errorf("podAllocator.allocate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && utils.ObjJSON(tt.want) != utils.ObjJSON(tt.fields.target.targetPod) {
				t.Errorf("podAllocator.allocate() want = %v, got %v", utils.ObjJSON(tt.want), utils.ObjJSON(tt.fields.target.targetPod))
				return
			}
		})
	}
}

func Test_podAllocator_delete(t *testing.T) {
	type fields struct {
		schedulerType string
		target        target
		current       current
		worker        *carbonv1.WorkerNode
		parser        podStatusParser
		syncer        podSpecSyncer
		executor      executor
	}
	ctl := gomock.NewController(t)
	defer ctl.Finish()

	syncer := NewMockpodSpecSyncer(ctl)
	executor := NewMockexecutor(ctl)
	syncer.EXPECT().setProhibit(gomock.Any(), gomock.Any()).MinTimes(1)
	executor.EXPECT().deletePod(gomock.Any(), gomock.Any()).Return(nil).MinTimes(1)
	executor1 := NewMockexecutor(ctl)
	executor1.EXPECT().deletePod(gomock.Any(), gomock.Any()).Return(nil).MinTimes(1)

	wantWithPrefer := newPod("test", "1", corev1.PodPending, "")
	wantWithPrefer.Labels = map[string]string{carbonv1.LabelKeyHippoPreference: "APP-PROHIBIT-600"}
	workerWithPrefer := newWorker("test", "1", true, "", nil, 1, 0)
	workerWithPrefer.Labels = map[string]string{
		carbonv1.LabelKeyHippoPreference: "APP-PROHIBIT-600",
	}
	workerWithPrefer.Status.ServiceOffline = true

	worker := newWorker("test", "1", true, "", nil, 1, 0)
	worker.Status.ServiceOffline = true
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "normal",
			fields: fields{
				current: current{
					currentPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
				},
				target: target{
					worker: workerWithPrefer,
				},
				syncer:   syncer,
				executor: executor,
			},
			wantErr: false,
		},
		{
			name: "nil pod",
			fields: fields{
				target: target{
					worker: workerWithPrefer,
				},
			},
			wantErr: false,
		},
		{
			name: "no prohibit",
			fields: fields{
				target: target{
					worker: worker,
				},
				current: current{
					currentPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
				},
				executor: executor1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &podAllocator{
				schedulerType: tt.fields.schedulerType,
				target:        tt.fields.target,
				current:       tt.fields.current,

				parser:   tt.fields.parser,
				syncer:   tt.fields.syncer,
				executor: tt.fields.executor,
			}
			if err := a.delete(); (err != nil) != tt.wantErr {
				t.Errorf("podAllocator.delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_podAllocator_syncPodMetas(t *testing.T) {
	type fields struct {
		schedulerType string
		target        target
		current       current
		worker        *carbonv1.WorkerNode
		executor      executor
	}
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	features.C2MutableFeatureGate.Set("SyncSubrsMetas=true")
	defer features.C2MutableFeatureGate.Set("SyncSubrsMetas=false")
	tests := []struct {
		name           string
		fields         fields
		wantErr        bool
		syncSubrsMetas bool
	}{
		{
			name:           "subrsNotSynced",
			syncSubrsMetas: true,
			fields: fields{
				current: current{
					currentPod: func() *corev1.Pod {
						pod := newPodWithDeclaredKeys("test", "1", corev1.PodPending, "")
						pod.ObjectMeta.Labels = map[string]string{
							"serverless.io/app-name":       "fengx",
							"serverless.io/instance-group": "feng_host",
							carbonv1.LabelServerlessAppId:  "test",
						}
						pod.ObjectMeta.Annotations = map[string]string{
							"app.hippo.io/appShortName":  "feng",
							"app.hippo.io/roleShortName": "fengy",
						}
						return pod
					}(),
				},
				target: target{
					worker: func() *carbonv1.WorkerNode {
						worker := newWorker("test", "1", true, "", nil, 1, 0)
						worker.ObjectMeta.Labels = map[string]string{
							"serverless.io/app-name": "feng",
						}
						worker.ObjectMeta.Annotations = map[string]string{
							"app.hippo.io/appShortName": "fengx",
						}
						worker.Spec.Template.Labels = map[string]string{
							"serverless.io/app-name": "fengy",
						}
						worker.Spec.Template.Annotations = map[string]string{
							"app.hippo.io/appShortName":            "fengy",
							"app.c2.io/biz-detail-partition-count": "5",
						}
						worker.Status.ResVersion = "2"
						sceneStatus := make([]carbonv1.AppNodeStatus, 0, 1)
						sceneStatus = append(sceneStatus, carbonv1.AppNodeStatus{
							AppId:         "test",
							StartedAt:     1695106366,
							PlanStatus:    1,
							ServiceStatus: 3,
						})
						serviceCondition := carbonv1.ServiceCondition{
							ServiceDetail: utils.ObjJSON(sceneStatus),
							Type:          carbonv1.ServiceGeneral,
						}
						worker.Status.ServiceConditions = make([]carbonv1.ServiceCondition, 0, 1)
						worker.Status.ServiceConditions = append(worker.Status.ServiceConditions, serviceCondition)
						return worker
					}(),
					targetPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
					podSpec: func() *corev1.PodSpec {
						pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
						return &pod.Spec
					}(),
				},
				executor: func() executor {
					executor := NewMockexecutor(ctl)
					sceneNodeStatus := carbonv1.AppNodeStatus{
						AppId:         "test",
						StartedAt:     1695106366,
						PlanStatus:    1,
						ServiceStatus: 3,
					}
					statusStr := utils.ObjJSON(sceneNodeStatus)
					executor.EXPECT().patchPod(gomock.Any(), testutils.FuncMockMatcher{
						Func: func(x interface{}) bool {
							data := x.(*util.CommonPatch)
							metadata := data.PatchData["metadata"]
							meta := metadata.(map[string]interface{})
							labels := meta["labels"].(map[string]string)
							annotations := meta["annotations"].(map[string]string)
							return labels["serverless.io/app-name"] == "fengy" && labels["serverless.io/instance-group"] == "NULL_HOLDER" &&
								annotations["app.hippo.io/appShortName"] == "fengy" && annotations["app.hippo.io/roleShortName"] == "NULL_HOLDER" &&
								annotations["app.c2.io/biz-detail-ready"] == "false" && annotations["app.c2.io/biz-detail-partition-count"] == "5" &&
								annotations[carbonv1.AnnoServerlessAppPlanStatus] == "1" && annotations[carbonv1.AnnoServerlessAppStatus] == statusStr
						},
					}).Return(nil).Times(1)
					return executor
				}(),
			},
		},
		{
			name:           "subrsNotSynced",
			syncSubrsMetas: true,
			fields: fields{
				current: current{
					currentPod: func() *corev1.Pod {
						pod := newPodWithDeclaredKeys("test", "1", corev1.PodPending, "")
						pod.ObjectMeta.Labels = map[string]string{
							"serverless.io/app-name":       "fengx",
							"serverless.io/instance-group": "feng_host",
							carbonv1.LabelServerlessAppId:  "test2",
						}
						pod.ObjectMeta.Annotations = map[string]string{
							"app.hippo.io/appShortName":  "feng",
							"app.hippo.io/roleShortName": "fengy",
						}
						return pod
					}(),
				},
				target: target{
					worker: func() *carbonv1.WorkerNode {
						worker := newWorker("test", "1", true, "", nil, 1, 0)
						worker.ObjectMeta.Labels = map[string]string{
							"serverless.io/app-name": "feng",
						}
						worker.ObjectMeta.Annotations = map[string]string{
							"app.hippo.io/appShortName": "fengx",
						}
						worker.Spec.Template.Labels = map[string]string{
							"serverless.io/app-name": "fengy",
						}
						worker.Spec.Template.Annotations = map[string]string{
							"app.hippo.io/appShortName":            "fengy",
							"app.c2.io/biz-detail-partition-count": "5",
						}
						worker.Status.ResVersion = "2"
						sceneStatus := make([]carbonv1.AppNodeStatus, 0, 1)
						sceneStatus = append(sceneStatus, carbonv1.AppNodeStatus{
							AppId:         "test",
							StartedAt:     1695106366,
							PlanStatus:    1,
							ServiceStatus: 3,
						})
						serviceCondition := carbonv1.ServiceCondition{
							ServiceDetail: utils.ObjJSON(sceneStatus),
							Type:          carbonv1.ServiceGeneral,
						}
						worker.Status.ServiceConditions = make([]carbonv1.ServiceCondition, 0, 1)
						worker.Status.ServiceConditions = append(worker.Status.ServiceConditions, serviceCondition)
						return worker
					}(),
					targetPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
					podSpec: func() *corev1.PodSpec {
						pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
						return &pod.Spec
					}(),
				},
				executor: func() executor {
					executor := NewMockexecutor(ctl)
					executor.EXPECT().patchPod(gomock.Any(), testutils.FuncMockMatcher{
						Func: func(x interface{}) bool {
							data := x.(*util.CommonPatch)
							metadata := data.PatchData["metadata"]
							meta := metadata.(map[string]interface{})
							labels := meta["labels"].(map[string]string)
							annotations := meta["annotations"].(map[string]string)
							return labels["serverless.io/app-name"] == "fengy" && labels["serverless.io/instance-group"] == "NULL_HOLDER" &&
								annotations["app.hippo.io/appShortName"] == "fengy" && annotations["app.hippo.io/roleShortName"] == "NULL_HOLDER" &&
								annotations["app.c2.io/biz-detail-ready"] == "false" && annotations["app.c2.io/biz-detail-partition-count"] == "5" &&
								annotations[carbonv1.AnnoServerlessAppPlanStatus] == "" && annotations[carbonv1.AnnoServerlessAppStatus] == ""
						},
					}).Return(nil).Times(1)
					return executor
				}(),
			},
		},
		{
			name:           "subrsNotSynced",
			syncSubrsMetas: true,
			fields: fields{
				current: current{
					currentPod: func() *corev1.Pod {
						pod := newPodWithDeclaredKeys("test", "1", corev1.PodPending, "")
						pod.ObjectMeta.Labels = map[string]string{
							"serverless.io/app-name":       "fengx",
							"serverless.io/instance-group": "feng_host",
							carbonv1.LabelServerlessAppId:  "test2",
						}
						pod.ObjectMeta.Annotations = map[string]string{
							"app.hippo.io/appShortName":  "feng",
							"app.hippo.io/roleShortName": "fengy",
						}
						return pod
					}(),
				},
				target: target{
					worker: func() *carbonv1.WorkerNode {
						worker := newWorker("test", "1", true, "", nil, 1, 0)
						worker.ObjectMeta.Labels = map[string]string{
							"serverless.io/app-name": "feng",
						}
						worker.ObjectMeta.Annotations = map[string]string{
							"app.hippo.io/appShortName": "fengx",
						}
						worker.Spec.Template.Labels = map[string]string{
							"serverless.io/app-name": "fengy",
						}
						worker.Spec.Template.Annotations = map[string]string{
							"app.hippo.io/appShortName":            "fengy",
							"app.c2.io/biz-detail-partition-count": "5",
						}
						worker.Status.ResVersion = "2"
						sceneStatus := make([]carbonv1.AppNodeStatus, 0, 2)
						sceneStatus = append(sceneStatus, carbonv1.AppNodeStatus{
							AppId:         "test",
							StartedAt:     1695106366,
							PlanStatus:    1,
							ServiceStatus: 3,
						})
						sceneStatus = append(sceneStatus, carbonv1.AppNodeStatus{
							AppId:         "test2",
							StartedAt:     1695106366,
							PlanStatus:    1,
							ServiceStatus: 3,
						})
						serviceCondition := carbonv1.ServiceCondition{
							ServiceDetail: utils.ObjJSON(sceneStatus),
							Type:          carbonv1.ServiceGeneral,
						}
						worker.Status.ServiceConditions = make([]carbonv1.ServiceCondition, 0, 1)
						worker.Status.ServiceConditions = append(worker.Status.ServiceConditions, serviceCondition)
						return worker
					}(),
					targetPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
					podSpec: func() *corev1.PodSpec {
						pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
						return &pod.Spec
					}(),
				},
				executor: func() executor {
					executor := NewMockexecutor(ctl)
					executor.EXPECT().patchPod(gomock.Any(), testutils.FuncMockMatcher{
						Func: func(x interface{}) bool {
							data := x.(*util.CommonPatch)
							metadata := data.PatchData["metadata"]
							meta := metadata.(map[string]interface{})
							labels := meta["labels"].(map[string]string)
							annotations := meta["annotations"].(map[string]string)
							return labels["serverless.io/app-name"] == "fengy" && labels["serverless.io/instance-group"] == "NULL_HOLDER" &&
								annotations["app.hippo.io/appShortName"] == "fengy" && annotations["app.hippo.io/roleShortName"] == "NULL_HOLDER" &&
								annotations["app.c2.io/biz-detail-ready"] == "false" && annotations["app.c2.io/biz-detail-partition-count"] == "5" &&
								annotations[carbonv1.AnnoServerlessAppPlanStatus] == "" && annotations[carbonv1.AnnoServerlessAppStatus] == ""
						},
					}).Return(nil).Times(1)
					return executor
				}(),
			},
		},
		{
			name:           "serverless app ready",
			syncSubrsMetas: true,
			fields: fields{
				current: current{
					currentPod: func() *corev1.Pod {
						pod := newPodWithDeclaredKeys("test", "1", corev1.PodPending, "")
						pod.ObjectMeta.Labels = map[string]string{
							"serverless.io/app-name":       "fengx",
							"serverless.io/instance-group": "feng_host",
							carbonv1.LabelServerlessAppId:  "test",
						}
						pod.ObjectMeta.Annotations = map[string]string{
							"app.hippo.io/appShortName":  "feng",
							"app.hippo.io/roleShortName": "fengy",
						}
						return pod
					}(),
				},
				target: target{
					worker: func() *carbonv1.WorkerNode {
						worker := newWorker("test", "1", true, "", nil, 1, 0)
						worker.ObjectMeta.Labels = map[string]string{
							"serverless.io/app-name": "feng",
						}
						worker.ObjectMeta.Annotations = map[string]string{
							"app.hippo.io/appShortName": "fengx",
						}
						worker.Spec.Template.Labels = map[string]string{
							"serverless.io/app-name": "fengy",
						}
						worker.Spec.Template.Annotations = map[string]string{
							"app.hippo.io/appShortName":            "fengy",
							"app.c2.io/biz-detail-partition-count": "5",
						}
						worker.Status.ResVersion = "2"
						sceneStatus1 := make([]carbonv1.AppNodeStatus, 0, 2)
						sceneStatus1 = append(sceneStatus1, carbonv1.AppNodeStatus{
							AppId:         "test",
							StartedAt:     1695106366,
							PlanStatus:    1,
							ServiceStatus: 3,
						})
						sceneStatus1 = append(sceneStatus1, carbonv1.AppNodeStatus{
							AppId:         "test2",
							StartedAt:     1695106366,
							PlanStatus:    1,
							ServiceStatus: 3,
						})
						sceneStatus2 := make([]carbonv1.AppNodeStatus, 0, 1)
						sceneStatus2 = append(sceneStatus2, carbonv1.AppNodeStatus{
							AppId:         "test",
							StartedAt:     1695106366,
							PlanStatus:    1,
							ServiceStatus: 3,
						})
						serviceCondition1 := carbonv1.ServiceCondition{
							ServiceDetail: utils.ObjJSON(sceneStatus1),
							Type:          carbonv1.ServiceGeneral,
						}
						serviceCondition2 := carbonv1.ServiceCondition{
							ServiceDetail: utils.ObjJSON(sceneStatus2),
							Type:          carbonv1.ServiceGeneral,
						}
						worker.Status.ServiceConditions = make([]carbonv1.ServiceCondition, 0, 2)
						worker.Status.ServiceConditions = append(worker.Status.ServiceConditions, serviceCondition1)
						worker.Status.ServiceConditions = append(worker.Status.ServiceConditions, serviceCondition2)
						return worker
					}(),
					targetPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
					podSpec: func() *corev1.PodSpec {
						pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
						return &pod.Spec
					}(),
				},
				executor: func() executor {
					executor := NewMockexecutor(ctl)
					sceneStatus := carbonv1.AppNodeStatus{
						AppId:         "test",
						StartedAt:     1695106366,
						PlanStatus:    1,
						ServiceStatus: 3,
					}
					sceneStatusStr := utils.ObjJSON(sceneStatus)
					executor.EXPECT().patchPod(gomock.Any(), testutils.FuncMockMatcher{
						Func: func(x interface{}) bool {
							data := x.(*util.CommonPatch)
							metadata := data.PatchData["metadata"]
							meta := metadata.(map[string]interface{})
							labels := meta["labels"].(map[string]string)
							annotations := meta["annotations"].(map[string]string)
							status := (data.PatchData["status"]).(map[string]interface{})
							conditions := (status["conditions"]).([]corev1.PodCondition)
							return labels["serverless.io/app-name"] == "fengy" && labels["serverless.io/instance-group"] == "NULL_HOLDER" &&
								annotations["app.hippo.io/appShortName"] == "fengy" && annotations["app.hippo.io/roleShortName"] == "NULL_HOLDER" &&
								annotations["app.c2.io/biz-detail-ready"] == "false" && annotations["app.c2.io/biz-detail-partition-count"] == "5" &&
								annotations[carbonv1.AnnoServerlessAppPlanStatus] == "1" && annotations[carbonv1.AnnoServerlessAppStatus] == sceneStatusStr &&
								len(conditions) == 2 && conditions[0].Status == corev1.ConditionTrue
						},
					}).Return(nil).Times(1)
					return executor
				}(),
			},
		},
		{
			name:           "serverless app not sync ReasonQueryError",
			syncSubrsMetas: true,
			fields: fields{
				current: current{
					currentPod: func() *corev1.Pod {
						pod := newPodWithDeclaredKeys("test", "1", corev1.PodPending, "")
						pod.ObjectMeta.Labels = map[string]string{
							carbonv1.LabelServerlessAppId: "test",
						}
						pod.ObjectMeta.Annotations = map[string]string{
							carbonv1.AnnoServerlessAppStatus: "{\"appId\":\"test\",\"startedAt\":1695106366,\"planStatus\":1,\"serviceStatus\":3,\"needBackup\":false}",
						}
						return pod
					}(),
				},
				target: target{
					worker: func() *carbonv1.WorkerNode {
						worker := newWorker("test", "1", true, "", nil, 1, 0)
						worker.Status.ResVersion = "2"
						serviceCondition1 := carbonv1.ServiceCondition{
							Type:   carbonv1.ServiceGeneral,
							Reason: carbonv1.ReasonQueryError,
						}
						worker.Status.ServiceConditions = make([]carbonv1.ServiceCondition, 0, 1)
						worker.Status.ServiceConditions = append(worker.Status.ServiceConditions, serviceCondition1)
						return worker
					}(),
					targetPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
					podSpec: func() *corev1.PodSpec {
						pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
						return &pod.Spec
					}(),
				},
				executor: func() executor {
					executor := NewMockexecutor(ctl)
					executor.EXPECT().patchPod(gomock.Any(), testutils.FuncMockMatcher{
						Func: func(x interface{}) bool {
							data := x.(*util.CommonPatch)
							metadata := data.PatchData["metadata"]
							meta := metadata.(map[string]interface{})
							annotations := meta["annotations"].(map[string]string)
							return annotations[carbonv1.AnnoServerlessAppPlanStatus] == "" && annotations[carbonv1.AnnoServerlessAppStatus] == ""
						},
					}).Return(nil).Times(1)
					return executor
				}(),
			},
		},
		{
			name:           "serverless app failed",
			syncSubrsMetas: true,
			fields: fields{
				current: current{
					currentPod: func() *corev1.Pod {
						pod := newPodWithDeclaredKeys("test", "1", corev1.PodPending, "")
						pod.ObjectMeta.Labels = map[string]string{
							carbonv1.LabelServerlessAppId: "test",
						}
						pod.ObjectMeta.Annotations = map[string]string{
							carbonv1.AnnoServerlessAppStatus: "{\"appId\":\"test\",\"startedAt\":1695106366,\"planStatus\":3,\"serviceStatus\":1,\"needBackup\":false}",
						}
						return pod
					}(),
				},
				target: target{
					worker: func() *carbonv1.WorkerNode {
						worker := newWorker("test", "1", true, "", nil, 1, 0)
						worker.Status.ResVersion = "2"
						sceneStatus1 := []carbonv1.AppNodeStatus{{
							AppId:         "test",
							StartedAt:     1695106366,
							PlanStatus:    2,
							ServiceStatus: 1,
						}}
						serviceCondition1 := carbonv1.ServiceCondition{
							Type:          carbonv1.ServiceGeneral,
							ServiceDetail: utils.ObjJSON(sceneStatus1),
						}
						worker.Status.ServiceConditions = make([]carbonv1.ServiceCondition, 0, 1)
						worker.Status.ServiceConditions = append(worker.Status.ServiceConditions, serviceCondition1)
						return worker
					}(),
					targetPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
					podSpec: func() *corev1.PodSpec {
						pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
						return &pod.Spec
					}(),
				},
				executor: func() executor {
					executor := NewMockexecutor(ctl)
					executor.EXPECT().patchPod(gomock.Any(), testutils.FuncMockMatcher{
						Func: func(x interface{}) bool {
							data := x.(*util.CommonPatch)
							metadata := data.PatchData["metadata"]
							meta := metadata.(map[string]interface{})
							annotations := meta["annotations"].(map[string]string)
							return annotations[carbonv1.AnnoServerlessAppPlanStatus] == "" && annotations[carbonv1.AnnoServerlessAppStatus] == ""
						},
					}).Return(nil).Times(1)
					return executor
				}(),
			},
		},
		{
			name:           "subrsNotSynced",
			syncSubrsMetas: true,
			fields: fields{
				current: current{
					currentPod: func() *corev1.Pod {
						pod := newPodWithDeclaredKeys("test", "1", corev1.PodPending, "")
						pod.ObjectMeta.Labels = map[string]string{
							"serverless.io/app-name":       "fengx",
							"serverless.io/instance-group": "feng_host",
							carbonv1.LabelServerlessAppId:  "test3",
						}
						pod.ObjectMeta.Annotations = map[string]string{
							"app.hippo.io/appShortName":  "feng",
							"app.hippo.io/roleShortName": "fengy",
						}
						return pod
					}(),
				},
				target: target{
					worker: func() *carbonv1.WorkerNode {
						worker := newWorker("test", "1", true, "", nil, 1, 0)
						worker.ObjectMeta.Labels = map[string]string{
							"serverless.io/app-name": "feng",
						}
						worker.ObjectMeta.Annotations = map[string]string{
							"app.hippo.io/appShortName": "fengx",
						}
						worker.Spec.Template.Labels = map[string]string{
							"serverless.io/app-name": "fengy",
						}
						worker.Spec.Template.Annotations = map[string]string{
							"app.hippo.io/appShortName":            "fengy",
							"app.c2.io/biz-detail-partition-count": "5",
						}
						worker.Status.ResVersion = "2"
						sceneStatus1 := make([]carbonv1.AppNodeStatus, 0, 2)
						sceneStatus1 = append(sceneStatus1, carbonv1.AppNodeStatus{
							AppId:         "test",
							StartedAt:     1695106366,
							PlanStatus:    1,
							ServiceStatus: 3,
						})
						sceneStatus1 = append(sceneStatus1, carbonv1.AppNodeStatus{
							AppId:         "test2",
							StartedAt:     1695106366,
							PlanStatus:    1,
							ServiceStatus: 3,
						})
						sceneStatus2 := make([]carbonv1.AppNodeStatus, 0, 1)
						sceneStatus2 = append(sceneStatus2, carbonv1.AppNodeStatus{
							AppId:         "test",
							StartedAt:     1695106366,
							PlanStatus:    1,
							ServiceStatus: 3,
						})
						serviceCondition1 := carbonv1.ServiceCondition{
							ServiceDetail: utils.ObjJSON(sceneStatus1),
							Type:          carbonv1.ServiceGeneral,
						}
						serviceCondition2 := carbonv1.ServiceCondition{
							ServiceDetail: utils.ObjJSON(sceneStatus2),
							Type:          carbonv1.ServiceGeneral,
						}
						worker.Status.ServiceConditions = make([]carbonv1.ServiceCondition, 0, 2)
						worker.Status.ServiceConditions = append(worker.Status.ServiceConditions, serviceCondition1)
						worker.Status.ServiceConditions = append(worker.Status.ServiceConditions, serviceCondition2)
						return worker
					}(),
					targetPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
					podSpec: func() *corev1.PodSpec {
						pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
						return &pod.Spec
					}(),
				},
				executor: func() executor {
					executor := NewMockexecutor(ctl)
					executor.EXPECT().patchPod(gomock.Any(), testutils.FuncMockMatcher{
						Func: func(x interface{}) bool {
							data := x.(*util.CommonPatch)
							metadata := data.PatchData["metadata"]
							meta := metadata.(map[string]interface{})
							labels := meta["labels"].(map[string]string)
							annotations := meta["annotations"].(map[string]string)
							return labels["serverless.io/app-name"] == "fengy" && labels["serverless.io/instance-group"] == "NULL_HOLDER" &&
								annotations["app.hippo.io/appShortName"] == "fengy" && annotations["app.hippo.io/roleShortName"] == "NULL_HOLDER" &&
								annotations["app.c2.io/biz-detail-ready"] == "false" && annotations["app.c2.io/biz-detail-partition-count"] == "5" &&
								annotations[carbonv1.AnnoServerlessAppPlanStatus] == "" && annotations[carbonv1.AnnoServerlessAppStatus] == ""
						},
					}).Return(nil).Times(1)
					return executor
				}(),
			},
		},
		{
			name:           "serverless AppId not match",
			syncSubrsMetas: true,
			fields: fields{
				current: current{
					currentPod: func() *corev1.Pod {
						pod := newPodWithDeclaredKeys("test", "1", corev1.PodPending, "")
						pod.ObjectMeta.Labels = map[string]string{
							"serverless.io/app-name":       "fengx",
							"serverless.io/instance-group": "feng_host",
							carbonv1.LabelServerlessAppId:  "test3",
						}
						pod.ObjectMeta.Annotations = map[string]string{
							"app.hippo.io/appShortName":          "feng",
							"app.hippo.io/roleShortName":         "fengy",
							carbonv1.AnnoServerlessAppPlanStatus: "test3_app_plan_status",
							carbonv1.AnnoServerlessAppStatus:     "test3_app_status",
						}
						return pod
					}(),
				},
				target: target{
					worker: func() *carbonv1.WorkerNode {
						worker := newWorker("test", "1", true, "", nil, 1, 0)
						worker.ObjectMeta.Labels = map[string]string{
							"serverless.io/app-name": "feng",
						}
						worker.ObjectMeta.Annotations = map[string]string{
							"app.hippo.io/appShortName": "fengx",
						}
						worker.Spec.Template.Labels = map[string]string{
							"serverless.io/app-name": "fengy",
						}
						worker.Spec.Template.Annotations = map[string]string{
							"app.hippo.io/appShortName":            "fengy",
							"app.c2.io/biz-detail-partition-count": "5",
						}
						worker.Status.ResVersion = "2"
						sceneStatus1 := make([]carbonv1.AppNodeStatus, 0, 2)
						sceneStatus1 = append(sceneStatus1, carbonv1.AppNodeStatus{
							AppId:         "test",
							StartedAt:     1695106366,
							PlanStatus:    1,
							ServiceStatus: 3,
						})
						sceneStatus1 = append(sceneStatus1, carbonv1.AppNodeStatus{
							AppId:         "test2",
							StartedAt:     1695106366,
							PlanStatus:    1,
							ServiceStatus: 3,
						})
						sceneStatus2 := make([]carbonv1.AppNodeStatus, 0, 1)
						sceneStatus2 = append(sceneStatus2, carbonv1.AppNodeStatus{
							AppId:         "test",
							StartedAt:     1695106366,
							PlanStatus:    1,
							ServiceStatus: 3,
						})
						serviceCondition1 := carbonv1.ServiceCondition{
							ServiceDetail: utils.ObjJSON(sceneStatus1),
							Type:          carbonv1.ServiceGeneral,
						}
						serviceCondition2 := carbonv1.ServiceCondition{
							ServiceDetail: utils.ObjJSON(sceneStatus2),
							Type:          carbonv1.ServiceGeneral,
						}
						worker.Status.ServiceConditions = make([]carbonv1.ServiceCondition, 0, 2)
						worker.Status.ServiceConditions = append(worker.Status.ServiceConditions, serviceCondition1, serviceCondition2)
						return worker
					}(),
					targetPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
					podSpec: func() *corev1.PodSpec {
						pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
						return &pod.Spec
					}(),
				},
				executor: func() executor {
					executor := NewMockexecutor(ctl)
					executor.EXPECT().patchPod(gomock.Any(), testutils.FuncMockMatcher{
						Func: func(x interface{}) bool {
							data := x.(*util.CommonPatch)
							metadata := data.PatchData["metadata"]
							meta := metadata.(map[string]interface{})
							labels := meta["labels"].(map[string]string)
							annotations := meta["annotations"].(map[string]string)
							status := (data.PatchData["status"]).(map[string]interface{})
							conditions := (status["conditions"]).([]corev1.PodCondition)
							return labels["serverless.io/app-name"] == "fengy" && labels["serverless.io/instance-group"] == "NULL_HOLDER" &&
								annotations["app.hippo.io/appShortName"] == "fengy" && annotations["app.hippo.io/roleShortName"] == "NULL_HOLDER" &&
								annotations["app.c2.io/biz-detail-ready"] == "false" && annotations["app.c2.io/biz-detail-partition-count"] == "5" &&
								annotations[carbonv1.AnnoServerlessAppPlanStatus] == "NULL_HOLDER" && annotations[carbonv1.AnnoServerlessAppStatus] == "NULL_HOLDER" &&
								len(conditions) == 2 && conditions[0].Status == corev1.ConditionFalse && conditions[1].Status == corev1.ConditionFalse

						},
					}).Return(nil).Times(1)
					return executor
				}(),
			},
		},
		{
			name:           "subrsCloseSync",
			syncSubrsMetas: false,
			fields: fields{
				current: current{
					currentPod: func() *corev1.Pod {
						pod := newPodWithDeclaredKeys("test", "1", corev1.PodPending, "")
						pod.ObjectMeta.Labels = map[string]string{}
						pod.ObjectMeta.Annotations = map[string]string{
							"app.hippo.io/appShortName":     "feng",
							"app.hippo.io/roleShortName":    "fengy",
							"app.c2.io/biz-detail-ready":    "false",
							"app.c2.io/update-status":       "{\"targetVersion\":\"1\",\"curVersion\":\"\",\"updateAt\":0,\"phase\":3,\"healthStatus\":\"HT_ALIVE\"}",
							"app.c2.io/update-phase-status": "3",
						}
						return pod
					}(),
				},
				target: target{
					worker: func() *carbonv1.WorkerNode {
						worker := newWorker("test", "1", false, "", nil, 1, 0)
						worker.ObjectMeta.Labels = map[string]string{}
						worker.ObjectMeta.Annotations = map[string]string{
							"app.hippo.io/appShortName": "fengx",
						}
						worker.Spec.Template.Labels = map[string]string{}
						worker.Spec.Template.Annotations = map[string]string{
							"app.hippo.io/appShortName": "fengy",
						}
						worker.Status.ResVersion = "2"
						return worker
					}(),
					targetPod: func() *corev1.Pod {
						pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
						pod.ObjectMeta.Annotations = map[string]string{
							"app.hippo.io/appShortName":  "feng",
							"app.hippo.io/roleShortName": "fengy",
							"app.c2.io/biz-detail-ready": "false",
						}
						return pod
					}(),
					podSpec: func() *corev1.PodSpec {
						pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
						return &pod.Spec
					}(),
				},
			},
		},
		{
			name:           "subrsSynced",
			syncSubrsMetas: true,
			fields: fields{
				current: current{
					currentPod: func() *corev1.Pod {
						pod := newPodWithDeclaredKeys("test", "1", corev1.PodPending, "")
						pod.ObjectMeta.Labels = map[string]string{
							"serverless.io/app-name": "feng",
						}
						pod.ObjectMeta.Annotations = map[string]string{
							"app.hippo.io/appShortName":     "feng",
							"app.c2.io/biz-detail-ready":    "false",
							"app.c2.io/update-status":       "{\"targetVersion\":\"1\",\"curVersion\":\"\",\"updateAt\":0,\"phase\":3,\"healthStatus\":\"HT_ALIVE\"}",
							"app.c2.io/update-phase-status": "3",
						}
						return pod
					}(),
				},
				target: target{
					worker: func() *carbonv1.WorkerNode {
						worker := newWorker("test", "1", false, "", nil, 1, 0)
						worker.ObjectMeta.Labels = map[string]string{
							"serverless.io/app-name": "fengx",
						}
						worker.Spec.Template.Labels = map[string]string{
							"serverless.io/app-name": "feng",
						}
						worker.ObjectMeta.Annotations = map[string]string{
							"app.hippo.io/appShortName": "fengy",
						}
						worker.Spec.Template.Annotations = map[string]string{
							"app.hippo.io/appShortName": "feng",
						}
						return worker
					}(),
					targetPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
					podSpec: func() *corev1.PodSpec {
						pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
						return &pod.Spec
					}(),
				},
			},
		},
		{
			name:           "SyncUpdateStatus ",
			syncSubrsMetas: true,
			fields: fields{
				current: current{
					currentPod: func() *corev1.Pod {
						pod := newPodWithDeclaredKeys("test", "1", corev1.PodPending, "")
						pod.ObjectMeta.Labels = map[string]string{}
						labels.AddLabel(pod.Annotations, carbonv1.AnnotationWorkerUpdateStatus, "{\"targetVersion\":\"version1\",\"curVersion\":\"version1\",\"updateAt\":1697715963,\"phase\":1,\"healthStatus\":\"HT_ALIVE\"}")
						return pod
					}(),
				},
				target: target{
					worker: func() *carbonv1.WorkerNode {
						worker := newWorker("test", "1", false, "", nil, 1, 0)
						worker.Spec.Version = "version1"
						worker.Status.Version = "version2"
						worker.Status.HealthStatus = "HT_ALIVE"
						worker.Annotations[carbonv1.AnnotationBufferSwapRecord] = "{\"swapAt\":1797715963}"
						worker.Status.ServiceOffline = true
						worker.Status.ServiceConditions = []carbonv1.ServiceCondition{{
							Type:   carbonv1.ServiceGeneral,
							Status: corev1.ConditionTrue,
						}}
						return worker
					}(),
					targetPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
					podSpec: func() *corev1.PodSpec {
						pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
						return &pod.Spec
					}(),
				},
				executor: func() executor {
					executor := NewMockexecutor(ctl)
					executor.EXPECT().patchPod(gomock.Any(), testutils.FuncMockMatcher{
						Func: func(x interface{}) bool {
							data := x.(*util.CommonPatch)
							metadata := data.PatchData["metadata"]
							meta := metadata.(map[string]interface{})
							annotations := meta["annotations"].(map[string]string)
							status := carbonv1.UpdateStatus{
								Phase:         carbonv1.PhaseEvicting,
								TargetVersion: "version1",
								CurVersion:    "version2",
								HealthStatus:  "HT_ALIVE",
								UpdateAt:      1797715963,
							}
							return annotations[carbonv1.AnnotationWorkerUpdateStatus] == utils.ObjJSON(status)
						},
					}).Return(nil).Times(1)
					return executor
				}(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &podAllocator{
				schedulerType: tt.fields.schedulerType,
				target:        tt.fields.target,
				current:       tt.fields.current,
				c:             &Controller{},
				executor:      tt.fields.executor,
			}
			if tt.syncSubrsMetas {
				features.C2MutableFeatureGate.Set("SyncSubrsMetas=true")
			} else {
				features.C2MutableFeatureGate.Set("SyncSubrsMetas=false")
			}
			if err := a.syncPodMetas(); (err != nil) != tt.wantErr {
				t.Errorf("podAllocator.syncPodMetas() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_podAllocator_syncPodSpec(t *testing.T) {
	type fields struct {
		schedulerType string
		target        target
		current       current
		worker        *carbonv1.WorkerNode
		parser        podStatusParser
		syncer        podSpecSyncer
		executor      executor
	}

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	tests := []struct {
		name       string
		fields     fields
		wantErr    bool
		assertFunc func(pod *corev1.Pod) bool
	}{
		{
			name: "no need",
			fields: fields{
				target: target{
					worker: func() *carbonv1.WorkerNode {
						worker := newWorker("test", "1", true, "", nil, 1, 0)
						worker.Spec.Version = "1"
						worker.Status.Version = "1"
						worker.Spec.ResVersion = "1"
						worker.Status.ResVersion = "1"
						return worker
					}(),
				},
				current: current{
					currentPod: newPodWithDeclaredKeys("test", "1", corev1.PodPending, ""),
				},
			},
		},
		{
			name: "no need",
			fields: fields{
				current: current{},
			},
		},
		{
			name: "no need ",
			fields: fields{
				current: current{
					currentPod: newPodWithDeclaredKeys("test", "1", corev1.PodPending, ""),
				},
				target: target{
					worker: func() *carbonv1.WorkerNode {
						worker := newWorker("test", "1", true, "", nil, 1, 0)
						worker.Spec.Version = "1"
						worker.Status.Version = "1"
						worker.Spec.ResVersion = "2"
						worker.Status.ResVersion = "1"
						return worker
					}(),
					targetPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
					podSpec: func() *corev1.PodSpec {
						pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
						return &pod.Spec
					}(),
				},
				syncer: func() podSpecSyncer {
					syncer := NewMockpodSpecSyncer(ctl)
					return syncer
				}(),
				executor: func() executor {
					executor := NewMockexecutor(ctl)
					return executor
				}(),
			},
		},
		{
			name: "no need ",
			fields: fields{
				current: current{
					currentPod: newPodWithDeclaredKeys("test", "1", corev1.PodPending, ""),
				},
				target: target{
					worker: func() *carbonv1.WorkerNode {
						worker := newWorker("test", "1", true, "", nil, 1, 0)
						worker.Spec.Version = "2"
						worker.Status.Version = "1"
						worker.Spec.ResVersion = "1"
						worker.Status.ResVersion = "1"
						return worker
					}(),
					targetPod: newPodWithOutUID("test", "1", corev1.PodPending, ""),
					podSpec: func() *corev1.PodSpec {
						pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
						return &pod.Spec
					}(),
				},
				syncer: func() podSpecSyncer {
					syncer := NewMockpodSpecSyncer(ctl)
					return syncer
				}(),
				executor: func() executor {
					executor := NewMockexecutor(ctl)
					return executor
				}(),
			},
		},
		{
			name: "updateResource",
			fields: fields{
				current: current{
					currentPod: newPodWithDeclaredKeys("test", "1", corev1.PodPending, ""),
				},
				target: target{
					worker: func() *carbonv1.WorkerNode {
						worker := newWorker("test", "1", true, "", nil, 1, 0)
						worker.Spec.Version = "1"
						worker.Status.Version = "1"
						worker.Spec.ResVersion = "2"
						worker.Status.ResVersion = "1"
						return worker
					}(),
					targetResourceVersion: "a",
					targetPod:             newPodWithOutUID("test", "1", corev1.PodPending, ""),
					podSpec: func() *corev1.PodSpec {
						pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
						return &pod.Spec
					}(),
				},
				syncer: func() podSpecSyncer {
					syncer := NewMockpodSpecSyncer(ctl)
					syncer.EXPECT().updateResource(gomock.Any(), gomock.Any()).MinTimes(1)
					syncer.EXPECT().MergeWebHooks(gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(1)
					return syncer
				}(),
				executor: func() executor {
					executor := NewMockexecutor(ctl)
					executor.EXPECT().updatePod(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).MinTimes(1)
					return executor
				}(),
			},
		},
		{
			name: "updateProcess",
			fields: fields{
				current: current{
					currentPod: newPodWithDeclaredKeys("test", "1", corev1.PodPending, ""),
				},
				target: target{
					worker: func() *carbonv1.WorkerNode {
						worker := newWorker("test", "1", true, "", nil, 1, 0)
						worker.Spec.Version = "2"
						worker.Status.Version = "1"
						worker.Spec.ResVersion = "1"
						worker.Status.ResVersion = "1"
						return worker
					}(),
					targetPodVersion: "a",
					targetPod:        newPodWithOutUID("test", "1", corev1.PodPending, ""),
					podSpec: func() *corev1.PodSpec {
						pod := newPodWithOutUID("test", "1", corev1.PodPending, "")
						return &pod.Spec
					}(),
				},
				syncer: func() podSpecSyncer {
					syncer := NewMockpodSpecSyncer(ctl)
					syncer.EXPECT().updateProcess(gomock.Any()).MinTimes(1)
					syncer.EXPECT().postSync(gomock.Any()).MinTimes(1)
					syncer.EXPECT().MergeWebHooks(gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(1)
					return syncer
				}(),
				executor: func() executor {
					executor := NewMockexecutor(ctl)
					executor.EXPECT().updatePod(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).MinTimes(1)
					return executor
				}(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &podAllocator{
				schedulerType: tt.fields.schedulerType,
				target:        tt.fields.target,
				current:       tt.fields.current,

				parser:   tt.fields.parser,
				syncer:   tt.fields.syncer,
				executor: tt.fields.executor,
			}
			a.pSpecMerger = &util.PodSpecMerger{}
			if err := a.syncPodSpec(); (err != nil) != tt.wantErr {
				t.Errorf("podAllocator.syncPodSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.assertFunc != nil && !tt.assertFunc(a.targetPod) {
				t.Errorf("podAllocator.syncPodSpec() assert error: %s", tt.name)
			}
		})
	}
}

func Test_podAllocator_syncWorkerAllocStatus(t *testing.T) {
	type fields struct {
		schedulerType string
		target        target
		current       current
		parser        podStatusParser
		syncer        podSpecSyncer
		executor      executor
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		want    carbonv1.WorkerNodeStatus
	}{
		{
			name: "sync",
			fields: fields{
				target: target{
					worker: &carbonv1.WorkerNode{
						Status: carbonv1.WorkerNodeStatus{},
					},
				},
				current: current{
					currentPod: &corev1.Pod{
						Spec: corev1.PodSpec{
							NodeName: "10.83.178.64-7707",
						},
						Status: corev1.PodStatus{},
					},
				},
				parser: &basePodStatusParser{},
			},
			wantErr: false,
			want: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					EntityAlloced: true,
					Phase:         carbonv1.Unknown,
					PodNotStarted: true,
				},
				AllocStatus:   carbonv1.WorkerUnAssigned,
				HealthStatus:  carbonv1.HealthUnKnown,
				ServiceStatus: carbonv1.ServiceUnAvailable,
			},
		},
		{
			name: "sync",
			fields: fields{
				target: target{
					worker: &carbonv1.WorkerNode{},
				},
				current: current{
					currentPod: &corev1.Pod{
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
				parser: &basePodStatusParser{},
			},
			wantErr: false,
			want: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					HostIP:        "1.1.1.1",
					IP:            "1.1.1.2",
					EntityAlloced: true,
					Phase:         carbonv1.Unknown,
				},
				AllocStatus:   carbonv1.WorkerUnAssigned,
				HealthStatus:  carbonv1.HealthUnKnown,
				ServiceStatus: carbonv1.ServiceUnAvailable,
			},
		},
		{
			name: "sync",
			fields: fields{
				target: target{
					worker: &carbonv1.WorkerNode{
						Status: carbonv1.WorkerNodeStatus{},
					},
				},
				current: current{
					currentPod: &corev1.Pod{
						Spec: corev1.PodSpec{
							NodeName:    "10.83.178.64-7707",
							HostNetwork: true,
						},
						Status: corev1.PodStatus{},
					},
				},
				parser: &basePodStatusParser{},
			},
			wantErr: false,
			want: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					EntityAlloced: true,
					Phase:         carbonv1.Unknown,
					PodNotStarted: true,
				},
				AllocStatus:   carbonv1.WorkerUnAssigned,
				HealthStatus:  carbonv1.HealthUnKnown,
				ServiceStatus: carbonv1.ServiceUnAvailable,
			},
		},
		{
			name: "sync",
			fields: fields{
				target: target{
					worker: &carbonv1.WorkerNode{
						Status: carbonv1.WorkerNodeStatus{
							AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
								EntityAlloced: true,
								Phase:         carbonv1.Unknown,
								PodNotStarted: true,
							},
							AllocStatus:   carbonv1.WorkerAssigned,
							HealthStatus:  carbonv1.HealthUnKnown,
							ServiceStatus: carbonv1.ServiceUnAvailable,
						},
					},
				},
				current: current{
					currentPod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							DeletionTimestamp: &metav1.Time{},
							Name:              "pod-NameA",
							UID:               "pod-UidA",
						},
						Spec: corev1.PodSpec{
							NodeName:    "10.83.178.64-7707",
							HostNetwork: true,
						},
						Status: corev1.PodStatus{},
					},
				},
				parser: &basePodStatusParser{},
			},
			wantErr: false,
			want: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					EntityAlloced: false,
					Phase:         carbonv1.Unknown,
					PodNotStarted: true,
					EntityUid:     "pod-UidA",
					EntityName:    "pod-NameA",
				},
				AllocStatus:   carbonv1.WorkerAssigned,
				HealthStatus:  carbonv1.HealthUnKnown,
				ServiceStatus: carbonv1.ServiceUnAvailable,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &podAllocator{
				schedulerType: tt.fields.schedulerType,
				target:        tt.fields.target,
				current:       tt.fields.current,

				parser:   tt.fields.parser,
				syncer:   tt.fields.syncer,
				executor: tt.fields.executor,
			}
			err := a.syncWorkerAllocStatus()
			if (err != nil) != tt.wantErr {
				t.Errorf("podAllocator.syncWorkerAllocStatus() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !reflect.DeepEqual(tt.want, tt.fields.target.worker.Status) {
				t.Errorf("hippoPodAllocator.syncWorkerAllocStatus() error = %v, want %v", utils.ObjJSON(tt.fields.target.worker.Status), utils.ObjJSON(tt.want))
			}
		})
	}
}

func Test_podAllocator_syncWorkerResourceStatus(t *testing.T) {
	type fields struct {
		schedulerType string
		target        target
		current       current
		worker        *carbonv1.WorkerNode
		parser        podStatusParser
		syncer        podSpecSyncer
		executor      executor
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		want    carbonv1.WorkerNodeStatus
	}{
		{
			name: "match",
			fields: fields{
				target: target{
					worker:                newWorker("tet", "", false, "", nil, 1, 0),
					targetResourceVersion: "1",
				},
				current: current{
					currentPod: func() *corev1.Pod {
						pod := newPodWithStatusExtend("test", "ip1", corev1.PodRunning, 2, "requirementID2", true)
						pod.Status.Conditions = append(pod.Status.Conditions, corev1.PodCondition{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						})
						return pod
					}(),
					currentResourceVersion: "1",
					currentResourceMatched: true,
				},
				parser: &basePodStatusParser{},
			},
			want: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					Phase:         carbonv1.Running,
					ResourceMatch: true,
					ResVersion:    "2",
				},
				AllocStatus:   carbonv1.WorkerAssigned,
				ServiceStatus: carbonv1.ServicePartAvailable,
				HealthStatus:  carbonv1.HealthAlive,
			},
		},
		{
			name: "not match",
			fields: fields{
				target: target{
					worker:                newWorker("tet", "", false, "", nil, 1, 0),
					targetResourceVersion: "1",
				},
				current: current{
					currentPod: func() *corev1.Pod {
						pod := newPodWithStatusExtend("test", "ip1", corev1.PodRunning, 2, "requirementID2", true)
						pod.Status.Conditions = append(pod.Status.Conditions, corev1.PodCondition{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						})
						return pod
					}(),
					currentResourceVersion: "1",
					currentResourceMatched: false,
				},
				parser: &basePodStatusParser{},
			},
			want: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					Phase: carbonv1.Running,
				},
				AllocStatus:   carbonv1.WorkerAssigned,
				ServiceStatus: carbonv1.ServicePartAvailable,
				HealthStatus:  carbonv1.HealthAlive,
			},
		},
		{
			name: "compatible not match",
			fields: fields{
				target: target{
					worker: func() *carbonv1.WorkerNode {
						worker := newWorker("tet", "", false, "", nil, 1, 0)
						worker.Status.ResVersion = worker.Spec.ResVersion
						return worker
					}(),
					targetResourceVersion: "1",
				},
				current: current{
					currentPod: func() *corev1.Pod {
						pod := newPodWithStatusExtend("test", "ip1", corev1.PodRunning, 2, "requirementID2", true)
						pod.Status.Conditions = append(pod.Status.Conditions, corev1.PodCondition{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						})
						return pod
					}(),
					currentResourceVersion: "1",
					currentResourceMatched: false,
				},
				parser: &basePodStatusParser{},
			},
			want: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					ResourceMatch: true,
					Phase:         carbonv1.Running,
					ResVersion:    "2",
				},
				AllocStatus:   carbonv1.WorkerAssigned,
				ServiceStatus: carbonv1.ServicePartAvailable,
				HealthStatus:  carbonv1.HealthAlive,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &podAllocator{
				schedulerType: tt.fields.schedulerType,
				target:        tt.fields.target,
				current:       tt.fields.current,

				parser:   tt.fields.parser,
				syncer:   tt.fields.syncer,
				executor: tt.fields.executor,
			}
			err := a.syncWorkerResourceStatus()
			if (err != nil) != tt.wantErr {
				t.Errorf("podAllocator.syncWorkerResourceStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
			tt.fields.target.worker.Status.LastResourceNotMatchtime = 0
			if !reflect.DeepEqual(tt.want, tt.fields.target.worker.Status) {
				t.Errorf("hippoPodAllocator.syncWorkerAllocStatus() error = %v, want %v", utils.ObjJSON(tt.fields.target.worker.Status), utils.ObjJSON(tt.want))
			}
		})
	}
}

func Test_podAllocator_syncWorkerProcessStatus(t *testing.T) {
	type fields struct {
		schedulerType string
		target        target
		current       current
		worker        *carbonv1.WorkerNode
		parser        podStatusParser
		syncer        podSpecSyncer
		executor      executor
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		want    carbonv1.WorkerNodeStatus
	}{
		{
			name: "match",
			fields: fields{
				target: target{
					worker: func() *carbonv1.WorkerNode {
						worker := newWorker("tet", "", false, "", nil, 1, 0)
						worker.Status.ResourceMatch = true
						return worker
					}(),
					targetProcessVersion: "1",
					targetContainersProcessVersion: map[string]string{
						"a": "a",
						"b": "b",
					},
				},
				current: current{
					currentPod: func() *corev1.Pod {
						pod := newPodWithStatusExtend("test", "ip1", corev1.PodRunning, 2, "requirementID2", true)
						pod.Status.Conditions = append(pod.Status.Conditions, corev1.PodCondition{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						})
						return pod
					}(),
					currentProcessVersion: "1",
					currentContainersProcessVersion: map[string]string{
						"a": "a",
						"b": "b",
					},
				},
				parser: &basePodStatusParser{},
			},
			want: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					Phase:         carbonv1.Running,
					ProcessMatch:  true,
					Version:       "1",
					ProcessReady:  true,
					ResourceMatch: true,
				},
				AllocStatus:   carbonv1.WorkerAssigned,
				ServiceStatus: carbonv1.ServicePartAvailable,
				HealthStatus:  carbonv1.HealthAlive,
			},
		},
		{
			name: "not match",
			fields: fields{
				target: target{
					worker:               newWorker("tet", "", false, "", nil, 1, 0),
					targetProcessVersion: "1",
					targetContainersProcessVersion: map[string]string{
						"a": "a",
						"b": "b",
					},
				},
				current: current{
					currentPod: func() *corev1.Pod {
						pod := newPodWithStatusExtend("test", "ip1", corev1.PodRunning, 2, "requirementID2", true)
						pod.Status.Conditions = append(pod.Status.Conditions, corev1.PodCondition{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						})
						return pod
					}(),
					currentProcessVersion: "1",
					currentContainersProcessVersion: map[string]string{
						"a": "a",
						"b": "b",
					},
				},
				parser: &basePodStatusParser{},
			},
			want: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					Phase:        carbonv1.Running,
					ProcessMatch: true,
					Version:      "",
					ProcessReady: true,
				},
				AllocStatus:   carbonv1.WorkerAssigned,
				ServiceStatus: carbonv1.ServicePartAvailable,
				HealthStatus:  carbonv1.HealthAlive,
			},
		},
		{
			name: "not match",
			fields: fields{
				target: target{
					worker:               newWorker("tet", "", false, "", nil, 1, 0),
					targetProcessVersion: "1",
					targetContainersProcessVersion: map[string]string{
						"a": "a",
						"b": "b",
					},
				},
				current: current{
					currentPod: func() *corev1.Pod {
						pod := newPodWithStatusExtend("test", "ip1", corev1.PodRunning, 2, "requirementID2", true)
						pod.Status.Conditions = append(pod.Status.Conditions, corev1.PodCondition{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						})
						return pod
					}(),
					currentProcessVersion: "2",
					currentContainersProcessVersion: map[string]string{
						"a": "a",
						"b": "b",
					},
				},
				parser: &basePodStatusParser{},
			},
			want: carbonv1.WorkerNodeStatus{
				AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
					Phase:        carbonv1.Running,
					ProcessReady: true,
				},
				AllocStatus:   carbonv1.WorkerAssigned,
				ServiceStatus: carbonv1.ServicePartAvailable,
				HealthStatus:  carbonv1.HealthAlive,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &podAllocator{
				schedulerType: tt.fields.schedulerType,
				target:        tt.fields.target,
				current:       tt.fields.current,

				parser:   tt.fields.parser,
				syncer:   tt.fields.syncer,
				executor: tt.fields.executor,
			}
			err := a.syncWorkerProcessStatus()
			if (err != nil) != tt.wantErr {
				t.Errorf("podAllocator.syncWorkerProcessStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
			tt.fields.target.worker.Status.LastProcessNotMatchtime = 0
			if !reflect.DeepEqual(tt.want, tt.fields.target.worker.Status) {
				t.Errorf("hippoPodAllocator.syncWorkerAllocStatus() error = %v, want %v", utils.ObjJSON(tt.fields.target.worker.Status), utils.ObjJSON(tt.want))
			}
		})
	}
}

func Test_podAllocator_addSuezWorkerEnv(t *testing.T) {
	type fields struct {
		schedulerType string
		target        target
		current       current
		parser        podStatusParser
		syncer        podSpecSyncer
		executor      executor
		pSpecMerger   *util.PodSpecMerger
		c             *Controller
	}
	type args struct {
		worker       *carbonv1.WorkerNode
		hippoPodSpec *carbonv1.HippoPodSpec
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *carbonv1.HippoPodSpec
	}{
		{
			name:   "normal",
			fields: fields{},
			args: args{
				hippoPodSpec: func() *carbonv1.HippoPodSpec {
					worker1 := newWorker("test", "", false, "", nil, 100, 1)
					worker1.Spec.RestartAfterResourceChange = utils.BoolPtr(false)
					hippoPodSpec1, _, _ := carbonv1.GetPodSpecFromTemplate(worker1.Spec.Template)
					return hippoPodSpec1
				}(),
				worker: func() *carbonv1.WorkerNode {
					worker1 := newWorker("test", "", false, "", nil, 100, 1)
					return worker1
				}(),
			},
			want: func() *carbonv1.HippoPodSpec {
				worker1 := newWorker("test", "", false, "", nil, 100, 1)
				worker1.Spec.RestartAfterResourceChange = utils.BoolPtr(false)
				hippoPodSpec1, _, _ := carbonv1.GetPodSpecFromTemplate(worker1.Spec.Template)
				return hippoPodSpec1
			}(),
		},
		{
			name:   "inject",
			fields: fields{},
			args: args{
				hippoPodSpec: func() *carbonv1.HippoPodSpec {
					worker1 := newWorker("test", "", false, "", nil, 100, 1)
					worker1.Spec.RestartAfterResourceChange = utils.BoolPtr(false)
					hippoPodSpec1, _, _ := carbonv1.GetPodSpecFromTemplate(worker1.Spec.Template)
					return hippoPodSpec1
				}(),
				worker: func() *carbonv1.WorkerNode {
					worker1 := newWorker("test", "", false, "", nil, 100, 1)
					worker1.Spec.Template.Labels = map[string]string{"app.c2.io/inject-identifier": "true"}
					return worker1
				}(),
			},
			want: func() *carbonv1.HippoPodSpec {
				worker1 := newWorker("test", "", false, "", nil, 100, 1)
				worker1.Spec.RestartAfterResourceChange = utils.BoolPtr(false)
				hippoPodSpec1, _, _ := carbonv1.GetPodSpecFromTemplate(worker1.Spec.Template)
				for i := range hippoPodSpec1.Containers {
					hippoPodSpec1.Containers[i].Env = []corev1.EnvVar{
						{
							Name:  "WORKER_IDENTIFIER_FOR_CARBON",
							Value: ":default:test",
						},
					}
				}
				return hippoPodSpec1
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &podAllocator{
				schedulerType: tt.fields.schedulerType,
				target:        tt.fields.target,
				current:       tt.fields.current,
				parser:        tt.fields.parser,
				syncer:        tt.fields.syncer,
				executor:      tt.fields.executor,
				pSpecMerger:   tt.fields.pSpecMerger,
				c:             tt.fields.c,
			}
			a.addSuezWorkerEnv(tt.args.worker, tt.args.hippoPodSpec)
			if !reflect.DeepEqual(tt.args.hippoPodSpec, tt.want) {
				t.Errorf("podAllocator.generatePodSpec() got2 = %v, want %v", utils.ObjJSON(tt.args.hippoPodSpec), utils.ObjJSON(tt.want))
			}
		})
	}
}

func Test_podAllocator_addPatchForInplaceUpdate(t *testing.T) {
	type fields struct {
		current current
		target  target
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
		want    func(patch *util.CommonPatch) bool
	}{
		{
			name: "updating_and_has_protectionFinalizer",
			fields: fields{
				current: current{
					currentPod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Finalizers: []string{"protection.pod.beta1.sigma.ali/alarming-on"},
						},
					},
					currentProcessVersion: "1",
				},
				target: target{targetProcessVersion: "2"},
			},
			wantErr: false,
			want: func(patch *util.CommonPatch) bool {
				exit, v := parseLabel(carbonv1.LabelFinalStateUpgrading, patch)
				if exit {
					return v == "true"
				}
				return false
			},
		},
		{
			name: "updating_and_has_not_protectionFinalizer",
			fields: fields{
				current: current{
					currentPod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Finalizers: []string{},
						},
					},
					currentProcessVersion: "1",
				},
				target: target{targetProcessVersion: "2"},
			},
			wantErr: false,
			want: func(patch *util.CommonPatch) bool {
				exit, _ := parseLabel(carbonv1.LabelFinalStateUpgrading, patch)
				if !exit {
					return true
				}
				return false
			},
		},
		{
			name: "not_update",
			fields: fields{
				current: current{
					currentPod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Finalizers: []string{"protection.pod.beta1.sigma.ali/alarming-on"},
						},
					},
					currentProcessVersion: "1",
				},
				target: target{targetProcessVersion: "1"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &podAllocator{
				current: tt.fields.current,
				target:  tt.fields.target,
			}
			a.worker = &carbonv1.WorkerNode{}
			a.currentPod.ObjectMeta.Finalizers = tt.fields.current.currentPod.Finalizers
			patch := &util.CommonPatch{PatchType: types.StrategicMergePatchType, PatchData: map[string]interface{}{}}
			a.addPatchForInplaceUpdate(patch)
			if tt.want != nil && !tt.want(patch) {
				t.Errorf("podAllocator.addPatchForInplaceUpdate() assert patch falied, name: %s", tt.name)
			}
		})
	}
}

func Test_podAllocator_resetWarmStandbyContainer(t *testing.T) {
	type fields struct {
		schedulerType string
		target        target
		current       current
		parser        podStatusParser
		syncer        podSpecSyncer
		executor      executor
		pSpecMerger   *util.PodSpecMerger
	}
	type args struct {
		worker       *carbonv1.WorkerNode
		hippoPodSpec *carbonv1.HippoPodSpec
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    *carbonv1.HippoPodSpec
	}{
		{
			name:   "normal",
			fields: fields{},
			args: args{
				hippoPodSpec: func() *carbonv1.HippoPodSpec {
					worker1 := newWorker("test", "", false, "", nil, 100, 1)
					worker1.Spec.RestartAfterResourceChange = utils.BoolPtr(false)
					hippoPodSpec1, _, _ := carbonv1.GetPodSpecFromTemplate(worker1.Spec.Template)
					return hippoPodSpec1
				}(),
				worker: func() *carbonv1.WorkerNode {
					worker1 := newWorker("test", "", false, "", nil, 100, 1)
					return worker1
				}(),
			},
			want: func() *carbonv1.HippoPodSpec {
				worker1 := newWorker("test", "", false, "", nil, 100, 1)
				worker1.Spec.RestartAfterResourceChange = utils.BoolPtr(false)
				hippoPodSpec1, _, _ := carbonv1.GetPodSpecFromTemplate(worker1.Spec.Template)
				return hippoPodSpec1
			}(),
			wantErr: false,
		},
		{
			name:   "warm",
			fields: fields{},
			args: args{
				hippoPodSpec: func() *carbonv1.HippoPodSpec {
					worker1 := newWorker("test", "", false, "", nil, 100, 1)
					worker1.Spec.RestartAfterResourceChange = utils.BoolPtr(false)
					hippoPodSpec1, _, _ := carbonv1.GetPodSpecFromTemplate(worker1.Spec.Template)
					return hippoPodSpec1
				}(),
				worker: func() *carbonv1.WorkerNode {
					worker1 := newWorker("test", "", false, "", nil, 100, 1)
					worker1.Spec.WorkerMode = carbonv1.WorkerModeTypeWarm
					return worker1
				}(),
			},
			want: func() *carbonv1.HippoPodSpec {
				worker1 := newWorker("test", "", false, "", nil, 100, 1)
				worker1.Spec.RestartAfterResourceChange = utils.BoolPtr(false)
				hippoPodSpec1, _, _ := carbonv1.GetPodSpecFromTemplate(worker1.Spec.Template)
				for i := range hippoPodSpec1.Containers {
					hippoPodSpec1.Containers[i].Env = []corev1.EnvVar{
						{
							Name:  "IS_WARM_STANDBY",
							Value: "true",
						},
					}
					hippoPodSpec1.Containers[i].Command = []string{"sleep", "1000d"}
				}
				return hippoPodSpec1
			}(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &podAllocator{
				schedulerType: tt.fields.schedulerType,
				target:        tt.fields.target,
				current:       tt.fields.current,
				parser:        tt.fields.parser,
				syncer:        tt.fields.syncer,
				executor:      tt.fields.executor,
				pSpecMerger:   tt.fields.pSpecMerger,
			}
			err := a.resetWarmStandbyContainer(tt.args.worker, tt.args.hippoPodSpec)
			if (err != nil) != tt.wantErr {
				t.Errorf("podAllocator.resetWarmStandbyContainer() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.hippoPodSpec, tt.want) {
				t.Errorf("podAllocator.generatePodSpec() got2 = %v, want %v", utils.ObjJSON(tt.args.hippoPodSpec), utils.ObjJSON(tt.want))
			}
		})
	}
}

func Test_podAllocator_generatePodSpec(t *testing.T) {
	type fields struct {
		schedulerType string
		target        target
		current       current
		worker        *carbonv1.WorkerNode
		parser        podStatusParser
		syncer        podSpecSyncer
		executor      executor
	}
	type args struct {
		worker            *carbonv1.WorkerNode
		hippoPodSpec      *carbonv1.HippoPodSpec
		version           string
		containersVersion map[string]string
	}

	ctl := gomock.NewController(t)
	defer ctl.Finish()
	syncer := NewMockpodSpecSyncer(ctl)
	syncer.EXPECT().preSync(gomock.Any(), gomock.Any(), gomock.Any()).MinTimes(3)

	res := resource.NewMilliQuantity(int64(1), resource.DecimalSI)
	res1 := resource.NewMilliQuantity(int64(1000), resource.DecimalSI)
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *corev1.PodSpec
		want1   *carbonv1.HippoPodSpecExtend
		want2   map[string]string
		want3   map[string]string
		wantErr bool
	}{
		{
			name: "generatePodSpec-cpusetmode-share",
			fields: fields{
				syncer: syncer,
			},
			args: args{
				hippoPodSpec: func() *carbonv1.HippoPodSpec {
					worker1 := newWorker("test", "", false, "", nil, 100, 1)
					worker1.Spec.RestartAfterResourceChange = utils.BoolPtr(false)
					hippoPodSpec1, _, _ := carbonv1.GetPodSpecFromTemplate(worker1.Spec.Template)
					return hippoPodSpec1
				}(),
				worker: func() *carbonv1.WorkerNode {
					worker1 := newWorker("test", "", false, "", nil, 100, 1)
					worker1.Spec.RestartAfterResourceChange = utils.BoolPtr(false)
					worker1.Labels = map[string]string{
						"app.hippo.io/app-name":    "app",
						"app.hippo.io/role-name":   "role",
						"sigma.ali/instance-group": "tpp",
					}
					return worker1
				}(),
				version: "1",
				containersVersion: map[string]string{
					"container": "a",
				},
			},
			want: &corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "container",
						Image: "image",
						Resources: corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								"cpu": *res,
							},
							Requests: map[corev1.ResourceName]resource.Quantity{
								"cpu": *res,
							},
						},
					},
				},
				NodeName: "node1",
				Priority: utils.Int32Ptr(100),
			},
			want1: &carbonv1.HippoPodSpecExtend{
				Containers: []carbonv1.ContainerHippoExternedWithName{
					{
						Name:       "container",
						InstanceID: "a",
						ContainerHippoExterned: carbonv1.ContainerHippoExterned{
							PreDeployImage: "preimage",
						},
					},
				},
				HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
					CpusetMode:     "SHARE",
					ContainerModel: "DOCKER",
				},
				InstanceID: "1",
			},
			want2: map[string]string{
				"app.hippo.io/app-name":    "app",
				"app.hippo.io/role-name":   "role",
				"sigma.ali/instance-group": "tpp",
			},
			want3: map[string]string{
				"app.hippo.io/app-name":                    "app",
				"app.hippo.io/role-name":                   "role",
				"pods.sigma.alibaba-inc.com/inject-pod-sn": "true",
			},
			wantErr: false,
		},
		{
			name: "generatePodSpec-cpusetmode-reserve",
			fields: fields{
				syncer: syncer,
			},
			args: args{
				hippoPodSpec: func() *carbonv1.HippoPodSpec {
					worker1 := newWorker("test", "", false, "", nil, 100, 1000)
					worker1.Spec.RestartAfterResourceChange = utils.BoolPtr(false)
					hippoPodSpec1, _, _ := carbonv1.GetPodSpecFromTemplate(worker1.Spec.Template)
					return hippoPodSpec1
				}(),
				worker: func() *carbonv1.WorkerNode {
					worker1 := newWorker("test", "", false, "", nil, 100, 1000)
					worker1.Spec.RestartAfterResourceChange = utils.BoolPtr(false)
					worker1.Labels = map[string]string{
						"app.hippo.io/app-name":    "app",
						"app.hippo.io/role-name":   "role",
						"sigma.ali/instance-group": "tpp",
					}
					return worker1
				}(),
				version: "1",
				containersVersion: map[string]string{
					"container": "a",
				},
			},
			want: &corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "container",
						Image: "image",
						Resources: corev1.ResourceRequirements{
							Limits: map[corev1.ResourceName]resource.Quantity{
								"cpu": *res1,
							},
							Requests: map[corev1.ResourceName]resource.Quantity{
								"cpu": *res1,
							},
						},
					},
				},
				NodeName: "node1",
				Priority: utils.Int32Ptr(100),
			},
			want1: &carbonv1.HippoPodSpecExtend{
				Containers: []carbonv1.ContainerHippoExternedWithName{
					{
						Name:       "container",
						InstanceID: "a",
						ContainerHippoExterned: carbonv1.ContainerHippoExterned{
							PreDeployImage: "preimage",
						},
					},
				},
				HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
					CpusetMode:     "RESERVED",
					ContainerModel: "DOCKER",
				},
				InstanceID: "1",
			},
			want2: map[string]string{
				"app.hippo.io/app-name":    "app",
				"app.hippo.io/role-name":   "role",
				"sigma.ali/instance-group": "tpp",
			},
			want3: map[string]string{
				"app.hippo.io/app-name":                    "app",
				"app.hippo.io/role-name":                   "role",
				"pods.sigma.alibaba-inc.com/inject-pod-sn": "true",
			},
			wantErr: false,
		},
		{
			name: "generatePodSpec",
			fields: fields{
				syncer: syncer,
			},
			args: args{
				hippoPodSpec: func() *carbonv1.HippoPodSpec {
					worker := newWorker("test", "", false, "", nil, 1, 0)
					worker.Spec.RestartAfterResourceChange = utils.BoolPtr(false)
					hippoPodSpec, _, _ := carbonv1.GetPodSpecFromTemplate(worker.Spec.Template)
					return hippoPodSpec
				}(),
				worker: func() *carbonv1.WorkerNode {
					worker := newWorker("test", "", false, "", nil, 1, 0)
					worker.Spec.RestartAfterResourceChange = utils.BoolPtr(false)
					worker.Labels = map[string]string{
						"app.hippo.io/app-name":    "app",
						"app.hippo.io/role-name":   "role",
						"sigma.ali/instance-group": "tpp",
					}
					return worker
				}(),
				version: "1",
				containersVersion: map[string]string{
					"container": "a",
				},
			},
			want: &corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "container",
						Image: "image",
					},
				},
				NodeName: "node1",
				Priority: utils.Int32Ptr(1),
			},
			want1: &carbonv1.HippoPodSpecExtend{
				Containers: []carbonv1.ContainerHippoExternedWithName{
					{
						Name:       "container",
						InstanceID: "a",
						ContainerHippoExterned: carbonv1.ContainerHippoExterned{
							PreDeployImage: "preimage",
						},
					},
				},
				HippoPodSpecExtendFields: carbonv1.HippoPodSpecExtendFields{
					CpusetMode:     "NONE",
					ContainerModel: "DOCKER",
				},
				InstanceID: "1",
			},
			want2: map[string]string{
				"app.hippo.io/app-name":    "app",
				"app.hippo.io/role-name":   "role",
				"sigma.ali/instance-group": "tpp",
			},
			want3: map[string]string{
				"app.hippo.io/app-name":                    "app",
				"app.hippo.io/role-name":                   "role",
				"pods.sigma.alibaba-inc.com/inject-pod-sn": "true",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &podAllocator{
				schedulerType: tt.fields.schedulerType,
				target:        tt.fields.target,
				current:       tt.fields.current,
				parser:        tt.fields.parser,
				syncer:        tt.fields.syncer,
				executor:      tt.fields.executor,
			}
			got, got1, got2, got3, err := a.generatePodSpec(tt.args.worker, tt.args.hippoPodSpec, tt.args.version, tt.args.containersVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("podAllocator.generatePodSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if utils.ObjJSON(got) != utils.ObjJSON(tt.want) {
				t.Errorf("podAllocator.generatePodSpec() got = %v, want %v", utils.ObjJSON(got), utils.ObjJSON(tt.want))
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("podAllocator.generatePodSpec() got1 = %v, want %v", utils.ObjJSON(got1), utils.ObjJSON(tt.want1))
			}
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("podAllocator.generatePodSpec() got2 = %v, want %v", got2, tt.want2)
			}
			if !reflect.DeepEqual(got3, tt.want3) {
				t.Errorf("podAllocator.generatePodSpec() got3 = %v, want %v", got3, tt.want3)
			}
		})
	}
}

func generateRestartRecords(len int, exceed bool) string {
	now := time.Now().Unix()
	records := make([]string, 0, len)
	for i := len; i > 0; i-- {
		record := now - (1<<i)*carbonv1.ExponentialBackoff
		if exceed {
			record += carbonv1.ExponentialBackoff
		}
		records = append(records, fmt.Sprintf("%d", record))
	}
	return strings.Join(records, ",")
}

func Test_podAllocator_isPodProcessFailed(t *testing.T) {
	type fields struct {
		schedulerType string
		target        target
		current       current
		parser        podStatusParser
		syncer        podSpecSyncer
		worker        *carbonv1.WorkerNode
		executor      executor
		pSpecMerger   *util.PodSpecMerger
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "failed",
			fields: fields{
				target: target{
					hippoPodSpec: &carbonv1.HippoPodSpec{
						Containers: []carbonv1.HippoContainer{
							{
								ContainerHippoExterned: carbonv1.ContainerHippoExterned{
									Configs: &carbonv1.ContainerConfig{
										RestartCountLimit: 10,
									},
								},
							},
						},
					},
				},
				current: current{
					currentPod: &corev1.Pod{
						Status: corev1.PodStatus{
							ContainerStatuses: []corev1.ContainerStatus{
								{
									Ready:        false,
									RestartCount: 11,
									LastTerminationState: corev1.ContainerState{
										Terminated: &corev1.ContainerStateTerminated{},
									},
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "normal",
			fields: fields{
				target: target{
					hippoPodSpec: &carbonv1.HippoPodSpec{
						Containers: []carbonv1.HippoContainer{
							{
								ContainerHippoExterned: carbonv1.ContainerHippoExterned{
									Configs: &carbonv1.ContainerConfig{},
								},
							},
						},
					},
				},
				current: current{
					currentPod: &corev1.Pod{
						Status: corev1.PodStatus{
							ContainerStatuses: []corev1.ContainerStatus{
								{
									Ready:        false,
									RestartCount: 11,
									LastTerminationState: corev1.ContainerState{
										Terminated: &corev1.ContainerStateTerminated{},
									},
								},
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "failed",
			fields: fields{
				target: target{
					hippoPodSpec: &carbonv1.HippoPodSpec{
						Containers: []carbonv1.HippoContainer{
							{
								ContainerHippoExterned: carbonv1.ContainerHippoExterned{
									Configs: &carbonv1.ContainerConfig{
										RestartCountLimit: 10,
									},
								},
							},
						},
					},
				},
				current: current{
					currentPod: &corev1.Pod{
						Status: corev1.PodStatus{
							ContainerStatuses: []corev1.ContainerStatus{
								{
									Ready:        true,
									RestartCount: 11,
									LastTerminationState: corev1.ContainerState{
										Terminated: &corev1.ContainerStateTerminated{},
									},
								},
							},
						},
					},
				},
			},
			want: false,
		},

		{
			name: "failed",
			fields: fields{
				target: target{
					hippoPodSpec: &carbonv1.HippoPodSpec{
						Containers: []carbonv1.HippoContainer{
							{
								ContainerHippoExterned: carbonv1.ContainerHippoExterned{
									Configs: &carbonv1.ContainerConfig{
										RestartCountLimit: 10,
									},
								},
							},
						},
					},
				},
				current: current{
					currentPod: &corev1.Pod{
						Status: corev1.PodStatus{
							ContainerStatuses: []corev1.ContainerStatus{
								{
									Ready:                false,
									RestartCount:         11,
									LastTerminationState: corev1.ContainerState{},
								},
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "failed",
			fields: fields{
				target: target{
					hippoPodSpec: &carbonv1.HippoPodSpec{
						Containers: []carbonv1.HippoContainer{
							{
								ContainerHippoExterned: carbonv1.ContainerHippoExterned{
									Configs: &carbonv1.ContainerConfig{
										RestartCountLimit: 10,
									},
								},
							},
						},
					},
				},
				current: current{
					currentPod: &corev1.Pod{
						Status: corev1.PodStatus{
							ContainerStatuses: []corev1.ContainerStatus{
								{
									Ready:        false,
									RestartCount: 1,
									LastTerminationState: corev1.ContainerState{
										Terminated: &corev1.ContainerStateTerminated{},
									},
								},
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "restartLimitation",
			fields: fields{
				target: target{
					hippoPodSpec: &carbonv1.HippoPodSpec{
						Containers: []carbonv1.HippoContainer{
							{
								Container: corev1.Container{Name: "main"},
							},
							{
								Container: corev1.Container{Name: "start"},
								ContainerHippoExterned: carbonv1.ContainerHippoExterned{
									Configs: &carbonv1.ContainerConfig{
										RestartCountLimit: 5,
									},
								},
							},
						},
					},
					worker: &carbonv1.WorkerNode{
						Status: carbonv1.WorkerNodeStatus{
							RestartRecords: map[string]string{
								"main":  generateRestartRecords(10, false),
								"start": generateRestartRecords(5, true),
							},
						},
					},
				},
				current: current{
					currentPod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								carbonv1.LabelKeyPodVersion: carbonv1.PodVersion3,
							},
						},
						Status: corev1.PodStatus{
							ContainerStatuses: []corev1.ContainerStatus{
								{
									Ready:        false,
									Name:         "main",
									RestartCount: 1,
									LastTerminationState: corev1.ContainerState{
										Terminated: &corev1.ContainerStateTerminated{},
									},
								},
								{
									Ready:        false,
									Name:         "start",
									RestartCount: 1,
									LastTerminationState: corev1.ContainerState{
										Terminated: &corev1.ContainerStateTerminated{},
									},
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "restartLimitation2",
			fields: fields{
				target: target{
					hippoPodSpec: &carbonv1.HippoPodSpec{
						Containers: []carbonv1.HippoContainer{
							{
								Container: corev1.Container{Name: "main"},
								ContainerHippoExterned: carbonv1.ContainerHippoExterned{
									Configs: &carbonv1.ContainerConfig{
										RestartCountLimit: 10,
									},
								},
							},
							{
								Container: corev1.Container{Name: "start"},
								ContainerHippoExterned: carbonv1.ContainerHippoExterned{
									Configs: &carbonv1.ContainerConfig{
										RestartCountLimit: 5,
									},
								},
							},
						},
					},
					worker: &carbonv1.WorkerNode{
						Status: carbonv1.WorkerNodeStatus{
							RestartRecords: map[string]string{
								"main": generateRestartRecords(9, false),
							},
						},
					},
				},
				current: current{
					currentPod: &corev1.Pod{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								carbonv1.LabelKeyPodVersion: carbonv1.PodVersion3,
							},
						},
						Status: corev1.PodStatus{
							ContainerStatuses: []corev1.ContainerStatus{
								{
									Ready:        false,
									Name:         "main",
									RestartCount: 1,
									LastTerminationState: corev1.ContainerState{
										Terminated: &corev1.ContainerStateTerminated{},
									},
								},
							},
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &podAllocator{
				schedulerType: tt.fields.schedulerType,
				target:        tt.fields.target,
				current:       tt.fields.current,
				parser:        tt.fields.parser,
				syncer:        tt.fields.syncer,
				executor:      tt.fields.executor,
				pSpecMerger:   tt.fields.pSpecMerger,
			}
			if got := a.isPodProcessFailed(); got != tt.want {
				t.Errorf("podAllocator.isPodProcessFailed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_podAllocator_setPodC2Labels(t *testing.T) {
	type fields struct {
		schedulerType string
		target        target
		current       current
		parser        podStatusParser
		syncer        podSpecSyncer
		executor      executor
		pSpecMerger   *util.PodSpecMerger
		c             *Controller
	}
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    *corev1.Pod
	}{
		{
			name: "test",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{},
					},
				},
			},
			want: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"alibabacloud.com/platform": "c2",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &podAllocator{
				schedulerType: tt.fields.schedulerType,
				target:        tt.fields.target,
				current:       tt.fields.current,
				parser:        tt.fields.parser,
				syncer:        tt.fields.syncer,
				executor:      tt.fields.executor,
				pSpecMerger:   tt.fields.pSpecMerger,
				c:             tt.fields.c,
			}
			if err := a.setPodC2Labels(tt.args.pod); (err != nil) != tt.wantErr {
				t.Errorf("podAllocator.setPodC2Labels() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.want, tt.args.pod) {
				t.Errorf("hippoPodAllocator.setPodC2Labels() error = %v, want %v", utils.ObjJSON(tt.args.pod), utils.ObjJSON(tt.want))
			}
		})
	}
}
func Test_syncStandbyStableStatus(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name    string
		args    args
		want    func(w *carbonv1.WorkerNode) bool
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				pod: func() *corev1.Pod {
					pod := newPod("test", "1", corev1.PodPending, "")
					pod.Annotations[carbonv1.AnnotationStandbyStableStatus] = "{\"standbyHours\":[2,3,4,5]}"
					pod.Annotations[carbonv1.AnnotationPodDeletionCost] = "4999"
					return pod
				}(),
			},
			want: func(w *carbonv1.WorkerNode) bool {
				return w.Status.AllocatorSyncedStatus.StandbyHours == "[2,3,4,5]" &&
					w.Status.AllocatorSyncedStatus.DeletionCost == 4999
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &podAllocator{
				current: current{
					currentPod: tt.args.pod,
				},
				target: target{worker: newWorker("tet", "ip1", false, "", nil, 1, 0)},
			}
			if err := a.syncStandbyStableStatus(); (err != nil) != tt.wantErr {
				t.Errorf("podAllocator.syncStandbyStableStatus() err:%v", err)
			}
			if !tt.want(a.worker) {
				t.Errorf("podAllocator.syncStandbyStableStatus() assert worker falied")
			}
		})
	}
}

func Test_addPatchForUpdateStatus(t *testing.T) {
	type args struct {
		pod    *corev1.Pod
		worker *carbonv1.WorkerNode
	}
	now := time.Now().Unix()
	tests := []struct {
		name string
		args args
		want func(patch *util.CommonPatch) bool
	}{
		{
			name: "normal",
			args: args{
				pod: func() *corev1.Pod {
					return newPod("test", "1", corev1.PodPending, "")
				}(),
				worker: func() *carbonv1.WorkerNode {
					worker := newWorker("test", "1", false, "", nil, 1, 0)
					carbonv1.SetBufferSwapRecord(worker, utils.ObjJSON(carbonv1.BufferSwapRecord{
						SwapAt: now,
					}))
					worker.Status.ResVersion = "2"
					worker.Status.ResourceMatch = true
					worker.Status.ProcessMatch = true
					worker.Status.HealthStatus = carbonv1.HealthAlive
					worker.Status.ProcessReady = true
					worker.Status.HealthCondition.Version = "1"
					worker.Status.Version = "1"
					worker.Status.Complete = true
					return worker
				}(),
			},
			want: func(patch *util.CommonPatch) bool {
				status, err := parseUpdateStatus(patch)
				if err != nil {
					return false
				}
				return status.UpdateAt == now && status.Phase == carbonv1.PhaseReady
			},
		},
		{
			name: "old version",
			args: args{
				pod: func() *corev1.Pod {
					pod := newPod("test", "1", corev1.PodPending, "")
					labels.AddLabel(pod.Annotations, carbonv1.AnnotationWorkerUpdateStatus, "{\"targetVersion\":\"1\",\"curVersion\":\"1\",\"updateAt\":1697715963,\"phase\":1,\"healthStatus\":\"HT_ALIVE\"}")
					return pod
				}(),
				worker: func() *carbonv1.WorkerNode {
					worker := newWorker("test", "1", false, "", nil, 1, 0)
					worker.Spec.Version = "1"
					worker.Status.Version = "1"
					worker.Status.Complete = false
					return worker
				}(),
			},
			want: func(patch *util.CommonPatch) bool {
				status, err := parseUpdateStatus(patch)
				if err != nil {
					return false
				}
				return status.UpdateAt == 1697715963 && status.Phase == carbonv1.PhaseDeploying
			},
		},
		{
			name: "service reclaim",
			args: args{
				pod: func() *corev1.Pod {
					pod := newPod("test", "1", corev1.PodPending, "")
					labels.AddLabel(pod.Annotations, carbonv1.AnnotationWorkerUpdateStatus, "{\"targetVersion\":\"1\",\"curVersion\":\"1\",\"updateAt\":1697715963,\"phase\":1,\"healthStatus\":\"HT_ALIVE\"}")
					return pod
				}(),
				worker: func() *carbonv1.WorkerNode {
					worker := newWorker("test", "1", false, "", nil, 1, 0)
					worker.Spec.Version = "1"
					worker.Status.Version = "1"
					worker.Status.Complete = false
					worker.Status.BadReason = carbonv1.BadReasonServiceReclaim
					return worker
				}(),
			},
			want: func(patch *util.CommonPatch) bool {
				status, err := parseUpdateStatus(patch)
				if err != nil {
					return false
				}
				return status.UpdateAt == 1697715963 && status.Phase == carbonv1.PhaseDeploying
			},
		},
		{
			name: "version inner match",
			args: args{
				pod: func() *corev1.Pod {
					pod := newPod("test", "1", corev1.PodPending, "")
					labels.AddLabel(pod.Annotations, carbonv1.AnnotationWorkerUpdateStatus, "{\"targetVersion\":\"1-3ws33\",\"curVersion\":\"1-3ws33\",\"updateAt\":1697715963,\"phase\":1,\"healthStatus\":\"HT_ALIVE\"}")
					return pod
				}(),
				worker: func() *carbonv1.WorkerNode {
					worker := newWorker("test", "1", false, "", nil, 1, 0)
					worker.Spec.Version = "2-3ws33"
					worker.Status.Version = "2-3ws33"
					return worker
				}(),
			},
			want: func(patch *util.CommonPatch) bool {
				status, err := parseUpdateStatus(patch)
				if err != nil {
					return false
				}
				return status.UpdateAt == 1697715963
			},
		},
		{
			name: "toRelease PhaseEvicting",
			args: args{
				pod: func() *corev1.Pod {
					pod := newPod("test", "1", corev1.PodPending, "")
					labels.AddLabel(pod.Annotations, carbonv1.AnnotationWorkerUpdateStatus, "{\"targetVersion\":\"1\",\"curVersion\":\"1\",\"updateAt\":1697715963,\"phase\":1,\"healthStatus\":\"HT_ALIVE\"}")
					return pod
				}(),
				worker: func() *carbonv1.WorkerNode {
					worker := newWorker("test", "1", true, "", nil, 1, 0)
					worker.Status.ServiceConditions = []carbonv1.ServiceCondition{
						{
							Type:   carbonv1.ServiceGeneral,
							Status: corev1.ConditionTrue,
						},
					}
					return worker
				}(),
			},
			want: func(patch *util.CommonPatch) bool {
				status, err := parseUpdateStatus(patch)
				if err != nil {
					return false
				}
				return status.UpdateAt == 1697715963 && status.Phase == carbonv1.PhaseEvicting
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &podAllocator{
				current: current{
					currentPod: tt.args.pod,
				},
				target: target{worker: tt.args.worker},
				c:      &Controller{},
			}
			patch := &util.CommonPatch{PatchType: types.StrategicMergePatchType, PatchData: map[string]interface{}{}}
			a.addPatchForUpdateStatus(patch)
			if !tt.want(patch) {
				t.Errorf("podAllocator.addPatchForUpdateStatus() assert patch falied")
			}
		})
	}
}

func parseLabel(labelKey string, patch *util.CommonPatch) (bool, string) {
	if _, ok := patch.PatchData["metadata"]; !ok {
		return false, ""
	}
	metadata := patch.PatchData["metadata"]
	if _, ok := metadata.(map[string]interface{}); !ok {
		return false, ""
	}
	metadataMap := metadata.(map[string]interface{})
	if _, ok := metadataMap["labels"].(map[string]string); !ok {
		return false, ""
	}
	labels := metadataMap["labels"].(map[string]string)
	if value, ok := labels[labelKey]; ok {
		return true, value
	}
	return false, ""
}

func parseUpdateStatus(patch *util.CommonPatch) (carbonv1.UpdateStatus, error) {
	var updateStatus carbonv1.UpdateStatus
	metadata := patch.PatchData["metadata"]
	meta := metadata.(map[string]interface{})
	annotations := meta["annotations"].(map[string]string)
	status := annotations[carbonv1.AnnotationWorkerUpdateStatus]
	err := json.Unmarshal([]byte(status), &updateStatus)
	return updateStatus, err
}
