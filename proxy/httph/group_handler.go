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

package httph

import (
	"fmt"
	"reflect"
	"time"

	"github.com/alibaba/kube-sharding/pkg/sdk"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils/restutil"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec"
	"github.com/alibaba/kube-sharding/proxy/service"
	restful "github.com/emicklei/go-restful"
	glog "k8s.io/klog"
)

const (
	paramGroupID  = "groupId"
	paramAppName  = "appName"
	paramZoneName = "zoneName"

	historyKeyFileCount   = 20
	historyTotalFileCount = 500
	historyRootPath       = "history"
)

// GroupHandler http handler to process Groups
type GroupHandler struct {
	groupService service.GroupService
	diffLogger   *utils.DiffLogger
}

// NewGroupHandler creates new group http handler
func NewGroupHandler(groupSvc service.GroupService) (*GroupHandler, error) {
	diffLogger, err := utils.NewDiffLogger(2000, func(msg string) { glog.Info(msg) }, historyKeyFileCount, historyTotalFileCount)
	if err != nil {
		return nil, err
	}
	return &GroupHandler{
		groupService: groupSvc,
		diffLogger:   diffLogger,
	}, nil
}

// NewWebService creates new web service.
func (c *GroupHandler) NewWebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Produces(any)
	ws.Consumes(any)
	ws.Path("/app/{appName}")
	ws.Route(ws.POST("/group/status").To(route.To(c.getGroup)))
	ws.Route(ws.PUT("/group/{groupId}").To(route.To(c.setGroup)))
	ws.Route(ws.POST("/group").To(route.To(c.newGroup)))
	ws.Route(ws.DELETE("/group/{groupId}").To(route.To(c.deleteGroup)))
	ws.Route(ws.POST("/groups").To(route.To(c.setGroups)))
	ws.Route(ws.POST("/group/{groupId}/scale").To(route.To(c.scaleGroup)))
	ws.Route(ws.POST("/group/{groupId}/scale/disable").To(route.To(c.disableScale)))

	return ws
}

func (c *GroupHandler) getGroup(request *restful.Request, response *restful.Response) restutil.Entity {
	startTime := time.Now()
	var err error
	appName := request.PathParameter(paramAppName)
	var schOpts *service.SchOptions
	schOpts, err = c.getSchOptions(request)
	if err != nil {
		return newPBResponse(nil, err, nil)
	}
	var groupIDs []string
	if err = request.ReadEntity(&groupIDs); err != nil {
		return newPBResponse(nil, err, nil)
	}
	statusList, err := c.groupService.GetGroup(appName, schOpts.SchType, groupIDs)
	glog.V(4).Infof("GetGroup appName: %s, groupIDs: %v, cost: %s, err: %v", appName, groupIDs, time.Now().Sub(startTime).String(), err)

	if nil != err {
		return newPBResponse(nil, err, nil)
	}
	return newPBResponse(statusList, nil, nil)
}

func (c *GroupHandler) deleteGroup(request *restful.Request, response *restful.Response) restutil.Entity {
	startTime := time.Now()
	var err error
	groupID := request.PathParameter(paramGroupID)
	appName := request.PathParameter(paramAppName)
	var schOpts *service.SchOptions
	schOpts, err = c.getSchOptions(request)
	if err != nil {
		return sdk.NewResponse(nil, err, nil)
	}
	err = c.groupService.DeleteGroup(appName, schOpts.SchType, groupID)
	glog.Infof("DeleteGroup, appName: %s, groupId: %s, cost: %s, err: %v", appName, groupID, time.Now().Sub(startTime).String(), err)
	return sdk.NewResponse(nil, err, nil)
}

func (c *GroupHandler) newGroup(request *restful.Request, response *restful.Response) restutil.Entity {
	startTime := time.Now()

	var err error
	appName := request.PathParameter(paramAppName)
	var schOpts *service.SchOptions
	schOpts, err = c.getSchOptions(request)
	if err != nil {
		return sdk.NewResponse(nil, err, nil)
	}

	var groupPlan typespec.GroupPlan
	if err = request.ReadEntity(&groupPlan); err != nil {
		return sdk.NewResponse(nil, err, nil)
	}
	if err = validateGroupPlans(appName, map[string]*typespec.GroupPlan{groupPlan.GroupID: &groupPlan}, schOpts.SchType); err != nil {
		glog.Errorf("validateGroupPlans with error %s, %v", appName, err)
		return sdk.NewResponse(nil, err, nil)
	}
	c.writeTargetHistory(appName, map[string]*typespec.GroupPlan{groupPlan.GroupID: &groupPlan})
	err = c.groupService.NewGroup(appName, schOpts, &groupPlan)
	glog.Infof("NewGroup appName: %s, cost: %s, err: %v", appName, time.Now().Sub(startTime).String(), err)
	if glog.V(4) {
		glog.Infof("Group detail: %s", utils.ObjJSON(&groupPlan))
	}
	return sdk.NewResponse(nil, err, nil)
}

func (c *GroupHandler) setGroups(request *restful.Request, response *restful.Response) restutil.Entity {
	startTime := time.Now()
	var err error
	appName := request.PathParameter(paramAppName)
	var schOpts *service.SchOptions
	schOpts, err = c.getSchOptions(request)
	if err != nil {
		return sdk.NewResponse(nil, err, nil)
	}

	var groupPlans map[string]*typespec.GroupPlan
	if err = request.ReadEntity(&groupPlans); err != nil {
		return sdk.NewResponse(nil, err, nil)
	}
	if err = validateGroupPlans(appName, groupPlans, schOpts.SchType); err != nil {
		return sdk.NewResponse(nil, err, nil)
	}
	if schOpts.SchType != service.SchTypeNode { // for performance reason
		c.writeTargetHistory(appName, groupPlans)
	}
	err = c.groupService.SetGroups(appName, schOpts, groupPlans)
	glog.Infof("SetGroups appName: %s, cost: %s, %+v, err: %v", appName, time.Now().Sub(startTime).String(), *schOpts, err)
	if glog.V(4) {
		glog.Infof("Group detail: %s", utils.ObjJSON(groupPlans))
	}
	return sdk.NewResponse(nil, err, nil)
}

func (c *GroupHandler) setGroup(request *restful.Request, response *restful.Response) restutil.Entity {
	startTime := time.Now()
	var err error
	groupID := request.PathParameter(paramGroupID)
	appName := request.PathParameter(paramAppName)
	var schOpts *service.SchOptions
	schOpts, err = c.getSchOptions(request)
	if err != nil {
		return sdk.NewResponse(nil, err, nil)
	}

	var groupPlan typespec.GroupPlan
	if err = request.ReadEntity(&groupPlan); err != nil {
		return sdk.NewResponse(nil, err, nil)
	}
	if err = validateGroupPlans(appName, map[string]*typespec.GroupPlan{groupPlan.GroupID: &groupPlan}, schOpts.SchType); err != nil {
		return sdk.NewResponse(nil, err, nil)
	}
	c.writeTargetHistory(appName, map[string]*typespec.GroupPlan{groupID: &groupPlan})
	err = c.groupService.SetGroup(appName, groupID, schOpts, &groupPlan)
	glog.Infof("SetGroup: appName: %s, groupID: %s, cost: %s, %+v, %v", appName, groupID, time.Now().Sub(startTime).String(), *schOpts, err)
	if glog.V(4) {
		glog.Infof("Group detail: %s", utils.ObjJSON(&groupPlan))
	}
	return sdk.NewResponse(nil, err, nil)
}

func (c *GroupHandler) scaleGroup(request *restful.Request, response *restful.Response) restutil.Entity {
	var err error
	groupID := request.PathParameter(paramGroupID)
	appName := request.PathParameter(paramAppName)
	zoneName := request.QueryParameter(paramZoneName)

	var scaleSchedulePlan carbonv1.ScaleSchedulePlan
	if err = request.ReadEntity(&scaleSchedulePlan); err != nil {
		glog.Errorf("scaleGroup ReadEntity with error %v", err)
		return sdk.NewResponse(nil, err, nil)
	}
	if err = validateScalePlans(groupID, &scaleSchedulePlan); err != nil {
		glog.Errorf("scaleGroup validateScalePlans with error %v", err)
		return sdk.NewResponse(nil, err, nil)
	}
	err = c.groupService.ScaleGroup(appName, groupID, zoneName, &scaleSchedulePlan)
	glog.Infof("ScaleGroup: appName: %s, groupID: %s, zoneName: %s, %s, %v",
		appName, groupID, zoneName, utils.ObjJSON(scaleSchedulePlan), err)
	return sdk.NewResponse(nil, err, nil)
}

func (c *GroupHandler) disableScale(request *restful.Request, response *restful.Response) restutil.Entity {
	var err error
	groupID := request.PathParameter(paramGroupID)
	appName := request.PathParameter(paramAppName)
	zoneName := request.QueryParameter(paramZoneName)

	if groupID == "" {
		err := fmt.Errorf("groupid can not be empty")
		glog.Errorf("disableScale %v", err)
		return sdk.NewResponse(nil, err, nil)
	}
	err = c.groupService.DisableScale(appName, groupID, zoneName)
	glog.Infof("DisableScale: appName: %s, groupID: %s, zoneName: %s, %v",
		appName, groupID, zoneName, err)
	return sdk.NewResponse(nil, err, nil)
}

func (c *GroupHandler) reclaimWorkerNodes(request *restful.Request, response *restful.Response) restutil.Entity {
	var err error
	appName := request.PathParameter(paramAppName)
	req := service.ReclaimWorkerNodeRequest{}
	if err = request.ReadEntity(&req); err != nil {
		return sdk.NewResponse(nil, err, nil)
	}
	if 0 == len(req.WorkerNodes) {
		return sdk.NewResponse(nil, nil, nil)
	}
	err = c.groupService.ReclaimWorkerNodesOnEviction(appName, &req)
	glog.Infof("reclaim workernodes: %s, %s, %v", appName, utils.ObjJSON(&req), err)
	return sdk.NewResponse(nil, err, nil)
}

func (c *GroupHandler) writeTargetHistory(appName string, plans map[string]*typespec.GroupPlan) {
	for _, plan := range plans {
		c.diffLogger.WriteHistoryFile(historyRootPath, appName+"-"+plan.GroupID, "target", plan,
			func(prev, obj interface{}) (bool, error) {
				return !reflect.DeepEqual(prev, obj), nil
			},
		)
	}
}

func (c *GroupHandler) getSchOptions(request *restful.Request) (*service.SchOptions, error) {
	opts := request.Request.Header.Get("schOptions")
	schOptions, err := service.ParseSchOptions(opts)
	if nil != err {
		return nil, err
	}
	if opts == "" {
		if request.Request.Header.Get("Single") == "1" {
			schOptions.SchType = service.SchTypeRole
		} else {
			schOptions.SchType = service.SchTypeGroup
		}
	}
	return schOptions, nil
}

func validateGroupPlans(app string, plans map[string]*typespec.GroupPlan, schType string) error {
	if len(plans) > carbonv1.MaxGroupsPerApp {
		err := fmt.Errorf("%s, groups exceed limits: %d, %d", app, len(plans), carbonv1.MaxGroupsPerApp)
		if schType != service.SchTypeNode {
			return err
		}
	}
	for _, plan := range plans {
		if plan != nil && len(plan.GroupID) > carbonv1.MaxNameLength {
			err := fmt.Errorf("%s, group id exceed max name length: %s", app, plan.GroupID)
			if schType != service.SchTypeNode {
				return err
			}
		}
	}
	return nil
}

func validateScalePlans(groupID string, plan *carbonv1.ScaleSchedulePlan) error {
	if groupID == "" {
		err := fmt.Errorf("groupid can not be empty")
		return err
	}
	if plan.Replicas == nil || *plan.Replicas == 0 {
		err := fmt.Errorf("replicas can not be empty")
		return err
	}
	return nil
}
