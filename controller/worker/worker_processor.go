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
	"fmt"
	"time"

	"k8s.io/kubernetes/pkg/apis/core"

	"github.com/alibaba/kube-sharding/common/features"
	"github.com/alibaba/kube-sharding/controller"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	glog "k8s.io/klog"

	"github.com/bluele/gcache"
	corev1 "k8s.io/api/core/v1"

	"github.com/alibaba/kube-sharding/controller/util"
	clientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
)

var (
	errResourceNotMatch = fmt.Errorf("resource not match")
	errProcessNotReady  = fmt.Errorf("process not ready")
)

// process single worker
type processor interface {
	process() (worker *carbonv1.WorkerNode, pod *corev1.Pod, err error)
}

func newDefaultProcessor(worker *carbonv1.WorkerNode, pod *corev1.Pod, c *Controller) processor {
	processor := newStateMachine(worker, pod, c)
	if nil == processor {
		return nil
	}
	return &defaultProcessor{
		worker:  worker,
		pod:     pod,
		machine: processor,
	}
}

type defaultProcessor struct {
	worker  *carbonv1.WorkerNode
	pod     *corev1.Pod
	machine workerStateMachine
}

func (p *defaultProcessor) process() (worker *carbonv1.WorkerNode, pod *corev1.Pod, err error) {
	defer reportWorkerLatencys(p.worker)
	err = p.machine.prepare()
	if nil != err {
		if glog.V(4) {
			glog.Infof("worker machine prepare error %s,%v", p.worker.Name, err)
		}
	}
	if p.worker != nil {
		if glog.V(4) {
			glog.Infof("process start worker status:%s", utils.ObjJSON(p.worker.Status))
		}
	}

	if carbonv1.IsWorkerUnAssigned(p.worker) {
		err = p.machine.dealwithUnAssigned()
		if nil != err {
			if glog.V(4) {
				glog.Infof("process worker unassigned %s,%v", p.worker.Name, err)
			}
		}
	}

	if carbonv1.IsWorkerAssigned(p.worker) {
		err = p.machine.dealwithAssigned()
		if nil != err {
			if glog.V(3) {
				glog.Infof("process worker assigned %s,%v", p.worker.Name, err)
			}
		}
	}

	if carbonv1.IsWorkerLost(p.worker) {
		err = p.machine.dealwithLost()
		if nil != err {
			if glog.V(3) {
				glog.Infof("process worker lost %s,%v", p.worker.Name, err)
			}
		}
	}

	if carbonv1.IsWorkerOfflining(p.worker) {
		err = p.machine.dealwithOfflining()
		if nil != err {
			if glog.V(3) {
				glog.Infof("process worker offlining %s,%v", p.worker.Name, err)
			}
		}
	}

	if carbonv1.IsWorkerReleasing(p.worker) {
		err = p.machine.dealwithReleasing()
		if nil != err {
			if glog.V(3) {
				glog.Infof("process worker releasing %s,%v", p.worker.Name, err)
			}
		}
	}

	if carbonv1.IsWorkerReleased(p.worker) {
		p.pod = nil
		err = p.machine.dealwithReleased()
		if nil != err {
			if glog.V(3) {
				glog.Infof("process worker released %s,%v", p.worker.Name, err)
			}
		}
	}
	if p.worker != nil {
		if glog.V(4) {
			glog.Infof("process end worker status:%s", utils.ObjJSON(p.worker.Status))
		}
	}
	if err = p.machine.postSync(); err != nil && glog.V(3) {
		glog.Errorf("post sync error %s,%v", p.worker.Name, err)
	}
	return p.worker, p.pod, err
}

// worker state machine
type workerStateMachine interface {
	prepare() error
	dealwithReleased() error
	dealwithReleasing() error
	dealwithOfflining() error
	dealwithLost() error
	dealwithUnAssigned() error
	dealwithAssigned() error
	postSync() error
}

type workerUpdater interface {
	processPlan(worker *carbonv1.WorkerNode) (bool, error)
	processHealthInfo(worker *carbonv1.WorkerNode) bool
	processServiceInfo(worker *carbonv1.WorkerNode) bool
	processUpdateGracefully(worker *carbonv1.WorkerNode) bool
}

func newStateMachine(worker *carbonv1.WorkerNode, pod *corev1.Pod, c *Controller) workerStateMachine {
	var allocator, err = createAllocator(worker, pod, c)
	if nil != err || nil == allocator {
		glog.Errorf("create alloctor %s, %v", worker.Name, err)
		return nil
	}
	machine := &defaultStateMachine{
		worker:    worker,
		pod:       pod,
		allocator: allocator,
		c:         c,
	}
	machine.u = machine
	return machine
}

type defaultStateMachine struct {
	worker    *carbonv1.WorkerNode
	pod       *corev1.Pod
	allocator workerAllocator
	u         workerUpdater
	c         *Controller
}

func (m *defaultStateMachine) prepare() error {
	err := m.allocator.syncWorkerAllocStatus()
	carbonv1.InitWorkerNode(m.worker)
	carbonv1.SyncWorkerInUpdating(m.worker)
	return err
}

func (m *defaultStateMachine) dealwithReleased() error {
	logK := m.getProcessLogKey()
	diffLogger.Log(logK, "processStep change to released, %s", m.worker.String())
	return nil
}

func (m *defaultStateMachine) dealwithReleasing() error {
	logK := m.getProcessLogKey()
	diffLogger.Log(logK, "processStep change to releasing")
	if nil != m.pod {
		err := m.allocator.delete()
		if nil != err {
			glog.Warningf("delete pod error %s,%v", m.pod.Name, err)
			return err
		}
		m.pod = nil
		carbonv1.SetWorkerEntityAlloced(m.worker, false)
		carbonv1.SetWorkerEntity(m.worker, "", "")
	}
	if carbonv1.IsWorkerEmpty(m.worker) {
		carbonv1.SetWorkerReleased(m.worker)
	}
	return nil
}

func (m *defaultStateMachine) dealwithOfflining() error {
	logK := m.getProcessLogKey()
	diffLogger.Log(logK, "processStep change to offlining")
	m.worker.Status.ServiceOffline = true
	carbonv1.SyncWorkerServiceHealthStatus(m.worker)
	if carbonv1.IsWorkerOfflined(m.worker) || carbonv1.IsWorkerIrrecoverable(m.worker) {
		carbonv1.SetWorkerReleasing(m.worker)
	}
	return nil
}

func (m *defaultStateMachine) dealwithLost() error {
	if carbonv1.IsWorkerToRelease(m.worker) {
		m.tryToReleaseWorker(m.worker)
	}
	return nil
}

func (m *defaultStateMachine) dealwithUnAssigned() error { // 同步service 避免直接available
	err := m.initServiceConditions(m.worker)
	if nil != err {
		return err
	}

	if carbonv1.IsWorkerEntityAlloced(m.worker) {
		if glog.V(4) {
			glog.Infof("workerNode %s assign slot, createTime %s, cost %s",
				m.worker.Name, m.worker.CreationTimestamp.String(), time.Since(m.worker.CreationTimestamp.Time).String())
		}
		if features.C2MutableFeatureGate.Enabled(features.WarmupNewStartingPod) {
			m.worker.Status.NeedWarmup = carbonv1.NeedWarmup(m.worker)
		}
		if carbonv1.IsBackupWorker(m.worker) && carbonv1.BackupOfPodNotEmpty(m.worker) {
			m.c.Eventf(m.pod, core.EventTypeNormal, "BackupPodOf", "Assigned backup pod of: %s, uid: %s.", m.worker.Spec.BackupOfPod.Name, m.worker.Spec.BackupOfPod.Uid)
		}
		m.worker.Status.InRestarting = true
		carbonv1.SetWorkerAssigned(m.worker)
	} else if carbonv1.IsWorkerEmpty(m.worker) && carbonv1.IsWorkerToRelease(m.worker) {
		if nil != m.pod {
			err := m.allocator.delete()
			if nil != err {
				glog.Warningf("delete pod error %s,%v", m.pod.Name, err)
				return err
			}
			m.pod = nil
		}
		carbonv1.SetWorkerReleased(m.worker)
	} else if !carbonv1.IsWorkerEmpty(m.worker) && carbonv1.IsWorkerToRelease(m.worker) &&
		!carbonv1.IsWorkerEntityAlloced(m.worker) {
		m.tryToReleaseWorker(m.worker)
	} else if m.needAllocate() {
		err = m.allocator.allocate()
		if nil != err {
			glog.Errorf("create pod error: %s, %v", utils.ObjJSON(m.worker), err)
			carbonv1.SetUnassignedReason(m.worker, carbonv1.UnassignedReasonCreatePodErr)
			key := m.worker.Namespace + "/" + m.worker.Name
			m.c.AddRateLimited(key)
		}
		return nil
	} else if m.pod != nil {
		reason := carbonv1.GetUnassignedResson(m.pod)
		carbonv1.SetUnassignedReason(m.worker, reason)
	}
	if nil != m.pod && carbonv1.IsWorkerUnAssigned(m.worker) && !carbonv1.IsWorkerToRelease(m.worker) {
		m.allocator.syncPodSpec()
	}
	return nil
}

func (m *defaultStateMachine) needAllocate() bool {
	if carbonv1.IsWorkerEmpty(m.worker) && !carbonv1.IsWorkerToRelease(m.worker) {
		if carbonv1.WorkerModeTypeCold == m.worker.Spec.WorkerMode {
		} else {
			if nil == m.pod {
				return true
			}
		}
	}
	return false
}

func (m *defaultStateMachine) getProcessLogKey() string {
	return diffLogKey(m.worker.Name, "ProcessStep")
}

func (m *defaultStateMachine) dealwithAssigned() error {
	carbonv1.SetUnassignedReason(m.worker, carbonv1.UnassignedReasonNone)
	defer func() {
		complete := carbonv1.IsWorkerComplete(m.worker)
		diffLogger.Log(diffLogKey(m.worker.Name, "complete"), ": %v", complete)
		controller.RecordCompleteLatency(&m.worker.ObjectMeta, m.worker.Kind,
			controllerAgentName, m.worker.Spec.Version, m.worker.Status.Complete)
		m.worker.Status.Complete = complete
		carbonv1.SyncWorkerInUpdating(m.worker)
	}()
	logK := m.getProcessLogKey()
	if carbonv1.IsWorkerToRelease(m.worker) {
		m.tryToReleaseWorker(m.worker)
		m.worker.Status.ProcessStep = carbonv1.StepToRelease
		diffLogger.Log(logK, "processStep change to "+string(carbonv1.StepToRelease))
		return nil
	}
	if !carbonv1.IsWorkerEntityAlloced(m.worker) {
		carbonv1.SetWorkerLost(m.worker)
		m.worker.Status.PodReady = false
		m.worker.Status.ProcessStep = carbonv1.StepLost
		diffLogger.Log(logK, "processStep change to "+string(carbonv1.StepLost))
		return nil
	}
	if m.pod != nil && m.pod.Spec.NodeName != "" {
		carbonv1.UnsetStandby2Active(m.worker) //#538079
	}
	syncRrResourceUpdateState(m.worker)

	m.worker.Status.IsFedWorker = false
	if m.pod != nil && m.pod.Labels != nil &&
		m.pod.Labels["fed.alibaba.com/local-cluster"] != "" && m.pod.Labels["fed.alibaba.com/remote-cluster"] != "" &&
		m.pod.Labels["fed.alibaba.com/local-cluster"] != m.pod.Labels["fed.alibaba.com/remote-cluster"] {
		m.worker.Status.IsFedWorker = true
	}

	m.worker.Status.ProcessStep = carbonv1.StepBegin

processUpdateGracefully:
	if !m.u.processUpdateGracefully(m.worker) {
		diffLogger.Log(logK, "processUpdateGracefully failed")
		return nil
	}
	m.worker.Status.ProcessStep = carbonv1.StepProcessUpdateGracefully

	ok, err := m.u.processPlan(m.worker)
	if nil != err {
		diffLogger.Log(logK, "processPlan error: %v", err)
		return err
	}
	if !ok {
		diffLogger.Log(logK, "processPlan failed")
		if m.offlineForRestart(m.worker) {
			glog.V(5).Infof("retry processUpdateGracefully before restart %s", m.worker.Name)
			goto processUpdateGracefully
		}
		return nil
	}
	m.worker.Status.ProcessStep = carbonv1.StepProcessPlan

	if !m.u.processHealthInfo(m.worker) {
		diffLogger.Log(logK, "processHealthInfo failed")
		return nil
	}
	m.worker.Status.ProcessStep = carbonv1.StepProcessHealthInfo

	if !m.u.processServiceInfo(m.worker) {
		diffLogger.Log(logK, "processServiceInfo failed")
		return nil
	}
	m.worker.Status.ProcessStep = carbonv1.StepProcessServiceInfo
	m.worker.Status.InRestarting = false
	return nil
}

func (m *defaultStateMachine) processPlan(worker *carbonv1.WorkerNode) (bool, error) {
	if carbonv1.IsWorkerRollback(worker) {
		glog.Warningf("worker:%s version rollback, %s -> %s", worker.Name, worker.Status.Version, worker.Spec.Version)
		worker.Status.Version = ""
		return false, nil
	}
	if carbonv1.IsWorkerInnerVerMatch(worker) {
		glog.Infof("worker:%s version inner match, %s -> %s", worker.Name, worker.Status.Version, worker.Spec.Version)
		worker.Status.Version = worker.Spec.Version
		return true, nil
	}
	err := m.allocator.syncPodSpec()
	if nil != err {
		glog.Warningf("sync pod spec error :%s,%v", m.worker.Name, err)
		return false, err
	}
	if m.offlineForRestart(worker) {
		glog.Warningf("offline for restart :%s", m.worker.Name)
		return false, nil
	}
	err = m.allocator.syncWorkerStatus()
	if nil != err {
		glog.Warningf("sync worker status error :%s,%v", m.worker.Name, err)
		return false, err
	}
	if carbonv1.IsWorkerVersionMisMatch(worker) {
		glog.Infof("version change: %s,%s,%s", worker.Name, worker.Spec.Version, worker.Status.Version)
		return false, nil
	}
	if carbonv1.IsWorkerModeMisMatch(worker) {
		glog.Infof("workermode change: %s,%s,%s", worker.Name, worker.Spec.WorkerMode, worker.Status.WorkerMode)
		return false, nil
	}
	if !carbonv1.IsWorkerDependencyReady(worker) {
		glog.Infof("worker dependency ready: %s,%v", worker.Name, worker.Spec.DependencyReady)
		return false, nil
	}
	return true, nil
}

func (m *defaultStateMachine) postSync() error {
	return m.allocator.syncPodMetas()
}

func syncSingleHealthInfo(worker *carbonv1.WorkerNode) {
	if len(worker.OwnerReferences) == 0 ||
		worker.Labels[carbonv1.GroupTypeKey] == carbonv1.GroupTypeReplica {
		worker.Status.HealthCondition.Status = carbonv1.GetHealthStatusByPhase(worker)
		worker.Status.HealthCondition.Version = worker.Status.Version
		worker.Status.HealthStatus = worker.Status.HealthCondition.Status
	}
}

func (m *defaultStateMachine) processHealthInfo(worker *carbonv1.WorkerNode) bool {
	syncSingleHealthInfo(worker)
	if carbonv1.IsWorkerHealthInfoReady(worker) && carbonv1.IsWorkerDependencyReady(worker) {
		worker.Status.WorkerReady = true
		worker.Status.LastWorkerNotReadytime = 0
		return true
	}
	if !carbonv1.IsWorkerHealthInfoReady(worker) {
		worker.Status.WorkerReady = false
		if worker.Status.LastWorkerNotReadytime == 0 {
			worker.Status.LastWorkerNotReadytime = time.Now().Unix()
		}
		m.processNeedWarmUp(worker)
	}
	return false
}

func (m *defaultStateMachine) processServiceInfo(worker *carbonv1.WorkerNode) bool {
	if carbonv1.IsWorkerGracefully(worker) || shouldGracefullyRestart(worker) || worker.Status.NeedWarmup || worker.Status.Warmup {
		logK := diffLogKey(worker.Name, "warmup")
		if worker.Status.NeedWarmup {
			if !carbonv1.IsWorkerRowComplete(worker) && features.C2MutableFeatureGate.Enabled(features.WarmupUnavailablePod) {
				diffLogger.Log(logK, "wait row complete to warmup")
				return false
			}
			worker.Status.Warmup = true
			worker.Status.NeedWarmup = false
			diffLogger.Log(logK, "start warmup ungracefully")
		}
		if worker.Status.ServiceOffline != !carbonv1.IsWorkerOnline(worker) {
			worker.Status.InRestarting = false
			worker.Status.ServiceOffline = !carbonv1.IsWorkerOnline(worker)
		}
		if !worker.Status.ServiceOffline && worker.Status.Warmup && worker.Status.LastWarmupStartTime != 0 {
			if time.Now().Unix()-worker.Status.LastWarmupStartTime > int64(worker.Spec.WarmupSeconds) {
				worker.Status.Warmup = false
				worker.Status.InWarmup = false
				worker.Status.LastWarmupStartTime = 0
				worker.Status.LastWarmupEndTime = time.Now().Unix()
				diffLogger.Log(logK, "end warmup")
			}
		}
	}

	carbonv1.SyncWorkerServiceHealthStatus(worker)
	return carbonv1.IsWorkerServiceMatch(worker)
}

func (m *defaultStateMachine) processUpdateGracefully(worker *carbonv1.WorkerNode) bool {
	if m.shouldUpdateGracefully(worker) {
		if carbonv1.IsWorkerVersionMisMatch(worker) || carbonv1.IsWorkerModeMisMatch(worker) {
			worker.Status.ServiceOffline = true
			carbonv1.SyncWorkerServiceHealthStatus(worker)
			return carbonv1.IsWorkerUnpublished(worker)
		}
	} else {
		worker.Status.ServiceOffline = !carbonv1.IsWorkerOnline(worker)
	}
	m.processNeedWarmUp(worker)
	return true
}

func (m *defaultStateMachine) processNeedWarmUp(worker *carbonv1.WorkerNode) {
	if features.C2MutableFeatureGate.Enabled(features.WarmupUnavailablePod) && !worker.Status.Warmup {
		if worker.Status.InUpdating && carbonv1.IsWorkerUnserviceable(worker) && !carbonv1.IsWorkerHealthInfoReady(worker) {
			glog.V(3).Infof("need warmup: %s,%s,%s", worker.Name, worker.Spec.Version, worker.Status.Version)
			worker.Status.NeedWarmup = carbonv1.NeedWarmup(worker)
		}
	}
	if worker.Status.NeedWarmup {
		if !carbonv1.IsWorkerGracefully(worker) {
			glog.V(3).Infof("need warmup: %s,%s,%s, set service offline", worker.Name, worker.Spec.Version, worker.Status.Version)
			worker.Status.ServiceOffline = true
		}
	}
}

func shouldGracefullyRestart(worker *carbonv1.WorkerNode) bool {
	if features.C2MutableFeatureGate.Enabled(features.UnPubRestartingNode) ||
		(features.C2MutableFeatureGate.Enabled(features.WarmupNewStartingPod) && carbonv1.NeedWarmup(worker)) {
		return true
	}
	return false
}

func (m *defaultStateMachine) offlineForRestart(worker *carbonv1.WorkerNode) bool {
	if m.needOfflineForRestart(worker) && !worker.Status.ServiceOffline &&
		(carbonv1.IsWorkerVersionMisMatch(m.worker) || carbonv1.IsWorkerModeMisMatch(m.worker)) {
		return true
	}
	return false
}

func (m *defaultStateMachine) needOfflineForRestart(worker *carbonv1.WorkerNode) bool {
	if shouldGracefullyRestart(worker) {
		if worker.Status.InRestarting {
			return true
		}
	}
	return false
}

func (m *defaultStateMachine) shouldUpdateGracefully(worker *carbonv1.WorkerNode) bool {
	if carbonv1.IsWorkerGracefully(worker) {
		return true
	}
	if m.needOfflineForRestart(worker) {
		return true
	}
	return false
}

func (m *defaultStateMachine) tryToReleaseWorker(worker *carbonv1.WorkerNode) {
	if carbonv1.IsWorkerGracefully(worker) {
		carbonv1.SetWorkerOfflining(worker)
	} else {
		carbonv1.SetWorkerReleasing(worker)
	}
}

func (m *defaultStateMachine) initServiceConditions(worker *carbonv1.WorkerNode) error {
	// 同步service 避免直接available
	if carbonv1.IsWorkerToRelease(worker) {
		return nil
	}
	if nil == m.c || nil == m.c.carbonclientset {
		glog.Infof("nil carbonclientset for initServiceConditions")
		return nil
	}
	services, err := getWorkerServices(m.c.rollingSetLister, m.c.carbonclientset, m.c.serviceLister, worker)
	if nil != err {
		glog.Warningf("GetWorkerServices error :%s,%v", worker.Name, err)
		return err
	}
	if len(services) != 0 && len(worker.Status.ServiceConditions) == 0 {
		for _, service := range services {
			if service.Spec.SoftDelete {
				continue
			}
			carbonv1.SetWorkerServiceHealthCondition(
				worker, service, "", corev1.ConditionFalse, 0,
				carbonv1.InitServiceFalseState, "", false, false)
		}
	}
	return nil
}

const (
	lruSize    = 1024
	lruTimeout = 10 * time.Second
)

var serviceCache = gcache.New(lruSize).LRU().Build()

func getWorkerServices(rslister listers.RollingSetLister, clientset clientset.Interface,
	lister listers.ServicePublisherLister, worker *carbonv1.WorkerNode) ([]*carbonv1.ServicePublisher, error) {
	rsID := carbonv1.GetRsID(worker)
	if "" == rsID {
		return nil, nil
	}
	if carbonv1.GetGangID(worker) == "" {
		cachedServices, err := serviceCache.Get(rsID)
		if nil == err && nil != cachedServices {
			if glog.V(4) {
				glog.Infof("get service form lru %v,%v", cachedServices, err)
			}
			return cachedServices.([]*carbonv1.ServicePublisher), nil
		}
	}

	services, err := util.GetWorkerServices(lister, worker)
	if nil != err {
		glog.Warningf("GetWorkerServices error :%s,%v", worker.Name, err)
		return nil, err
	}
	if carbonv1.GetGangID(worker) == "" {
		serviceCache.SetWithExpire(rsID, services, lruTimeout)
	}
	return services, err
}
