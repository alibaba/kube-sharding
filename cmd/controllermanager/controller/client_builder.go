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
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	glog "k8s.io/klog"

	carbonclientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	"github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/scheme"
	memclient "github.com/alibaba/kube-sharding/pkg/memkube/client"
	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	clientset "k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

// ClientBuilder allows you to get clients and configs for controllers
type ClientBuilder interface {
	Client(name string) (clientset.Interface, error)
	ClientOrDie(name string) clientset.Interface
	CarbonClient(name string) (carbonclientset.Interface, error)
	CarbonClientOrDie(name string) carbonclientset.Interface
	DynamicClient(name string) (dynamic.Interface, error)
	DynamicClientOrDie(name string) dynamic.Interface
}

// SimpleControllerClientBuilder returns a fixed client with different user agents
type SimpleControllerClientBuilder struct {
	ClientConfig *restclient.Config
	localClient  memclient.Client
	memObjs      []string
}

// Config set the config
func (b SimpleControllerClientBuilder) Config(name string) (*restclient.Config, error) {
	clientConfig := *b.ClientConfig
	return restclient.AddUserAgent(&clientConfig, name), nil
}

// ConfigOrDie set the config or die
func (b SimpleControllerClientBuilder) ConfigOrDie(name string) *restclient.Config {
	clientConfig, err := b.Config(name)
	if err != nil {
		glog.Fatal(err)
	}
	return clientConfig
}

// Client build a client
func (b SimpleControllerClientBuilder) Client(name string) (clientset.Interface, error) {
	clientConfig, err := b.Config(name)
	if err != nil {
		return nil, err
	}
	return clientset.NewForConfig(clientConfig)
}

// ClientOrDie build a client or die
func (b SimpleControllerClientBuilder) ClientOrDie(name string) clientset.Interface {
	client, err := b.Client(name)
	if err != nil {
		glog.Fatal(err)
	}
	return client
}

// CarbonClient build a client or die
func (b SimpleControllerClientBuilder) CarbonClient(name string) (carbonclientset.Interface, error) {
	clientConfig, err := b.Config(name)
	if err != nil {
		return nil, err
	}
	clientConfig.ContentType = ""
	return carbonclientset.NewForConfig(clientConfig)
}

// CarbonClientOrDie build a client or die
func (b SimpleControllerClientBuilder) CarbonClientOrDie(name string) carbonclientset.Interface {
	client, err := b.CarbonClient(name)
	if err != nil {
		glog.Fatal(err)
	}
	return client
}

// DynamicClient build a dynamic client
func (b SimpleControllerClientBuilder) DynamicClient(name string) (dynamic.Interface, error) {
	clientConfig, err := b.Config(name)
	if err != nil {
		return nil, err
	}
	scheme := mem.NewResourceScheme(scheme.Scheme)
	gvks := make([]schema.GroupVersionKind, 0)
	for _, o := range b.memObjs {
		gvks = append(gvks, getGroupVersionKind(strings.ToLower(o)))
	}
	return memclient.NewDynamicMixResource(b.localClient, clientConfig, scheme, gvks)
}

// DynamicClientOrDie build a dynamic client or die
func (b SimpleControllerClientBuilder) DynamicClientOrDie(name string) dynamic.Interface {
	client, err := b.DynamicClient(name)
	if err != nil {
		glog.Fatal(err)
	}
	return client
}
