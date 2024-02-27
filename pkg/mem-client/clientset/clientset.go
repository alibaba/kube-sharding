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

package clientset

import (
	carbonv1 "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/typed/carbon/v1"
	memclient "github.com/alibaba/kube-sharding/pkg/memkube/client"
	discovery "k8s.io/client-go/discovery"
	rest "k8s.io/client-go/rest"
	flowcontrol "k8s.io/client-go/util/flowcontrol"
)

// Clientset contains the clients for groups. Each group has exactly one
// version included in a Clientset.
type Clientset struct {
	*discovery.DiscoveryClient
	carbonV1 carbonv1.CarbonV1Interface
}

// CarbonV1 retrieves the CarbonV1Client
func (c *Clientset) CarbonV1() carbonv1.CarbonV1Interface {
	return c.carbonV1
}

// Carbon retrieves the default version of CarbonClient.
// Deprecated: Please explicitly pick a version.
func (c *Clientset) Carbon() carbonv1.CarbonV1Interface {
	return c.carbonV1
}

// Discovery retrieves the DiscoveryClient
func (c *Clientset) Discovery() discovery.DiscoveryInterface {
	if c == nil {
		return nil
	}
	return c.DiscoveryClient
}

// NewForConfig creates a new Clientset for the given config.
func NewForConfig(c *rest.Config, localClient memclient.Client, memObjs []string) (*Clientset, error) {
	configShallowCopy := *c
	if configShallowCopy.RateLimiter == nil && configShallowCopy.QPS > 0 {
		configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)
	}
	var cs Clientset
	var err error
	carbonV1, err := carbonv1.NewForConfig(&configShallowCopy)
	cs.carbonV1 = carbonV1
	if err != nil {
		return nil, err
	}
	cs.DiscoveryClient, err = discovery.NewDiscoveryClientForConfig(&configShallowCopy)
	if err != nil {
		return nil, err
	}
	return &cs, nil
}
