/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/scheme"
	rest "k8s.io/client-go/rest"
)

type CarbonV1Interface interface {
	RESTClient() rest.Interface
	CarbonJobsGetter
	RollingSetsGetter
	ServicePublishersGetter
	ShardGroupsGetter
	TemporaryConstraintsGetter
	WorkerNodesGetter
	WorkerNodeEvictionsGetter
}

// CarbonV1Client is used to interact with features provided by the carbon.taobao.com group.
type CarbonV1Client struct {
	restClient rest.Interface
}

func (c *CarbonV1Client) CarbonJobs(namespace string) CarbonJobInterface {
	return newCarbonJobs(c, namespace)
}

func (c *CarbonV1Client) RollingSets(namespace string) RollingSetInterface {
	return newRollingSets(c, namespace)
}

func (c *CarbonV1Client) ServicePublishers(namespace string) ServicePublisherInterface {
	return newServicePublishers(c, namespace)
}

func (c *CarbonV1Client) ShardGroups(namespace string) ShardGroupInterface {
	return newShardGroups(c, namespace)
}

func (c *CarbonV1Client) TemporaryConstraints(namespace string) TemporaryConstraintInterface {
	return newTemporaryConstraints(c, namespace)
}

func (c *CarbonV1Client) WorkerNodes(namespace string) WorkerNodeInterface {
	return newWorkerNodes(c, namespace)
}

func (c *CarbonV1Client) WorkerNodeEvictions(namespace string) WorkerNodeEvictionInterface {
	return newWorkerNodeEvictions(c, namespace)
}

// NewForConfig creates a new CarbonV1Client for the given config.
func NewForConfig(c *rest.Config) (*CarbonV1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &CarbonV1Client{client}, nil
}

// NewForConfigOrDie creates a new CarbonV1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *CarbonV1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new CarbonV1Client for the given RESTClient.
func New(c rest.Interface) *CarbonV1Client {
	return &CarbonV1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *CarbonV1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
