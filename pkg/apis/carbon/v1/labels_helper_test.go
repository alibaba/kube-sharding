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

package v1

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/alibaba/kube-sharding/common/features"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMergeMap(t *testing.T) {
	type args struct {
		mainMaps map[string]string
		subMaps  map[string]string
		keys     []string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "merge",
			args: args{
				mainMaps: map[string]string{"a": "a", "b": "b"},
				subMaps:  map[string]string{"a": "b", "c": "c"},
				keys:     []string{"a", "c"},
			},
			want: map[string]string{"a": "b", "b": "b", "c": "c"},
		},

		{
			name: "not merge",
			args: args{
				mainMaps: map[string]string{"a": "a", "b": "b"},
				subMaps:  map[string]string{"a": "b", "b": "c", "d": ""},
				keys:     []string{"a", "c", "d"},
			},
			want: map[string]string{"a": "b", "b": "b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PatchMap(tt.args.mainMaps, tt.args.subMaps, tt.args.keys...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PatchMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCloneMap(t *testing.T) {
	type args struct {
		maps map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "merge",
			args: args{
				maps: map[string]string{"a": "a", "b": "b"},
			},
			want: map[string]string{"a": "a", "b": "b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CloneMap(tt.args.maps); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CloneMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResourceNameHash(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "not hash",
			args: args{
				name: "a",
			},
			want: "a",
		},
		{
			name: "hash",
			args: args{
				name: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
			},
			want: "15061fc3840896c5299a5b8cc1cf5b5f",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResourceNameHash(tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("ResourceNameHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ResourceNameHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddUniqueLabelHashKey(t *testing.T) {
	type args struct {
		labels map[string]string
		key    string
		value  string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
	}{
		{
			name: "add",
			args: args{
				labels: map[string]string{},
				key:    "a",
				value:  "b",
			},
			want: map[string]string{"a": "b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := AddUniqueLabelHashKey(tt.args.labels, tt.args.key, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("AddUniqueLabelHashKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.labels, tt.want) {
				t.Errorf("AddUniqueLabelHashKey() = %v, want %v", tt.args.labels, tt.want)
			}
		})
	}
}

func TestAddRsUniqueLabelHashKey(t *testing.T) {
	type args struct {
		labels map[string]string
		rsName string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
	}{
		{
			name: "add",
			args: args{
				labels: map[string]string{},
				rsName: "b",
			},
			want: map[string]string{"rs-version-hash": "b"},
		},
		{
			name: "add",
			args: args{
				labels: map[string]string{},
				rsName: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
			},
			want: map[string]string{"rs-version-hash": "15061fc3840896c5299a5b8cc1cf5b5f"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := AddRsUniqueLabelHashKey(tt.args.labels, tt.args.rsName); (err != nil) != tt.wantErr {
				t.Errorf("AddRsUniqueLabelHashKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.labels, tt.want) {
				t.Errorf("AddUniqueLabelHashKey() = %v, want %v", tt.args.labels, tt.want)
			}
		})
	}
}

func TestAddReplicaUniqueLabelHashKey(t *testing.T) {
	type args struct {
		labels map[string]string
		rsName string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
	}{
		{
			name: "add",
			args: args{
				labels: map[string]string{},
				rsName: "b",
			},
			want: map[string]string{"replica-version-hash": "b"},
		},
		{
			name: "add",
			args: args{
				labels: map[string]string{},
				rsName: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
			},
			want: map[string]string{"replica-version-hash": "15061fc3840896c5299a5b8cc1cf5b5f"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := AddReplicaUniqueLabelHashKey(tt.args.labels, tt.args.rsName); (err != nil) != tt.wantErr {
				t.Errorf("AddReplicaUniqueLabelHashKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
		if !reflect.DeepEqual(tt.args.labels, tt.want) {
			t.Errorf("AddUniqueLabelHashKey() = %v, want %v", tt.args.labels, tt.want)
		}
	}
}

func TestGetRsStringSelector(t *testing.T) {
	type args struct {
		rsName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "add",
			args: args{
				rsName: "b",
			},
			want: "rs-version-hash=b",
		},
		{
			name: "add",
			args: args{
				rsName: "abcdefghijklmnopqrstuvwxyz123456789-abcdefghijklmnopqrstuvwxyz123456789",
			},
			want: "rs-version-hash=02ff47a51528ab6199e75ec402f2012c",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRsStringSelector(tt.args.rsName); got != tt.want {
				t.Errorf("AddReplicaUniqueLabelHashKey() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetReplicaID(t *testing.T) {
	type args struct {
		object metav1.Object
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "get",
			args: args{
				object: &metav1.ObjectMeta{
					Labels: map[string]string{
						"replica-version-hash": "a",
					},
				},
			},
			want: "a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetReplicaID(tt.args.object); got != tt.want {
				t.Errorf("GetReplicaUniqueLabelHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddWorkerUniqueLabelHashKey(t *testing.T) {
	type args struct {
		labels map[string]string
		rsName string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
	}{
		{
			name: "add",
			args: args{
				labels: map[string]string{},
				rsName: "b",
			},
			want: map[string]string{"worker-version-hash": "b"},
		},
		{
			name: "add",
			args: args{
				labels: map[string]string{},
				rsName: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
			},
			want: map[string]string{"worker-version-hash": "15061fc3840896c5299a5b8cc1cf5b5f"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := AddWorkerUniqueLabelHashKey(tt.args.labels, tt.args.rsName); (err != nil) != tt.wantErr {
				t.Errorf("AddWorkerUniqueLabelHashKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.labels, tt.want) {
				t.Errorf("AddUniqueLabelHashKey() = %v, want %v", tt.args.labels, tt.want)
			}
		})
	}
}

func TestAddGroupUniqueLabelHashKey(t *testing.T) {
	type args struct {
		labels map[string]string
		rsName string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]string
		wantErr bool
	}{
		{
			name: "add",
			args: args{
				labels: map[string]string{},
				rsName: "b",
			},
			want: map[string]string{"shardgroup": "b"},
		},
		{
			name: "add",
			args: args{
				labels: map[string]string{},
				rsName: "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
			},
			want: map[string]string{"shardgroup": "15061fc3840896c5299a5b8cc1cf5b5f"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := AddGroupUniqueLabelHashKey(tt.args.labels, tt.args.rsName); (err != nil) != tt.wantErr {
				t.Errorf("AddGroupUniqueLabelHashKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.labels, tt.want) {
				t.Errorf("AddUniqueLabelHashKey() = %v, want %v", tt.args.labels, tt.want)
			}
		})
	}
}

func TestAddSelectorHashKey(t *testing.T) {
	type args struct {
		selector *metav1.LabelSelector
		key      string
		value    string
	}
	tests := []struct {
		name    string
		args    args
		want    *metav1.LabelSelector
		wantErr bool
	}{
		{
			name: "add",
			args: args{
				selector: &metav1.LabelSelector{},
				key:      "b",
				value:    "a",
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"b": "a"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := AddSelectorHashKey(tt.args.selector, tt.args.key, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("AddSelectorHashKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.selector, tt.want) {
				t.Errorf("AddUniqueLabelHashKey() = %v, want %v", tt.args.selector, tt.want)
			}
		})
	}
}

func TestAddRsSelectorHashKey(t *testing.T) {
	type args struct {
		selector *metav1.LabelSelector
		rsName   string
	}
	tests := []struct {
		name    string
		args    args
		want    *metav1.LabelSelector
		wantErr bool
	}{
		{
			name: "add",
			args: args{
				selector: &metav1.LabelSelector{},
				rsName:   "b",
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"rs-version-hash": "b"},
			},
		},
		{
			name: "add",
			args: args{
				selector: &metav1.LabelSelector{},
				rsName:   "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"rs-version-hash": "15061fc3840896c5299a5b8cc1cf5b5f"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := AddRsSelectorHashKey(tt.args.selector, tt.args.rsName); (err != nil) != tt.wantErr {
				t.Errorf("AddRsSelectorHashKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.selector, tt.want) {
				t.Errorf("AddUniqueLabelHashKey() = %v, want %v", tt.args.selector, tt.want)
			}
		})
	}
}

func TestAddReplicaSelectorHashKey(t *testing.T) {
	type args struct {
		selector *metav1.LabelSelector
		rsName   string
	}
	tests := []struct {
		name    string
		args    args
		want    *metav1.LabelSelector
		wantErr bool
	}{
		{
			name: "add",
			args: args{
				selector: &metav1.LabelSelector{},
				rsName:   "b",
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"replica-version-hash": "b"},
			},
		},
		{
			name: "add",
			args: args{
				selector: &metav1.LabelSelector{},
				rsName:   "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"replica-version-hash": "15061fc3840896c5299a5b8cc1cf5b5f"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := AddReplicaSelectorHashKey(tt.args.selector, tt.args.rsName); (err != nil) != tt.wantErr {
				t.Errorf("AddReplicaSelectorHashKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.selector, tt.want) {
				t.Errorf("AddUniqueLabelHashKey() = %v, want %v", tt.args.selector, tt.want)
			}
		})
	}
}

func TestAddWorkerSelectorHashKey(t *testing.T) {
	type args struct {
		selector *metav1.LabelSelector
		rsName   string
	}
	tests := []struct {
		name    string
		args    args
		want    *metav1.LabelSelector
		wantErr bool
	}{
		{
			name: "add",
			args: args{
				selector: &metav1.LabelSelector{},
				rsName:   "b",
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"worker-version-hash": "b"},
			},
		},
		{
			name: "add",
			args: args{
				selector: &metav1.LabelSelector{},
				rsName:   "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"worker-version-hash": "15061fc3840896c5299a5b8cc1cf5b5f"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := AddWorkerSelectorHashKey(tt.args.selector, tt.args.rsName); (err != nil) != tt.wantErr {
				t.Errorf("AddWorkerSelectorHashKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.selector, tt.want) {
				t.Errorf("AddUniqueLabelHashKey() = %v, want %v", tt.args.selector, tt.want)
			}
		})
	}
}

func TestAddGroupSelectorHashKey(t *testing.T) {
	type args struct {
		selector *metav1.LabelSelector
		rsName   string
	}
	tests := []struct {
		name    string
		args    args
		want    *metav1.LabelSelector
		wantErr bool
	}{
		{
			name: "add",
			args: args{
				selector: &metav1.LabelSelector{},
				rsName:   "b",
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"shardgroup": "b"},
			},
		},
		{
			name: "add",
			args: args{
				selector: &metav1.LabelSelector{},
				rsName:   "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"shardgroup": "15061fc3840896c5299a5b8cc1cf5b5f"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := AddGroupSelectorHashKey(tt.args.selector, tt.args.rsName); (err != nil) != tt.wantErr {
				t.Errorf("AddGroupSelectorHashKey() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.selector, tt.want) {
				t.Errorf("AddUniqueLabelHashKey() = %v, want %v", tt.args.selector, tt.want)
			}
		})
	}
}

func TestCopyWorkerAndReplicaLabels(t *testing.T) {
	type args struct {
		mainMaps map[string]string
		subMaps  map[string]string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "copy",
			args: args{
				mainMaps: map[string]string{
					"a": "a",
				},
				subMaps: map[string]string{
					"b":                    "b",
					"replica-version-hash": "c",
					"worker-version-hash":  "d",
				},
			},
			want: map[string]string{
				"a":                    "a",
				"replica-version-hash": "c",
				"worker-version-hash":  "d",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CopyWorkerAndReplicaLabels(tt.args.mainMaps, tt.args.subMaps); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CopyWorkerAndReplicaLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateWorkerLabels(t *testing.T) {
	type args struct {
		labels      map[string]string
		replicaName string
		workerName  string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "copy",
			args: args{
				labels: map[string]string{
					"a": "a",
				},
				replicaName: "c",
				workerName:  "d",
			},
			want: map[string]string{
				"a":                     "a",
				"replica-version-hash":  "c",
				"worker-version-hash":   "d",
				"app.c2.io/worker-role": "current",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateWorkerLabels(tt.args.labels, tt.args.replicaName, tt.args.workerName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateWorkerLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateWorkerSelector(t *testing.T) {
	type args struct {
		selector    *metav1.LabelSelector
		replicaName string
		workerName  string
	}
	tests := []struct {
		name string
		args args
		want *metav1.LabelSelector
	}{
		{
			name: "add",
			args: args{
				selector:    &metav1.LabelSelector{MatchLabels: map[string]string{"a": "a"}},
				replicaName: "b",
				workerName:  "c",
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"a": "a", "replica-version-hash": "b", "worker-version-hash": "c"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateWorkerSelector(tt.args.selector, tt.args.replicaName, tt.args.workerName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateWorkerSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCloneSelector(t *testing.T) {
	type args struct {
		selector *metav1.LabelSelector
	}
	tests := []struct {
		name string
		args args
		want *metav1.LabelSelector
	}{
		{
			name: "clone",
			args: args{
				selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"a": "a", "replica-version-hash": "b", "worker-version-hash": "c"},
				},
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"a": "a", "replica-version-hash": "b", "worker-version-hash": "c"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CloneSelector(tt.args.selector); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CloneSelector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRsID(t *testing.T) {
	type args struct {
		object metav1.Object
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "get",
			args: args{
				object: &metav1.ObjectMeta{
					Labels: map[string]string{
						"rs-version-hash": "a",
					},
				},
			},
			want: "a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRsID(tt.args.object); got != tt.want {
				t.Errorf("GetRsID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetShardGroupID(t *testing.T) {
	type args struct {
		object metav1.Object
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "get",
			args: args{
				object: &metav1.ObjectMeta{
					Labels: map[string]string{
						"shardgroup": "a",
					},
				},
			},
			want: "a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetShardGroupID(tt.args.object); got != tt.want {
				t.Errorf("GetShardGroupID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetClusterName(t *testing.T) {
	type args struct {
		object metav1.Object
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "get",
			args: args{
				object: &metav1.ObjectMeta{
					Labels: map[string]string{
						"app.hippo.io/cluster-name": "a",
					},
				},
			},
			want: "a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetClusterName(tt.args.object); got != tt.want {
				t.Errorf("GetClusterName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAppName(t *testing.T) {
	type args struct {
		object metav1.Object
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "get from label",
			args: args{
				object: &metav1.ObjectMeta{
					Labels: map[string]string{
						"app.hippo.io/app-name": "a",
					},
				},
			},
			want: "a",
		},
		{
			name: "get from annotation",
			args: args{
				object: &metav1.ObjectMeta{
					Annotations: map[string]string{
						"app.hippo.io/app-name": "a",
					},
				},
			},
			want: "a",
		},
		{
			name: "get from annotation first",
			args: args{
				object: &metav1.ObjectMeta{
					Annotations: map[string]string{
						"app.hippo.io/app-name": "a",
					},
					Labels: map[string]string{
						"app.hippo.io/app-name": "b",
					},
				},
			},
			want: "a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetAppName(tt.args.object); got != tt.want {
				t.Errorf("GetAppName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetGroupName(t *testing.T) {
	type args struct {
		object metav1.Object
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "get from label",
			args: args{
				object: &metav1.ObjectMeta{
					Labels: map[string]string{
						"app.hippo.io/group-name": "a",
					},
				},
			},
			want: "a",
		},
		{
			name: "get from annotation",
			args: args{
				object: &metav1.ObjectMeta{
					Annotations: map[string]string{
						"app.hippo.io/group-name": "a",
					},
				},
			},
			want: "a",
		},
		{
			name: "get from annotation first",
			args: args{
				object: &metav1.ObjectMeta{
					Annotations: map[string]string{
						"app.hippo.io/group-name": "a",
					},
					Labels: map[string]string{
						"app.hippo.io/group-name": "b",
					},
				},
			},
			want: "a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetGroupName(tt.args.object); got != tt.want {
				t.Errorf("GetGroupName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRoleName(t *testing.T) {
	type args struct {
		object metav1.Object
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "get from label",
			args: args{
				object: &metav1.ObjectMeta{
					Labels: map[string]string{
						"app.hippo.io/role-name": "a",
					},
				},
			},
			want: "a",
		},
		{
			name: "get from annotation",
			args: args{
				object: &metav1.ObjectMeta{
					Annotations: map[string]string{
						"app.hippo.io/role-name": "a",
					},
				},
			},
			want: "a",
		},
		{
			name: "get from annotation first",
			args: args{
				object: &metav1.ObjectMeta{
					Annotations: map[string]string{
						"app.hippo.io/role-name": "a",
					},
					Labels: map[string]string{
						"app.hippo.io/role-name": "b",
					},
				},
			},
			want: "a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetRoleName(tt.args.object); got != tt.want {
				t.Errorf("GetRoleName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGeC2tRoleName(t *testing.T) {
	type args struct {
		object metav1.Object
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "get",
			args: args{
				object: &metav1.ObjectMeta{
					Labels: map[string]string{
						"app.hippo.io/group-name": "g",
						"app.hippo.io/role-name":  "g.a",
					},
				},
			},
			want: "a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetC2RoleName(tt.args.object); got != tt.want {
				t.Errorf("GetRoleName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetHippoLabelKeys(t *testing.T) {
	type args struct {
		object metav1.Object
		key    string
		value  string
	}
	tests := []struct {
		name string
		args args
		want metav1.Object
	}{
		{
			name: "add",
			args: args{
				object: &metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				key:   "a",
				value: "b",
			},
			want: &metav1.ObjectMeta{
				Labels: map[string]string{"a": "b"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setLabel(tt.args.object, tt.args.key, tt.args.value)
		})
		if !reflect.DeepEqual(tt.args.object, tt.want) {
			t.Errorf("setLabel() = %v, want %v", tt.args.object, tt.want)
		}
	}
}

func TestSetClusterName(t *testing.T) {
	type args struct {
		object      metav1.Object
		clusterName string
	}
	tests := []struct {
		name string
		args args
		want metav1.Object
	}{
		{
			name: "add",
			args: args{
				object: &metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				clusterName: "b",
			},
			want: &metav1.ObjectMeta{
				Labels: map[string]string{"app.hippo.io/cluster-name": "b"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetClusterName(tt.args.object, tt.args.clusterName)
		})
		if !reflect.DeepEqual(tt.args.object, tt.want) {
			t.Errorf("setLabel() = %v, want %v", tt.args.object, tt.want)
		}
	}
}

func TestSetAppName(t *testing.T) {
	type args struct {
		object  metav1.Object
		appName string
	}
	tests := []struct {
		name string
		args args
		want metav1.Object
	}{
		{
			name: "add",
			args: args{
				object: &metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				appName: "b",
			},
			want: &metav1.ObjectMeta{
				Labels: map[string]string{"app.hippo.io/app-name": "b"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetAppName(tt.args.object, tt.args.appName)
		})
		if !reflect.DeepEqual(tt.args.object, tt.want) {
			t.Errorf("setLabel() = %v, want %v", tt.args.object, tt.want)
		}
	}
}

func TestSetGroupName(t *testing.T) {
	type args struct {
		object    metav1.Object
		groupName string
	}
	tests := []struct {
		name string
		args args
		want metav1.Object
	}{
		{
			name: "add",
			args: args{
				object: &metav1.ObjectMeta{
					Labels: map[string]string{},
				},
				groupName: "b",
			},
			want: &metav1.ObjectMeta{
				Labels: map[string]string{"app.hippo.io/group-name": "b"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetGroupName(tt.args.object, tt.args.groupName)
		})
		if !reflect.DeepEqual(tt.args.object, tt.want) {
			t.Errorf("setLabel() = %v, want %v", tt.args.object, tt.want)
		}
	}
}

func TestSetHippoLabelSelector(t *testing.T) {
	type args struct {
		selector *metav1.LabelSelector
		key      string
		value    string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    *metav1.LabelSelector
	}{
		{
			name: "add",
			args: args{
				selector: &metav1.LabelSelector{},
				key:      "a",
				value:    "b",
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"a": "b"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SetHippoLabelSelector(tt.args.selector, tt.args.key, tt.args.value); (err != nil) != tt.wantErr {
				t.Errorf("SetHippoLabelSelector() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.selector, tt.want) {
				t.Errorf("setLabel() = %v, want %v", tt.args.selector, tt.want)
			}
		})
	}
}

func TestSetClusterSelector(t *testing.T) {
	type args struct {
		selector    *metav1.LabelSelector
		clusterName string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    *metav1.LabelSelector
	}{
		{
			name: "add",
			args: args{
				selector:    &metav1.LabelSelector{},
				clusterName: "b",
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.hippo.io/cluster-name": "b"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SetClusterSelector(tt.args.selector, tt.args.clusterName); (err != nil) != tt.wantErr {
				t.Errorf("SetClusterSelector() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.selector, tt.want) {
				t.Errorf("setLabel() = %v, want %v", tt.args.selector, tt.want)
			}
		})
	}
}

func TestSetSetAppSelector(t *testing.T) {
	type args struct {
		selector *metav1.LabelSelector
		appName  string
		hashFlag bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    *metav1.LabelSelector
	}{
		{
			name: "add",
			args: args{
				selector: &metav1.LabelSelector{},
				appName:  "b",
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.hippo.io/app-name": "b"},
			},
		},
		{
			name: "hash label",
			args: args{
				selector: &metav1.LabelSelector{},
				appName:  "b",
				hashFlag: true,
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.c2.io/app-name-hash": "b"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features.C2MutableFeatureGate.Set(fmt.Sprintf("HashLabelValue=%t", tt.args.hashFlag))
			if err := SetAppSelector(tt.args.selector, tt.args.appName); (err != nil) != tt.wantErr {
				t.Errorf("SetAppCluster() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.selector, tt.want) {
				t.Errorf("setLabel() = %v, want %v", tt.args.selector, tt.want)
			}
		})
	}
}

func TestSetGroupSelector(t *testing.T) {
	type args struct {
		selector  *metav1.LabelSelector
		groupName string
		hashFlag  bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    *metav1.LabelSelector
	}{
		{
			name: "add",
			args: args{
				selector:  &metav1.LabelSelector{},
				groupName: "b",
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.hippo.io/group-name": "b"},
			},
		},
		{
			name: "hash label",
			args: args{
				selector:  &metav1.LabelSelector{},
				groupName: "b",
				hashFlag:  true,
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.c2.io/group-name-hash": "b"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features.C2MutableFeatureGate.Set(fmt.Sprintf("HashLabelValue=%t", tt.args.hashFlag))
			if err := SetGroupSelector(tt.args.selector, tt.args.groupName); (err != nil) != tt.wantErr {
				t.Errorf("SetGroupSelector() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.selector, tt.want) {
				t.Errorf("setLabel() = %v, want %v", tt.args.selector, tt.want)
			}
		})
	}
}

func TestSetRoleSelector(t *testing.T) {
	type args struct {
		selector *metav1.LabelSelector
		roleName string
		hashFlag bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    *metav1.LabelSelector
	}{
		{
			name: "add",
			args: args{
				selector: &metav1.LabelSelector{},
				roleName: "b",
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.hippo.io/role-name": "b"},
			},
		},
		{
			name: "hash label",
			args: args{
				selector: &metav1.LabelSelector{},
				roleName: "b",
				hashFlag: true,
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.c2.io/role-name-hash": "b"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features.C2MutableFeatureGate.Set(fmt.Sprintf("HashLabelValue=%t", tt.args.hashFlag))
			if err := SetRoleSelector(tt.args.selector, tt.args.roleName); (err != nil) != tt.wantErr {
				t.Errorf("SetRoleSelector() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.selector, tt.want) {
				t.Errorf("setLabel() = %v, want %v", tt.args.selector, tt.want)
			}
		})
	}
}

func TestSetGroupsSelector(t *testing.T) {
	type args struct {
		selector *metav1.LabelSelector
		groupIDs []string
		hashFlag bool
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    *metav1.LabelSelector
	}{
		{
			name: "add",
			args: args{
				selector: &metav1.LabelSelector{},
				groupIDs: []string{"b"},
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.hippo.io/group-name": "b"},
			},
		},
		{
			name: "adds",
			args: args{
				selector: &metav1.LabelSelector{},
				groupIDs: []string{"b", "a"},
			},
			want: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      LabelKeyGroupName,
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"b", "a"},
				}},
			},
		},
		{
			name: "hash label",
			args: args{
				selector: &metav1.LabelSelector{},
				groupIDs: []string{"b", "a"},
				hashFlag: true,
			},
			want: &metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{{
					Key:      LabelKeyGroupNameHash,
					Operator: metav1.LabelSelectorOpIn,
					Values:   []string{"b", "a"},
				}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features.C2MutableFeatureGate.Set(fmt.Sprintf("HashLabelValue=%t", tt.args.hashFlag))
			if err := SetGroupsSelector(tt.args.selector, tt.args.groupIDs...); (err != nil) != tt.wantErr {
				t.Errorf("SetGroupsSelector() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.selector, tt.want) {
				t.Errorf("setLabel() = %v, want %v", tt.args.selector, tt.want)
			}
		})
	}
}

func TestSetObjectAppRoleName(t *testing.T) {
	type args struct {
		object     metav1.Object
		app        string
		role       string
		compatible bool
	}
	tests := []struct {
		name string
		args args
		want metav1.ObjectMeta
	}{
		{
			name: "annotation nil",
			args: args{
				object:     &metav1.ObjectMeta{},
				app:        "app1",
				role:       "role1",
				compatible: false,
			},
			want: metav1.ObjectMeta{
				Annotations: map[string]string{
					LabelKeyAppName:  "app1",
					LabelKeyRoleName: "role1",
				},
			},
		},
		{
			name: "annotation not nil",
			args: args{
				object:     &metav1.ObjectMeta{Annotations: map[string]string{}},
				app:        "app1",
				role:       "role1",
				compatible: false,
			},
			want: metav1.ObjectMeta{
				Annotations: map[string]string{
					LabelKeyAppName:  "app1",
					LabelKeyRoleName: "role1",
				},
			},
		},
		{
			name: "both annotation and label",
			args: args{
				object:     &metav1.ObjectMeta{},
				app:        "app1",
				role:       "role1",
				compatible: true,
			},
			want: metav1.ObjectMeta{
				Annotations: map[string]string{
					LabelKeyAppName:  "app1",
					LabelKeyRoleName: "role1",
				},
				Labels: map[string]string{
					LabelKeyAppName:  "app1",
					LabelKeyRoleName: "role1",
				},
			},
		},
	}
	// reset
	defer func() { features.C2MutableFeatureGate.Set(fmt.Sprintf("HashLabelValue=false")) }()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features.C2MutableFeatureGate.Set(fmt.Sprintf("HashLabelValue=%t", !tt.args.compatible))
			SetObjectAppRoleName(tt.args.object, tt.args.app, tt.args.role)
			if !reflect.DeepEqual(tt.args.object, &tt.want) {
				t.Errorf("SetObjectAppRoleName verify failed, \nactual: %s \nwant: %s", utils.ObjJSON(tt.args.object), utils.ObjJSON(tt.want))
			}
		})
	}
}

func TestLabelValueHash(t *testing.T) {
	type args struct {
		v      string
		enable bool
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "disable",
			args: args{
				v:      testLongString("A", LabelValueMaxLength*2),
				enable: false,
			},
			want: testLongString("A", LabelValueMaxLength*2),
		},
		{
			name: "original",
			args: args{
				v:      testLongString("A", LabelValueMaxLength),
				enable: true,
			},
			want: testLongString("A", LabelValueMaxLength),
		},
		{
			name: "hash long",
			args: args{
				v:      testLongString("A", LabelValueMaxLength*2),
				enable: true,
			},
			want: "b2dd2bb10ade5f63319ddff03481e50e",
		},
		{
			name: "hash invalid",
			args: args{
				v:      "_hello",
				enable: true,
			},
			want: "0b04160a73c420c48ad7a33f42911fbd",
		},
		{
			name: "hash invalid and long",
			args: args{
				v:      testLongString("=", LabelValueMaxLength*2),
				enable: true,
			},
			want: "b2f0c9da22daa4e391f2aea7800c963c",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := LabelValueHash(tt.args.v, tt.args.enable); got != tt.want {
				t.Errorf("LabelValueHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetObjectRoleHash(t *testing.T) {
	type args struct {
		object metav1.Object
		app    string
		role   string
	}
	tests := []struct {
		name string
		args args
		want metav1.Object
	}{
		{
			name: "original",
			args: args{
				object: &metav1.ObjectMeta{},
				app:    "app1",
				role:   "role1",
			},
			want: &metav1.ObjectMeta{
				Labels: map[string]string{
					LabelKeyAppNameHash:  "app1",
					LabelKeyRoleNameHash: "role1",
				},
			},
		},
		{
			name: "hash",
			args: args{
				object: &metav1.ObjectMeta{},
				app:    "_a",
				role:   "_r",
			},
			want: &metav1.ObjectMeta{
				Labels: map[string]string{
					LabelKeyAppNameHash:  "5c855e094bdf284e55e9d16627ddd64b",
					LabelKeyRoleNameHash: "364641d04574146d9f88001e66b4410f",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetObjectRoleHash(tt.args.object, tt.args.app, tt.args.role)
			if !reflect.DeepEqual(tt.args.object, tt.want) {
				t.Errorf("SetObjectRoleHash verify failed, \nactual: %s \nwant: %s", utils.ObjJSON(tt.args.object), utils.ObjJSON(tt.want))
			}
		})
	}
}

func testLongString(b string, n int) string {
	var s string
	for ; n > 0; n-- {
		s = s + b
	}
	return s
}

func TestSetObjectGroupHash(t *testing.T) {
	type args struct {
		object metav1.Object
		app    string
		group  string
	}
	tests := []struct {
		name string
		args args
		want metav1.Object
	}{
		{
			name: "original",
			args: args{
				object: &metav1.ObjectMeta{},
				app:    "app1",
				group:  "group1",
			},
			want: &metav1.ObjectMeta{
				Labels: map[string]string{
					LabelKeyAppNameHash:   "app1",
					LabelKeyGroupNameHash: "group1",
				},
			},
		},
		{
			name: "hash",
			args: args{
				object: &metav1.ObjectMeta{},
				app:    "_a",
				group:  "_g",
			},
			want: &metav1.ObjectMeta{
				Labels: map[string]string{
					LabelKeyAppNameHash:   "5c855e094bdf284e55e9d16627ddd64b",
					LabelKeyGroupNameHash: "13e9f0f7ec00ce86e65f26031d80b7ba",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetObjectGroupHash(tt.args.object, tt.args.app, tt.args.group)
			if !reflect.DeepEqual(tt.args.object, tt.want) {
				t.Errorf("SetObjectGroupHash verify failed, \nactual: %s \nwant: %s", utils.ObjJSON(tt.args.object), utils.ObjJSON(tt.want))
			}
		})
	}
}

func TestMakeNameLegal(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test",
			args: args{
				name: "app",
			},
			want: "app",
		},
		{
			name: "test",
			args: args{
				name: "app_1",
			},
			want: "app-1",
		},
		{
			name: "test",
			args: args{
				name: "app_1_",
			},
			want: "app-1-s",
		},
		{
			name: "test",
			args: args{
				name: "app_1_------------------------------------------------------------",
			},
			want: "67c053a8edae630fdde448fa71334367",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MakeNameLegal(tt.args.name); got != tt.want {
				t.Errorf("MakeNameLegal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseInjectLabels(t *testing.T) {
	type args struct {
		injectLabels string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "normal",
			args: args{
				injectLabels: "foo=bar,foo1=bar1",
			},
			want: map[string]string{
				"foo":  "bar",
				"foo1": "bar1",
			},
		},
		{
			name: "invalid",
			args: args{
				injectLabels: "foo=bar,foo1",
			},
			want: map[string]string{
				"foo": "bar",
			},
		},
		{
			name: "nil",
			args: args{
				injectLabels: "",
			},
			want: map[string]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseInjectLabels(tt.args.injectLabels); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseInjectLabels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsBelowMinHealth(t *testing.T) {
	rollingset := newRollingset()
	assert.False(t, IsBelowMinHealth(rollingset))
	rollingset.Spec.Replicas = utils.Int32Ptr(10)
	assert.False(t, IsBelowMinHealth(rollingset))
	unavailable := intstr.FromString(fmt.Sprintf("%d%%", 20))
	rollingset.Spec.Strategy.RollingUpdate = &v1.RollingUpdateDeployment{
		MaxUnavailable: &unavailable,
	}
	rollingset.Status.AvailableReplicas = 8
	assert.False(t, IsBelowMinHealth(rollingset))
	rollingset.Spec.ScaleSchedulePlan = &ScaleSchedulePlan{
		Replicas: utils.Int32Ptr(12),
	}
	assert.True(t, IsBelowMinHealth(rollingset))
	rollingset.Status.AvailableReplicas = 10
	assert.False(t, IsBelowMinHealth(rollingset))
}

func TestGetInstanceGroup(t *testing.T) {
	type args struct {
		object metav1.Object
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normal",
			args: args{
				object: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"sigma.ali/instance-group":     "sky-1",
							"sigma.ali/fed-instance-group": "sky-fed",
						},
					},
				},
			},
			want: "sky-1",
		},
		{
			name: "fed",
			args: args{
				object: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"sigma.ali/fed-instance-group": "sky-fed",
						},
					},
				},
			},
			want: "sky-fed",
		},
		{
			name: "empty",
			args: args{
				object: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{},
					},
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetInstanceGroup(tt.args.object); got != tt.want {
				t.Errorf("GetInstanceGroup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAppUnit(t *testing.T) {
	type args struct {
		object metav1.Object
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normal",
			args: args{
				object: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"sigma.alibaba-inc.com/app-unit":     "unit-1",
							"sigma.alibaba-inc.com/fed-app-unit": "unit-fed",
						},
					},
				},
			},
			want: "unit-1",
		},
		{
			name: "fed",
			args: args{
				object: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"sigma.alibaba-inc.com/fed-app-unit": "unit-fed",
						},
					},
				},
			},
			want: "unit-fed",
		},
		{
			name: "empty",
			args: args{
				object: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{},
					},
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetAppUnit(tt.args.object); got != tt.want {
				t.Errorf("GetAppUnit() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAppStage(t *testing.T) {
	type args struct {
		object metav1.Object
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normal",
			args: args{
				object: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"sigma.alibaba-inc.com/app-stage":     "stage-pre",
							"sigma.alibaba-inc.com/fed-app-stage": "stage-fed",
						},
					},
				},
			},
			want: "stage-pre",
		},
		{
			name: "fed",
			args: args{
				object: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"sigma.alibaba-inc.com/fed-app-stage": "stage-fed",
						},
					},
				},
			},
			want: "stage-fed",
		},
		{
			name: "empty",
			args: args{
				object: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{},
					},
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetAppStage(tt.args.object); got != tt.want {
				t.Errorf("GetAppUnit() = %v, want %v", got, tt.want)
			}
		})
	}
}
