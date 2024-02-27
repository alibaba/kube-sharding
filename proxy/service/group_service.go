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
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/carbon"
)

// GroupService operates on groups
type GroupService interface {
	GetAppGroup(appName, schType string) ([]*carbon.GroupStatus, error)
	GetGroup(appName, schType string, groupIds []string) ([]*carbon.GroupStatus, error)
	DeleteGroup(appName, schType, groupID string) error
	NewGroup(appName string, schOpts *SchOptions, groupPlan *typespec.GroupPlan) error
	SetGroups(appName string, schOpts *SchOptions, groupPlans map[string]*typespec.GroupPlan) error
	SetGroup(appName, groupID string, schOpts *SchOptions, groupPlan *typespec.GroupPlan) error
	ScaleGroup(appName, groupID, zoneName string, scalePlan *carbonv1.ScaleSchedulePlan) error
	DisableScale(appName, groupID, zoneName string) error
	ReclaimWorkerNodesOnEviction(appName string, req *ReclaimWorkerNodeRequest) error
}
