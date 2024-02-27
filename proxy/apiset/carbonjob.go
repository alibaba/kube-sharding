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

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	v1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	carbonclientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	"github.com/alibaba/kube-sharding/transfer"
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// CarbonJobAPIs wrap carbonjob apis
type CarbonJobAPIs interface {
	GetCarbonJob(namespace, name string) (*v1.CarbonJob, error)
	GetCarbonJobs(namespace, appName string) ([]v1.CarbonJob, error)
	CreateCarbonJob(cb *v1.CarbonJob) (*v1.CarbonJob, error)
	UpdateCarbonJob(cb *v1.CarbonJob) error
	DeleteCarbonJob(cb *v1.CarbonJob) error
}

type carbonJobAPIs struct {
	clientset carbonclientset.Interface
}

// NewCarbonJobAPIs creates carbonjob apis
func NewCarbonJobAPIs(clientset carbonclientset.Interface) CarbonJobAPIs {
	return &carbonJobAPIs{
		clientset: clientset,
	}
}

func (c *carbonJobAPIs) GetCarbonJob(namespace, name string) (*v1.CarbonJob, error) {
	name = transfer.EscapeName(name)
	carbonJob, err := c.clientset.CarbonV1().CarbonJobs(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if nil != carbonJob {
		if glog.V(4) {
			glog.Infof("Get carbonJob %s %s success, groupCount %d", namespace, name, carbonJob.GetTaskSize())
		}
	}
	return carbonJob, err
}

func (c *carbonJobAPIs) GetCarbonJobs(namespace, appName string) ([]v1.CarbonJob, error) {
	labelSelector := metav1.SetAsLabelSelector(labels.Set{})
	v1.SetHippoLabelSelector(labelSelector, v1.LabelKeyAppNameHash, v1.LabelValueHash(appName, true))
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		glog.Errorf("Format label selector error: %s, %v", appName, err)
		return nil, err
	}
	carbonJobs, err := c.clientset.CarbonV1().CarbonJobs(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: selector.String()})
	if nil != err {
		glog.Errorf("get carbonjobs error : %s, %v", appName, err)
		return nil, err
	}
	return carbonJobs.Items, err
}

func (c *carbonJobAPIs) CreateCarbonJob(cb *v1.CarbonJob) (*v1.CarbonJob, error) {
	v, err := c.clientset.CarbonV1().CarbonJobs(cb.GetNamespace()).Create(context.Background(), cb, metav1.CreateOptions{})
	if glog.V(4) {
		glog.Infof("ResourceManager createCarbonJob cb %s, err: %v", utils.ObjJSON(cb), err)
	}
	return v, err
}

func (c *carbonJobAPIs) UpdateCarbonJob(cb *v1.CarbonJob) error {
	_, err := c.clientset.CarbonV1().CarbonJobs(cb.GetNamespace()).Update(context.Background(), cb, metav1.UpdateOptions{})
	if glog.V(4) {
		glog.Infof("ResourceManager updateCarbonJob cb %s, err: %v", utils.ObjJSON(cb), err)
	}
	return err
}

func (c *carbonJobAPIs) DeleteCarbonJob(cb *v1.CarbonJob) error {
	err := c.clientset.CarbonV1().CarbonJobs(cb.GetNamespace()).Delete(context.Background(), cb.GetName(), metav1.DeleteOptions{})
	if glog.V(4) {
		glog.Infof("ResourceManager deleteCarbonJob cb %s, err: %v", utils.ObjJSON(cb), err)
	}
	return err
}
