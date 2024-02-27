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

package service

import (
	carbonclientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	memclientset "github.com/alibaba/kube-sharding/pkg/mem-client/clientset"
	memclient "github.com/alibaba/kube-sharding/pkg/memkube/client"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	glog "k8s.io/klog"
)

// simpleControllerClientBuilder returns a fixed client with different user agents
type simpleControllerClientBuilder struct {
	// clientConfig is a skeleton config to clone and use as the basis for each controller client
	clientConfig *restclient.Config
}

// Config set the config
func (b simpleControllerClientBuilder) Config(name string) (*restclient.Config, error) {
	clientConfig := *b.clientConfig
	return restclient.AddUserAgent(&clientConfig, name), nil
}

// ConfigOrDie set the config or die
func (b simpleControllerClientBuilder) ConfigOrDie(name string) *restclient.Config {
	clientConfig, err := b.Config(name)
	if err != nil {
		glog.Fatal(err)
	}
	return clientConfig
}

// Client build a client
func (b simpleControllerClientBuilder) Client(name string) (clientset.Interface, error) {
	clientConfig, err := b.Config(name)
	if err != nil {
		return nil, err
	}
	return clientset.NewForConfig(clientConfig)
}

// ClientOrDie build a client or die
func (b simpleControllerClientBuilder) ClientOrDie(name string) clientset.Interface {
	client, err := b.Client(name)
	if err != nil {
		glog.Fatal(err)
	}
	return client
}

// CarbonClient build a client or die
func (b simpleControllerClientBuilder) CarbonClient(name string) (carbonclientset.Interface, error) {
	clientConfig, err := b.Config(name)
	if err != nil {
		return nil, err
	}
	clientConfig.ContentType = ""
	return carbonclientset.NewForConfig(clientConfig)
}

// CarbonClientOrDie build a client or die
func (b simpleControllerClientBuilder) CarbonClientOrDie(name string) carbonclientset.Interface {
	client, err := b.CarbonClient(name)
	if err != nil {
		glog.Fatal(err)
	}
	return client
}

// RemoteCarbonMixClient build a mix client or die
func (b simpleControllerClientBuilder) RemoteCarbonMixClient(name string, localClient memclient.Client, memObjs []string) (carbonclientset.Interface, error) {
	clientConfig, err := b.Config(name)
	if err != nil {
		return nil, err
	}
	clientConfig.ContentType = ""
	return memclientset.NewForConfig(clientConfig, localClient, memObjs)
}

// RemoteCarbonMixClientOrDie build a mix client or die
func (b simpleControllerClientBuilder) RemoteCarbonMixClientOrDie(name string, localClient memclient.Client, memObjs []string) carbonclientset.Interface {
	client, err := b.RemoteCarbonMixClient(name, localClient, memObjs)
	if err != nil {
		glog.Fatal(err)
	}
	return client
}
