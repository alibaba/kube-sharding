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
	"testing"
	"time"

	"github.com/alibaba/kube-sharding/common/features"
	"github.com/alibaba/kube-sharding/controller"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labelsutil "k8s.io/kubernetes/pkg/util/labels"
)

func Test_isWorkersSame(t *testing.T) {
	type args struct {
		current *carbonv1.WorkerNode
		backup  *carbonv1.WorkerNode
	}
	workersame1 := newTestWorker("test", "1.1.1.1", nil)
	workersame2 := newTestWorker("test", "1.1.1.1", nil)
	workersame1.Spec.Version = "1"
	workersame1.Spec.ResVersion = "1"
	workersame2.Spec.Version = "1"
	workersame2.Spec.ResVersion = "1"
	workernotsame1 := newTestWorker("test", "1.1.1.1", nil)
	workernotsame2 := newTestWorker("test", "1.1.1.1", nil)
	workernotsame1.Spec.Version = "1"
	workernotsame1.Spec.ResVersion = "1"
	workernotsame2.Spec.Version = "1"
	workernotsame2.Spec.ResVersion = "2"
	workernotsame11 := newTestWorker("test", "1.1.1.1", nil)
	workernotsame12 := newTestWorker("test", "1.1.1.1", nil)
	workernotsame11.Spec.Version = "1"
	workernotsame11.Spec.ResVersion = "1"
	workernotsame12.Spec.Version = "2"
	workernotsame12.Spec.ResVersion = "1"

	workernotsame21 := newTestWorker("test", "1.1.1.1", nil)
	workernotsame22 := newTestWorker("test", "1.1.1.1", nil)
	workernotsame21.Spec.Version = "1"
	workernotsame21.Spec.ResVersion = "1"
	workernotsame22.Spec.Version = "1"
	workernotsame22.Spec.ResVersion = "1"
	workernotsame22.Labels = map[string]string{carbonv1.LabelKeyQuotaGroupID: "1"}
	workernotsame22.Labels = map[string]string{carbonv1.LabelKeyQuotaGroupID: "2"}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "same",
			args: args{
				current: workersame1,
				backup:  workersame2,
			},
			want: true,
		},
		{
			name: "not_same",
			args: args{
				current: workernotsame1,
				backup:  workernotsame2,
			},
			want: false,
		},
		{
			name: "not_same",
			args: args{
				current: workernotsame11,
				backup:  workernotsame12,
			},
			want: false,
		},
		{
			name: "not_same",
			args: args{
				current: workernotsame21,
				backup:  workernotsame22,
			},
			want: false,
		},
		{
			name: "dependency_same",
			args: args{
				current: func() *carbonv1.WorkerNode {
					workernotsame31 := newTestWorker("test", "1.1.1.1", nil)
					workernotsame31.Spec.Version = "1"
					workernotsame31.Spec.ResVersion = "1"
					workernotsame31.Spec.DependencyReady = utils.BoolPtr(true)
					return workernotsame31
				}(),
				backup: func() *carbonv1.WorkerNode {
					workernotsame32 := newTestWorker("test", "1.1.1.1", nil)
					workernotsame32.Spec.Version = "1"
					workernotsame32.Spec.ResVersion = "1"
					workernotsame32.Spec.DependencyReady = utils.BoolPtr(true)
					return workernotsame32
				}(),
			},
			want: true,
		},
		{
			name: "dependency_same",
			args: args{
				current: func() *carbonv1.WorkerNode {
					workernotsame31 := newTestWorker("test", "1.1.1.1", nil)
					workernotsame31.Spec.Version = "1"
					workernotsame31.Spec.ResVersion = "1"
					workernotsame31.Spec.DependencyReady = utils.BoolPtr(true)
					workernotsame31.Spec.RowComplete = utils.BoolPtr(true)
					return workernotsame31
				}(),
				backup: func() *carbonv1.WorkerNode {
					workernotsame32 := newTestWorker("test", "1.1.1.1", nil)
					workernotsame32.Spec.Version = "1"
					workernotsame32.Spec.ResVersion = "1"
					workernotsame32.Spec.DependencyReady = utils.BoolPtr(true)
					workernotsame32.Spec.RowComplete = utils.BoolPtr(true)
					return workernotsame32
				}(),
			},
			want: true,
		},
		{
			name: "dependency_not_same",
			args: args{
				current: func() *carbonv1.WorkerNode {
					workernotsame31 := newTestWorker("test", "1.1.1.1", nil)
					workernotsame31.Spec.Version = "1"
					workernotsame31.Spec.ResVersion = "1"
					workernotsame31.Spec.DependencyReady = utils.BoolPtr(true)
					return workernotsame31
				}(),
				backup: func() *carbonv1.WorkerNode {
					workernotsame32 := newTestWorker("test", "1.1.1.1", nil)
					workernotsame32.Spec.Version = "1"
					workernotsame32.Spec.ResVersion = "1"
					workernotsame32.Spec.DependencyReady = utils.BoolPtr(true)
					workernotsame32.Spec.RowComplete = utils.BoolPtr(true)
					return workernotsame32
				}(),
			},
			want: false,
		},
		{
			name: "dependency_not_same",
			args: args{
				current: func() *carbonv1.WorkerNode {
					workernotsame31 := newTestWorker("test", "1.1.1.1", nil)
					workernotsame31.Spec.Version = "1"
					workernotsame31.Spec.ResVersion = "1"
					workernotsame31.Spec.DependencyReady = utils.BoolPtr(true)
					return workernotsame31
				}(),
				backup: func() *carbonv1.WorkerNode {
					workernotsame32 := newTestWorker("test", "1.1.1.1", nil)
					workernotsame32.Spec.Version = "1"
					workernotsame32.Spec.ResVersion = "1"
					workernotsame32.Spec.DependencyReady = utils.BoolPtr(true)
					workernotsame32.Spec.RowComplete = utils.BoolPtr(false)
					return workernotsame32
				}(),
			},
			want: false,
		},
		{
			name: "dependency_not_same",
			args: args{
				current: func() *carbonv1.WorkerNode {
					workernotsame31 := newTestWorker("test", "1.1.1.1", nil)
					workernotsame31.Spec.Version = "1"
					workernotsame31.Spec.ResVersion = "1"
					workernotsame31.Spec.DependencyReady = utils.BoolPtr(true)
					return workernotsame31
				}(),
				backup: func() *carbonv1.WorkerNode {
					workernotsame32 := newTestWorker("test", "1.1.1.1", nil)
					workernotsame32.Spec.Version = "1"
					workernotsame32.Spec.ResVersion = "1"
					workernotsame32.Spec.DependencyReady = utils.BoolPtr(false)
					return workernotsame32
				}(),
			},
			want: false,
		},
		{
			name: "workermode_not_same",
			args: args{
				current: func() *carbonv1.WorkerNode {
					workernotsame31 := newTestWorker("test", "1.1.1.1", nil)
					workernotsame31.Spec.Version = "1"
					workernotsame31.Spec.ResVersion = "1"
					workernotsame31.Spec.WorkerMode = carbonv1.WorkerModeTypeHot
					return workernotsame31
				}(),
				backup: func() *carbonv1.WorkerNode {
					workernotsame32 := newTestWorker("test", "1.1.1.1", nil)
					workernotsame32.Spec.Version = "1"
					workernotsame32.Spec.ResVersion = "1"
					return workernotsame32
				}(),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isWorkersSame(tt.args.current, tt.args.backup); got != tt.want {
				t.Errorf("isWorkersSame() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_syncWorkers(t *testing.T) {
	type args struct {
		current *carbonv1.WorkerNode
		backup  *carbonv1.WorkerNode
	}
	worker := newTestWorker("test", "1.1.1.1", map[string]string{"foo": "bar"})
	worker.Labels[carbonv1.WorkerRoleKey] = carbonv1.CurrentWorkerKey
	worker.Labels[carbonv1.LabelKeyQuotaGroupID] = "1"

	metaStr := `{"metadata":{"name":"nginx-deployment","creationTimestamp":null,"labels":{"app":"nginx"}},"spec":{"containers":null}}`
	err := json.Unmarshal([]byte(metaStr), &worker.Spec.Template)
	if nil != err {
		t.Errorf("Unmarshal error :%v", err)
	}
	pair := newTestWorker("backup", "1.1.1.1", map[string]string{"foo": "bar"})
	worker.Spec.OwnerGeneration = 1

	worker1 := newTestWorker("test", "1.1.1.1", map[string]string{"foo": "bar"})
	worker1.Labels[carbonv1.WorkerRoleKey] = carbonv1.CurrentWorkerKey

	pair1 := newTestWorker("backup", "1.1.1.1", map[string]string{"foo": "bar"})
	err = json.Unmarshal([]byte(metaStr), &pair1.Spec.Template)
	if nil != err {
		t.Errorf("Unmarshal error :%v", err)
	}
	pair1.Spec.OwnerGeneration = 1
	pair1.Labels[carbonv1.LabelKeyQuotaGroupID] = "1"

	tests := []struct {
		name string
		args args
	}{
		{
			name: "sync",
			args: args{
				current: worker,
				backup:  pair,
			},
		},
		{
			name: "reverse-sync",
			args: args{
				current: worker1,
				backup:  pair1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncWorkers(tt.args.current, tt.args.backup)
			str := utils.ObjJSON(tt.args.current.Spec.Template)
			if len(str) != len(metaStr) {
				t.Errorf("sync = %v, want %v", str, metaStr)
			}
			if carbonv1.GetQuotaID(tt.args.current) != "1" {
				t.Errorf("sync = %v, want %v", carbonv1.GetQuotaID(tt.args.current), "1")
			}
			str = utils.ObjJSON(tt.args.backup.Spec.Template)
			if len(str) != len(metaStr) {
				t.Errorf("sync = %v, want %v", str, metaStr)
			}
			if carbonv1.GetQuotaID(tt.args.backup) != "1" {
				t.Errorf("sync = %v, want %v", carbonv1.GetQuotaID(tt.args.backup), "1")
			}
		})
	}
}

func Test_couldSwapWorkers(t *testing.T) {
	type args struct {
		current *carbonv1.WorkerNode
		backup  *carbonv1.WorkerNode
	}

	currentDirect := newTestWorker("test", "1.1.1.1", nil)
	backupDirect := newTestWorker("test", "1.1.1.1", nil)
	backupDirect.Spec.RecoverStrategy = carbonv1.DirectReleasedRecoverStrategy
	currentComplete := newTestWorker("test", "1.1.1.1", nil)
	backupComplete := newTestWorker("test", "1.1.1.1", nil)
	setWorkerComplete(backupComplete)
	currentNotComplete := newTestWorker("test", "1.1.1.1", nil)
	backupNotComplete := newTestWorker("test", "1.1.1.1", nil)
	currentNotSame := newTestWorker("test", "1.1.1.1", nil)
	backupNotSame := newTestWorker("test", "1.1.1.1", nil)
	backupNotSame.Spec.Version = "1"

	backupReady := backupComplete.DeepCopy()
	backupReady.Status.Complete = false
	backupReady.Status.ServiceStatus = carbonv1.ServiceUnAvailable
	currentLost := currentNotComplete.DeepCopy()
	currentLost.Status.AllocStatus = carbonv1.WorkerLost
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "direct release",
			args: args{
				current: currentDirect,
				backup:  backupDirect,
			},
			want: false,
		},
		{
			name: "backup complete",
			args: args{
				current: currentComplete,
				backup:  backupComplete,
			},
			want: true,
		},
		{
			name: "current lost",
			args: args{
				current: currentComplete,
				backup:  backupComplete,
			},
			want: true,
		},
		{
			name: "backup not complete",
			args: args{
				current: currentNotComplete,
				backup:  backupNotComplete,
			},
			want: false,
		},
		{
			name: "backup not same",
			args: args{
				current: currentNotSame,
				backup:  backupNotSame,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := couldSwapWorkers(tt.args.current, tt.args.backup); got != tt.want {
				t.Errorf("couldSwapWorkers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_defaultAdjuster_adjust(t *testing.T) {
	type fields struct {
		worker *carbonv1.WorkerNode
		pair   *carbonv1.WorkerNode
		c      *Controller
	}

	workerRelease, pairRelease := newTestWorkerPair("test", "1.1.1.1", map[string]string{"foo": "bar"})
	workerRelease.Spec.ToDelete = true
	workerReleaseWant := workerRelease.DeepCopy()
	workerReleaseWant.Annotations = map[string]string{
		"alibabacloud.com/evict-failed-msg": "backup NotAvailable",
	}
	pairReleaseWant := pairRelease.DeepCopy()
	pairReleaseWant.Status.ToRelease = true
	workerReleaseWant.Status.ToRelease = true
	workerReleaseReverseWant := workerRelease.DeepCopy()
	workerReleaseReverseWant.Status.ToRelease = true

	workerCreateBackup, pairCreateBackup := newTestWorkerPair("test1", "1.1.1.1", map[string]string{"foo": "bar"})
	workerCreateBackup.Status.AllocStatus = carbonv1.WorkerLost
	workerCreateBackupWant := workerCreateBackup.DeepCopy()
	workerCreateBackupWant.Labels["app.hippo.io/preference"] = "APP-PROHIBIT-600"
	workerCreateBackupWant.Status.BadReason = 1
	workerCreateBackupWant.Status.BadReasonMessage = "lost"

	fmt.Println(utils.ObjJSON(workerCreateBackup))
	pairCreateBackupWant := pairCreateBackup.DeepCopy()
	pairCreateBackupWant.Status = carbonv1.WorkerNodeStatus{}
	pairCreateBackupWant.Annotations = map[string]string{}
	carbonv1.SetWorkerBackup(pairCreateBackupWant)
	pairCreateBackupWant.Labels["worker-version-hash"] = "test1-b"
	pairCreateBackupWant.Spec.Selector = labelsutil.CloneSelectorAndAddLabel(
		&metav1.LabelSelector{}, carbonv1.DefaultWorkernodeUniqueLabelKey, "test1-b")

	workerCreateBackupSyncStandby, pairCreateBackupSyncStandby := newTestWorkerPair("test2", "1.1.1.1", map[string]string{"foo": "bar"})
	workerCreateBackupSyncStandby.Status.AllocStatus = carbonv1.WorkerLost
	workerCreateBackupSyncStandby.Status.PodStandbyStatus = carbonv1.PodStandbyStatus{
		UseOrder:     1,
		StandbyHours: []int64{10, 11},
	}
	workerCreateBackupSyncStandbyWant := workerCreateBackupSyncStandby.DeepCopy()
	workerCreateBackupSyncStandbyWant.Labels["app.hippo.io/preference"] = "APP-PROHIBIT-600"
	workerCreateBackupSyncStandbyWant.Status.BadReason = 1
	workerCreateBackupSyncStandbyWant.Status.BadReasonMessage = "lost"
	workerCreateBackupSyncStandbyWant.Annotations = map[string]string{}

	fmt.Println(utils.ObjJSON(workerCreateBackupSyncStandby))
	pairCreateBackupSyncStandbyWant := pairCreateBackupSyncStandby.DeepCopy()
	pairCreateBackupSyncStandbyWant.Status = carbonv1.WorkerNodeStatus{}
	carbonv1.SetWorkerBackup(pairCreateBackupSyncStandbyWant)
	pairCreateBackupSyncStandbyWant.Labels["worker-version-hash"] = "test2-b"
	pairCreateBackupSyncStandbyWant.Spec.Selector = labelsutil.CloneSelectorAndAddLabel(
		&metav1.LabelSelector{}, carbonv1.DefaultWorkernodeUniqueLabelKey, "test2-b")
	pairCreateBackupSyncStandbyWant.Annotations = map[string]string{}

	workerSwap, pairSwap := newTestWorkerPair("test6", "1.1.1.1", map[string]string{"foo": "bar"})
	setWorkerComplete(pairSwap)
	workerSwapWant := workerSwap.DeepCopy()
	workerSwapWant.Annotations = map[string]string{
		"alibabacloud.com/evict-failed-msg": "backup Complete",
	}
	carbonv1.SetWorkerBackup(workerSwapWant)
	pairSwapWant := pairSwap.DeepCopy()
	carbonv1.SetWorkerCurrent(pairSwapWant)

	workerSwap1, pairSwap1 := newTestWorkerPair("test5", "1.1.1.1", map[string]string{"foo": "bar"})
	setWorkerComplete(pairSwap1)
	workerSwapWant1 := workerSwap1.DeepCopy()
	workerSwapWant1.Annotations = map[string]string{
		"alibabacloud.com/evict-failed-msg": "backup Complete",
	}
	carbonv1.SetWorkerBackup(workerSwapWant1)
	pairSwapWant1 := pairSwap1.DeepCopy()
	carbonv1.SetWorkerCurrent(pairSwapWant1)
	workerSwapWant1.Annotations = map[string]string{
		"alibabacloud.com/evict-failed-msg": "backup Complete",
	}

	workerReleaseBakcup, pairReleaseBakcup := newTestWorkerPair("test", "1.1.1.1", map[string]string{"foo": "bar"})
	setWorkerComplete(workerReleaseBakcup)
	workerReleaseBakcupWant := workerReleaseBakcup.DeepCopy()
	workerReleaseBakcupWant.Annotations = map[string]string{}
	//delete(workerReleaseBakcupWant.Labels, "app.hippo.io/preference")
	pairReleaseBakcupWant := pairReleaseBakcup.DeepCopy()
	pairReleaseBakcupWant.Status.ToRelease = true

	directRelease, _ := newTestWorkerPair("test3", "1.1.1.1", map[string]string{"foo": "bar"})
	directRelease.Status.AllocStatus = carbonv1.WorkerLost
	directRelease.Spec.RecoverStrategy = carbonv1.DirectReleasedRecoverStrategy
	directReleaseWant := directRelease.DeepCopy()
	directReleaseWant.Status.ToRelease = true
	directReleaseWant.Status.ServiceOffline = true
	directReleaseWant.Status.ServiceOffline = true
	directReleaseWant.Status.BadReason = 1
	directReleaseWant.Status.BadReasonMessage = "lost"
	directReleaseWant.Labels[carbonv1.LabelKeyHippoPreference] = "APP-PROHIBIT-600"

	directReleaseNoQuota, _ := newTestWorkerPair("test4", "1.1.1.1", map[string]string{"foo": "bar"})
	directReleaseNoQuota.Status.HealthStatus = carbonv1.HealthDead
	directReleaseNoQuota.Labels["rs-version-hash"] = "test"
	directReleaseNoQuota.Spec.BrokenRecoverQuotaConfig = &carbonv1.BrokenRecoverQuotaConfig{
		MaxFailedCount: utils.Int32Ptr(1),
		TimeWindow:     utils.Int32Ptr(60),
	}
	hasRecoverQuota(directReleaseNoQuota)
	directReleaseWantNoQuota := directReleaseNoQuota.DeepCopy()
	directReleaseWantNoQuota.Status.BadReason = 2
	directReleaseWantNoQuota.Status.BadReasonMessage = "dead"
	directReleaseWantNoQuota.Labels[carbonv1.LabelKeyHippoPreference] = "APP-PROHIBIT-600"
	directReleaseWantNoQuota.Annotations = map[string]string{}
	workerDonothing, _ := newTestWorkerPair("test7", "1.1.1.1", map[string]string{"foo": "bar"})
	setWorkerComplete(workerDonothing)
	workerDonothingBakcupWant := workerDonothing.DeepCopy()
	workerDonothingBakcupWant.Annotations = map[string]string{}

	workerNotRecover, _ := newTestWorkerPair("test8", "1.1.1.1", map[string]string{"foo": "bar"})
	workerNotRecover.Status.AllocStatus = carbonv1.WorkerLost
	workerNotRecover.Spec.RecoverStrategy = carbonv1.NotRecoverStrategy
	workerNotRecoverWant := workerNotRecover.DeepCopy()
	workerNotRecoverWant.Annotations = map[string]string{}
	workerNotRecoverWant.Status.BadReason = 1
	workerNotRecoverWant.Status.BadReasonMessage = "lost"
	workerNotRecoverWant.Labels[carbonv1.LabelKeyHippoPreference] = "APP-PROHIBIT-600"

	workerSync, pairSync := newTestWorkerPair("test", "1.1.1.1", map[string]string{"foo": "bar"})
	setWorkerComplete(workerSync)
	setWorkerComplete(pairSync)
	workerSync.Spec.Version = "1111"
	workerSyncWant := workerSync.DeepCopy()
	pairSyncWant := pairSync.DeepCopy()
	pairSyncWant.Spec.Version = "1111"

	var c = &Controller{}
	c.DefaultController = *controller.NewDefaultController(nil, nil, "a", controllerKind, c)
	tests := []struct {
		name    string
		fields  fields
		want    *carbonv1.WorkerNode
		want1   *carbonv1.WorkerNode
		want2   bool
		wantErr bool
	}{
		{
			name: "release",
			fields: fields{
				worker: workerRelease,
				pair:   pairRelease,
			},
			want:    workerReleaseWant,
			want1:   pairReleaseWant,
			wantErr: false,
		},
		{
			name: "release-reverse",
			fields: fields{
				worker: pairRelease,
				pair:   workerRelease,
			},
			want:    pairReleaseWant,
			want1:   workerReleaseReverseWant,
			wantErr: false,
		},
		{
			name: "direct release",
			fields: fields{
				worker: directRelease,
				pair:   nil,
			},
			want:    directReleaseWant,
			want1:   nil,
			wantErr: false,
		},
		{
			name: "no quota",
			fields: fields{
				worker: directReleaseNoQuota,
				pair:   nil,
			},
			want:    directReleaseWantNoQuota,
			want1:   nil,
			wantErr: false,
		},
		{
			name: "swap",
			fields: fields{
				worker: workerSwap,
				pair:   pairSwap,
			},
			want:    workerSwapWant,
			want1:   pairSwapWant,
			wantErr: false,
		},
		{
			name: "swap-reverse",
			fields: fields{
				worker: pairSwap1,
				pair:   workerSwap1,
			},
			want:    pairSwapWant1,
			want1:   workerSwapWant1,
			wantErr: false,
		},
		{
			name: "releasebackup",
			fields: fields{
				worker: workerReleaseBakcup,
				pair:   pairReleaseBakcupWant,
			},
			want:    workerReleaseBakcupWant,
			want1:   pairReleaseBakcupWant,
			wantErr: false,
		},
		{
			name: "releasebackup-reverse",
			fields: fields{
				worker: pairReleaseBakcupWant,
				pair:   workerReleaseBakcup,
			},
			want:    pairReleaseBakcupWant,
			want1:   workerReleaseBakcupWant,
			wantErr: false,
		},
		{
			name: "do nothing",
			fields: fields{
				worker: workerDonothing,
				pair:   nil,
			},
			want:    workerDonothingBakcupWant,
			want1:   nil,
			wantErr: false,
		},
		{
			name: "not recover",
			fields: fields{
				worker: workerNotRecover,
				pair:   nil,
			},
			want:    workerNotRecoverWant,
			want1:   nil,
			wantErr: false,
		},
		{
			name: "sync",
			fields: fields{
				worker: workerSync,
				pair:   pairSync,
			},
			want:    workerSyncWant,
			want1:   pairSyncWant,
			want2:   true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &defaultAdjuster{
				worker: tt.fields.worker,
				pair:   tt.fields.pair,
				c:      c,
			}
			got, got1, got2, err := a.adjust()
			if (err != nil) != tt.wantErr {
				t.Errorf("defaultAdjuster.adjust() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			tt.want.Status.BecomeCurrentTime = got.Status.BecomeCurrentTime
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("defaultAdjuster.adjust() got = %v, want %v", utils.ObjJSON(got), utils.ObjJSON(tt.want))
			}
			if nil != tt.want1 && nil != got1 {
				tt.want1.Status.BecomeCurrentTime = got1.Status.BecomeCurrentTime
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("defaultAdjuster.adjust() got1 = %v, want %v", utils.ObjJSON(got1), utils.ObjJSON(tt.want1))
			}
			if !reflect.DeepEqual(got2, tt.want2) {
				t.Errorf("defaultAdjuster.adjust() got2 = %v, want2 %v", utils.ObjJSON(got2), utils.ObjJSON(tt.want2))
			}
		})
	}
}

func Test_createBackup(t *testing.T) {
	type args struct {
		current *carbonv1.WorkerNode
		backup  *carbonv1.WorkerNode
		pod     *corev1.Pod
	}

	worker := newTestWorker("worker-a", "1.1.1.1", nil)
	worker.Labels = map[string]string{
		carbonv1.DefaultReplicaUniqueLabelKey: "worker",
	}
	worker.Spec.Selector = &metav1.LabelSelector{}
	metaStr := `{"metadata":{"name":"nginx-deployment","labels":{"app": "nginx"}}}`
	err := json.Unmarshal([]byte(metaStr), &worker.Spec.Template)
	if nil != err {
		t.Errorf("Unmarshal error :%v", err)
	}

	workerBackup := worker.DeepCopy()
	workerBackup.Name = "worker-b"
	workerBackup.UID = ""
	workerBackup.Annotations = map[string]string{}
	carbonv1.SetWorkerBackup(workerBackup)
	workerBackup.Labels["worker-version-hash"] = "worker-b"
	workerBackup.Spec.Selector = labelsutil.CloneSelectorAndAddLabel(
		worker.Spec.Selector, carbonv1.DefaultWorkernodeUniqueLabelKey, "worker-b")
	workerBackup.Spec.BackupOfPod.Uid = "UID"
	workerBackup.Status = carbonv1.WorkerNodeStatus{}

	var pod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			UID: "UID",
		},
	}
	tests := []struct {
		name       string
		args       args
		wantErr    bool
		wantWorker *carbonv1.WorkerNode
	}{
		{
			name: "add",
			args: args{
				current: worker,
				pod:     pod,
			},
			wantErr:    false,
			wantWorker: workerBackup,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotWorker, err := createBackup(tt.args.current, tt.args.backup, tt.args.pod)
			if (err != nil) != tt.wantErr {
				t.Errorf("createBackup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotWorker, tt.wantWorker) {
				t.Errorf("createBackup() = %v, want %v", utils.ObjJSON(gotWorker), utils.ObjJSON(tt.wantWorker))
			}
		})
	}
}

func Test_getInt32Value(t *testing.T) {
	type args struct {
		ptr *int32
	}
	var a int32 = 10
	tests := []struct {
		name string
		args args
		want int32
	}{
		{
			name: "normal",
			args: args{
				ptr: &a,
			},
			want: a,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getInt32Value(tt.args.ptr); got != tt.want {
				t.Errorf("getInt32Value() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_defaultAdjuster_shouldReleaseBackupNow(t *testing.T) {
	type fields struct {
		c      *Controller
		worker *carbonv1.WorkerNode
		pair   *carbonv1.WorkerNode
	}
	type args struct {
		backup *carbonv1.WorkerNode
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   time.Duration
		want1  bool
	}{
		{
			name: "default delay release",
			args: args{
				backup: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						VersionPlan: carbonv1.VersionPlan{
							BroadcastPlan: carbonv1.BroadcastPlan{
								WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
									DelayDeleteBackupSeconds: 0,
								},
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocStatus:   carbonv1.WorkerAssigned,
						ServiceStatus: carbonv1.ServiceAvailable,
					},
				},
			},
			want:  time.Second * (40),
			want1: false,
		},
		{
			name: "release now with no delay",
			args: args{
				backup: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						VersionPlan: carbonv1.VersionPlan{
							BroadcastPlan: carbonv1.BroadcastPlan{
								WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
									DelayDeleteBackupSeconds: 0,
								},
							},
						},
					},
				},
			},
			want:  time.Second * 0,
			want1: true,
		},
		{
			name: "release now with delay",
			args: args{
				backup: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						VersionPlan: carbonv1.VersionPlan{
							BroadcastPlan: carbonv1.BroadcastPlan{
								WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
									DelayDeleteBackupSeconds: 300,
								},
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						WorkerStateChangeRecoder: carbonv1.WorkerStateChangeRecoder{
							LastDeleteBackupTime: time.Now().Unix() - 300,
						},
					},
				},
			},
			want:  time.Second * 0,
			want1: true,
		},
		{
			name: "release now with delay",
			args: args{
				backup: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						VersionPlan: carbonv1.VersionPlan{
							BroadcastPlan: carbonv1.BroadcastPlan{
								WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
									DelayDeleteBackupSeconds: 300,
								},
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						WorkerStateChangeRecoder: carbonv1.WorkerStateChangeRecoder{
							LastDeleteBackupTime: time.Now().Unix() - 310,
						},
					},
				},
			},
			want:  time.Second * 0,
			want1: true,
		},
		{
			name: "delay release",
			args: args{
				backup: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						VersionPlan: carbonv1.VersionPlan{
							BroadcastPlan: carbonv1.BroadcastPlan{
								WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
									DelayDeleteBackupSeconds: 300,
								},
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocStatus: carbonv1.WorkerAssigned,
						WorkerStateChangeRecoder: carbonv1.WorkerStateChangeRecoder{
							LastDeleteBackupTime: time.Now().Unix() - 50,
						},
						BadReason: carbonv1.BadReasonDead,
					},
				},
			},
			want:  time.Second * (260),
			want1: false,
		},
		{
			name: "delay release",
			args: args{
				backup: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						VersionPlan: carbonv1.VersionPlan{
							BroadcastPlan: carbonv1.BroadcastPlan{
								WorkerSchedulePlan: carbonv1.WorkerSchedulePlan{
									DelayDeleteBackupSeconds: 300,
								},
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocStatus: carbonv1.WorkerAssigned,
					},
				},
			},
			want:  time.Second * (40),
			want1: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &defaultAdjuster{
				c:      tt.fields.c,
				worker: tt.fields.worker,
				pair:   tt.fields.pair,
			}
			got, got1 := a.shouldReleaseBackupNow(tt.args.backup)
			if got != tt.want {
				t.Errorf("defaultAdjuster.shouldReleaseBackupNow() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("defaultAdjuster.shouldReleaseBackupNow() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_shouldRecover(t *testing.T) {
	type args struct {
		current *carbonv1.WorkerNode
		backup  *carbonv1.WorkerNode
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "should recover",
			args: args{
				current: &carbonv1.WorkerNode{
					Status: carbonv1.WorkerNodeStatus{
						AllocStatus: carbonv1.WorkerAssigned,
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Phase: carbonv1.Failed,
						},
					},
				},
				backup: nil,
			},
			want: true,
		},
		{
			name: "should not recover",
			args: args{
				current: &carbonv1.WorkerNode{
					Status: carbonv1.WorkerNodeStatus{
						AllocStatus: carbonv1.WorkerAssigned,
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Phase: carbonv1.Failed,
						},
						ToRelease: true,
					},
				},
				backup: nil,
			},
			want: false,
		},
		{
			name: "should not recover",
			args: args{
				current: &carbonv1.WorkerNode{
					Status: carbonv1.WorkerNodeStatus{
						AllocStatus: carbonv1.WorkerAssigned,
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Phase: carbonv1.Failed,
						},
					},
				},
				backup: &carbonv1.WorkerNode{
					Status: carbonv1.WorkerNodeStatus{
						AllocStatus: carbonv1.WorkerAssigned,
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Phase: carbonv1.Failed,
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldRecover(tt.args.current, tt.args.backup, ""); got != tt.want {
				t.Errorf("shouldRecover() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_recoverWithoutCreatebackup(t *testing.T) {
	type args struct {
		current *carbonv1.WorkerNode
		pod     *corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "test",
			args: args{
				current: &carbonv1.WorkerNode{
					Status: carbonv1.WorkerNodeStatus{
						AllocStatus: carbonv1.WorkerAssigned,
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							IP: "1.2.3.4",
						},
					},
				},
			},
			want: false,
		},
		{
			name: "test",
			args: args{
				current: &carbonv1.WorkerNode{
					Status: carbonv1.WorkerNodeStatus{
						AllocStatus: carbonv1.WorkerUnAssigned,
					},
				},
			},
			want: true,
		},
		{
			name: "test",
			args: args{
				current: &carbonv1.WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.Time{
							Time: time.Now().Add(time.Second * -1201),
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocStatus: carbonv1.WorkerAssigned,
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Phase: carbonv1.Pending,
						},
					},
				},
			},
			want: false,
		},
		{
			name: "test",
			args: args{
				current: &carbonv1.WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app.hippo.io/pod-version": "v3.0"},
						CreationTimestamp: metav1.Time{
							Time: time.Now().Add(time.Second * -1201),
						},
					},
					Spec: carbonv1.WorkerNodeSpec{
						Version: "3",
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocStatus: carbonv1.WorkerAssigned,
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Phase:   carbonv1.Pending,
							Version: "2",
						},
					},
				},
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app.hippo.io/pod-version": "v2.5"},
					},
				},
			},
			want: true,
		},
		{
			name: "test",
			args: args{
				current: &carbonv1.WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app.hippo.io/pod-version": "v3.0"},
						CreationTimestamp: metav1.Time{
							Time: time.Now().Add(time.Second * -1201),
						},
					},
					Spec: carbonv1.WorkerNodeSpec{
						Version: "3",
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocStatus: carbonv1.WorkerAssigned,
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Phase:   carbonv1.Pending,
							Version: "2",
						},
					},
				},
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app.hippo.io/pod-version": "v3.0"},
					},
				},
			},
			want: false,
		},
		{
			name: "test",
			args: args{
				current: &carbonv1.WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app.hippo.io/pod-version": "v3.0"},
						CreationTimestamp: metav1.Time{
							Time: time.Now().Add(time.Second * -1201),
						},
					},
					Spec: carbonv1.WorkerNodeSpec{
						Version: "3",
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocStatus: carbonv1.WorkerAssigned,
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Phase:   carbonv1.Pending,
							Version: "3",
						},
					},
				},
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"app.hippo.io/pod-version": "v2.5"},
					},
				},
			},
			want: false,
		},
		{
			name: "test",
			args: args{
				current: &carbonv1.WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.Time{
							Time: time.Now().Add(time.Second * -1),
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						AllocStatus: carbonv1.WorkerAssigned,
						AllocatorSyncedStatus: carbonv1.AllocatorSyncedStatus{
							Phase: carbonv1.Pending,
						},
					},
				},
			},
			want: true,
		},
	}
	features.C2MutableFeatureGate.Set("UpdatePodVersion3WithRelease=true")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := recoverWithoutCreatebackup(tt.args.current, tt.args.pod); got != tt.want {
				t.Errorf("recoverWithoutCreatebackup() = %v, want %v", got, tt.want)
			}
		})
	}
}
