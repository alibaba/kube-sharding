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

package carbonjob

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/alibaba/kube-sharding/controller"
	"github.com/alibaba/kube-sharding/controller/util"
	mock_util "github.com/alibaba/kube-sharding/controller/util/mock"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	clientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	informers "github.com/alibaba/kube-sharding/pkg/client/informers/externalversions/carbon/v1"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	mock_v1 "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1/mock"
	"github.com/alibaba/kube-sharding/transfer"
	"github.com/golang/mock/gomock"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

func TestController_getWorkerNodesFromCarbonJob(t *testing.T) {
	workers := []*carbonv1.WorkerNode{
		{},
		{},
	}

	namespace := "namespace"
	carbonJob := &carbonv1.CarbonJob{}
	carbonJob.Namespace = namespace
	carbonJob.Name = "1"

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type fields struct {
		GroupVersionKind  schema.GroupVersionKind
		carbonclientset   clientset.Interface
		kubeclientset     kubernetes.Interface
		podLister         corelisters.PodLister
		podListerSynced   cache.InformerSynced
		podInformer       coreinformers.PodInformer
		carbonJobLister   listers.CarbonJobLister
		carbonJobSynced   cache.InformerSynced
		carbonJobInformer informers.CarbonJobInformer
		resourceManager   util.ResourceManager
	}
	type args struct {
		carbonJob *carbonv1.CarbonJob
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []*carbonv1.WorkerNode
		wantErr bool
	}{
		{
			name: "success get",
			fields: fields{
				resourceManager: func() util.ResourceManager {
					rm := mock_util.NewMockResourceManager(ctrl)
					rm.EXPECT().ListWorkerNodeByOwner(carbonJob.GetLabels(), carbonv1.LabelKeyCarbonJobName).Return(workers, nil)
					return rm
				}(),
			},
			args: args{
				carbonJob: carbonJob,
			},
			want:    workers,
			wantErr: false,
		},
		{
			name: "failed get",
			fields: fields{
				resourceManager: func() util.ResourceManager {
					rm := mock_util.NewMockResourceManager(ctrl)
					rm.EXPECT().ListWorkerNodeByOwner(carbonJob.GetLabels(), carbonv1.LabelKeyCarbonJobName).Return(nil, fmt.Errorf("failed"))
					return rm
				}(),
			},
			args: args{
				carbonJob: carbonJob,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "success get, not found, ",
			fields: fields{
				resourceManager: func() util.ResourceManager {
					rm := mock_util.NewMockResourceManager(ctrl)
					rm.EXPECT().ListWorkerNodeByOwner(carbonJob.GetLabels(), carbonv1.LabelKeyCarbonJobName).Return([]*carbonv1.WorkerNode{}, nil)
					return rm
				}(),
			},
			args: args{
				carbonJob: carbonJob,
			},
			want:    []*carbonv1.WorkerNode{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				GroupVersionKind:  tt.fields.GroupVersionKind,
				carbonclientset:   tt.fields.carbonclientset,
				kubeclientset:     tt.fields.kubeclientset,
				podLister:         tt.fields.podLister,
				podListerSynced:   tt.fields.podListerSynced,
				podInformer:       tt.fields.podInformer,
				carbonJobLister:   tt.fields.carbonJobLister,
				carbonJobSynced:   tt.fields.carbonJobSynced,
				carbonJobInformer: tt.fields.carbonJobInformer,
				DefaultController: controller.DefaultController{
					ResourceManager: tt.fields.resourceManager,
				},
			}
			got, err := c.getWorkerNodesFromCarbonJob(tt.args.carbonJob)
			if (err != nil) != tt.wantErr {
				t.Errorf("Controller.getWorkerNodesFromCarbonJob() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Controller.getWorkerNodesFromCarbonJob() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestController_DeleteSubObj(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	name := "test"
	namespace := "namespae"

	workerNode := &carbonv1.WorkerNode{}
	workerNode.Name = "test"

	type fields struct {
		GroupVersionKind       schema.GroupVersionKind
		resourceManager        util.ResourceManager
		carbonclientset        clientset.Interface
		kubeclientset          kubernetes.Interface
		podLister              corelisters.PodLister
		podListerSynced        cache.InformerSynced
		podInformer            coreinformers.PodInformer
		workerNodeLister       listers.WorkerNodeLister
		workerNodeListerSynced cache.InformerSynced
		workerNodeInformer     informers.WorkerNodeInformer
		carbonJobLister        listers.CarbonJobLister
		carbonJobSynced        cache.InformerSynced
		carbonJobInformer      informers.CarbonJobInformer
		specConverter          transfer.WorkerNodeSpecConverter
	}
	type args struct {
		namespace string
		name      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "get failed, error",
			fields: fields{
				resourceManager: func() util.ResourceManager {
					rm := mock_util.NewMockResourceManager(ctrl)
					return rm
				}(),
				workerNodeLister: func() listers.WorkerNodeLister {
					namespaceLister := mock_v1.NewMockWorkerNodeNamespaceLister(ctrl)
					namespaceLister.EXPECT().Get(name).Return(nil, fmt.Errorf("failed"))
					lister := mock_v1.NewMockWorkerNodeLister(ctrl)
					lister.EXPECT().WorkerNodes(namespace).Return(namespaceLister)
					return lister
				}(),
			},
			args: args{
				namespace: namespace,
				name:      name,
			},
			wantErr: true,
		},
		{
			name: "get failed, not exist",
			fields: fields{
				resourceManager: func() util.ResourceManager {
					rm := mock_util.NewMockResourceManager(ctrl)
					return rm
				}(),
				workerNodeLister: func() listers.WorkerNodeLister {
					namespaceLister := mock_v1.NewMockWorkerNodeNamespaceLister(ctrl)
					namespaceLister.EXPECT().Get(name).Return(nil, errors.NewNotFound(schema.GroupResource{}, ""))
					lister := mock_v1.NewMockWorkerNodeLister(ctrl)
					lister.EXPECT().WorkerNodes(namespace).Return(namespaceLister)
					return lister
				}(),
			},
			args: args{
				namespace: namespace,
				name:      name,
			},
			wantErr: false,
		},
		{
			name: "get success, delete failed",
			fields: fields{
				resourceManager: func() util.ResourceManager {
					rm := mock_util.NewMockResourceManager(ctrl)
					rm.EXPECT().DeleteWorkerNode(workerNode).Return(fmt.Errorf("failed"))
					return rm
				}(),
				workerNodeLister: func() listers.WorkerNodeLister {
					namespaceLister := mock_v1.NewMockWorkerNodeNamespaceLister(ctrl)
					namespaceLister.EXPECT().Get(name).Return(workerNode, nil)
					lister := mock_v1.NewMockWorkerNodeLister(ctrl)
					lister.EXPECT().WorkerNodes(namespace).Return(namespaceLister)
					return lister
				}(),
			},
			args: args{
				namespace: namespace,
				name:      name,
			},
			wantErr: true,
		},
		{
			name: "get success, delete not exist",
			fields: fields{
				resourceManager: func() util.ResourceManager {
					rm := mock_util.NewMockResourceManager(ctrl)
					rm.EXPECT().DeleteWorkerNode(workerNode).Return(errors.NewNotFound(schema.GroupResource{}, ""))
					return rm
				}(),
				workerNodeLister: func() listers.WorkerNodeLister {
					namespaceLister := mock_v1.NewMockWorkerNodeNamespaceLister(ctrl)
					namespaceLister.EXPECT().Get(name).Return(workerNode, nil)
					lister := mock_v1.NewMockWorkerNodeLister(ctrl)
					lister.EXPECT().WorkerNodes(namespace).Return(namespaceLister)
					return lister
				}(),
			},
			args: args{
				namespace: namespace,
				name:      name,
			},
			wantErr: false,
		},
		{
			name: "get success, delete success",
			fields: fields{
				resourceManager: func() util.ResourceManager {
					rm := mock_util.NewMockResourceManager(ctrl)
					rm.EXPECT().DeleteWorkerNode(workerNode).Return(nil)
					return rm
				}(),
				workerNodeLister: func() listers.WorkerNodeLister {
					namespaceLister := mock_v1.NewMockWorkerNodeNamespaceLister(ctrl)
					namespaceLister.EXPECT().Get(name).Return(workerNode, nil)
					lister := mock_v1.NewMockWorkerNodeLister(ctrl)
					lister.EXPECT().WorkerNodes(namespace).Return(namespaceLister)
					return lister
				}(),
			},
			args: args{
				namespace: namespace,
				name:      name,
			},
			wantErr: false,
		}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				GroupVersionKind: tt.fields.GroupVersionKind,
				DefaultController: controller.DefaultController{
					ResourceManager: tt.fields.resourceManager,
				},
				carbonclientset:        tt.fields.carbonclientset,
				kubeclientset:          tt.fields.kubeclientset,
				podLister:              tt.fields.podLister,
				podListerSynced:        tt.fields.podListerSynced,
				podInformer:            tt.fields.podInformer,
				workerNodeLister:       tt.fields.workerNodeLister,
				workerNodeListerSynced: tt.fields.workerNodeListerSynced,
				workerNodeInformer:     tt.fields.workerNodeInformer,
				carbonJobLister:        tt.fields.carbonJobLister,
				carbonJobSynced:        tt.fields.carbonJobSynced,
				carbonJobInformer:      tt.fields.carbonJobInformer,
				specConverter:          tt.fields.specConverter,
			}
			if err := c.DeleteSubObj(tt.args.namespace, tt.args.name); (err != nil) != tt.wantErr {
				t.Errorf("Controller.DeleteSubObj() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
