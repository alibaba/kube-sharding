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

// Package app implements a server that runs a set of active
// components.  This includes replication controllers, service endpoints and
// nodes.
package app

import (
	"net/http"

	"github.com/alibaba/kube-sharding/cmd/controllermanager/controller"

	"github.com/alibaba/kube-sharding/controller/carbonjob"
	healthcheck "github.com/alibaba/kube-sharding/controller/healthcheck"
	"github.com/alibaba/kube-sharding/controller/rollingset"
	publisher "github.com/alibaba/kube-sharding/controller/service-publisher"
	"github.com/alibaba/kube-sharding/controller/shardgroup"
	"github.com/alibaba/kube-sharding/controller/worker"
	workernodeeviction "github.com/alibaba/kube-sharding/controller/workernode-eviction"
	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	// _ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

// StartWorkerController start the worker controller
func StartWorkerController(ctx controller.Context) (http.Handler, bool, error) {
	ctl := worker.NewController(
		ctx.CarbonClient,
		ctx.KubeClient,
		ctx.CarbonInformerFactory.Carbon().V1().WorkerNodes(),
		ctx.CarbonInformerFactory.Carbon().V1().ServicePublishers(),
		ctx.CarbonInformerFactory.Carbon().V1().RollingSets(),
		ctx.InformerFactory.Core().V1().Pods(),
		ctx.SystemInformerFactory.Core().V1().ConfigMaps(),
		ctx.WriteLabels,
	)
	go ctl.Run(int(ctx.ConcurrentSyncs*20), ctx.Stop, ctx.Waiter)
	return nil, true, nil
}

// StartRollingsetController start the rollingset controller
func StartRollingsetController(ctx controller.Context) (http.Handler, bool, error) {
	go rollingset.NewController(
		ctx.Cluster,
		ctx.KubeClient,
		ctx.CarbonClient,
		ctx.CarbonInformerFactory.Carbon().V1().RollingSets(),
		ctx.CarbonInformerFactory.Carbon().V1().WorkerNodes(),
		ctx.WriteLabels,
	).Run(int(ctx.ConcurrentSyncs), ctx.Stop, ctx.Waiter)
	return nil, true, nil
}

// StartShardgroupController start the shardGroup controller
func StartShardgroupController(ctx controller.Context) (http.Handler, bool, error) {
	go shardgroup.NewController(
		ctx.KubeClient,
		ctx.CarbonClient,
		ctx.CarbonInformerFactory.Carbon().V1().ShardGroups(),
		ctx.CarbonInformerFactory.Carbon().V1().RollingSets(),
		ctx.CarbonInformerFactory.Carbon().V1().WorkerNodes(),
	).Run(int(ctx.ConcurrentSyncs), ctx.Stop, ctx.Waiter)
	return nil, true, nil
}

// StartServicePublisher start the replica controller
func StartServicePublisher(ctx controller.Context) (http.Handler, bool, error) {
	go publisher.NewController(
		ctx.KubeClient,
		ctx.CarbonClient,
		ctx.CarbonInformerFactory.Carbon().V1().WorkerNodes(),
		ctx.CarbonInformerFactory.Carbon().V1().ServicePublishers(),
		ctx.InformerFactory.Core().V1().Services(),
	).Run(int(ctx.ConcurrentSyncs), ctx.Stop, ctx.Waiter)
	return nil, true, nil
}

// StartHealthChecker start the health checker
func StartHealthChecker(ctx controller.Context) (http.Handler, bool, error) {
	go healthcheck.NewController(
		ctx.KubeClient,
		ctx.CarbonClient,
		ctx.CarbonInformerFactory.Carbon().V1().RollingSets(),
		ctx.CarbonInformerFactory.Carbon().V1().WorkerNodes(),
	).Run(1, ctx.Stop, ctx.Waiter)
	return nil, true, nil
}

// StartCarbonJobController StartCarbonJobController
func StartCarbonJobController(ctx controller.Context) (http.Handler, bool, error) {
	go carbonjob.NewController(
		ctx.KubeClient,
		ctx.CarbonClient,
		ctx.InformerFactory.Core().V1().Pods(),
		ctx.CarbonInformerFactory.Carbon().V1().WorkerNodes(),
		ctx.CarbonInformerFactory.Carbon().V1().CarbonJobs(),
	).Run(int(ctx.ConcurrentSyncs), ctx.Stop, ctx.Waiter)
	return nil, true, nil
}

// StartWorkerNodeEvictionController StartWorkerNodeEvictionController
func StartWorkerNodeEvictionController(ctx controller.Context) (http.Handler, bool, error) {
	go workernodeeviction.NewController(
		ctx.KubeClient,
		ctx.CarbonClient,
		ctx.InformerFactory.Core().V1().Pods(),
		ctx.CarbonInformerFactory.Carbon().V1().WorkerNodes(),
		ctx.CarbonInformerFactory.Carbon().V1().WorkerNodeEvictions(),
	).Run(int(ctx.ConcurrentSyncs), ctx.Stop, ctx.Waiter)
	return nil, true, nil
}

// NewControllerInitializers is a public map of named controller groups (you can start more than one in an init func)
// paired to their InitFunc.  This allows for structured downstream composition and subdivision.
func NewControllerInitializers() map[string]controller.InitFunc {
	controllers := map[string]controller.InitFunc{}
	controllers["rollingset"] = StartRollingsetController
	controllers["publisher"] = StartServicePublisher
	controllers["healthcheck"] = StartHealthChecker
	controllers["shardgroup"] = StartShardgroupController
	controllers["worker"] = StartWorkerController
	controllers["carbonjob"] = StartCarbonJobController
	controllers["workereviction"] = StartWorkerNodeEvictionController
	return controllers
}
