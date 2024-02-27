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
	"math"
	"reflect"
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	clientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/integer"
)

func newRollingSetNotWithSched(specReplicas int32) *carbonv1.RollingSet {

	rollingSet := new(carbonv1.RollingSet)
	rollingSet.Spec.SchedulePlan.Replicas = &specReplicas
	//rollingSet.Spec.SchedulePlan.Strategy.RollingUpdate.MaxSurge = &intstr.FromInt(3)
	//rollingSet.Spec.SchedulePlan.Strategy.RollingUpdate.MaxUnavailable = &intstr.IntOrString{Type: intstr.Int, IntVal: maxUnavailable}

	return rollingSet
}

func NewRollingset(specReplicas int32, maxSurge int, maxUnavailable int, latestVersionRatio int32) *carbonv1.RollingSet {

	rollingSet := new(carbonv1.RollingSet)
	rollingSet.Spec.SchedulePlan.Replicas = &specReplicas
	//rollingSet.Spec.SchedulePlan.Strategy.RollingUpdate.MaxSurge = &intstr.FromInt(3)
	//rollingSet.Spec.SchedulePlan.Strategy.RollingUpdate.MaxUnavailable = &intstr.IntOrString{Type: intstr.Int, IntVal: maxUnavailable}

	rollingSet.Spec.SchedulePlan.Strategy.RollingUpdate = new(v1.RollingUpdateDeployment)
	maxSurge2 := intstr.FromInt(maxSurge)
	rollingSet.Spec.SchedulePlan.Strategy.RollingUpdate.MaxSurge = &maxSurge2

	maxUnavailable2 := intstr.FromInt(maxUnavailable)
	rollingSet.Spec.SchedulePlan.Strategy.RollingUpdate.MaxUnavailable = &maxUnavailable2

	rollingSet.Spec.LatestVersionRatio = &latestVersionRatio

	rollingSet.Spec.Strategy.Type = "RollingUpdate"
	return rollingSet
}

func NewRollingsetNotWithMaxUnavailable(specReplicas int32, maxSurge int, latestVersionRatio int32) *carbonv1.RollingSet {

	rollingSet := new(carbonv1.RollingSet)
	rollingSet.Spec.SchedulePlan.Replicas = &specReplicas
	//rollingSet.Spec.SchedulePlan.Strategy.RollingUpdate.MaxSurge = &intstr.FromInt(3)
	//rollingSet.Spec.SchedulePlan.Strategy.RollingUpdate.MaxUnavailable = &intstr.IntOrString{Type: intstr.Int, IntVal: maxUnavailable}

	rollingSet.Spec.SchedulePlan.Strategy.RollingUpdate = new(v1.RollingUpdateDeployment)
	maxSurge2 := intstr.FromInt(maxSurge)
	rollingSet.Spec.SchedulePlan.Strategy.RollingUpdate.MaxSurge = &maxSurge2

	rollingSet.Spec.LatestVersionRatio = &latestVersionRatio

	rollingSet.Spec.Strategy.Type = "RollingUpdate"
	return rollingSet
}

func TestRoundToInt32(t *testing.T) {
	a := integer.RoundToInt32(1.2)
	if a != 1 {
		t.Errorf("RoundToInt32() = %v, want %v", a, 1)
	}

	a = integer.RoundToInt32(1.7)
	if a != 2 {
		t.Errorf("RoundToInt32() = %v, want %v", a, 2)
	}

	a = integer.RoundToInt32(1.5)
	if a != 2 {
		t.Errorf("RoundToInt32() = %v, want %v", a, 2)
	}
	a = integer.RoundToInt32(1)
	if a != 1 {
		t.Errorf("RoundToInt32() = %v, want %v", a, 1)
	}
	a = integer.RoundToInt32(10.9)
	if a != 11 {
		t.Errorf("RoundToInt32() = %v, want %v", a, 11)
	}

	a = int32(math.Ceil(1.2))
	if a != 2 {
		t.Errorf("RoundToInt32() = %v, want %v", a, 2)
	}
}

func TestController_hasSmoothFinalizer(t *testing.T) {
	type fields struct {
		kubeclientset    kubernetes.Interface
		carbonclientset  clientset.Interface
		rollingSetLister listers.RollingSetLister
		rollingSetSynced cache.InformerSynced

		replicaSynced cache.InformerSynced
		workqueue     workqueue.RateLimitingInterface
		recorder      record.EventRecorder
	}
	type args struct {
		rollingset *carbonv1.RollingSet
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "test1",
			args: args{rollingset: newRollingsetFinalizer("")},
			want: false,
		},
		{
			name: "test2",
			args: args{rollingset: newRollingsetFinalizer("smoothDeletion")},
			want: true,
		},
		{
			name: "test3",
			args: args{rollingset: newRollingsetFinalizer("SmoothDeletion")},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			if got := HasSmoothFinalizer(tt.args.rollingset); got != tt.want {
				t.Errorf("Controller.hasSmoothFinalizer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecordRollingsetHistory(t *testing.T) {
	type args struct {
		rollingset *carbonv1.RollingSet
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test",
			args: args{rollingset: newRollingSet("version-new")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RecordRollingsetHistory(tt.args.rollingset)
		})
	}
}

func TestEqualsMap(t *testing.T) {
	var c, d map[string]int = nil, make(map[string]int)
	if reflect.DeepEqual(c, d) {
		t.Errorf("equals, c %v", c)
	}
}

func TestEqualsMap2(t *testing.T) {
	var c, d map[string]int = make(map[string]int), make(map[string]int)
	if !reflect.DeepEqual(c, d) {
		t.Errorf("not equals, c %v", c)
	}
}

func TestEqualsMap3(t *testing.T) {
	var c, d map[string]int = make(map[string]int), make(map[string]int)
	c = d
	if !reflect.DeepEqual(c, d) {
		t.Errorf("not equals, c %v", c)
	}
}

func TestEqualsMap4(t *testing.T) {
	var c, d map[string]int = make(map[string]int), make(map[string]int)
	c["key"] = 1
	d["key"] = 1
	if !reflect.DeepEqual(c, d) {
		t.Errorf("not equals, c %v", c)
	}
}

func Test_copyGeneralAnnotations(t *testing.T) {
	type args struct {
		src map[string]string
		dst map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "copy",
			args: args{
				src: map[string]string{
					"app.hippo.io/app-name":   "app",
					"app.hippo.io/role-name":  "role",
					"app.hippo.io/group-name": "group",
				},
				dst: map[string]string{
					"other": "app",
				},
			},
			want: map[string]string{
				"app.hippo.io/app-name":   "app",
				"app.hippo.io/role-name":  "role",
				"app.hippo.io/group-name": "group",
				"other":                   "app",
			},
		},
		{
			name: "copy",
			args: args{
				src: map[string]string{
					"app.hippo.io/app-name": "app",
					"other1":                "other1",
				},
				dst: map[string]string{
					"other": "app",
				},
			},
			want: map[string]string{
				"app.hippo.io/app-name": "app",
				"other":                 "app",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copyGeneralAnnotations(tt.args.src, tt.args.dst)
			if !reflect.DeepEqual(tt.args.dst, tt.want) {
				t.Errorf("compareGeneralAnnotations() = %v, want %v", tt.args.dst, tt.want)
			}
		})
	}
}

func Test_compareGeneralAnnotations(t *testing.T) {
	type args struct {
		src map[string]string
		dst map[string]string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "true",
			args: args{
				src: map[string]string{
					"app.hippo.io/app-name": "app",
				},
				dst: map[string]string{
					"app.hippo.io/app-name": "app",
					"other":                 "app",
				},
			},
			want: true,
		},
		{
			name: "true",
			args: args{
				src: map[string]string{
					"app.hippo.io/app-name":   "app",
					"app.hippo.io/role-name":  "role",
					"app.hippo.io/group-name": "group",
				},
				dst: map[string]string{
					"app.hippo.io/app-name":   "app",
					"app.hippo.io/role-name":  "role",
					"app.hippo.io/group-name": "group",
					"other":                   "app",
				},
			},
			want: true,
		},
		{
			name: "true",
			args: args{
				src: map[string]string{
					"app.hippo.io/app-name":   "app",
					"app.hippo.io/role-name":  "role",
					"app.hippo.io/group-name": "group",
				},
				dst: map[string]string{
					"app.hippo.io/app-name":  "app",
					"app.hippo.io/role-name": "role",
					"other":                  "app",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareGeneralAnnotations(tt.args.src, tt.args.dst); got != tt.want {
				t.Errorf("compareGeneralAnnotations() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_copyGeneralMetas(t *testing.T) {
	type args struct {
		replica            *carbonv1.Replica
		generalLabels      map[string]string
		generalAnnotations map[string]string
	}
	tests := []struct {
		name            string
		args            args
		wantLabels      map[string]string
		wantAnnotations map[string]string
	}{
		{
			name: "normal",
			args: args{
				replica: &carbonv1.Replica{
					WorkerNode: carbonv1.WorkerNode{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"oldLable":            "oldLabel",
								"worker-version-hash": "workerName",
							},
							Annotations: map[string]string{
								"oldAnno": "oldAnno",
							},
						},
					},
				},
				generalLabels: map[string]string{
					"newLabel":        "otherLabel",
					"rs-version-hash": "rsname",
				},
				generalAnnotations: map[string]string{
					"otherAnnotation":        "otherAnnotation",
					"app.hippo.io/app-name":  "app",
					"app.hippo.io/role-name": "role",
				},
			},
			wantLabels: map[string]string{
				"newLabel":            "otherLabel",
				"rs-version-hash":     "rsname",
				"worker-version-hash": "workerName",
			},
			wantAnnotations: map[string]string{
				"oldAnno":                "oldAnno",
				"app.hippo.io/app-name":  "app",
				"app.hippo.io/role-name": "role",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			copyGeneralMetas(tt.args.replica, tt.args.generalLabels, tt.args.generalAnnotations)

			if !reflect.DeepEqual(tt.args.replica.Labels, tt.wantLabels) {
				t.Errorf("compareGeneralAnnotations() = %v, want %v", tt.args.replica.Labels, tt.wantLabels)
			}
			if !reflect.DeepEqual(tt.args.replica.Annotations, tt.wantAnnotations) {
				t.Errorf("compareGeneralAnnotations() = %v, want %v", tt.args.replica.Annotations, tt.wantAnnotations)
			}
		})
	}
}

func Test_NonVersionedKeys(t *testing.T) {
	rs := &carbonv1.RollingSet{
		Spec: carbonv1.RollingSetSpec{
			VersionPlan: carbonv1.VersionPlan{
				SignedVersionPlan: carbonv1.SignedVersionPlan{
					Template: &carbonv1.HippoPodTemplate{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"app.hippo.io/app-name": "app",
							},
						},
					},
				},
			},
		},
	}
	replica := newReplica("rsname", 1111, 1111)
	replica.Spec.Template = &carbonv1.HippoPodTemplate{ObjectMeta: metav1.ObjectMeta{}}
	keys := mapset.NewThreadUnsafeSet()
	keys.Add("app.hippo.io/app-name")
	assert.False(t, carbonv1.IsNonVersionedKeysMatch(rs.Spec.Template, replica.Spec.Template, keys))
	syncNonVersionedMetas(rs, replica, keys)
	assert.True(t, carbonv1.IsNonVersionedKeysMatch(rs.Spec.Template, replica.Spec.Template, keys))
	keys.Add("app.hippo.io/role-name")
	replica.Spec.Template.Labels["app.hippo.io/role-name"] = "role"
	assert.False(t, carbonv1.IsNonVersionedKeysMatch(rs.Spec.Template, replica.Spec.Template, keys))
	syncNonVersionedMetas(rs, replica, keys)
	assert.True(t, carbonv1.IsNonVersionedKeysMatch(rs.Spec.Template, replica.Spec.Template, keys))
}
