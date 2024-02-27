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

// Package options provides the flags used for the controller manager.
package options

import (
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/alibaba/kube-sharding/cmd/controllermanager/config"
	"k8s.io/client-go/tools/clientcmd"
	apiserverflag "k8s.io/component-base/cli/flag"

	k8sconfig "k8s.io/component-base/config"

	"github.com/spf13/pflag"
)

// KubeControllerManagerOptions is the main context object for the kube-controller manager.
type KubeControllerManagerOptions struct {
	Generic *GenericControllerManagerConfigurationOptions

	Master     string
	Kubeconfig string

	LabelSelector   string
	WriteLabels     string
	Namespace       string
	Cluster         string
	Name            string
	ConcurrentSyncs int32

	//ExtendConfig define the exetrnal config file path for controllers
	ExtendConfig string

	KmonAddress string

	// delay duration before controllers start after became to leader
	DelayStart metav1.Duration

	MemObjs                string
	MemFsURL               string
	MemFsBackupURL         string
	MemSyncPeriodMs        int32
	MemTimeoutMs           int32
	MemBatcherHashKeyNum   uint32
	MemBatcherGroupKey     string
	MemStorePoolWorkerSize int
	MemStorePoolQueueSize  int
}

// NewKubeControllerManagerOptions creates a new KubeControllerManagerOptions with a default config.
func NewKubeControllerManagerOptions() (*KubeControllerManagerOptions, error) {
	componentConfig, err := NewDefaultComponentConfig()
	if err != nil {
		return nil, err
	}

	s := KubeControllerManagerOptions{
		Generic: NewGenericControllerManagerConfigurationOptions(componentConfig.Generic),
	}

	return &s, nil
}

// NewDefaultComponentConfig returns kube-controller manager configuration object.
func NewDefaultComponentConfig() (*config.Config, error) {
	var config config.Config
	config.Generic.ClientConnection.ContentType = "application/vnd.kubernetes.protobuf"
	return &config, nil
}

// Flags returns flags for a specific APIServer by section name
func (s *KubeControllerManagerOptions) Flags(allControllers []string, disabledByDefaultControllers []string) apiserverflag.NamedFlagSets {
	fss := apiserverflag.NamedFlagSets{}
	s.Generic.AddFlags(&fss, allControllers, disabledByDefaultControllers)

	fs := fss.FlagSet("misc")
	fs.StringVar(&s.Master, "master", s.Master, "The address of the Kubernetes API server (overrides any value in kubeconfig).")
	fs.StringVar(&s.Kubeconfig, "kubeconfig", s.Kubeconfig, "Path to kubeconfig file with authorization and master location information.")

	fs.StringVar(&s.LabelSelector, "selector", s.LabelSelector, "The selector of controller selected and watched.")
	fs.StringVar(&s.WriteLabels, "write-labels", s.WriteLabels, "The labels of controller write to resource.")
	fs.StringVar(&s.Namespace, "namespace", s.Namespace, "The namespace of controller selected and watched.")
	fs.StringVar(&s.Cluster, "cluster", s.Cluster, "The cluster of controller.")
	fs.StringVar(&s.Name, "name", s.Cluster, "The name of controllermanager, used for leader elect.")
	fs.Int32Var(&s.ConcurrentSyncs, "concurrent", 5, "Concurrent workers count.")
	fs.StringVar(&s.ExtendConfig, "extendconfig", s.ExtendConfig, "Path to extend config file.")

	fs.DurationVar(&s.DelayStart.Duration, "delay-start", s.DelayStart.Duration, "Delay duration before controllers start after became to leader.")
	fs.StringVar(&s.KmonAddress, "kmon-address", s.KmonAddress, "The address of the Kmonitor server.")

	memFs := fss.FlagSet("mem")
	memFs.StringVar(&s.MemObjs, "mem-objs", s.MemObjs, "The mem-objs of memkube resource.")
	memFs.StringVar(&s.MemFsURL, "mem-fsurl", s.MemFsURL, "The fsurl of memkube persister url.")
	memFs.StringVar(&s.MemFsBackupURL, "mem-fsbackupurl", s.MemFsBackupURL, "The fsurl of memkube persister backup url.")
	memFs.Int32Var(&s.MemSyncPeriodMs, "mem-syncperiod-ms", s.MemSyncPeriodMs, "The mem-sync-period of memkube persist sync period.")
	memFs.Int32Var(&s.MemTimeoutMs, "mem-timeout-ms", s.MemTimeoutMs, "The mem-timeout-ms of memkube client timeout.")
	memFs.Uint32Var(&s.MemBatcherHashKeyNum, "mem-batcher-hash-key-num", s.MemBatcherHashKeyNum, "The mem batcher hash key num")
	memFs.StringVar(&s.MemBatcherGroupKey, "mem-batcher-group-key", s.MemBatcherGroupKey, "The mem batcher group key.")
	memFs.IntVar(&s.MemStorePoolWorkerSize, "mem-store-pool-worker-size", s.MemStorePoolWorkerSize, "The mem store pool worker size.")
	memFs.IntVar(&s.MemStorePoolQueueSize, "mem-store-pool-queue-size", s.MemStorePoolQueueSize, "The mem store pool queue size.")
	return fss
}

// ApplyTo fills up controller manager config with options.
func (s *KubeControllerManagerOptions) ApplyTo(c *config.Config) error {
	if err := s.Generic.ApplyTo(&c.Generic); err != nil {
		return err
	}

	return nil
}

// Config return a controller manager config objective
func (s KubeControllerManagerOptions) Config() (*config.Config, error) {
	kubeconfig, err := clientcmd.BuildConfigFromFlags(s.Master, s.Kubeconfig)
	if err != nil {
		return nil, err
	}
	kubeconfig.ContentConfig.ContentType = s.Generic.ClientConnection.ContentType
	kubeconfig.QPS = s.Generic.ClientConnection.QPS
	kubeconfig.Burst = int(s.Generic.ClientConnection.Burst)

	var memObjArray []string
	if s.MemObjs != "" {
		memObjArray = strings.Split(strings.ToLower(s.MemObjs), ",")
	}

	c := &config.Config{
		Kubeconfig:             kubeconfig,
		Namespace:              s.Namespace,
		Cluster:                s.Cluster,
		Name:                   s.Name,
		LabelSelector:          s.LabelSelector,
		WriteLabels:            s.WriteLabels,
		ConcurrentSyncs:        s.ConcurrentSyncs,
		ExtendConfig:           s.ExtendConfig,
		DelayStart:             s.DelayStart,
		KmonAddress:            s.KmonAddress,
		MemObjs:                memObjArray,
		MemFsURL:               s.MemFsURL,
		MemFsBackupURL:         s.MemFsBackupURL,
		MemSyncPeriodMs:        s.MemSyncPeriodMs,
		MemTimeoutMs:           s.MemTimeoutMs,
		MemBatcherHashKeyNum:   s.MemBatcherHashKeyNum,
		MemBatcherGroupKey:     s.MemBatcherGroupKey,
		MemStorePoolWorkerSize: s.MemStorePoolWorkerSize,
		MemStorePoolQueueSize:  s.MemStorePoolQueueSize,
	}
	if err := s.ApplyTo(c); err != nil {
		return nil, err
	}

	return c, nil
}

// GenericControllerManagerConfigurationOptions holds the options which are generic.
type GenericControllerManagerConfigurationOptions struct {
	Port             int32
	Address          string
	MinResyncPeriod  metav1.Duration
	ClientConnection k8sconfig.ClientConnectionConfiguration
	LeaderElection   k8sconfig.LeaderElectionConfiguration
	Controllers      []string
}

// NewGenericControllerManagerConfigurationOptions returns generic configuration default values for both
// the kube-controller-manager and the cloud-contoller-manager. Any common changes should
// be made here. Any individual changes should be made in that controller.
func NewGenericControllerManagerConfigurationOptions(cfg config.GenericControllerManagerConfiguration) *GenericControllerManagerConfigurationOptions {
	o := &GenericControllerManagerConfigurationOptions{
		Port:             cfg.Port,
		Address:          cfg.Address,
		ClientConnection: cfg.ClientConnection,
		LeaderElection:   cfg.LeaderElection,
		Controllers:      cfg.Controllers,
	}

	return o
}

// AddFlags adds flags related to generic for controller manager to the specified FlagSet.
func (o *GenericControllerManagerConfigurationOptions) AddFlags(fss *apiserverflag.NamedFlagSets, allControllers, disabledByDefaultControllers []string) {
	if o == nil {
		return
	}

	genericfs := fss.FlagSet("generic")
	genericfs.DurationVar(&o.MinResyncPeriod.Duration, "min-resync-period", o.MinResyncPeriod.Duration, "The resync period in reflectors will be random between MinResyncPeriod and 2*MinResyncPeriod.")
	genericfs.StringVar(&o.ClientConnection.ContentType, "kube-api-content-type", o.ClientConnection.ContentType, "Content type of requests sent to apiserver.")
	genericfs.Float32Var(&o.ClientConnection.QPS, "kube-api-qps", o.ClientConnection.QPS, "QPS to use while talking with kubernetes apiserver.")
	genericfs.Int32Var(&o.ClientConnection.Burst, "kube-api-burst", o.ClientConnection.Burst, "Burst to use while talking with kubernetes apiserver.")

	genericfs.StringSliceVar(&o.Controllers, "controllers", o.Controllers, "A list of controllers to enable. '*' enables all on-by-default controllers, 'foo' enables the controller named 'foo'.")
	genericfs.StringVar(&o.Address, "address", "0.0.0.0", "the http server address of the controller.")
	genericfs.Int32Var(&o.Port, "port", o.Port, "the http server port of the controller.")

	bindFlags(&o.LeaderElection, genericfs)
}

// bindFlags binds the LeaderElectionConfiguration struct fields to a flagset
func bindFlags(l *k8sconfig.LeaderElectionConfiguration, fs *pflag.FlagSet) {
	fs.BoolVar(&l.LeaderElect, "leader-elect", true, ""+
		"Start a leader election client and gain leadership before "+
		"executing the main loop. Enable this when running replicated "+
		"components for high availability.")
	fs.DurationVar(&l.LeaseDuration.Duration, "leader-elect-lease-duration", time.Second*15, ""+
		"The duration that non-leader candidates will wait after observing a leadership "+
		"renewal until attempting to acquire leadership of a led but unrenewed leader "+
		"slot. This is effectively the maximum duration that a leader can be stopped "+
		"before it is replaced by another candidate. This is only applicable if leader "+
		"election is enabled.")
	fs.DurationVar(&l.RenewDeadline.Duration, "leader-elect-renew-deadline", time.Second*10, ""+
		"The interval between attempts by the acting master to renew a leadership slot "+
		"before it stops leading. This must be less than or equal to the lease duration. "+
		"This is only applicable if leader election is enabled.")
	fs.DurationVar(&l.RetryPeriod.Duration, "leader-elect-retry-period", time.Second*2, ""+
		"The duration the clients should wait between attempting acquisition and renewal "+
		"of a leadership. This is only applicable if leader election is enabled.")
	fs.StringVar(&l.ResourceLock, "leader-elect-resource-lock", "configmaps", ""+
		"The type of resource object that is used for locking during "+
		"leader election. Supported options are `endpoints` (default) and `configmaps`.")
}

// ApplyTo fills up generic config with options.
func (o *GenericControllerManagerConfigurationOptions) ApplyTo(cfg *config.GenericControllerManagerConfiguration) error {
	if o == nil {
		return nil
	}

	cfg.Port = o.Port
	cfg.Address = o.Address
	if "" == cfg.Address {
		cfg.Address = "0.0.0.0"
	}
	cfg.ClientConnection = o.ClientConnection
	cfg.LeaderElection = o.LeaderElection
	cfg.Controllers = o.Controllers
	cfg.MinResyncPeriod = o.MinResyncPeriod
	return nil
}
