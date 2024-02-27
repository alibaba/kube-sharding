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
	"reflect"
	"testing"

	"github.com/alibaba/kube-sharding/controller/util"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	clientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	carbonfake "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/fake"
	carboninformers "github.com/alibaba/kube-sharding/pkg/client/informers/externalversions"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/controller"
)

func TestController_onDeleteWorker(t *testing.T) {
	type fields struct {
		GroupVersionKind schema.GroupVersionKind

		kubeclientset               kubernetes.Interface
		carbonclientset             clientset.Interface
		podLister                   corelisters.PodLister
		podListerSynced             cache.InformerSynced
		podInformer                 coreinformers.PodInformer
		workerLister                listers.WorkerNodeLister
		workerListerSynced          cache.InformerSynced
		serviceLister               listers.ServicePublisherLister
		serviceListerSynced         cache.InformerSynced
		rollingSetLister            listers.RollingSetLister
		rollingSetSynced            cache.InformerSynced
		reserveResourceListerSynced cache.InformerSynced
	}
	type args struct {
		obj interface{}
	}
	worker := newTestWorker("test", "1", nil)
	carbonfakeclient := carbonfake.NewSimpleClientset([]runtime.Object{}...)
	carboninformersFactory := carboninformers.NewSharedInformerFactory(carbonfakeclient, controller.NoResyncPeriodFunc())
	workerLister := carboninformersFactory.Carbon().V1().WorkerNodes().Lister()

	//carboninformersFactory.Carbon().V1().
	//	WorkerNodes().Informer().GetIndexer().Add(worker)

	pod := newPod("test", "1", corev1.PodPending, "")
	fakeclient := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := informers.NewSharedInformerFactory(fakeclient, controller.NoResyncPeriodFunc())
	podLister := informersFactory.Core().V1().Pods().Lister()
	informersFactory.Core().V1().Pods().Informer().GetIndexer().Add(pod)
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "worker",
			fields: fields{
				podLister:       podLister,
				workerLister:    workerLister,
				kubeclientset:   fakeclient,
				carbonclientset: carbonfakeclient,
			},
			args: args{
				obj: worker,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				GroupVersionKind:    tt.fields.GroupVersionKind,
				kubeclientset:       tt.fields.kubeclientset,
				carbonclientset:     tt.fields.carbonclientset,
				podLister:           tt.fields.podLister,
				podListerSynced:     tt.fields.podListerSynced,
				podInformer:         tt.fields.podInformer,
				workerLister:        tt.fields.workerLister,
				workerListerSynced:  tt.fields.workerListerSynced,
				serviceLister:       tt.fields.serviceLister,
				serviceListerSynced: tt.fields.serviceListerSynced,
				rollingSetLister:    tt.fields.rollingSetLister,
				rollingSetSynced:    tt.fields.rollingSetSynced,
			}
			c.DefaultController.ResourceManager = util.NewSimpleResourceManager(tt.fields.kubeclientset, tt.fields.carbonclientset, nil, nil, nil)
			c.onDeleteWorker(tt.args.obj)
		})
	}
}

func TestController_GetObj(t *testing.T) {
	type fields struct {
		GroupVersionKind schema.GroupVersionKind

		kubeclientset       kubernetes.Interface
		carbonclientset     clientset.Interface
		podLister           corelisters.PodLister
		podListerSynced     cache.InformerSynced
		podInformer         coreinformers.PodInformer
		workerLister        listers.WorkerNodeLister
		workerListerSynced  cache.InformerSynced
		serviceLister       listers.ServicePublisherLister
		serviceListerSynced cache.InformerSynced
		rollingSetLister    listers.RollingSetLister
		rollingSetSynced    cache.InformerSynced
	}

	worker := newTestWorker("test", "1", nil)
	carbonfakeclient := carbonfake.NewSimpleClientset([]runtime.Object{}...)
	carboninformersFactory := carboninformers.NewSharedInformerFactory(carbonfakeclient, controller.NoResyncPeriodFunc())
	workerLister := carboninformersFactory.Carbon().V1().WorkerNodes().Lister()
	carboninformersFactory.Carbon().V1().
		WorkerNodes().Informer().GetIndexer().Add(worker)

	pod := newPod("test", "1", corev1.PodPending, "")
	fakeclient := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := informers.NewSharedInformerFactory(fakeclient, controller.NoResyncPeriodFunc())
	podLister := informersFactory.Core().V1().Pods().Lister()
	informersFactory.Core().V1().Pods().Informer().GetIndexer().Add(pod)
	type args struct {
		namespace string
		key       string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "get",
			fields: fields{
				podLister:    podLister,
				workerLister: workerLister,
			},
			args: args{
				namespace: worker.Namespace,
				key:       worker.Name,
			},
			want:    worker,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				GroupVersionKind: tt.fields.GroupVersionKind,

				kubeclientset:       tt.fields.kubeclientset,
				carbonclientset:     tt.fields.carbonclientset,
				podLister:           tt.fields.podLister,
				podListerSynced:     tt.fields.podListerSynced,
				podInformer:         tt.fields.podInformer,
				workerLister:        tt.fields.workerLister,
				workerListerSynced:  tt.fields.workerListerSynced,
				serviceLister:       tt.fields.serviceLister,
				serviceListerSynced: tt.fields.serviceListerSynced,
				rollingSetLister:    tt.fields.rollingSetLister,
				rollingSetSynced:    tt.fields.rollingSetSynced,
			}
			got, err := c.GetObj(tt.args.namespace, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Controller.GetObj() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Controller.GetObj() = %s, want %s", utils.ObjJSON(got), utils.ObjJSON(tt.want))
			}
		})
	}
}

func TestController_WaitForCacheSync(t *testing.T) {
	type fields struct {
		GroupVersionKind schema.GroupVersionKind

		kubeclientset       kubernetes.Interface
		carbonclientset     clientset.Interface
		podLister           corelisters.PodLister
		podListerSynced     cache.InformerSynced
		podInformer         coreinformers.PodInformer
		workerLister        listers.WorkerNodeLister
		workerListerSynced  cache.InformerSynced
		serviceLister       listers.ServicePublisherLister
		serviceListerSynced cache.InformerSynced
		rollingSetLister    listers.RollingSetLister
		rollingSetSynced    cache.InformerSynced
	}
	type args struct {
		stopCh <-chan struct{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				GroupVersionKind: tt.fields.GroupVersionKind,

				kubeclientset:       tt.fields.kubeclientset,
				carbonclientset:     tt.fields.carbonclientset,
				podLister:           tt.fields.podLister,
				podListerSynced:     tt.fields.podListerSynced,
				podInformer:         tt.fields.podInformer,
				workerLister:        tt.fields.workerLister,
				workerListerSynced:  tt.fields.workerListerSynced,
				serviceLister:       tt.fields.serviceLister,
				serviceListerSynced: tt.fields.serviceListerSynced,
				rollingSetLister:    tt.fields.rollingSetLister,
				rollingSetSynced:    tt.fields.rollingSetSynced,
			}
			if got := c.WaitForCacheSync(tt.args.stopCh); got != tt.want {
				t.Errorf("Controller.WaitForCacheSync() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestController_Sync(t *testing.T) {
	type fields struct {
		GroupVersionKind schema.GroupVersionKind

		kubeclientset       kubernetes.Interface
		carbonclientset     clientset.Interface
		podLister           corelisters.PodLister
		podListerSynced     cache.InformerSynced
		podInformer         coreinformers.PodInformer
		workerLister        listers.WorkerNodeLister
		workerListerSynced  cache.InformerSynced
		serviceLister       listers.ServicePublisherLister
		serviceListerSynced cache.InformerSynced
		rollingSetLister    listers.RollingSetLister
		rollingSetSynced    cache.InformerSynced
	}
	type args struct {
		key string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				GroupVersionKind: tt.fields.GroupVersionKind,

				kubeclientset:       tt.fields.kubeclientset,
				carbonclientset:     tt.fields.carbonclientset,
				podLister:           tt.fields.podLister,
				podListerSynced:     tt.fields.podListerSynced,
				podInformer:         tt.fields.podInformer,
				workerLister:        tt.fields.workerLister,
				workerListerSynced:  tt.fields.workerListerSynced,
				serviceLister:       tt.fields.serviceLister,
				serviceListerSynced: tt.fields.serviceListerSynced,
				rollingSetLister:    tt.fields.rollingSetLister,
				rollingSetSynced:    tt.fields.rollingSetSynced,
			}
			if err := c.Sync(tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Controller.Sync() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestController_updateWorkers(t *testing.T) {
	type fields struct {
		GroupVersionKind schema.GroupVersionKind

		kubeclientset       kubernetes.Interface
		carbonclientset     clientset.Interface
		podLister           corelisters.PodLister
		podListerSynced     cache.InformerSynced
		podInformer         coreinformers.PodInformer
		workerLister        listers.WorkerNodeLister
		workerListerSynced  cache.InformerSynced
		serviceLister       listers.ServicePublisherLister
		serviceListerSynced cache.InformerSynced
		rollingSetLister    listers.RollingSetLister
		rollingSetSynced    cache.InformerSynced
	}
	type args struct {
		worker           *carbonv1.WorkerNode
		targetWorker     *carbonv1.WorkerNode
		pairWorker       *carbonv1.WorkerNode
		targetPairWorker *carbonv1.WorkerNode
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				GroupVersionKind: tt.fields.GroupVersionKind,

				kubeclientset:       tt.fields.kubeclientset,
				carbonclientset:     tt.fields.carbonclientset,
				podLister:           tt.fields.podLister,
				podListerSynced:     tt.fields.podListerSynced,
				podInformer:         tt.fields.podInformer,
				workerLister:        tt.fields.workerLister,
				workerListerSynced:  tt.fields.workerListerSynced,
				serviceLister:       tt.fields.serviceLister,
				serviceListerSynced: tt.fields.serviceListerSynced,
				rollingSetLister:    tt.fields.rollingSetLister,
				rollingSetSynced:    tt.fields.rollingSetSynced,
			}
			if err := c.updateWorkers(tt.args.worker, tt.args.targetWorker, tt.args.pairWorker, tt.args.targetPairWorker); (err != nil) != tt.wantErr {
				t.Errorf("Controller.updateWorkers() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestController_updateWorker(t *testing.T) {
	type fields struct {
		GroupVersionKind schema.GroupVersionKind

		kubeclientset       kubernetes.Interface
		carbonclientset     clientset.Interface
		podLister           corelisters.PodLister
		podListerSynced     cache.InformerSynced
		podInformer         coreinformers.PodInformer
		workerLister        listers.WorkerNodeLister
		workerListerSynced  cache.InformerSynced
		serviceLister       listers.ServicePublisherLister
		serviceListerSynced cache.InformerSynced
		rollingSetLister    listers.RollingSetLister
		rollingSetSynced    cache.InformerSynced
	}
	type args struct {
		before *carbonv1.WorkerNode
		after  *carbonv1.WorkerNode
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				GroupVersionKind: tt.fields.GroupVersionKind,

				kubeclientset:       tt.fields.kubeclientset,
				carbonclientset:     tt.fields.carbonclientset,
				podLister:           tt.fields.podLister,
				podListerSynced:     tt.fields.podListerSynced,
				podInformer:         tt.fields.podInformer,
				workerLister:        tt.fields.workerLister,
				workerListerSynced:  tt.fields.workerListerSynced,
				serviceLister:       tt.fields.serviceLister,
				serviceListerSynced: tt.fields.serviceListerSynced,
				rollingSetLister:    tt.fields.rollingSetLister,
				rollingSetSynced:    tt.fields.rollingSetSynced,
			}
			if err := c.updateWorker(tt.args.before, tt.args.after); (err != nil) != tt.wantErr {
				t.Errorf("Controller.updateWorker() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestController_getPodsForWorker(t *testing.T) {
	type fields struct {
		GroupVersionKind schema.GroupVersionKind

		kubeclientset       kubernetes.Interface
		carbonclientset     clientset.Interface
		podLister           corelisters.PodLister
		podListerSynced     cache.InformerSynced
		podInformer         coreinformers.PodInformer
		workerLister        listers.WorkerNodeLister
		workerListerSynced  cache.InformerSynced
		serviceLister       listers.ServicePublisherLister
		serviceListerSynced cache.InformerSynced
		rollingSetLister    listers.RollingSetLister
		rollingSetSynced    cache.InformerSynced
	}

	worker := newTestWorker("test-a", "1", nil)
	carbonfakeclient := carbonfake.NewSimpleClientset([]runtime.Object{}...)
	carboninformersFactory := carboninformers.NewSharedInformerFactory(carbonfakeclient, controller.NoResyncPeriodFunc())
	workerLister := carboninformersFactory.Carbon().V1().WorkerNodes().Lister()
	carboninformersFactory.Carbon().V1().
		WorkerNodes().Informer().GetIndexer().Add(worker)

	pod := newPod("test-a-abcd", "1", corev1.PodPending, "")
	fakeclient := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := informers.NewSharedInformerFactory(fakeclient, controller.NoResyncPeriodFunc())
	podLister := informersFactory.Core().V1().Pods().Lister()
	informersFactory.Core().V1().Pods().Informer().GetIndexer().Add(pod)
	type args struct {
		worker *carbonv1.WorkerNode
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *corev1.Pod
		wantErr bool
	}{
		{
			name: "get",
			fields: fields{
				podLister:    podLister,
				workerLister: workerLister,
			},
			args: args{
				worker: worker,
			},
			want:    pod,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				GroupVersionKind: tt.fields.GroupVersionKind,

				kubeclientset:       tt.fields.kubeclientset,
				carbonclientset:     tt.fields.carbonclientset,
				podLister:           tt.fields.podLister,
				podListerSynced:     tt.fields.podListerSynced,
				podInformer:         tt.fields.podInformer,
				workerLister:        tt.fields.workerLister,
				workerListerSynced:  tt.fields.workerListerSynced,
				serviceLister:       tt.fields.serviceLister,
				serviceListerSynced: tt.fields.serviceListerSynced,
				rollingSetLister:    tt.fields.rollingSetLister,
				rollingSetSynced:    tt.fields.rollingSetSynced,
			}
			got, err := c.getPodsForWorker(tt.args.worker)
			if (err != nil) != tt.wantErr {
				t.Errorf("Controller.getPodsForWorker() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Controller.getPodsForWorker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateName(t *testing.T) {
	type args struct {
		prefix string
		suffix string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "gen",
			args: args{
				prefix: "1",
				suffix: "-2",
			},
			want: "1-2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateName(tt.args.prefix, tt.args.suffix); got != tt.want {
				t.Errorf("generateName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getPairWorkerName(t *testing.T) {
	type args struct {
		replicaName    string
		currWorkerName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "genbak",
			args: args{
				replicaName:    "1",
				currWorkerName: "1-a",
			},
			want: "1-b",
		},
		{
			name: "genbak",
			args: args{
				replicaName:    "1",
				currWorkerName: "1-b",
			},
			want: "1-a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getPairWorkerName(tt.args.currWorkerName); got != tt.want {
				t.Errorf("getPairWorkerName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestController_getPairWorker(t *testing.T) {
	type fields struct {
		GroupVersionKind schema.GroupVersionKind

		kubeclientset       kubernetes.Interface
		carbonclientset     clientset.Interface
		podLister           corelisters.PodLister
		podListerSynced     cache.InformerSynced
		podInformer         coreinformers.PodInformer
		workerLister        listers.WorkerNodeLister
		workerListerSynced  cache.InformerSynced
		serviceLister       listers.ServicePublisherLister
		serviceListerSynced cache.InformerSynced
		rollingSetLister    listers.RollingSetLister
		rollingSetSynced    cache.InformerSynced
	}

	worker := newTestWorker("worker-a", "1", nil)
	worker.Labels = map[string]string{
		carbonv1.DefaultReplicaUniqueLabelKey: "worker",
	}
	worker.Spec.Selector = &metav1.LabelSelector{}
	worker1 := worker.DeepCopy()
	worker1.Name = "worker-b"
	carbonfakeclient := carbonfake.NewSimpleClientset([]runtime.Object{}...)
	carboninformersFactory := carboninformers.NewSharedInformerFactory(carbonfakeclient, controller.NoResyncPeriodFunc())
	workerLister := carboninformersFactory.Carbon().V1().WorkerNodes().Lister()
	carboninformersFactory.Carbon().V1().
		WorkerNodes().Informer().GetIndexer().Add(worker)
	carboninformersFactory.Carbon().V1().
		WorkerNodes().Informer().GetIndexer().Add(worker1)

	pod := newPod("worker-a", "1", corev1.PodPending, "")
	fakeclient := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := informers.NewSharedInformerFactory(fakeclient, controller.NoResyncPeriodFunc())
	podLister := informersFactory.Core().V1().Pods().Lister()
	informersFactory.Core().V1().Pods().Informer().GetIndexer().Add(pod)

	type args struct {
		worker *carbonv1.WorkerNode
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *carbonv1.WorkerNode
		wantErr bool
	}{
		{
			name: "get",
			fields: fields{
				podLister:    podLister,
				workerLister: workerLister,
			},
			args: args{
				worker: worker,
			},
			want:    worker1,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				GroupVersionKind: tt.fields.GroupVersionKind,

				kubeclientset:       tt.fields.kubeclientset,
				carbonclientset:     tt.fields.carbonclientset,
				podLister:           tt.fields.podLister,
				podListerSynced:     tt.fields.podListerSynced,
				podInformer:         tt.fields.podInformer,
				workerLister:        tt.fields.workerLister,
				workerListerSynced:  tt.fields.workerListerSynced,
				serviceLister:       tt.fields.serviceLister,
				serviceListerSynced: tt.fields.serviceListerSynced,
				rollingSetLister:    tt.fields.rollingSetLister,
				rollingSetSynced:    tt.fields.rollingSetSynced,
			}
			got, err := c.getPairWorker(tt.args.worker)
			if (err != nil) != tt.wantErr {
				t.Errorf("Controller.getPairWorker() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Controller.getPairWorker() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestController_getResources(t *testing.T) {
	type fields struct {
		GroupVersionKind schema.GroupVersionKind

		kubeclientset       kubernetes.Interface
		carbonclientset     clientset.Interface
		podLister           corelisters.PodLister
		podListerSynced     cache.InformerSynced
		podInformer         coreinformers.PodInformer
		workerLister        listers.WorkerNodeLister
		workerListerSynced  cache.InformerSynced
		serviceLister       listers.ServicePublisherLister
		serviceListerSynced cache.InformerSynced
		rollingSetLister    listers.RollingSetLister
		rollingSetSynced    cache.InformerSynced
	}

	worker := newTestWorker("worker-a", "1", nil)
	worker.Labels = map[string]string{
		carbonv1.DefaultReplicaUniqueLabelKey: "worker",
	}
	worker.Spec.Selector = &metav1.LabelSelector{}
	worker1 := worker.DeepCopy()
	worker1.Name = "worker-b"
	carbonfakeclient := carbonfake.NewSimpleClientset([]runtime.Object{}...)
	carboninformersFactory := carboninformers.NewSharedInformerFactory(carbonfakeclient, controller.NoResyncPeriodFunc())
	workerLister := carboninformersFactory.Carbon().V1().WorkerNodes().Lister()
	carboninformersFactory.Carbon().V1().
		WorkerNodes().Informer().GetIndexer().Add(worker)
	carboninformersFactory.Carbon().V1().
		WorkerNodes().Informer().GetIndexer().Add(worker1)

	pod := newPod("worker-a", "1", corev1.PodPending, "")
	fakeclient := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := informers.NewSharedInformerFactory(fakeclient, controller.NoResyncPeriodFunc())
	podLister := informersFactory.Core().V1().Pods().Lister()
	informersFactory.Core().V1().Pods().Informer().GetIndexer().Add(pod)

	type args struct {
		key string
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantWorker *carbonv1.WorkerNode
		wantPair   *carbonv1.WorkerNode
		wantPod    *corev1.Pod
		wantErr    bool
	}{
		{
			name: "get",
			fields: fields{
				podLister:    podLister,
				workerLister: workerLister,
			},
			args: args{
				key: "default/worker-a",
			},
			wantWorker: func() *carbonv1.WorkerNode {
				w := worker.DeepCopy()
				w.Status.EntityName = "worker-a"
				return w
			}(),
			wantPair: worker1,
			wantPod:  pod,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				GroupVersionKind: tt.fields.GroupVersionKind,

				kubeclientset:       tt.fields.kubeclientset,
				carbonclientset:     tt.fields.carbonclientset,
				podLister:           tt.fields.podLister,
				podListerSynced:     tt.fields.podListerSynced,
				podInformer:         tt.fields.podInformer,
				workerLister:        tt.fields.workerLister,
				workerListerSynced:  tt.fields.workerListerSynced,
				serviceLister:       tt.fields.serviceLister,
				serviceListerSynced: tt.fields.serviceListerSynced,
				rollingSetLister:    tt.fields.rollingSetLister,
				rollingSetSynced:    tt.fields.rollingSetSynced,
			}
			gotWorker, gotPair, gotPod, err := c.getResources(tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Controller.getResources() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotWorker, tt.wantWorker) {
				t.Errorf("Controller.getResources() gotWorker = %v, want %v", gotWorker, tt.wantWorker)
			}
			if !reflect.DeepEqual(gotPair, tt.wantPair) {
				t.Errorf("Controller.getResources() gotPair = %v, want %v", gotPair, tt.wantPair)
			}
			if !reflect.DeepEqual(gotPod, tt.wantPod) {
				t.Errorf("Controller.getResources() gotPod = %v, want %v", gotPod, tt.wantPod)
			}
		})
	}
}

func TestController_DeleteSubObj(t *testing.T) {
	type fields struct {
		GroupVersionKind schema.GroupVersionKind

		kubeclientset               kubernetes.Interface
		carbonclientset             clientset.Interface
		podLister                   corelisters.PodLister
		podListerSynced             cache.InformerSynced
		podInformer                 coreinformers.PodInformer
		workerLister                listers.WorkerNodeLister
		workerListerSynced          cache.InformerSynced
		serviceLister               listers.ServicePublisherLister
		serviceListerSynced         cache.InformerSynced
		rollingSetLister            listers.RollingSetLister
		rollingSetSynced            cache.InformerSynced
		reserveResourceListerSynced cache.InformerSynced
	}

	carbonfakeclient := carbonfake.NewSimpleClientset([]runtime.Object{}...)
	carboninformersFactory := carboninformers.NewSharedInformerFactory(carbonfakeclient, controller.NoResyncPeriodFunc())
	workerLister := carboninformersFactory.Carbon().V1().WorkerNodes().Lister()

	pod := newPod("test", "1", corev1.PodPending, "")
	fakeclient := fake.NewSimpleClientset([]runtime.Object{}...)
	informersFactory := informers.NewSharedInformerFactory(fakeclient, controller.NoResyncPeriodFunc())
	podLister := informersFactory.Core().V1().Pods().Lister()
	informersFactory.Core().V1().Pods().Informer().GetIndexer().Add(pod)

	type args struct {
		namespace string
		key       string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "worker",
			fields: fields{
				podLister:       podLister,
				workerLister:    workerLister,
				kubeclientset:   fakeclient,
				carbonclientset: carbonfakeclient,
			},
			args: args{
				namespace: pod.Namespace,
				key:       pod.Name,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Controller{
				GroupVersionKind:    tt.fields.GroupVersionKind,
				kubeclientset:       tt.fields.kubeclientset,
				carbonclientset:     tt.fields.carbonclientset,
				podLister:           tt.fields.podLister,
				podListerSynced:     tt.fields.podListerSynced,
				podInformer:         tt.fields.podInformer,
				workerLister:        tt.fields.workerLister,
				workerListerSynced:  tt.fields.workerListerSynced,
				serviceLister:       tt.fields.serviceLister,
				serviceListerSynced: tt.fields.serviceListerSynced,
				rollingSetLister:    tt.fields.rollingSetLister,
				rollingSetSynced:    tt.fields.rollingSetSynced,
			}
			c.DefaultController.ResourceManager = util.NewSimpleResourceManager(tt.fields.kubeclientset, tt.fields.carbonclientset, nil, nil, nil)
			if err := c.DeleteSubObj(tt.args.namespace, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Controller.DeleteSubObj() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
