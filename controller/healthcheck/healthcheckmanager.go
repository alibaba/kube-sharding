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
	"net/http"
	"sync"
	"time"

	"github.com/alibaba/kube-sharding/common"
	"github.com/alibaba/kube-sharding/controller/util"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	glog "k8s.io/klog"
)

var (
	healthcheckInterval = 5000
)

//Manager 结构体，负责管理多个rollingset的心跳任务
type Manager struct {
	// carbonclientset       *clientset.Interface
	// workernodeLister     *listers.WorkerNodeLister
	healthCheckTaskMap map[string]*Task
	lock               sync.Mutex
	//心跳http连接池， 心跳任务公用这个线程池
	httpClient       *common.HTTPClient
	workernodeLister listers.WorkerNodeLister
	executor         *utils.AsyncExecutor
	resourceManager  util.ResourceManager
}

//HealthCheckerInfo 保存HealthChecker的信息
type HealthCheckerInfo struct {
	healthChecker    *carbonv1.HealthCheckerConfig
	latestUpdateTime time.Time
	createdTime      time.Time
}

//NewHealthCheckManager 创建一个Manager
func NewHealthCheckManager(resourceManager util.ResourceManager,
	workernodeLister listers.WorkerNodeLister) Manager {
	healthCheckClient := &http.Client{
		Transport: &http.Transport{}}
	httpClient := common.NewWithHTTPClient(healthCheckClient)
	//总控协程池
	executor := utils.NewExecutor(20480)
	return Manager{
		resourceManager:    resourceManager,
		healthCheckTaskMap: make(map[string]*Task),
		httpClient:         httpClient,
		executor:           executor,
		workernodeLister:   workernodeLister,
	}
}

//Process key在healthCheckerChanMap中则更新任务， 否则创建新的task
func (hm *Manager) Process(key string, rollingSet *carbonv1.RollingSet) error {
	hm.lock.Lock()
	defer hm.lock.Unlock()
	task, ok := hm.healthCheckTaskMap[key]
	if ok {
		//更新
		if glog.V(4) {
			glog.Infof("healthcheck task already exist %s", key)
		}
		task.doUpdate(rollingSet)
		glog.V(4).Infof("healthcheck config update, key: %s, config: %v", key, utils.ObjJSON(rollingSet.Spec.HealthCheckerConfig))
	} else {
		glog.Infof("healthcheck startTask, key: %s, config: %v", key, utils.ObjJSON(rollingSet.Spec.HealthCheckerConfig))
		//新增，新启线程
		hm.startTask(key, rollingSet)
	}
	HealthCheckTaskCount.WithLabelValues().Set(float64(len(hm.healthCheckTaskMap)))
	return nil
}

func (hm *Manager) startTask(key string, rollingSet *carbonv1.RollingSet) {
	//判断rollingSet是否包含有效的selector参数
	if !validSelector(rollingSet) {
		glog.Errorf("unvalid selector :%s", utils.ObjJSON(rollingSet))
		return
	}
	//新增，新启线程
	now := time.Now()
	healthCheckerInfo := HealthCheckerInfo{rollingSet.Spec.HealthCheckerConfig, now, now}
	task := NewHealthCheckTask(&healthCheckerInfo, rollingSet, hm.resourceManager, hm.httpClient, hm.executor, hm.workernodeLister)
	task.doUpdate(rollingSet)
	hm.healthCheckTaskMap[key] = task
	go task.run(healthcheckInterval)
	glog.Infof("add healthChecker and run work, key=%s, map.size=%v", key, len(hm.healthCheckTaskMap))
}

//Close 关闭所有任务，并清空healthCheckerChanMap
func (hm *Manager) Close() error {
	hm.lock.Lock()
	defer hm.lock.Unlock()
	for key, v := range hm.healthCheckTaskMap {
		glog.Infof("close healthChecker task, key=%s", key)
		v.Stop()
		glog.Infof("delete healthChecker task from map, key=%s", key)
		delete(hm.healthCheckTaskMap, key)
	}
	HealthCheckTaskCount.WithLabelValues().Set(float64(len(hm.healthCheckTaskMap)))
	return nil
}

//移除被删除的healthChecker通道
func (hm *Manager) removeTask(key string) error {
	hm.lock.Lock()
	defer hm.lock.Unlock()
	if value, ok := hm.healthCheckTaskMap[key]; ok && value != nil {
		glog.Infof("close healthCheckerCh, key=%s", key)
		value.Stop()
		glog.Infof("delete healthChecker from map, key=%s", key)
		delete(hm.healthCheckTaskMap, key)
	}
	HealthCheckTaskCount.WithLabelValues().Set(float64(len(hm.healthCheckTaskMap)))
	return nil
}

//判断rollingSet是否包含有效的selector参数
func validSelector(rollingSet *carbonv1.RollingSet) bool {
	selector, err := metav1.LabelSelectorAsMap(rollingSet.Spec.Selector)
	if err != nil {
		return false
	}
	if selector[carbonv1.DefaultRollingsetUniqueLabelKey] == "" {
		return false
	}
	return true
}
