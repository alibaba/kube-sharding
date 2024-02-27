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
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	corev1 "k8s.io/api/core/v1"
	glog "k8s.io/klog"
)

type scheduler interface {
	schedule() (
		worker *carbonv1.WorkerNode, pair *carbonv1.WorkerNode,
		pod *corev1.Pod, err error)
}

func newWorkerScheduler(worker *carbonv1.WorkerNode,
	pair *carbonv1.WorkerNode, pod *corev1.Pod, c *Controller) scheduler {
	return newDefaultWorkerScheduler(worker, pair, pod, c)
}

func newDefaultWorkerScheduler(worker *carbonv1.WorkerNode,
	pair *carbonv1.WorkerNode, pod *corev1.Pod, c *Controller) scheduler {
	processor := newDefaultProcessor(worker, pod, c)
	if nil == processor {
		return nil
	}
	var scheduler = defaultWorkerScheduler{
		worker:    worker,
		pair:      pair,
		pod:       pod,
		adjuster:  newDefaultAdjuster(worker, pair, pod, c),
		c:         c,
		processor: processor,
	}
	return &scheduler
}

type defaultWorkerScheduler struct {
	worker    *carbonv1.WorkerNode
	pair      *carbonv1.WorkerNode
	pod       *corev1.Pod
	c         *Controller
	adjuster  workerAdjuster
	processor processor
}

func (s *defaultWorkerScheduler) schedule() (worker *carbonv1.WorkerNode, pair *carbonv1.WorkerNode, pod *corev1.Pod, err error) {
	if s.worker != nil {
		s.worker.Status.NotScheduleSeconds = getNotScheduleTime(s.pod)
		glog.V(4).Infof("getNotScheduleTime %s,%d,%d ", s.worker.Name, getNotScheduleTime(s.pod), s.worker.Status.NotScheduleSeconds)
	}
	var synced bool
	s.worker, s.pair, synced, err = s.adjuster.adjust()
	if nil != err {
		if nil != s.worker {
			glog.Warningf("worker: %s adjust with error: %v", s.worker.Name, err)
		}
		return
	}
	if synced {
		return s.worker, s.pair, nil, nil
	}
	worker, pod, err = s.processor.process()
	if nil != worker {
		worker.Status.Score = carbonv1.ScoreOfWorker(worker)
	}
	return worker, s.pair, pod, err
}
