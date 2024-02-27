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

package config

import (
	"errors"
	"time"

	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	informerv1 "k8s.io/client-go/informers/core/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	glog "k8s.io/klog"
)

var NotFoundError = errors.New("not found config")

// KubeConfiger parse configs in 1 ConfigMap
type KubeConfiger struct {
	name   string
	lister listerv1.ConfigMapNamespaceLister
}

// KubeConfigerFactory creates KubeConfiger
type KubeConfigerFactory struct {
	informerFactory informers.SharedInformerFactory
	configMapLister listerv1.ConfigMapLister
	ns              string
}

// NewKubeConfigerFactory creates a kubenetes configer factory.
func NewKubeConfigerFactory(client kubeclientset.Interface, namespace string, selectorKv map[string]string) *KubeConfigerFactory {
	resyncPeriod := time.Duration(0)
	tweakListOpts := func(opts *metav1.ListOptions) {
		if len(selectorKv) > 0 {
			opts.LabelSelector = metav1.FormatLabelSelector(&metav1.LabelSelector{MatchLabels: selectorKv})
		}
	}
	glog.Infof("Create kube config factory: %s, %+v", namespace, selectorKv)
	configmapInformers := informers.NewFilteredSharedInformerFactory(client, resyncPeriod, namespace, tweakListOpts)
	return &KubeConfigerFactory{
		informerFactory: configmapInformers,

		ns: namespace,
	}
}

// NewKubeConfigerFactory creates a kubenetes configer factory from configmap lister.
func NewKubeConfigerFactoryFromLister(lister listerv1.ConfigMapLister, namespace string) *KubeConfigerFactory {
	glog.Infof("Create kube config factory: %s", namespace)
	return &KubeConfigerFactory{
		configMapLister: lister,
		ns:              namespace,
	}
}

// Start starts the configer factory.
func (f *KubeConfigerFactory) Start(stopCh <-chan struct{}) {
	// register before start
	if f.configMapLister != nil {
		return
	}
	if f.informerFactory == nil {
		return
	}
	f.informer().Informer()
	f.informerFactory.Start(stopCh)
	glog.Infof("Wait configMapFactory ready...")
	cache.WaitForCacheSync(stopCh, f.informer().Informer().HasSynced)
}

// Configer gets a kube configer.
func (f *KubeConfigerFactory) Configer(name string) Configer {
	if f.configMapLister != nil {
		return &KubeConfiger{
			name:   name,
			lister: f.configMapLister.ConfigMaps(f.ns),
		}
	}
	return &KubeConfiger{
		name:   name,
		lister: f.informer().Lister().ConfigMaps(f.ns),
	}
}

func (f *KubeConfigerFactory) informer() informerv1.ConfigMapInformer {
	return f.informerFactory.Core().V1().ConfigMaps()
}

// Start starts the configer
func (c *KubeConfiger) Start(<-chan struct{}) {

}

// Get gets a config in ConfigMap.
func (c *KubeConfiger) Get(key string, result interface{}) error {
	s, err := c.GetString(key)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(s), result)
}

// GetVal get string value from configmap
func (c *KubeConfiger) GetString(key string) (string, error) {
	configmap, err := c.lister.Get(c.name)
	if err != nil {
		return "", err
	}
	if configmap.Data == nil {
		return "", errors.New("empty configMap data")
	}
	s, ok := configmap.Data[key]
	if !ok {
		return "", NotFoundError
	}
	return s, nil
}

// GetData return all datas
func (c *KubeConfiger) GetData() (map[string]string, error) {
	configmap, err := c.lister.Get(c.name)
	if err != nil {
		return nil, err
	}
	return configmap.Data, nil
}
