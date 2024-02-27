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

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	goflag "flag"

	"github.com/alibaba/kube-sharding/cmd/controllermanager/config"
	"github.com/alibaba/kube-sharding/cmd/controllermanager/controller"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/client/clientset/versioned/scheme"
	memclient "github.com/alibaba/kube-sharding/pkg/memkube/client"
	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	"github.com/alibaba/kube-sharding/pkg/memkube/memorize"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	utilflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	glog "k8s.io/klog"
)

type options struct {
	kubefile     string
	namespace    string
	descfile     string
	dry          bool
	memFsurl     string
	memResources string
}

var (
	opts options

	rootCmd = &cobra.Command{
		Use:   "memorize",
		Short: "memorize",
		Long:  "Memorize kubenetes resources to mem-kube resources.",
		Run: func(cmd *cobra.Command, args []string) {
			run()
		},
	}
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&opts.kubefile, "kubeconfig", "", "", "kubernetes config file")
	rootCmd.PersistentFlags().StringVarP(&opts.namespace, "namespace", "", "", "kubernetes namespace")
	rootCmd.PersistentFlags().StringVarP(&opts.descfile, "desc", "", "", "memorized descriptors file")
	rootCmd.PersistentFlags().BoolVarP(&opts.dry, "dry", "", false, "dry run")

	rootCmd.PersistentFlags().StringVarP(&opts.memFsurl, "mem-fsurl", "", "", "mem-kube fs-url")
	rootCmd.PersistentFlags().StringVarP(&opts.memResources, "mem-resources", "", "", "mem-kube resources, seperated by comma")
}

type app struct {
	memclient  *memclient.DynamicClient
	kubeclient dynamic.Interface
	namespace  string
}

func newApp(opts *options, stopCh <-chan struct{}) (*app, error) {
	app := &app{
		namespace: opts.namespace,
	}
	conf, err := app.parseConfig(opts)
	if err != nil {
		return nil, err
	}
	if err = app.init(conf, stopCh); err != nil {
		return nil, err
	}
	return app, nil
}

func (a *app) parseConfig(opts *options) (*config.Config, error) {
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", opts.kubefile)
	if err != nil {
		glog.Errorf("Build kube config error: %s, %v", opts.kubefile, err)
		return nil, err
	}
	kubeconfig.QPS = 10000
	kubeconfig.Burst = 10000
	kubeconfig.Timeout = time.Second * time.Duration(90)
	restconf := restclient.AddUserAgent(kubeconfig, "memorize")
	return &config.Config{
		Kubeconfig:   restconf,
		MemFsURL:     opts.memFsurl,
		MemObjs:      strings.Split(strings.ToLower(opts.memResources), ","),
		MemTimeoutMs: 10 * 1000,
		Namespace:    opts.namespace,
	}, nil
}

func (a *app) init(conf *config.Config, stopCh <-chan struct{}) error {
	var err error
	a.kubeclient, err = dynamic.NewForConfig(conf.Kubeconfig)
	if err != nil {
		glog.Errorf("Create kubernetes dynamic client error: %v", err)
		return err
	}
	client, err := controller.CreateLocalClient(conf, stopCh, true)
	if err != nil {
		glog.Errorf("Create local mem-client error: %v", err)
		return err
	}
	if err = client.Start(stopCh); err != nil {
		glog.Errorf("Start mem-client error: %v", err)
		return err
	}
	scheme.Scheme.AddKnownTypes(v1.SchemeGroupVersion, &v1.Pod{}, &v1.PodList{})
	scheme := mem.NewResourceScheme(scheme.Scheme)
	a.memclient = memclient.NewDynamicClient(client, scheme)
	return nil
}

func (a *app) process(descriptors []*memorize.Descriptor, dry bool) error {
	apiCreator := memorize.NewDynamicReaderWriterCreator(a.namespace, a.kubeclient, a.memclient)
	processors, err := memorize.CreateNodeProcessors(carbonv1.SchemeGroupVersion, descriptors, apiCreator)
	if err != nil {
		return err
	}
	return processors.Process(&memorize.ProcessOptions{Dry: dry, NumWorkers: 10})
}

func (a *app) loadDescriptors(file string) ([]*memorize.Descriptor, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var descriptors []*memorize.Descriptor
	if err = json.Unmarshal(b, &descriptors); err != nil {
		return nil, err
	}
	return descriptors, nil
}

func run() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGUSR1)
	stopCh := make(chan struct{})
	defer close(stopCh)

	app, err := newApp(&opts, stopCh)
	if err != nil {
		glog.Errorf("New app failed: %v", err)
		return
	}
	descs, err := app.loadDescriptors(opts.descfile)
	if err != nil {
		glog.Errorf("Load descriptors error: %v", err)
		return
	}
	go func() {
		defer close(sigCh)
		if err = app.process(descs, opts.dry); err != nil {
			glog.Errorf("Process error: %v", err)
			return
		}
		glog.Info("Memorize success")
	}()
	<-sigCh // wait for SIGINT
}

func main() {
	pflag.CommandLine.SetNormalizeFunc(utilflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	logs.InitLogs()
	defer logs.FlushLogs()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
