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
	reflect "reflect"

	"github.com/alibaba/kube-sharding/controller/util"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"k8s.io/apimachinery/pkg/api/errors"

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	corev1 "k8s.io/api/core/v1"
	glog "k8s.io/klog"
)

type executor interface {
	createPod(pod *corev1.Pod) (*corev1.Pod, error)
	updatePod(worker *carbonv1.WorkerNode, before, after *corev1.Pod) (*corev1.Pod, error)
	patchPod(pod *corev1.Pod, patch *util.CommonPatch) error
	deletePod(pod *corev1.Pod, preUpdate bool) error
}

type baseExecutor struct {
	grace      bool
	controller *Controller
}

func (e *baseExecutor) deletePod(pod *corev1.Pod, preUpdate bool) error {
	var hasC2Finalizer = false
	var newFinalizers = []string{}
	for i := range pod.Finalizers {
		if pod.Finalizers[i] == C2DeleteProtectionFinalizer {
			hasC2Finalizer = true
			continue
		}
		newFinalizers = append(newFinalizers, pod.Finalizers[i])
	}
	if hasC2Finalizer {
		pod.Finalizers = newFinalizers
	}
	if preUpdate || hasC2Finalizer {
		err := e.controller.ResourceManager.UpdatePodSpec(pod)
		if nil != err && !errors.IsNotFound(err) {
			glog.Errorf("update pod %s,%v", pod.Name, err)
			return err
		}
	}
	err := e.controller.ResourceManager.DeletePod(pod, e.grace)
	if nil != err && !errors.IsNotFound(err) {
		glog.Errorf("delete pod %s,%v", pod.Name, err)
		return err
	}
	glog.Infof("delete pod %s, grace: %v, error: %v", pod.Name, e.grace, err)
	if nil != err && !errors.IsNotFound(err) {
		glog.Errorf("delete pod %s,%v", pod.Name, err)
		return err
	}

	return nil
}

func (e *baseExecutor) patchPod(pod *corev1.Pod, patch *util.CommonPatch) error {
	if patch.SubResources == nil {
		patch.SubResources = []string{}
	}
	err := e.controller.ResourceManager.PatchPod(pod, patch.PatchType, patch.SimpleData(), patch.SubResources)
	glog.Infof("patch Pod: %s, content: %s, err:%v", pod.Name, patch.String(), err)
	if nil != err && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (e *baseExecutor) createPod(pod *corev1.Pod) (*corev1.Pod, error) {
	newPod, err := e.controller.ResourceManager.CreatePod(pod)
	glog.Infof("create pod %s, %v", utils.ObjJSON(pod), err)
	if nil != err {
		return nil, err
	}
	return newPod, nil
}

func (e *baseExecutor) createConstraint(constraint *carbonv1.TemporaryConstraint) error {
	return nil
}

func (e *baseExecutor) deleteConstraint(constraint *carbonv1.TemporaryConstraint) error {
	return nil
}

func (e *baseExecutor) listConstraint() ([]*carbonv1.TemporaryConstraint, error) {
	return nil, nil
}

type updateExecutor struct {
	baseExecutor
}

func (e *updateExecutor) updatePod(worker *carbonv1.WorkerNode, before, after *corev1.Pod) (*corev1.Pod, error) {
	if nil == before || nil == after {
		glog.Infof("nil pod: before: %v, after: %v", before, after)
		return nil, nil
	}
	if nil == e.controller || nil == e.controller.ResourceManager {
		return nil, nil
	}
	carbonv1.SyncCrdTime(&before.ObjectMeta, &after.ObjectMeta)
	if utils.ObjJSON(before.Spec) != utils.ObjJSON(after.Spec) || utils.ObjJSON(before.ObjectMeta) != utils.ObjJSON(after.ObjectMeta) {
		glog.Infof("update pod %s, before:  %s, after:  %s", after.Name, utils.ObjJSON(before), utils.ObjJSON(after))
		if e.shouldReclaim(worker, before, after) {
			if carbonv1.IsWorkerVersionMisMatch(worker) {
				glog.Infof("%s modify fields not allowed to be modified, set worker reclaim", worker.Name)
				worker.Status.InternalReclaim = true
			}
			return before, nil
		}
		err := e.controller.ResourceManager.UpdatePodSpec(after)
		if nil != err && !errors.IsNotFound(err) {
			glog.Warningf("update pod error:%s,%v", after.Name, err)
			if errors.IsForbidden(err) || errors.IsInvalid(err) || carbonv1.IsRejectByWebhook(err) {
				if carbonv1.IsWorkerVersionMisMatch(worker) {
					glog.Warningf("%s update failed, set reclaim", after.Name)
					worker.Status.InternalReclaim = true
				}
				return before, nil
			}
			return nil, err
		}
	} else {
		if glog.V(4) {
			glog.Infof("worker UpdatePodSpec %s, same, cant update", after.Name)
		}
	}
	worker.Status.InternalReclaim = false
	return before, nil
}

func (e *updateExecutor) isPodAppRoleNameChanged(before, after *corev1.Pod) bool {
	if carbonv1.GetAppName(before) != carbonv1.GetAppName(after) ||
		carbonv1.GetRoleName(before) != carbonv1.GetRoleName(after) {
		return true
	}
	return false
}

func (e *updateExecutor) isPodInplaceUpdateForbidden(before, after *corev1.Pod) bool {
	if carbonv1.GetLabelValue(before, carbonv1.LabelInplaceUpdateForbidden) == "true" {
		for i := range before.Spec.Containers {
			for j := range after.Spec.Containers {
				if before.Spec.Containers[i].Name == after.Spec.Containers[j].Name {
					if before.Spec.Containers[i].Image != after.Spec.Containers[j].Image {
						return true
					}
				}
			}
		}
	}
	return false
}

func (e *updateExecutor) isPodReplaceNodeVersionChanged(before, after *corev1.Pod) bool {
	if carbonv1.GetReplaceNodeVersion(before) != carbonv1.GetReplaceNodeVersion(after) {
		return true
	}
	return false
}

func (e *updateExecutor) isAsiPodChanged(before, after *corev1.Pod) bool {
	if carbonv1.GetPodVersion(before) != carbonv1.GetPodVersion(after) &&
		(carbonv1.GetPodVersion(before) == carbonv1.PodVersion3 || carbonv1.GetPodVersion(after) == carbonv1.PodVersion3) {
		return true
	}
	return false
}

func (e *updateExecutor) isSkylineChanged(before, after *corev1.Pod) bool {
	if e.valueChanged(carbonv1.GetInstanceGroup(before), carbonv1.GetInstanceGroup(after)) ||
		e.valueChanged(carbonv1.GetAppUnit(before), carbonv1.GetAppUnit(after)) ||
		e.valueChanged(carbonv1.GetAppStage(before), carbonv1.GetAppStage(after)) {
		return true
	}
	return false
}

func (e *updateExecutor) valueChanged(before, after string) bool {
	return before != after && before != ""
}

func (e *updateExecutor) shouldReclaim(worker *carbonv1.WorkerNode, before, after *corev1.Pod) bool {
	if e.isPodAppRoleNameChanged(before, after) {
		glog.Infof("app role change, set reclaim: %s", before.Name)
		return true
	}
	if e.isAsiPodChanged(before, after) {
		glog.Infof("asi pod change, set reclaim: %s", before.Name)
		return true
	}
	if e.isSkylineChanged(before, after) {
		glog.Infof("skyline change, set reclaim: %s", before.Name)
		return true
	}
	if e.isPodReplaceNodeVersionChanged(before, after) {
		glog.Infof("replace-node-version change, set reclaim: %s", before.Name)
		return true
	}
	if carbonv1.IsWorkerResVersionMisMatch(worker) && before.Spec.NodeName == "" && !carbonv1.InStandby2Active(worker) {
		glog.Infof("3.0 update not alloced pod resource, set reclaim: %s", before.Name)
		return true
	}
	if e.isPodInplaceUpdateForbidden(before, after) {
		glog.Infof("3.0 update inplace update forbidden, set reclaim: %s", before.Name)
		return true
	}
	return false
}

type recreateExecutor struct {
	baseExecutor
}

func (e *recreateExecutor) updatePod(worker *carbonv1.WorkerNode, before, after *corev1.Pod) (*corev1.Pod, error) {
	var pod *corev1.Pod
	if before == nil || after == nil {
		return nil, nil
	}
	if e.controller == nil || e.controller.ResourceManager == nil {
		return nil, nil
	}
	if !carbonv1.IsWorkerVersionMisMatch(worker) {
		glog.V(5).Infof("worker version match, no need to updatePod, name:%s", worker.Name)
		return nil, nil
	}
	carbonv1.SyncCrdTime(&before.ObjectMeta, &after.ObjectMeta)
	if !reflect.DeepEqual(before.Spec, after.Spec) || !reflect.DeepEqual(before.ObjectMeta, after.ObjectMeta) {
		glog.Infof("update pod %s, before:%s, after:%s", after.Name, utils.ObjJSON(before), utils.ObjJSON(after))
		err := e.controller.ResourceManager.DeletePod(after, e.grace)
		glog.Infof("delete pod for update, %s, %v", utils.ObjJSON(after), err)
		if nil != err && !errors.IsNotFound(err) {
			return nil, err
		}
		after.Spec.NodeName = ""
		carbonv1.ClearObjectMeta(&after.ObjectMeta)

		e.resetWorkerStatus(worker)
		glog.Infof("reset workernode status, %s", worker.Name)
	}
	return pod, nil
}

func (e *recreateExecutor) resetWorkerStatus(worker *carbonv1.WorkerNode) {
	worker.Spec.Releasing = false
	worker.Spec.Reclaim = false
	worker.Status = carbonv1.WorkerNodeStatus{}
	delete(worker.Labels, carbonv1.LabelKeyHippoPreference)
	carbonv1.SetWorkerUnAssigned(worker)
}
