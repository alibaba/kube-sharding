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

package controller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/alibaba/kube-sharding/cmd/controllermanager/config"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	carbonscheme "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/scheme"
	"github.com/pborman/uuid"
	"k8s.io/client-go/kubernetes/scheme"
	glog "k8s.io/klog"

	memclient "github.com/alibaba/kube-sharding/pkg/memkube/client"
	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	//MemkubeRegisterPrefix memkube service node register prefix
	MemkubeRegisterPrefix = "memkube:register:v1"

	defaultMemSyncPeriodMs = 500

	defaultBatcherHashedNum = 40
)

func isCarbonJobMode(memObjs []string) bool {
	for _, s := range memObjs {
		if s == "carbonjobs" {
			return true
		}
	}
	return false
}

// CreateLocalClient create a mem-kube local client.
func CreateLocalClient(s *config.Config, stopCh <-chan struct{}, silentWatch bool) (*memclient.LocalClient, error) {
	glog.Infof("Create LocalClient memObjs:%s memFsURL:%s memSyncPeriodMs:%v", s.MemObjs, s.MemFsURL, s.MemSyncPeriodMs)
	if s.MemFsURL == "" {
		return nil, errors.New("Illegal memFsURL")
	}
	if s.MemSyncPeriodMs <= 0 {
		s.MemSyncPeriodMs = defaultMemSyncPeriodMs
	}
	if s.MemBatcherHashKeyNum <= 0 {
		s.MemBatcherHashKeyNum = defaultBatcherHashedNum
	}
	syncPeriodMs := time.Millisecond * time.Duration(s.MemSyncPeriodMs)
	timeoutMs := syncPeriodMs * 2
	if s.MemTimeoutMs > 0 {
		timeoutMs = time.Millisecond * time.Duration(s.MemTimeoutMs)
	}
	client := memclient.NewLocalClient(timeoutMs)
	persisterFactory, _ := mem.NewNamespacePersisterFactory(s.MemFsURL, s.MemFsBackupURL, s.Namespace)

	// verify storage
	if err := persisterFactory.Check(); err != nil {
		return nil, err
	}

	carbonJobMode := isCarbonJobMode(s.MemObjs)

	for _, obj := range s.MemObjs {
		resource := strings.ToLower(obj)
		gvk := getGroupVersionKind(resource)
		if gvk.Empty() {
			glog.Errorf("Illegal memObj:%s", resource)
			return nil, errors.New("Illegal memObj")
		}
		persister, err := persisterFactory.NewFsPersister(gvk, nil)
		if err != nil {
			glog.Errorf("New FsPersister failed gvk:%s err:%v", gvk.String(), err)
			return nil, err
		}
		opts := []mem.StoreOption{
			mem.WithStorePersistPeriod(syncPeriodMs),
		}
		if silentWatch {
			opts = append(opts, mem.WithSilentWatchConsumer(stopCh))
		}
		if s.MemStorePoolWorkerSize > 0 && s.MemStorePoolQueueSize > 0 {
			opts = append(opts, mem.WithStoreConcurrentPersist(s.MemStorePoolWorkerSize, s.MemStorePoolQueueSize))
		}
		var batcher mem.Batcher
		if carbonJobMode {
			batcher = getCarbonBatcher(resource, s.MemBatcherHashKeyNum)
		} else if s.MemBatcherGroupKey != "" {
			batcher = getBatcherByGroupKey(s.MemBatcherGroupKey)
		} else {
			batcher = getBatcher(resource)
		}
		store := mem.NewStore(getScheme(resource), gvk, s.Namespace, batcher, persister, opts...)
		client.Register(resource, s.Namespace, store)
		glog.Infof("Register store resource:%s namespace:%s", resource, s.Namespace)
	}
	glog.Info("Create LocalClient finished")
	return client, nil
}

func getGroupVersionKind(resource string) schema.GroupVersionKind {
	if resource == "" {
		return schema.GroupVersionKind{}
	}
	switch resource {
	case "pods":
		return v1.SchemeGroupVersion.WithKind("Pod")
	case "workernodes":
		return carbonv1.SchemeGroupVersion.WithKind("WorkerNode")
	case "rollingsets":
		return carbonv1.SchemeGroupVersion.WithKind("RollingSet")
	case "shardgroups":
		return carbonv1.SchemeGroupVersion.WithKind("ShardGroup")
	case "carbonjobs":
		return carbonv1.SchemeGroupVersion.WithKind("CarbonJob")
	default:
		return schema.GroupVersionKind{}
	}
}

func getBatcher(resource string) mem.Batcher {
	funcBatcher := mem.FuncBatcher{}
	if resource == "scales" || resource == "groups" {
		funcBatcher.KeyFunc = mem.KeyFuncLabelValue("app.scaler.io/scheduler") //TODO use hash?
	} else {
		funcBatcher.KeyFunc = mem.KeyFuncLabelValue(carbonv1.DefaultShardGroupUniqueLabelKey)
	}
	return &funcBatcher
}

func getBatcherByGroupKey(batcherGroupKey string) mem.Batcher {
	funcBatcher := mem.FuncBatcher{}
	funcBatcher.KeyFunc = mem.KeyFuncLabelValue(batcherGroupKey)
	return &funcBatcher
}

func getCarbonBatcher(resource string, hashNum uint32) mem.Batcher {
	funcBatcher := mem.FuncBatcher{}
	funcBatcher.KeyFunc = mem.HashedNameFuncLabelValue(carbonv1.LabelKeyCarbonJobName, hashNum)
	return &funcBatcher
}

func getScalerBatcher(resource string, hashNum uint32) mem.Batcher {
	funcBatcher := mem.FuncBatcher{}
	funcBatcher.KeyFunc = mem.KeyFuncLabelValue("app.scaler.io/scheduler")
	return &funcBatcher
}

func getScheme(resource string) *runtime.Scheme {
	if "pods" == resource {
		return scheme.Scheme
	}
	return carbonscheme.Scheme
}

func getServerID(s *config.Config) string {
	hostIP, _ := utils.GetLocalIP()
	sign, _ := utils.SignatureWithMD5(string(uuid.NewUUID()))
	if s.LabelSelector != "" && strings.HasPrefix(s.LabelSelector, carbonv1.ControllersShardKey) {
		//app.c2.io/controllers-shard in (xxx,xxx)
		shardIDsStr := s.LabelSelector[strings.Index(s.LabelSelector, "(")+1 : strings.Index(s.LabelSelector, ")")]
		return MemkubeRegisterPrefix + ":" + sign + "_" + hostIP + "_" + strconv.Itoa(int(s.Generic.Port)) + "_" + s.Namespace + "_" + shardIDsStr
	}
	return MemkubeRegisterPrefix + ":" + sign + "_" + hostIP + "_" + strconv.Itoa(int(s.Generic.Port)) + "_" + s.Namespace
}
