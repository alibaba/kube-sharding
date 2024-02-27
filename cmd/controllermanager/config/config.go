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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	restclient "k8s.io/client-go/rest"
	config "k8s.io/component-base/config"
)

// GenericControllerManagerConfiguration holds configuration for a generic controller-manager
type GenericControllerManagerConfiguration struct {
	// port is the port that the controller-manager's http service runs on.
	Port int32
	// address is the IP address to serve on (set to 0.0.0.0 for all interfaces).
	Address string
	// minResyncPeriod is the resync period in reflectors; will be random between
	// minResyncPeriod and 2*minResyncPeriod.
	MinResyncPeriod metav1.Duration
	// ClientConnection specifies the kubeconfig file and client connection
	// settings for the proxy server to use when communicating with the apiserver.
	ClientConnection config.ClientConnectionConfiguration
	// leaderElection defines the configuration of leader election client.
	LeaderElection config.LeaderElectionConfiguration
	// Controllers is the list of controllers to enable or disable
	// '*' means "all enabled by default controllers"
	// 'foo' means "enable 'foo'"
	// '-foo' means "disable 'foo'"
	// first item for a particular name wins
	Controllers []string
}

// Config is the main context object for the controller manager.
type Config struct {
	// Generic holds configuration for a generic controller-manager
	Generic GenericControllerManagerConfiguration

	// the rest config for the master
	Kubeconfig *restclient.Config

	// LabelSelector and Namespace defines the scope of controller
	LabelSelector string
	WriteLabels   string
	Namespace     string
	Cluster       string
	Name          string

	// Current numbers
	ConcurrentSyncs int32

	//ExtendConfig define the exetrnal config file path for controllers
	ExtendConfig string

	// delay duration before controllers start after became to leader
	DelayStart metav1.Duration

	// KmonAdress kmon service host
	// default=localhost
	KmonAddress string

	MemObjs                []string
	MemFsURL               string
	MemFsBackupURL         string
	MemSyncPeriodMs        int32
	MemTimeoutMs           int32
	MemBatcherHashKeyNum   uint32
	MemBatcherGroupKey     string
	MemStorePoolWorkerSize int
	MemStorePoolQueueSize  int
}

// TweakListOptionsFunc used to defines the scope of controller
func (c *Config) TweakListOptionsFunc(options *v1.ListOptions) {
	if "" == options.LabelSelector && "" != c.LabelSelector {
		options.LabelSelector = c.LabelSelector
	}
}
