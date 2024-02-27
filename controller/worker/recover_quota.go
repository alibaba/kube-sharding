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
	"container/list"
	"sync"
	"time"

	"github.com/bluele/gcache"
	glog "k8s.io/klog"
)

const (
	workerLruSize    = 10240
	workerLruTimeout = 5 * time.Second
)

var defaultRecoverQuotas = &recoverQuotas{
	quotas:         make(map[string]*timeRecoverQuota),
	parallelQuotas: make(map[string]*parallelRecoverQuota),
	workerCache:    gcache.New(workerLruSize).LRU().Build(),
}

type timeRecoverQuota struct {
	recovers *list.List
	sync.RWMutex
}

type parallelRecoverQuota struct {
	parallelRecovers map[string]bool
	sync.RWMutex
}

func newParallelRecoverQuota() *parallelRecoverQuota {
	return &parallelRecoverQuota{
		parallelRecovers: map[string]bool{},
	}
}

func newRecoverQuota() *timeRecoverQuota {
	var timeRecoverQuota = &timeRecoverQuota{
		recovers: list.New(),
	}
	return timeRecoverQuota
}

func (r *timeRecoverQuota) require(maxFailedCount *int32, timeWindow *int32) bool {
	r.Lock()
	defer r.Unlock()
	if nil == maxFailedCount {
		return true
	}
	if 0 == *maxFailedCount {
		if glog.V(4) {
			glog.Infof("maxfailedcount is zero and return false")
		}
		return false
	}
	if nil == timeWindow || 0 == *timeWindow {
		return true
	}
	var front *list.Element
	for {
		front = r.recovers.Front()
		if nil == front {
			break
		}
		requireTime := front.Value.(int64)
		if time.Now().Unix()-requireTime > int64(*timeWindow) {
			r.recovers.Remove(front)
			continue
		}
		break
	}

	if r.recovers.Len() >= int(*maxFailedCount) {
		if glog.V(4) {
			glog.Infof("maxfailedcount is %d and recovers is %d and front is %d", int(*maxFailedCount), r.recovers.Len(), front.Value.(int64))
		}
		return false
	}

	r.recovers.PushBack(time.Now().Unix())
	return true
}

func (r *parallelRecoverQuota) require(replica string, maxFailedCount *int32) bool {
	if maxFailedCount == nil {
		return true
	}
	r.Lock()
	defer r.Unlock()
	glog.V(4).Infof("parallelRecoverQuota require %s,%v,%d", replica, r.parallelRecovers, *maxFailedCount)
	if r.parallelRecovers[replica] {
		return true
	}
	if len(r.parallelRecovers) >= int(*maxFailedCount) {
		return false
	}
	r.parallelRecovers[replica] = true
	glog.Infof("required recover quota for %s", replica)
	return true
}

func (r *parallelRecoverQuota) giveback(replica string) bool {
	r.Lock()
	defer r.Unlock()
	if _, ok := r.parallelRecovers[replica]; ok {
		glog.Infof("giveback recover quota for %s", replica)
		delete(r.parallelRecovers, replica)
	}
	return true
}

type recoverQuotas struct {
	quotas         map[string]*timeRecoverQuota
	parallelQuotas map[string]*parallelRecoverQuota
	sync.RWMutex
	workerCache gcache.Cache
	workerLock  sync.RWMutex
}

func (r *recoverQuotas) getQuota(key string) *timeRecoverQuota {
	r.Lock()
	defer r.Unlock()
	if quota, ok := r.quotas[key]; ok {
		return quota
	}
	quota := newRecoverQuota()
	r.quotas[key] = quota
	return quota
}

func (r *recoverQuotas) getParallelQuota(key string) *parallelRecoverQuota {
	r.Lock()
	defer r.Unlock()
	if quota, ok := r.parallelQuotas[key]; ok {
		return quota
	}
	quota := newParallelRecoverQuota()
	r.parallelQuotas[key] = quota
	return quota
}

func (r *recoverQuotas) require(key string, maxFailedCount *int32, timeWindow *int32) bool {
	quota := r.getQuota(key)
	return quota.require(maxFailedCount, timeWindow)
}

func (r *recoverQuotas) requireParallel(key, replica string, maxFailedCount *int32) bool {
	quota := r.getParallelQuota(key)
	return quota.require(replica, maxFailedCount)
}

func (r *recoverQuotas) givebackParallel(key, replica string) bool {
	quota := r.getParallelQuota(key)
	return quota.giveback(replica)
}

func (r *recoverQuotas) requireWorkerQuota(key string) bool {
	r.workerLock.Lock()
	defer r.workerLock.Unlock()
	ret, _ := r.workerCache.Get(key)
	if ret != nil {
		return false
	}
	r.workerCache.SetWithExpire(key, true, workerLruTimeout)
	return true
}
