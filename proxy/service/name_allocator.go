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
	"strconv"

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	"github.com/alibaba/kube-sharding/transfer"
	"k8s.io/apimachinery/pkg/api/errors"
	glog "k8s.io/klog"
)

type nameAllocator struct {
	groupLister      listers.ShardGroupLister
	rollingsetLister listers.RollingSetLister
}

func (c *nameAllocator) allocateShardGroupName(appName, namespace, groupID string) (string, error) {
	shardGroupName := transfer.SubGroupName(groupID)
	oldShardGroup, err := c.groupLister.ShardGroups(namespace).Get(shardGroupName)
	if err != nil {
		if errors.IsNotFound(err) {
			selector, err := newHippoKeySelector("", appName, groupID, "", nil, nil)
			appGroups, err := c.groupLister.ShardGroups(namespace).List(selector)
			if err != nil {
				return "", err
			}
			if len(appGroups) > 0 {
				return appGroups[0].Name, nil
			}
			return shardGroupName, nil
		}
		return "", err
	}
	if carbonv1.GetAppName(oldShardGroup) == appName &&
		carbonv1.GetGroupName(oldShardGroup) == groupID {
		return shardGroupName, nil
	}

	selector, err := newHippoKeySelector("", appName, groupID, "", nil, nil)
	appGroups, err := c.groupLister.ShardGroups(namespace).List(selector)
	if err != nil {
		return "", err
	}
	if len(appGroups) > 0 {
		return appGroups[0].Name, nil
	}

	//走重名
	for i := 0; i < 100; i++ {
		newShardGroupName := shardGroupName + "-" + strconv.Itoa(i)
		oldShardGroup, err = c.groupLister.ShardGroups(namespace).Get(newShardGroupName)
		if nil != err {
			if errors.IsNotFound(err) {
				glog.Infof("GetShardGroupName rename:%s, app:%s, groupID:%s", newShardGroupName, appName, groupID)
				return newShardGroupName, nil
			}
			return "", err
		}
	}
	return "", fmt.Errorf("getShardGroupName conflict")
}

func (c *nameAllocator) allocateRollingSetName(appName, namespace, groupID string) (string, error) {
	rollingSetName, err := transfer.GenerateRollingsetName(appName, groupID, groupID, transfer.SchTypeRole)
	if nil != err {
		return "", err
	}
	hippoRoleName := transfer.GenerateHippoRoleName(groupID, groupID)
	oldRollingset, err := c.rollingsetLister.RollingSets(namespace).Get(rollingSetName)
	if err != nil {
		if errors.IsNotFound(err) {
			selector, err := newHippoKeySelector("", appName, "", hippoRoleName, nil, nil)
			rollingSets, err := c.rollingsetLister.RollingSets(namespace).List(selector)
			if err != nil {
				return "", err
			}
			if len(rollingSets) > 0 {
				return rollingSets[0].Name, nil
			}
			return rollingSetName, nil
		}
		return "", err
	}
	if carbonv1.GetAppName(oldRollingset) == appName &&
		carbonv1.GetGroupName(oldRollingset) == groupID {
		return rollingSetName, nil
	}

	selector, err := newHippoKeySelector("", appName, "", hippoRoleName, nil, nil)
	rollingSets, err := c.rollingsetLister.RollingSets(namespace).List(selector)
	if err != nil {
		return "", err
	}
	if len(rollingSets) > 0 {
		return rollingSets[0].Name, nil
	}

	//走重名
	for i := 0; i < 100; i++ {
		newRollingSetName := rollingSetName + "-" + strconv.Itoa(i)
		oldRollingset, err = c.rollingsetLister.RollingSets(namespace).Get(newRollingSetName)
		if nil != err {
			if errors.IsNotFound(err) {
				glog.Infof("allocateRollingSetName rename:%s, app:%s, groupID:%s", newRollingSetName, appName, hippoRoleName)
				return newRollingSetName, nil
			}
			return "", err
		}
	}
	return "", fmt.Errorf("allocateRollingSetName conflict")

}

func (c *nameAllocator) allocateGangRollingSetName(appName, namespace, groupID string) (string, error) {
	rollingSetName, err := transfer.GenerateRollingsetName(appName, groupID, groupID, transfer.SchTypeRole)
	if nil != err {
		return "", err
	}
	oldRollingset, err := c.rollingsetLister.RollingSets(namespace).Get(rollingSetName)
	if err != nil {
		if errors.IsNotFound(err) {
			selector, err := newHippoKeySelector("", appName, groupID, "", nil, nil)
			rollingSets, err := c.rollingsetLister.RollingSets(namespace).List(selector)
			if err != nil {
				return "", err
			}
			if len(rollingSets) > 0 {
				return rollingSets[0].Name, nil
			}
			return rollingSetName, nil
		}
		return "", err
	}
	if carbonv1.GetAppName(oldRollingset) == appName &&
		carbonv1.GetGroupName(oldRollingset) == groupID {
		return rollingSetName, nil
	}

	selector, err := newHippoKeySelector("", appName, groupID, "", nil, nil)
	rollingSets, err := c.rollingsetLister.RollingSets(namespace).List(selector)
	if err != nil {
		return "", err
	}
	if len(rollingSets) > 0 {
		return rollingSets[0].Name, nil
	}

	//走重名
	for i := 0; i < 100; i++ {
		newRollingSetName := rollingSetName + "-" + strconv.Itoa(i)
		oldRollingset, err = c.rollingsetLister.RollingSets(namespace).Get(newRollingSetName)
		if nil != err {
			if errors.IsNotFound(err) {
				glog.Infof("allocateRollingSetName rename:%s, app:%s, groupID:%s", newRollingSetName, appName, groupID)
				return newRollingSetName, nil
			}
			return "", err
		}
	}
	return "", fmt.Errorf("allocateRollingSetName conflict")

}
