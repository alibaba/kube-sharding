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
	"reflect"
	"strconv"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/transfer"
	gomock "github.com/golang/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestRollingSet(appname, groupname, quota string, index int, schType string, single bool) carbonv1.RollingSet {
	rolename := groupname
	if 0 != index {
		rolename = strconv.Itoa(index)
	}
	name, _ := transfer.GenerateRollingsetName(appname, groupname, rolename, schType)
	role := carbonv1.RollingSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: quota,
			Name:      name,
			Labels: map[string]string{
				carbonv1.LabelKeyAppName:                 appname,
				carbonv1.LabelKeyRoleName:                groupname + "." + rolename,
				carbonv1.LabelKeyGroupName:               groupname,
				carbonv1.LabelKeyQuotaGroupID:            quota,
				carbonv1.DefaultRollingsetUniqueLabelKey: name,
				carbonv1.DefaultShardGroupUniqueLabelKey: transfer.SubGroupName(groupname),
			},
		},
	}
	if !single {
		role.OwnerReferences = []metav1.OwnerReference{{}}
	}
	return role
}

func newTestService(appname, groupname, rolename, quota string, count int, schType string, single bool) []carbonv1.ServicePublisher {
	var services = make([]carbonv1.ServicePublisher, count)
	shardGroupName := transfer.SubGroupName(groupname)
	roleName, _ := transfer.GenerateRollingsetName(appname, groupname, rolename, schType)
	for i := range services {
		name := roleName + "-" + strconv.Itoa(i)
		if !single {
			name = utils.AppendString(transfer.SubGroupName(groupname), ".", name)
		}
		services[i] = carbonv1.ServicePublisher{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: quota,
				Name:      name,
				Labels: map[string]string{
					carbonv1.DefaultShardGroupUniqueLabelKey: shardGroupName,
					carbonv1.DefaultRollingsetUniqueLabelKey: roleName,
				},
			},
		}
	}
	return services
}

type roleAndServices struct {
	role     carbonv1.RollingSet
	services []carbonv1.ServicePublisher
}

func newTestRoleAndService(appname, groupname, quota string, index, count int, single bool) roleAndServices {
	schType := SchTypeRole
	if !single {
		schType = SchTypeGroup
	}
	role := newTestRollingSet(appname, groupname, quota, index, schType, single)
	services := newTestService(appname, groupname, strconv.Itoa(index), quota, count, schType, single)
	return roleAndServices{
		role:     role,
		services: services,
	}
}

func newTestGroupRoleAndService(appname, groupname, quota string, shards, count int) []roleAndServices {
	results := make([]roleAndServices, shards)
	for i := 1; i <= shards; i++ {
		results[i-1] = newTestRoleAndService(appname, groupname, quota, i, count, false)
	}
	return results
}

func newTestShardGroupRoles(appname, groupname, quota string, shards int) []carbonv1.RollingSet {
	var roles = make([]carbonv1.RollingSet, shards)
	for i := 1; i <= shards; i++ {
		roles[i-1] = newTestRollingSet(appname, groupname, quota, i, SchTypeGroup, false)
	}
	return roles
}

func newTestShardGroup(appname, groupname, quota string, shards, serviceCount int) (
	carbonv1.ShardGroup, []roleAndServices, []carbonv1.ServicePublisher) {
	name := transfer.SubGroupName(groupname)
	roles := newTestGroupRoleAndService(appname, groupname, quota, shards, serviceCount)
	var services = []carbonv1.ServicePublisher{}
	for i := range roles {
		services = append(services, roles[i].services...)
	}
	return carbonv1.ShardGroup{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: quota,
			Name:      name,
			Labels: map[string]string{
				carbonv1.LabelKeyAppName:                 appname,
				carbonv1.LabelKeyGroupName:               groupname,
				carbonv1.LabelKeyQuotaGroupID:            quota,
				carbonv1.DefaultShardGroupUniqueLabelKey: name,
			},
		},
	}, roles, services
}

func Test_k8sAdapter_deleteSingleRollingset(t *testing.T) {
	type fields struct {
		cluster string
		manager resourceManager
	}
	type args struct {
		appName string
		groupID string
	}
	role := newTestRollingSet("test-app", "test-group", "igraph", 0, SchTypeRole, true)
	service := newTestService("test-app", "test-group", "test-group", "igraph", 1, SchTypeRole, true)
	role1 := newTestRollingSet("test-app", "test-group", "cainiao", 0, SchTypeRole, true)
	service1 := newTestService("test-app", "test-group", "test-group", "cainiao", 1, SchTypeRole, true)
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	manager := NewMockresourceManager(ctl)
	manager.EXPECT().
		listRollingSet("test-app", "test-group", true).
		Return([]carbonv1.RollingSet{role}, nil).MaxTimes(2)
	manager.EXPECT().
		getRollingsetServices("test-app", "igraph", "test-group").
		Return(service, nil)
	manager.EXPECT().
		deleteService("igraph", "test-group-0").
		Return(nil)
	manager.EXPECT().
		deleteRollingSet("igraph", "test-group").
		Return(nil)

	manager1 := NewMockresourceManager(ctl)
	manager1.EXPECT().
		listRollingSet("test-app", "test-group", true).
		Return([]carbonv1.RollingSet{role, role1}, nil).MaxTimes(2)
	manager1.EXPECT().
		getRollingsetServices("test-app", "igraph", "test-group").
		Return(service, nil)
	manager1.EXPECT().
		deleteService("igraph", "test-group-0").
		Return(nil)
	manager1.EXPECT().
		deleteRollingSet("igraph", "test-group").
		Return(nil)
	manager1.EXPECT().
		getRollingsetServices("test-app", "cainiao", "test-group").
		Return(service1, nil)
	manager1.EXPECT().
		deleteService("cainiao", "test-group-0").
		Return(nil)
	manager1.EXPECT().
		deleteRollingSet("cainiao", "test-group").
		Return(nil)

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "normal",
			fields: fields{
				manager: manager,
			},
			args: args{
				appName: "test-app",
				groupID: "test-group",
			},
			wantErr: false,
		},
		{
			name: "dup",
			fields: fields{
				manager: manager1,
			},
			args: args{
				appName: "test-app",
				groupID: "test-group",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &k8sAdapter{
				cluster: tt.fields.cluster,
				manager: tt.fields.manager,
			}
			err := c.deleteSingleRollingset(tt.args.appName, tt.args.groupID)
			if (err != nil) != tt.wantErr {
				t.Errorf("k8sAdapter.deleteSingleRollingset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_k8sAdapter_deleteShardGroup(t *testing.T) {
	group, _, services := newTestShardGroup("test-app", "test-group", "igraph", 2, 2)
	group1, _, services1 := newTestShardGroup("test-app", "test-group", "cainiao", 2, 2)
	ctl := gomock.NewController(t)
	defer ctl.Finish()
	manager := NewMockresourceManager(ctl)
	manager.EXPECT().
		listGroup("test-app", "test-group").
		Return([]carbonv1.ShardGroup{group}, nil).MaxTimes(2)
	manager.EXPECT().
		getServiceByGroupname("test-app", "igraph", "test-group").
		Return(services, nil)
	manager.EXPECT().
		deleteService("igraph", "test-group.1-0").
		Return(nil)
	manager.EXPECT().
		deleteService("igraph", "test-group.1-1").
		Return(nil)
	manager.EXPECT().
		deleteService("igraph", "test-group.2-0").
		Return(nil)
	manager.EXPECT().
		deleteService("igraph", "test-group.2-1").
		Return(nil)
	manager.EXPECT().
		deleteShardGroup("igraph", "test-group").
		Return(nil)

	manager1 := NewMockresourceManager(ctl)
	manager1.EXPECT().
		listGroup("test-app", "test-group").
		Return([]carbonv1.ShardGroup{group, group1}, nil).MaxTimes(2)
	manager1.EXPECT().
		getServiceByGroupname("test-app", "igraph", "test-group").
		Return(services, nil)
	manager1.EXPECT().
		deleteService("igraph", "test-group.1-0").
		Return(nil)
	manager1.EXPECT().
		deleteService("igraph", "test-group.1-1").
		Return(nil)
	manager1.EXPECT().
		deleteService("igraph", "test-group.2-0").
		Return(nil)
	manager1.EXPECT().
		deleteService("igraph", "test-group.2-1").
		Return(nil)
	manager1.EXPECT().
		deleteShardGroup("igraph", "test-group").
		Return(nil)
	manager1.EXPECT().
		getServiceByGroupname("test-app", "cainiao", "test-group").
		Return(services1, nil)
	manager1.EXPECT().
		deleteService("cainiao", "test-group.1-0").
		Return(nil)
	manager1.EXPECT().
		deleteService("cainiao", "test-group.1-1").
		Return(nil)
	manager1.EXPECT().
		deleteService("cainiao", "test-group.2-0").
		Return(nil)
	manager1.EXPECT().
		deleteService("cainiao", "test-group.2-1").
		Return(nil)
	manager1.EXPECT().
		deleteShardGroup("cainiao", "test-group").
		Return(nil)

	type fields struct {
		cluster string
		manager resourceManager
	}
	type args struct {
		appName string
		groupID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "normal",
			fields: fields{
				manager: manager,
			},
			args: args{
				appName: "test-app",
				groupID: "test-group",
			},
			wantErr: false,
		},
		{
			name: "dup",
			fields: fields{
				manager: manager1,
			},
			args: args{
				appName: "test-app",
				groupID: "test-group",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &k8sAdapter{
				cluster: tt.fields.cluster,
				manager: tt.fields.manager,
			}
			err := c.deleteShardGroup(tt.args.appName, tt.args.groupID)
			if (err != nil) != tt.wantErr {
				t.Errorf("k8sAdapter.deleteShardGroup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_parseSchOptions(t *testing.T) {
	type args struct {
		str string
		zk  string
	}
	tests := []struct {
		name    string
		args    args
		want    *SchOptions
		wantErr bool
	}{
		{
			name: "single kv",
			args: args{
				str: "schType=role",
			},
			want:    &SchOptions{SchType: "role"},
			wantErr: false,
		},
		{
			name: "multiple kvs",
			args: args{
				str: "schType=group;proxy=tcp:abc.com:80,zkAddress=zfs://host:port/path",
			},
			want: &SchOptions{
				SchType:   "group",
				ZKAddress: "",
			},
			wantErr: false,
		},
		{
			name: "invaliad type",
			args: args{
				str: "schType=abc,zkAddress=zfs://host:port/path",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSchOptions(tt.args.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseSchOptions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseSchOptions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_k8sAdapter_filterDeleteRoleService(t *testing.T) {
	type fields struct {
		cluster string
		manager resourceManager
		syncer  k8sSyncer
	}
	type args struct {
		services []carbonv1.ServicePublisher
		group    *carbonv1.ShardGroup
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []carbonv1.ServicePublisher
	}{
		{
			name: "normal",
			args: args{
				services: []carbonv1.ServicePublisher{
					carbonv1.ServicePublisher{
						ObjectMeta: metav1.ObjectMeta{
							Name: "group.role1.aaa",
							Labels: map[string]string{
								"rs-version-hash": "group.role1",
							},
						},
					},
					carbonv1.ServicePublisher{
						ObjectMeta: metav1.ObjectMeta{
							Name: "group.role2.aaa",
							Labels: map[string]string{
								"rs-version-hash": "group.role2",
							},
						},
					},
					carbonv1.ServicePublisher{
						ObjectMeta: metav1.ObjectMeta{
							Name: "group.role3.aaa",
							Labels: map[string]string{
								"rs-version-hash": "group.role3",
							},
						},
					},
				},
				group: &carbonv1.ShardGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name: "group",
					},
					Spec: carbonv1.ShardGroupSpec{
						ShardTemplates: map[string]carbonv1.ShardTemplate{
							"role1": carbonv1.ShardTemplate{},
							"role2": carbonv1.ShardTemplate{},
						},
					},
				},
			},
			want: []carbonv1.ServicePublisher{
				carbonv1.ServicePublisher{
					ObjectMeta: metav1.ObjectMeta{
						Name: "group.role1.aaa",
						Labels: map[string]string{
							"rs-version-hash": "group.role1",
						},
					},
				},
				carbonv1.ServicePublisher{
					ObjectMeta: metav1.ObjectMeta{
						Name: "group.role2.aaa",
						Labels: map[string]string{
							"rs-version-hash": "group.role2",
						},
					},
				},
			},
		},
		{
			name: "normal",
			args: args{
				services: []carbonv1.ServicePublisher{
					carbonv1.ServicePublisher{
						ObjectMeta: metav1.ObjectMeta{
							Name: "group.role1.aaa",
							Labels: map[string]string{
								"rs-version-hash": "76b5ea7ba261954c6d0f7816e97ea3ce",
							},
						},
					},
					carbonv1.ServicePublisher{
						ObjectMeta: metav1.ObjectMeta{
							Name: "group.role2.aaa",
							Labels: map[string]string{
								"rs-version-hash": "76b5ea7ba261954c6d0f7816e97ea3ce",
							},
						},
					},
				},
				group: &carbonv1.ShardGroup{
					ObjectMeta: metav1.ObjectMeta{
						Name: "ele-newretail-rank-2-ele-newretail-rank2-ea119-ele-newretail-rank2",
					},
					Spec: carbonv1.ShardGroupSpec{
						ShardTemplates: map[string]carbonv1.ShardTemplate{
							"partition-0": carbonv1.ShardTemplate{},
						},
					},
				},
			},
			want: []carbonv1.ServicePublisher{
				carbonv1.ServicePublisher{
					ObjectMeta: metav1.ObjectMeta{
						Name: "group.role1.aaa",
						Labels: map[string]string{
							"rs-version-hash": "76b5ea7ba261954c6d0f7816e97ea3ce",
						},
					},
				},
				carbonv1.ServicePublisher{
					ObjectMeta: metav1.ObjectMeta{
						Name: "group.role2.aaa",
						Labels: map[string]string{
							"rs-version-hash": "76b5ea7ba261954c6d0f7816e97ea3ce",
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &k8sAdapter{
				cluster: tt.fields.cluster,
				manager: tt.fields.manager,
				syncer:  tt.fields.syncer,
			}
			if got := c.filterDeleteRoleService(tt.args.services, tt.args.group); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("k8sAdapter.filterDeleteRoleService() = %v, want %v", got, tt.want)
			}
		})
	}
}
