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
	"github.com/spf13/pflag"
)

var opts struct {
	memObjs    string
	memTimeout int
	timeout    int // also used in batcher
	selector   string
	namespace  string

	nsRouteStrategy string
	nsRouteConfKey  string // Namespace/ConfigMap-name
	globalConfigKey string

	memPodNamespaces string // load pods from remote memkube server

	maxSubrsRollingSteps int // subrs rolling steps

	subrsCopyGroupRSKeys string // subrs copy group rs labels keys
}

// InitFlags setup command line flags.
func InitFlags(flagset *pflag.FlagSet) {
	flagset.StringVar(&opts.memObjs, "mem-objs", opts.memObjs, "The mem-objs of memkube resource.")
	flagset.IntVar(&opts.memTimeout, "mem-timeout-second", opts.memTimeout, "The mem-timeout-second of memkube client timeout.")
	flagset.IntVar(&opts.timeout, "proxy-k8s-timeout", 90, "query k8s apiserver timeout")
	flagset.IntVar(&opts.maxSubrsRollingSteps, "max-subrs-rolling-steps", 10, "max steps of subrs to rollings")
	flagset.StringVar(&opts.selector, "selector", "", "cache selector")
	flagset.StringVar(&opts.nsRouteStrategy, "ns-route-strategy", "", "namespace route strategy")
	flagset.StringVar(&opts.nsRouteConfKey, "ns-route-conf", "", "config routing key name in format: namespace/key-name")

	flagset.StringVar(&opts.namespace, "namespace", "", "cache namespace")
	flagset.StringVar(&opts.memPodNamespaces, "mem-pod-namespaces", "", "which namespaces in `n1,n2` format, to load pods from remote memkube server")

	flagset.StringVar(&opts.subrsCopyGroupRSKeys, "subrs-copy-grouprs-keys", "", "sync subrs copy group rollingset labels keys")
}
