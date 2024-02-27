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

package apiset

import (
	"context"
	"sort"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	v1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	carbonclientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	"github.com/alibaba/kube-sharding/transfer"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkerEvictionAPIs operates on WorkerNodeEviction
type WorkerEvictionAPIs interface {
	SetWorkerNodeEviction(eviction *v1.WorkerNodeEviction) error
}

type workerEviction struct {
	clientset carbonclientset.Interface
}

// NewWorkerEvictionAPIs creates a new worker node eviction apis.
func NewWorkerEvictionAPIs(clientset carbonclientset.Interface) WorkerEvictionAPIs {
	return &workerEviction{clientset}
}

func (w *workerEviction) SetWorkerNodeEviction(eviction *v1.WorkerNodeEviction) error {
	api := w.clientset.CarbonV1().WorkerNodeEvictions(eviction.GetNamespace())
	_, err := api.Get(context.Background(), eviction.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			glog.Errorf("Get node eviction erorr: %s, %v", eviction.Name, err)
			return err
		}
		_, err = api.Create(context.Background(), eviction, metav1.CreateOptions{})
		if glog.V(4) {
			glog.Infof("Create node eviction: %s, %+v, ret: %v", eviction.Name, eviction.Spec, err)
		}
		return err
	}
	// No updates here.
	return nil
}

// NewWorkerNodeEvictionSpec creates WorkerNodeEviction spec.
func NewWorkerNodeEvictionSpec(appName, namespace, cluster string, workerNodes []*carbonv1.ReclaimWorkerNode, pref *v1.HippoPrefrence) (*v1.WorkerNodeEviction, error) {
	sort.SliceStable(workerNodes, func(i, j int) bool {
		return workerNodes[i].WorkerNodeID < workerNodes[j].WorkerNodeID
	})
	eviction := &v1.WorkerNodeEviction{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Labels: map[string]string{
				v1.LabelKeyClusterName: cluster,
				v1.LabelKeyAppName:     appName,
			},
		},
		Spec: v1.WorkerNodeEvictionSpec{
			AppName:     appName,
			WorkerNodes: workerNodes,
			Pref:        pref,
		},
	}
	h, err := utils.SignatureShort(&eviction.Spec)
	if err != nil {
		return nil, err
	}
	// Make an uniq name to optimize the writing.
	eviction.Name = transfer.EscapeName(appName) + "-" + h
	return eviction, nil
}
