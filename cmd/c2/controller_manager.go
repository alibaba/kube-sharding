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

// The controller manager is responsible for monitoring replication
// controllers, and creating corresponding pods to achieve the desired
// state.  It uses the API to listen for new controllers and to create/delete
// pods.
package main

import (
	goflag "flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/alibaba/kube-sharding/cmd/c2/app"
	"github.com/alibaba/kube-sharding/cmd/controllermanager/controller"
	logrotate "github.com/alibaba/kube-sharding/pkg/ago-util/logging/rotate"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/spf13/pflag"
	utilflag "k8s.io/component-base/cli/flag"
	glog "k8s.io/klog"
)

func main() {
	utils.ReplaceStderr("stderr.rep", true)
	rand.Seed(time.Now().UTC().UnixNano())
	command := controller.NewControllerManagerCommand(app.NewControllerInitializers())

	pflag.CommandLine.SetNormalizeFunc(utilflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)

	defer logrotate.CloseAll()
	defer glog.Flush()

	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
