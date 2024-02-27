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
	"context"
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/alibaba/kube-sharding/pkg/memkube/client"
	"github.com/alibaba/kube-sharding/pkg/memkube/mem"
	"github.com/alibaba/kube-sharding/pkg/memkube/rpc"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon"
	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	"github.com/emicklei/go-restful"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	glog "k8s.io/klog"
)

var opts struct {
	port int
	// e.g: zfs://host:port/memkube/demo?connectTimeout=10
	fs string

	nodesCount int // > 0 to launch test nodes writer
	shardCount int
	namespace  string
	byIndex    bool
	dupCache   bool
}

const (
	shardLabelKey = "shard-index"
	updateTmKey   = "updated-at"
)

func init() {
	flag.IntVar(&opts.nodesCount, "n", opts.nodesCount, "node count to test")
	flag.IntVar(&opts.shardCount, "shd", 1, "node shard count to benchmark test")
	flag.IntVar(&opts.port, "p", 8176, "http port to listen")
	flag.BoolVar(&opts.byIndex, "idx", opts.byIndex, "list nodes by index")
	flag.BoolVar(&opts.dupCache, "dup", opts.dupCache, "duplicate cache")
	flag.StringVar(&opts.fs, "f", "", "file system url to persist")
	flag.StringVar(&opts.namespace, "s", "ns1", "namespace")
}

func initNodes(store mem.Store, n int, shd int, ns string, wait time.Duration) {
	for i := 0; i < n; i++ {
		node := &spec.WorkerNode{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("node-%d", i),
				Namespace: ns,
				Labels: map[string]string{
					shardLabelKey: fmt.Sprintf("shard-%d", i/shd),
				},
			},
			Nick: fmt.Sprintf("nick-%d", i),
			Age:  i,
			Status: spec.WorkerNodeStatus{
				Health: spec.HealthCondition{
					Metas:   map[string]string{"name": "emily willlis"},
					Success: true,
				},
			},
		}
		if _, err := store.Add(node); err != nil {
			glog.Errorf("Add node error: %s, %v", node.Name, err)
		}
	}
	time.Sleep(wait)
}

func assert(exp bool, msg string) {
	if !exp {
		panic(errors.New(msg))
	}
}

func verifyNodes(lister carbon.WorkerNodeNamespaceLister, n int, ns string) {
	nodes, err := lister.List(labels.Everything())
	assert(err == nil, "list workerNodes error")
	glog.Infof("List workerNodes count: %d", len(nodes))
	for _, node := range nodes {
		glog.Infof("List node: %+v", *node)
	}
}

func listNodes(idx int, wi carbon.WorkerNodeInterface, lister carbon.WorkerNodeNamespaceLister, indexer cache.Indexer) ([]*spec.WorkerNode, error) {
	shardStr := fmt.Sprintf("shard-%d", idx)
	if opts.byIndex {
		items, err := indexer.Index(shardLabelKey, shardStr)
		if err != nil {
			glog.Warningf("Index nodes error: %v", err)
			return nil, err
		}
		nodes := make([]*spec.WorkerNode, 0, len(items))
		for _, item := range items {
			nodes = append(nodes, item.(*spec.WorkerNode))
		}
		return nodes, nil
	}
	selector := labels.SelectorFromSet(map[string]string{
		shardLabelKey: shardStr,
	})
	nodes, err := lister.List(selector)
	if err != nil {
		glog.Warningf("List nodes error: %v", err)
		return nil, err
	}
	return nodes, nil
}

func runShardNodeOp(idx int, wi carbon.WorkerNodeInterface, lister carbon.WorkerNodeNamespaceLister, indexer cache.Indexer, ns string, readonly bool) {
	nodes, err := listNodes(idx, wi, lister, indexer)
	if err != nil {
		return
	}
	if readonly {
		return
	}
	var wg sync.WaitGroup
	for i := range nodes {
		node := nodes[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			node0 := node.DeepCopyObject().(*spec.WorkerNode)
			node0.Labels[updateTmKey] = strconv.FormatInt(time.Now().UnixNano()/1000/1000, 10)
			node0.Age++
			_, err = wi.Update(node0)
			if err != nil {
				glog.Warningf("Update node error: %s, %v", node0.Name, err)
			}
		}()
	}
	wg.Wait()
}

func launchOp(idx int, wi carbon.WorkerNodeInterface, lister carbon.WorkerNodeNamespaceLister, indexer cache.Indexer, ns string, readonly bool, stopCh <-chan struct{}) {
	tm := rand.Int() % 200
	time.Sleep(time.Millisecond * time.Duration(tm))
	glog.Infof("Launch node Op %d, readonly %v", idx, readonly)
	go wait.Until(func() {
		runShardNodeOp(idx, wi, lister, indexer, ns, readonly)
	}, time.Second*3, stopCh)
}

func bindEventHandler(informer cache.SharedInformer) {
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if glog.V(5) {
				glog.Infof("Handler on object added: %+v", obj)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if glog.V(5) {
				glog.Infof("Handler on object updated: %+v", newObj)
			}
			node := newObj.(*spec.WorkerNode)
			i, _ := strconv.ParseInt(node.Labels[updateTmKey], 10, 64)
			if elapsed := time.Now().UnixNano()/1000/1000 - i; i > 0 && elapsed > 1000 {
				glog.Warningf("Recv updated event delayed: %s, %d ms, ver %s", node.Name, elapsed, node.ResourceVersion)
			}
		},
		DeleteFunc: func(obj interface{}) {
			if glog.V(5) {
				glog.Infof("Handle to delete object: %+v", obj)
			}
		},
	})
}

func startService(client client.Client, stopCh chan struct{}) {
	wsContainer := restful.NewContainer()
	scheme := mem.NewResourceScheme(spec.Scheme)

	service := rpc.NewService(client, scheme, spec.Codecs, &rpc.ServiceOptions{})

	webSvc := service.GetHandlerService()
	wsContainer.Add(webSvc)
	wsContainer.EnableContentEncoding(true)
	var mux http.ServeMux
	mux.Handle("/", wsContainer)
	for p, h := range utils.PProfHandlers() {
		mux.Handle(p, h)
	}
	server := &http.Server{Addr: fmt.Sprintf(":%d", opts.port), Handler: &mux}
	go server.ListenAndServe()
	glog.Infof("Web server starts")
	go func() {
		<-stopCh
		server.Shutdown(context.Background())
	}()
}

func setupErrHandlers() {
	utilruntime.ErrorHandlers = append(utilruntime.ErrorHandlers, func(err error) {
		glog.ErrorDepth(2, err)
	})
}

func shardIndexerFunc(obj interface{}) ([]string, error) {
	if s, ok := obj.(string); ok {
		return []string{s}, nil
	}
	m, err := meta.Accessor(obj)
	if err != nil {
		return []string{""}, err
	}
	return []string{m.GetLabels()[shardLabelKey]}, nil
}

func main() {
	setupErrHandlers()
	flag.Parse()
	rand.Seed(time.Now().Unix())
	defer glog.Flush()
	stopCh := make(chan struct{})
	namespace := opts.namespace
	count := opts.nodesCount
	syncPeriod := time.Millisecond * 500
	kind := mem.Kind(&spec.WorkerNode{})
	// all resources share the same local client in server.
	client := client.NewLocalClient(syncPeriod * 10)
	// create persister
	persister, err := mem.NewFsPersister(opts.fs, "", namespace, spec.SchemeGroupVersion.WithKind(kind), nil)
	if err != nil {
		panic(err)
	}
	batcher := &mem.FuncBatcher{
		KeyFunc: mem.KeyFuncLabelValue(shardLabelKey),
	}
	// for a specific resource in some namespace.
	store := mem.NewStore(spec.Scheme, spec.SchemeGroupVersion.WithKind(kind), namespace, batcher, persister,
		mem.WithStorePersistPeriod(syncPeriod),
		mem.WithStoreConcurrentPersist(opts.shardCount, 1))
	client.Register("workernodes", namespace, store)
	if err := store.Start(stopCh); err != nil {
		panic("start store failed:" + err.Error())
	}
	if count > 0 {
		// init some sample data
		initNodes(store, count, opts.shardCount, namespace, syncPeriod*2)
	}

	// create the workerNodeInterface
	workerNodeInterface := carbon.NewWorkerNodeInterface(client, namespace)
	// create an informer.
	informer := carbon.NewWorkerNodeInformer(client, namespace, 30*time.Second)
	bindEventHandler(informer.Informer())
	indexer := informer.Informer().GetIndexer()
	indexer.AddIndexers(cache.Indexers{shardLabelKey: shardIndexerFunc})
	lister := informer.Lister().WorkerNodes(namespace)
	if opts.dupCache {
		// optimize issue http://github.com/alibaba/kube-sharding/issues/113265
		indexer = mem.DuplicateInformerCache(informer.Informer(), nil)
		lister = carbon.NewWorkerNodeLister(indexer).WorkerNodes(namespace)
	}
	// run the informer and wait the first list call to cache.
	glog.Infof("Run the informer and wait cache synced")
	go informer.Informer().Run(stopCh)
	cache.WaitForCacheSync(stopCh, informer.Informer().HasSynced)
	if count > 0 {
		// verify
		glog.Infof("Cache synced, verify now")
		verifyNodes(lister, count, namespace)

		for i := 0; i < opts.shardCount; i++ {
			go launchOp(i, workerNodeInterface, lister, indexer, namespace, false, stopCh)
		}
		for i := 0; i < opts.shardCount; i++ {
			go launchOp(i, workerNodeInterface, lister, indexer, namespace, true, stopCh)
		}
	}

	startService(client, stopCh)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGUSR1)

	<-sigChan
	glog.Infof("quit by signal")
	close(sigChan)
	close(stopCh)
}
