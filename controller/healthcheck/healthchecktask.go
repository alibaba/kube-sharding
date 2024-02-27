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
	"context"
	"reflect"
	"strconv"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/alibaba/kube-sharding/common"
	"github.com/alibaba/kube-sharding/controller/util"
	"github.com/alibaba/kube-sharding/pkg/ago-util/monitor/metricgc"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	glog "k8s.io/klog"
)

// BatchCheckTimeOut 批量执行check超时时间
var BatchCheckTimeOut = time.Second * 5

// DefaultLostCountThreshold Lost阈值
var DefaultLostCountThreshold = int32(5)

// DefaultLostTimeout Lost超时时间
var DefaultLostTimeout = int64(300)

//Helper 结构体， 传入给healthchecker使用
type Helper interface {
	Sync(workernode *carbonv1.WorkerNode, healthCondition *carbonv1.HealthCondition) bool
	GetWorkerNode(workernode *carbonv1.WorkerNode) (*carbonv1.WorkerNode, error)
	GetHTTPClient() *common.HTTPClient
}

//TaskCondition 保存Task任务的执行状态
type TaskCondition struct {
	taskID              string
	lastUpdateTime      time.Time
	lastProcessTime     time.Time
	lastWorkerNodeCount int32
	count               int64
}

//NewTaskCondition 创建一个NewTaskCondition
func NewTaskCondition(rsName string) *TaskCondition {
	c := TaskCondition{taskID: getTaskID(rsName), count: 1}
	return &c
}

func (tc *TaskCondition) getTaskProccessID() string {
	return tc.taskID + "-" + strconv.FormatInt(tc.count, 10)
}

//Task 结构体，管理一个rollingset的心跳
type Task struct {
	healthCheckerInfo *HealthCheckerInfo

	httpClient      *common.HTTPClient
	executor        *utils.AsyncExecutor
	resourceManager util.ResourceManager

	rollingSet *carbonv1.RollingSet

	//本线程中的副本，chan接收到的值更新到此
	healthCheckerConfig *carbonv1.HealthCheckerConfig
	workernodeLister    listers.WorkerNodeLister
	healthCheckerRunner HealthChecker
	taskCondition       *TaskCondition
	utils.LoopJob
	lock sync.RWMutex
}

//NewHealthCheckTask 创建一个Task
func NewHealthCheckTask(
	healthCheckerInfo *HealthCheckerInfo,
	rollingSet *carbonv1.RollingSet,
	resourceManager util.ResourceManager,
	httpClient *common.HTTPClient,
	executor *utils.AsyncExecutor,
	workernodeLister listers.WorkerNodeLister) *Task {
	return &Task{
		healthCheckerInfo: healthCheckerInfo,
		rollingSet:        rollingSet,
		resourceManager:   resourceManager,
		httpClient:        httpClient,
		executor:          executor,
		taskCondition:     NewTaskCondition(rollingSet.Name),
		workernodeLister:  workernodeLister,
	}
}

func (ht *Task) run(interval int) error {
	glog.Infof("start healthcheck task %s", utils.ObjJSON(ht.healthCheckerInfo))
	ht.LoopJob.Start()
	go func() {
		defer ht.MarkStop()
		for !ht.CheckStop(interval) {
			if err := ht.processBatch(); err != nil {
				glog.Errorf("Health check task run failed: %v", err)
			}
		}
	}()

	return nil
}

func (ht *Task) processBatch() error {
	ht.lock.RLock()
	defer ht.lock.RUnlock()

	ht.taskCondition.lastProcessTime = time.Now()
	selector, err := metav1.LabelSelectorAsMap(ht.rollingSet.Spec.Selector)
	if err != nil {
		return err
	}

	workernodes, err := ht.resourceManager.ListWorkerNodeForRS(selector)
	if err != nil || len(workernodes) == 0 {
		return err
	}
	workerCount := len(workernodes)
	ht.taskCondition.lastWorkerNodeCount = int32(workerCount)
	taskProccessID := ht.taskCondition.getTaskProccessID()

	ctx, cancel := context.WithTimeout(context.Background(), common.GetBatcherTimeout(BatchCheckTimeOut))
	defer cancel()
	batcher := utils.NewBatcherWithExecutor(ht.executor)
	commands := make([]*utils.Command, 0, workerCount)
	for _, workernode := range workernodes {
		w := workernode
		f1 := func() {
			ht.healthCheckerRunner.doCheck(w, ht.healthCheckerConfig, ht.rollingSet)
		}
		command, err := batcher.Go(ctx, false, f1)
		HealthCheckCounter.WithLabelValues(
			carbonv1.GetObjectBaseScopes(ht.rollingSet)...,
		).Inc()
		if err != nil {
			glog.Warningf("[%s] executor.Queue err: %v", taskProccessID, err)
		}
		commands = append(commands, command)
	}
	batcher.Wait()
	finishedCount := 0
	for _, command := range commands {
		if command != nil && command.ExecutorError == nil {
			finishedCount++
		}
	}
	elapsed := time.Since(ht.taskCondition.lastProcessTime)
	costMs := float64(elapsed.Nanoseconds() / 1000000)
	if glog.V(4) {
		glog.Infof("[%s] healthcheck proccessBatch, rs name:%v, selector:%v, worker count: %v, finished count: %v,  elapsed: %vms", taskProccessID, ht.rollingSet.Name, selector, workerCount, finishedCount, costMs)
	}
	metricgc.WithLabelValuesSummary(HealthCheckTaskLatency,
		carbonv1.GetClusterName(ht.rollingSet),
		carbonv1.GetAppName(ht.rollingSet),
		ht.rollingSet.Name).Observe(costMs)
	ht.taskCondition.count++
	return nil
}

func (ht *Task) doUpdate(rollingSet *carbonv1.RollingSet) {
	ht.decompressVersionPlan(rollingSet)
	ht.lock.Lock()
	defer ht.lock.Unlock()
	ht.rollingSet = rollingSet
	if ht.healthCheckerInfo == nil {
		ht.healthCheckerInfo = &HealthCheckerInfo{}
	}
	ht.healthCheckerInfo.healthChecker = rollingSet.Spec.HealthCheckerConfig
	ht.healthCheckerInfo.latestUpdateTime = time.Now()
	ht.doUpdateHealthCheckerConfig(rollingSet.Spec.HealthCheckerConfig)
}

func (ht *Task) decompressVersionPlan(rollingSet *carbonv1.RollingSet) {
	versionPlan := &rollingSet.Spec.VersionPlan
	if versionPlan.CustomInfo != "" {
		versionPlan.CompressedCustomInfo = ""
	}
	if versionPlan.CompressedCustomInfo != "" {
		customInfo, err := utils.DecompressStringToString(versionPlan.CompressedCustomInfo)
		if err != nil || customInfo == "" {
			glog.Errorf("get finalVersionPlan decompress  %s, %v", rollingSet.Name, err)
			return
		}
		versionPlan.CustomInfo = customInfo
		versionPlan.CompressedCustomInfo = ""
	}
}

func (ht *Task) doUpdateHealthCheckerConfig(healthCheckerConfig *carbonv1.HealthCheckerConfig) {
	ht.taskCondition.lastUpdateTime = time.Now()
	ht.healthCheckerConfig = healthCheckerConfig

	if ht.healthCheckerConfig != nil && ht.healthCheckerConfig.Type != "" &&
		ht.healthCheckerConfig.Type != carbonv1.DefaultHealth {
		//修正LostCountThreshold， LostTimeout
		if ht.healthCheckerConfig.Lv7Config.LostCountThreshold <= 0 {
			ht.healthCheckerConfig.Lv7Config.LostCountThreshold = DefaultLostCountThreshold
		}
		if ht.healthCheckerConfig.Lv7Config.LostTimeout <= 0 {
			ht.healthCheckerConfig.Lv7Config.LostTimeout = DefaultLostTimeout
		}

		if ht.healthCheckerConfig.Type == carbonv1.Lv7Health {
			ht.healthCheckerRunner = NewLv7HealthChecker(ht, ht.rollingSet)
			glog.V(4).Infof("[%s] update healthCheckerRunner, type: %v, config: %v", ht.taskCondition.taskID, ht.healthCheckerRunner.getTypeName(), utils.ObjJSON(ht.healthCheckerConfig))
			return
		} else if ht.healthCheckerConfig.Type == carbonv1.AdvancedLv7Health {
			ht.healthCheckerRunner = NewAdvancedLv7HealthChecker(ht, ht.rollingSet)
			glog.V(4).Infof("[%s] update healthCheckerRunner, type: %v, config: %v", ht.taskCondition.taskID, ht.healthCheckerRunner.getTypeName(), utils.ObjJSON(ht.healthCheckerConfig))
			return
		}
	}
	ht.healthCheckerRunner = NewDefaultHealthChecker(ht)
	glog.V(4).Infof("[%s] update healthCheckerRunner, type: %v", ht.taskCondition.taskID, ht.healthCheckerRunner.getTypeName())
}

//Sync 同步workernode对象状态
func (ht *Task) Sync(workernode *carbonv1.WorkerNode, healthCondition *carbonv1.HealthCondition) bool {
	worker, _ := ht.GetWorkerNode(workernode)
	if nil != worker {
		workernode = worker
	}
	workernodeCopy := workernode.DeepCopy()

	carbonv1.SetWorkerHealthStatus(workernodeCopy, healthCondition)
	var err error
	if !reflect.DeepEqual(workernodeCopy.Status, workernode.Status) {
		err = ht.resourceManager.UpdateWorkerStatus(workernodeCopy)
	}
	return err == nil
}

// GetWorkerNode GetWorkerNode
func (ht *Task) GetWorkerNode(workernode *carbonv1.WorkerNode) (*carbonv1.WorkerNode, error) {
	if nil == ht || nil == ht.workernodeLister || nil == workernode {
		return nil, nil
	}
	worker, err := ht.workernodeLister.WorkerNodes(workernode.Namespace).Get(workernode.Name)
	return worker, err
}

// GetHTTPClient 返回httpclient对象
func (ht *Task) GetHTTPClient() *common.HTTPClient {
	return ht.httpClient
}

//IsEquals 比较两个healthCondition是否相同
func IsEquals(healthCondition1 *carbonv1.HealthCondition, healthCondition2 *carbonv1.HealthCondition) bool {
	if healthCondition1 == nil {
		return healthCondition2 == nil
	}
	return reflect.DeepEqual(healthCondition1, healthCondition2)
}

//生成一个调度任务的id
func getTaskID(namespace string) string {
	id := "t-" + namespace + "-" + common.GetCommonID()
	return id
}
