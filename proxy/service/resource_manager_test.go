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

package service

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	carbonclientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	carbonfake "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/fake"
	carboninformers "github.com/alibaba/kube-sharding/pkg/client/informers/externalversions"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	"github.com/alibaba/kube-sharding/transfer"
	assert "github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubeclientset "k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	restclient "k8s.io/client-go/rest"
	core "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/controller"
)

func filterInformerActions(actions []core.Action) []core.Action {
	ret := []core.Action{}
	for _, action := range actions {
		if action.Matches("list", "shardgroups") ||
			action.Matches("list", "rollingsets") ||
			action.Matches("list", "pods") ||
			action.Matches("list", "replicas") ||
			action.Matches("list", "workernodes") ||
			action.Matches("list", "namespacemigrations") ||
			action.Matches("watch", "namespacemigrations") ||
			action.Matches("watch", "shardgroups") ||
			action.Matches("watch", "rollingsets") ||
			action.Matches("watch", "pods") ||
			action.Matches("watch", "replicas") ||
			action.Matches("watch", "workernodes") {
			continue
		}
		ret = append(ret, action)
	}

	return ret
}

func Test_defaultResourceManager_getRollingsetByGroupIDs(t *testing.T) {
	fakeclient := carbonfake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(fakeclient, controller.NoResyncPeriodFunc())
	rollingsetLister := informersFactory.Carbon().V1().RollingSets().Lister()

	role := newTestRollingSet("test-app", "test-group", "igraph", 0, SchTypeRole, true)
	role1 := newTestRollingSet("test-app", "test-group", "cainiao", 0, SchTypeRole, true)
	role2 := newTestRollingSet("test-app", "test-group1", "cainiao", 0, SchTypeRole, true)
	role1.CreationTimestamp = metav1.Time{Time: time.Now()}
	informersFactory.Carbon().V1().
		RollingSets().Informer().GetIndexer().Add(&role)
	informersFactory.Carbon().V1().
		RollingSets().Informer().GetIndexer().Add(&role1)
	informersFactory.Carbon().V1().
		RollingSets().Informer().GetIndexer().Add(&role2)

	roles := newTestShardGroupRoles("test-app", "test-shardgroup", "igraph", 2)
	roles1 := newTestShardGroupRoles("test-app", "test-shardgroup", "cainiao", 2)
	roles1[0].CreationTimestamp = metav1.Time{Time: time.Now()}
	roles1[1].CreationTimestamp = metav1.Time{Time: time.Now()}
	roles2 := newTestShardGroupRoles("test-app", "test-shardgroup1", "cainiao", 3)
	fmt.Println(utils.ObjJSON(roles2))
	informersFactory.Carbon().V1().
		RollingSets().Informer().GetIndexer().Add(&roles[0])
	informersFactory.Carbon().V1().
		RollingSets().Informer().GetIndexer().Add(&roles[1])
	informersFactory.Carbon().V1().
		RollingSets().Informer().GetIndexer().Add(&roles1[0])
	informersFactory.Carbon().V1().
		RollingSets().Informer().GetIndexer().Add(&roles1[1])
	informersFactory.Carbon().V1().
		RollingSets().Informer().GetIndexer().Add(&roles2[2])
	type fields struct {
		cluster             string
		cached              bool
		kubeConfig          *restclient.Config
		kubeClientSet       kubeclientset.Interface
		carbonClientSet     carbonclientset.Interface
		workerLister        listers.WorkerNodeLister
		workerListerSynced  cache.InformerSynced
		serviceLister       listers.ServicePublisherLister
		serviceListerSynced cache.InformerSynced
		rollingSetLister    listers.RollingSetLister
		rollingSetSynced    cache.InformerSynced
		groupLister         listers.ShardGroupLister
		groupSynced         cache.InformerSynced
		podLister           corelisters.PodLister
		podListerSynced     cache.InformerSynced
	}
	type args struct {
		appName  string
		groupIds []string
		single   bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []carbonv1.RollingSet
		wantErr bool
	}{
		{
			name: "single",
			fields: fields{
				carbonClientSet:  fakeclient,
				rollingSetLister: rollingsetLister,
			},
			args: args{
				appName:  "test-app",
				groupIds: []string{"test-group"},
				single:   true,
			},
			want:    []carbonv1.RollingSet{role},
			wantErr: false,
		},
		// {
		// 	name: "single2",
		// 	fields: fields{
		// 		carbonClientSet:  fakeclient,
		// 		rollingSetLister: rollingsetLister,
		// 		cached:           true,
		// 	},
		// 	args: args{
		// 		appName:  "test-app",
		// 		groupIds: []string{},
		// 		single:   true,
		// 	},
		// 	want:    []carbonv1.RollingSet{role, role2, roles1[0], roles1[1], roles2[0], roles2[1]},
		// 	wantErr: false,
		// },
		{
			name: "group",
			fields: fields{
				carbonClientSet:  fakeclient,
				rollingSetLister: rollingsetLister,
			},
			args: args{
				appName:  "test-app",
				groupIds: []string{"test-shardgroup"},
				single:   false,
			},
			want:    roles,
			wantErr: false,
		},
		{
			name: "group",
			fields: fields{
				carbonClientSet:  fakeclient,
				rollingSetLister: rollingsetLister,
				cached:           true,
			},
			args: args{
				appName:  "test-app",
				groupIds: []string{},
				single:   false,
			},
			want:    append(roles, roles2[2]),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &defaultResourceManager{
				cluster:             tt.fields.cluster,
				kubeClientSet:       tt.fields.kubeClientSet,
				carbonClientSet:     tt.fields.carbonClientSet,
				workerLister:        tt.fields.workerLister,
				workerListerSynced:  tt.fields.workerListerSynced,
				serviceLister:       tt.fields.serviceLister,
				serviceListerSynced: tt.fields.serviceListerSynced,
				rollingSetLister:    tt.fields.rollingSetLister,
				rollingSetSynced:    tt.fields.rollingSetSynced,
				groupLister:         tt.fields.groupLister,
				groupSynced:         tt.fields.groupSynced,
				podLister:           tt.fields.podLister,
				podListerSynced:     tt.fields.podListerSynced,
			}
			got, err := c.getRollingsetByGroupIDs(tt.args.appName, tt.args.groupIds, tt.args.single)
			if (err != nil) != tt.wantErr {
				t.Errorf("defaultResourceManager.getRollingsetByGroupIDs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			sort.Sort(SortenRoles(got))
			sort.Sort(SortenRoles(tt.want))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("defaultResourceManager.getRollingsetByGroupIDs() = %v, want %v", utils.ObjJSON(got), utils.ObjJSON(tt.want))
			}
		})
	}
}

// SortenRoles SortenRoles
type SortenRoles []carbonv1.RollingSet

//Len()
func (s SortenRoles) Len() int {
	return len(s)
}

//Less()
func (s SortenRoles) Less(i, j int) bool {
	return s[i].Name < s[j].Name
}

//Swap()
func (s SortenRoles) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func Test_newHippoKeySelector(t *testing.T) {
	cluster := "cluster"
	appName := "appName"
	role := "roleName"
	groupIDs := []string{
		"groupId1",
		"groupId2",
	}
	roleNames := []string{
		"role1",
		"role2",
	}
	type args struct {
		cluster   string
		app       string
		group     string
		role      string
		groupIDs  []string
		roleNames []string
	}
	tests := []struct {
		name string
		args args
		want *metav1.LabelSelector
	}{
		{
			name: "group ids nil",
			args: args{
				app:      appName,
				cluster:  cluster,
				role:     role,
				group:    "group",
				groupIDs: nil,
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					carbonv1.LabelKeyAppName:     appName,
					carbonv1.LabelKeyClusterName: cluster,
					carbonv1.LabelKeyGroupName:   "group",
					carbonv1.LabelKeyRoleName:    role,
				},
				MatchExpressions: []metav1.LabelSelectorRequirement{},
			},
		},
		{
			name: "normal",
			args: args{
				app:      appName,
				cluster:  cluster,
				role:     role,
				groupIDs: groupIDs,
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					carbonv1.LabelKeyAppName:     appName,
					carbonv1.LabelKeyClusterName: cluster,
					carbonv1.LabelKeyRoleName:    role,
				},
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      carbonv1.LabelKeyGroupName,
						Operator: metav1.LabelSelectorOpIn,
						Values:   groupIDs,
					},
				},
			},
		},
		{
			name: "normal",
			args: args{
				app:       appName,
				cluster:   cluster,
				role:      role,
				roleNames: roleNames,
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					carbonv1.LabelKeyAppName:     appName,
					carbonv1.LabelKeyClusterName: cluster,
					carbonv1.LabelKeyRoleName:    role,
				},
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      carbonv1.LabelKeyRoleName,
						Operator: metav1.LabelSelectorOpIn,
						Values:   roleNames,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, _ := metav1.LabelSelectorAsSelector(tt.want)
			if got, _ := newHippoKeySelector(tt.args.cluster, tt.args.app, tt.args.group, tt.args.role, tt.args.groupIDs, tt.args.roleNames); !reflect.DeepEqual(got, want) {
				t.Errorf("k8sAdapter.makeGroupLabelSelector() = %v, want %v", got.String(), tt.want.String())
			}
		})
	}
}

func Test_newC2ObjectSelector(t *testing.T) {
	type args struct {
		app   string
		group string
		role  string
	}
	tests := []struct {
		name string
		args args
		want *metav1.LabelSelector
	}{
		{
			name: "normal",
			args: args{
				app:   "rtp_share_admin_ea119",
				group: "ele-newretail-rank-2-ele-newretail-rank2-ea119-ele-newretail-rank2",
			},
			want: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					carbonv1.LabelKeyAppName: "rtp_share_admin_ea119",
					"shardgroup":             "d0c71d203d0e283c870e902398e6da47",
				},
				MatchExpressions: []metav1.LabelSelectorRequirement{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want, _ := metav1.LabelSelectorAsSelector(tt.want)
			if got, _ := newC2ObjectSelector(tt.args.app, tt.args.group, tt.args.role); !reflect.DeepEqual(got, want) {
				t.Errorf("k8sAdapter.makeGroupLabelSelector() = %v, want %v", got.String(), tt.want.String())
			}
		})
	}
}

func Test_defaultResourceManager_getShardGroupName(t *testing.T) {
	assert := assert.New(t)
	fakeclient := carbonfake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(fakeclient, controller.NoResyncPeriodFunc())
	groupLister := informersFactory.Carbon().V1().ShardGroups().Lister()
	group1, _, _ := newTestShardGroup("test_app", "test_group", "igraph", 1, 1)
	group2, _, _ := newTestShardGroup("test_app2", "test_group_2", "igraph", 1, 1)
	group3, _, _ := newTestShardGroup("test_app", "test_group_1", "igraph", 1, 1)
	informersFactory.Carbon().V1().
		ShardGroups().Informer().GetIndexer().Add(&group1)
	informersFactory.Carbon().V1().
		ShardGroups().Informer().GetIndexer().Add(&group2)
	informersFactory.Carbon().V1().
		ShardGroups().Informer().GetIndexer().Add(&group3)
	c := &defaultResourceManager{
		nameAllocator: &nameAllocator{groupLister: groupLister},
	}
	groupName, err := c.getShardGroupName("test_app", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group", groupName)
	groupName, err = c.getShardGroupName("test_app2", "igraph", "test_group_2")
	assert.NoError(err, "")
	assert.Equal("test-group-2", groupName)
	groupName, err = c.getShardGroupName("test_app3", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group-0", groupName)
	groupName, err = c.getShardGroupName("test_app", "igraph", "test_group_1")
	assert.NoError(err, "")
	assert.Equal("test-group-1", groupName)
	groupName, err = c.getShardGroupName("test_app2", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group-0", groupName)

	//添加同名不同app的group
	group4, _, _ := newTestShardGroup("test_app2", "test_group", "igraph", 1, 1)
	group4.Name = "test-group-999"
	group5, _, _ := newTestShardGroup("test_app3", "test_group", "igraph", 1, 1)
	group5.Name = "test-group-0"
	informersFactory.Carbon().V1().
		ShardGroups().Informer().GetIndexer().Add(&group4)
	informersFactory.Carbon().V1().
		ShardGroups().Informer().GetIndexer().Add(&group5)

	groupName, err = c.getShardGroupName("test_app", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group", groupName)

	//bugfix case
	groupName, err = c.getShardGroupName("test_app2", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group-999", groupName)

	groupName, err = c.getShardGroupName("test_app3", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group-0", groupName)

	groupName, err = c.getShardGroupName("test_app4", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group-3", groupName)

	groupName, err = c.getShardGroupName("test_app5", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group-3", groupName)

	informersFactory.Carbon().V1().
		ShardGroups().Informer().GetIndexer().Delete(&group1)

	informersFactory.Carbon().V1().
		ShardGroups().Informer().GetIndexer().Delete(&group3)

	informersFactory.Carbon().V1().
		ShardGroups().Informer().GetIndexer().Delete(&group5)

	groupName, err = c.getShardGroupName("test_app", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group", groupName)
	groupName, err = c.getShardGroupName("test_app2", "igraph", "test_group_0")
	assert.NoError(err, "")
	assert.Equal("test-group-0", groupName)
	groupName, err = c.getShardGroupName("test_app3", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group", groupName)
	groupName, err = c.getShardGroupName("test_app", "igraph", "test_group_1")
	assert.NoError(err, "")
	assert.Equal("test-group-1", groupName)
}

func Test_defaultResourceManager_getRollingSetName(t *testing.T) {
	assert := assert.New(t)
	fakeclient := carbonfake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := carboninformers.NewSharedInformerFactory(fakeclient, controller.NoResyncPeriodFunc())
	rollingsetLister := informersFactory.Carbon().V1().RollingSets().Lister()
	rollingset1 := newTestRollingSet("test_app", "test_group", "igraph", 0, transfer.SchTypeRole, true)
	rollingset2 := newTestRollingSet("test_app2", "test_group_2", "igraph", 0, transfer.SchTypeRole, true)
	rollingset3 := newTestRollingSet("test_app", "test_group_1", "igraph", 0, transfer.SchTypeRole, true)
	informersFactory.Carbon().V1().
		RollingSets().Informer().GetIndexer().Add(&rollingset1)
	informersFactory.Carbon().V1().
		RollingSets().Informer().GetIndexer().Add(&rollingset2)
	informersFactory.Carbon().V1().
		RollingSets().Informer().GetIndexer().Add(&rollingset3)
	c := &defaultResourceManager{
		nameAllocator: &nameAllocator{rollingsetLister: rollingsetLister},
	}
	rollingsetName, err := c.getRollingSetName("test_app", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group", rollingsetName)
	rollingsetName, err = c.getRollingSetName("test_app2", "igraph", "test_group_2")
	assert.NoError(err, "")
	assert.Equal("test-group-2", rollingsetName)
	rollingsetName, err = c.getRollingSetName("test_app3", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group-0", rollingsetName)
	rollingsetName, err = c.getRollingSetName("test_app", "igraph", "test_group_1")
	assert.NoError(err, "")
	assert.Equal("test-group-1", rollingsetName)
	rollingsetName, err = c.getRollingSetName("test_app2", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group-0", rollingsetName)

	//添加同名不同app的group
	rollingset4 := newTestRollingSet("test_app2", "test_group", "igraph", 0, transfer.SchTypeRole, true)
	rollingset4.Name = "test-group-999"
	rollingset5 := newTestRollingSet("test_app3", "test_group", "igraph", 0, transfer.SchTypeRole, true)
	rollingset5.Name = "test-group-0"
	informersFactory.Carbon().V1().
		RollingSets().Informer().GetIndexer().Add(&rollingset4)
	informersFactory.Carbon().V1().
		RollingSets().Informer().GetIndexer().Add(&rollingset5)

	rollingsetName, err = c.getRollingSetName("test_app", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group", rollingsetName)

	//bugfix case
	rollingsetName, err = c.getRollingSetName("test_app2", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group-999", rollingsetName)

	rollingsetName, err = c.getRollingSetName("test_app3", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group-0", rollingsetName)

	rollingsetName, err = c.getRollingSetName("test_app4", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group-3", rollingsetName)

	rollingsetName, err = c.getRollingSetName("test_app5", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group-3", rollingsetName)

	informersFactory.Carbon().V1().
		RollingSets().Informer().GetIndexer().Delete(&rollingset1)

	informersFactory.Carbon().V1().
		RollingSets().Informer().GetIndexer().Delete(&rollingset3)

	informersFactory.Carbon().V1().
		RollingSets().Informer().GetIndexer().Delete(&rollingset5)

	rollingsetName, err = c.getRollingSetName("test_app", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group", rollingsetName)
	rollingsetName, err = c.getRollingSetName("test_app2", "igraph", "test_group_0")
	assert.NoError(err, "")
	assert.Equal("test-group-0", rollingsetName)
	rollingsetName, err = c.getRollingSetName("test_app3", "igraph", "test_group")
	assert.NoError(err, "")
	assert.Equal("test-group", rollingsetName)
	rollingsetName, err = c.getRollingSetName("test_app", "igraph", "test_group_1")
	assert.NoError(err, "")
	assert.Equal("test-group-1", rollingsetName)
}
