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
	"flag"
	"fmt"
	"os"

	goflag "flag"

	"github.com/alibaba/kube-sharding/cmd/controllermanager/options"
	"github.com/alibaba/kube-sharding/common"
	"github.com/alibaba/kube-sharding/common/features"
	logrotate "github.com/alibaba/kube-sharding/pkg/ago-util/logging/rotate"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/proxy"
	"github.com/alibaba/kube-sharding/transfer"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	apiserverflag "k8s.io/component-base/cli/flag"
	utilflag "k8s.io/component-base/cli/flag"
	glog "k8s.io/klog"
)

func main() {
	utils.ReplaceStderr("stderr.rep", true)
	logFlags := flag.NewFlagSet("log", flag.ExitOnError)
	command := &cobra.Command{
		Use:     "carbon-proxy",
		Short:   "carbon-proxy",
		Long:    "carbon-proxy is used to adapt carbon2 to carbon3(c2).",
		Version: utils.GetVersion().String(),
		Run: func(command *cobra.Command, args []string) {
			if err := proxy.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}
	initFlags(command, logFlags)
	defer logrotate.CloseAll()
	defer glog.Flush()

	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func initFlags(command *cobra.Command, logFlags *flag.FlagSet) {
	glog.InitFlags(logFlags)
	logrotate.PatchKlogFlags(logFlags)

	namedFlagSets := apiserverflag.NamedFlagSets{}
	namedFlagSets.FlagSet("log").AddGoFlagSet(logFlags)
	common.InitFlags(namedFlagSets.FlagSet("common"))
	transfer.InitFlags(namedFlagSets.FlagSet("spec-transfer"))
	carbonv1.InitFlags(namedFlagSets.FlagSet("misc"))
	proxy.InitFlags(namedFlagSets.FlagSet("misc"))
	features.InitFlags(namedFlagSets.FlagSet("feature-gates"))

	pflag.CommandLine.SetNormalizeFunc(utilflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	options.PrintNamedFlagsetsSections(namedFlagSets, command, "c2-proxy")
}
