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
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/alibaba/kube-sharding/common"
	"github.com/alibaba/kube-sharding/common/config"
	nsroute "github.com/alibaba/kube-sharding/common/router"
	"github.com/alibaba/kube-sharding/controller/util"
	"github.com/alibaba/kube-sharding/proxy/apiset"

	glog "k8s.io/klog"

	goerrors "errors"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec"
	carbonclientset "github.com/alibaba/kube-sharding/pkg/client/clientset/versioned"
	listers "github.com/alibaba/kube-sharding/pkg/client/listers/carbon/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"
	informercorev1 "k8s.io/client-go/informers/core/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"

	carboninformers "github.com/alibaba/kube-sharding/pkg/client/informers/externalversions"
	"k8s.io/client-go/informers"
	corelisters "k8s.io/client-go/listers/core/v1"
)

var (
	proxyConfigMapLabel = map[string]string{"app.c2.io/configs": "proxy"}
)

type resourceManager interface {
	//namespace
	createNamespace(namespace string) error
	checkNamespace(appName string, namespaces []string) ([]string, error)
	getTargetNamespace(appName string) (string, error)

	//rollingset
	listRollingSet(appName, groupID string, single bool) ([]carbonv1.RollingSet, error)
	listZoneRollingSet(appName, groupID, zoneName string) ([]*carbonv1.RollingSet, error)
	getRollingsetByGroupIDs(
		appName string, groupIds []string, single bool) (
		[]carbonv1.RollingSet, error)
	listRollingsetBySelector(namespace string, selector labels.Selector) ([]carbonv1.RollingSet, error)
	syncRollingset(namespace string, rollingset *carbonv1.RollingSet) error
	getRollingset(namespace, name string) (*carbonv1.RollingSet, error)
	deleteRollingSet(namespace, name string) error
	updateRollingset(namespace string, rollingset *carbonv1.RollingSet) error
	createRollingset(namespace string, rollingset *carbonv1.RollingSet) error

	//shardgroup
	listGroup(appName, groupID string) ([]carbonv1.ShardGroup, error)
	syncShardGroup(namespace string, shardGroup *carbonv1.ShardGroup) error
	deleteShardGroup(namespace, name string) error
	//services
	getRollingsetServices(
		appName, namespace, rollingsetName string) (
		[]carbonv1.ServicePublisher, error)
	getServiceByGroupname(
		appName, namespace, groupName string) (
		[]carbonv1.ServicePublisher, error)
	syncServices(
		namespace string,
		oldServices []carbonv1.ServicePublisher,
		services []*carbonv1.ServicePublisher) error
	deleteService(namespace, name string) error

	listWorkerNode(
		namespace string, selector labels.Selector) (
		[]carbonv1.WorkerNode, error)
	syncWorkerNode(namespace string, workerNode *carbonv1.WorkerNode, immutableNode bool) error
	getWorkerNode(namespace, name string) (*carbonv1.WorkerNode, error)
	listPod(namespace string, selector labels.Selector) ([]corev1.Pod, error)
	getShardGroupName(appName, namespace, groupID string) (string, error)
	getRollingSetName(appName, namespace, groupID string) (string, error)
	getGangRollingSetName(appName, namespace, groupID string) (string, error)

	getWorkerEvictionAPIs() apiset.WorkerEvictionAPIs
	getCarbonJobAPIs() apiset.CarbonJobAPIs
}

var _ resourceManager = &defaultResourceManager{}

type defaultResourceManager struct {
	cluster       string
	memObjMap     map[string]bool
	nsRouter      nsroute.Router
	nameAllocator *nameAllocator

	kubeClientSet        kubeclientset.Interface
	carbonClientSet      carbonclientset.Interface
	kubeClientSetCache   kubeclientset.Interface
	carbonClientSetCache carbonclientset.Interface
	dynamicInterface     dynamic.Interface

	workerLister          listers.WorkerNodeLister
	workerListerSynced    cache.InformerSynced
	serviceLister         listers.ServicePublisherLister
	serviceListerSynced   cache.InformerSynced
	rollingSetLister      listers.RollingSetLister
	rollingSetSynced      cache.InformerSynced
	groupLister           listers.ShardGroupLister
	groupSynced           cache.InformerSynced
	podLister             corelisters.PodLister
	podListerSynced       cache.InformerSynced
	configMapLister       corelisters.ConfigMapLister
	configMapListerSynced cache.InformerSynced
	podInformer           informercorev1.PodInformer
}

func createNSRouter(stopCh <-chan struct{}, kubeClient kubeclientset.Interface) (nsroute.Router, error) {
	if opts.nsRouteStrategy == "simple" {
		return nsroute.NewSimpleRouter(), nil
	} else if opts.nsRouteStrategy != "" && opts.nsRouteStrategy != "config" {
		return nil, goerrors.New("invalid namespace route strategy")
	}
	ns, name := getConfigMapName(opts.nsRouteConfKey)
	kubeConfFactory := config.NewKubeConfigerFactory(kubeClient, ns, proxyConfigMapLabel)
	kubeConfFactory.Start(stopCh)
	nsRouter := nsroute.NewConfigRouter(kubeConfFactory.Configer(name))
	return nsRouter, nil
}

func getConfigMapName(config string) (namespace, name string) {
	namespace = carbonv1.SystemNamespace
	vec := strings.Split(config, "/")
	if len(vec) == 1 {
		name = vec[0]
	} else if len(vec) == 2 {
		namespace, name = vec[0], vec[1]
	}
	return
}

func newClientBuilder(kubeConfigFile, contentType string, timeout time.Duration) simpleControllerClientBuilder {
	kubeconfig, err := clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		glog.Fatalf("newClientBuilder with error: %s, %v", kubeConfigFile, err)
	}
	kubeconfig.QPS = 10000
	kubeconfig.Burst = 10000
	kubeconfig.Timeout = timeout
	kubeconfig.ContentType = contentType
	rootClientBuilder := simpleControllerClientBuilder{
		clientConfig: kubeconfig,
	}
	return rootClientBuilder
}

func newResourceManager(cluster, kubeConfigFile, contentType string, stopCh <-chan struct{}) (*defaultResourceManager, error) {
	rootClientBuilder := newClientBuilder(kubeConfigFile, contentType, time.Second*time.Duration(opts.timeout))
	cacheClientBuilder := newClientBuilder(kubeConfigFile, contentType, 0)
	kubeClient := cacheClientBuilder.ClientOrDie("carbon-proxy")

	nsRouter, err := createNSRouter(stopCh, kubeClient)
	if err != nil {
		return nil, err
	}
	err = common.InitGlobalKubeConfiger(opts.namespace, kubeClient, stopCh)
	if err != nil {
		glog.Fatalf("error init global kubeconfiger: %v", err)
	}
	configInformers := informers.NewFilteredSharedInformerFactory(kubeClient, time.Duration(0), opts.namespace, nil)
	configmapInformer := configInformers.Core().V1().ConfigMaps()
	leaseInformer := configInformers.Coordination().V1().Leases()

	mixClient := createMixClient(rootClientBuilder, configmapInformer, leaseInformer, cluster)
	carbonClientSetCache := createMixClient(cacheClientBuilder, configmapInformer, leaseInformer, cluster)

	var defaultResourceManager = defaultResourceManager{
		cluster:               cluster,
		kubeClientSet:         kubeClient,
		carbonClientSet:       mixClient,
		kubeClientSetCache:    kubeClient,
		carbonClientSetCache:  carbonClientSetCache,
		configMapLister:       configmapInformer.Lister(),
		configMapListerSynced: configmapInformer.Informer().HasSynced,
		nsRouter:              nsRouter,
	}

	glog.Info("Wait resourceManager ready...")
	configInformers.Start(stopCh)
	for {
		if ok := cache.WaitForCacheSync(stopCh,
			defaultResourceManager.configMapListerSynced,
		); !ok {
			time.Sleep(time.Second)
			continue
		}
		break
	}

	defaultResourceManager.initWithCache(stopCh)
	defaultResourceManager.nameAllocator = &nameAllocator{
		groupLister:      defaultResourceManager.groupLister,
		rollingsetLister: defaultResourceManager.rollingSetLister,
	}
	glog.Infof("New resource manager success.")
	return &defaultResourceManager, nil
}

func (c *defaultResourceManager) initWithCache(stopCh <-chan struct{}) {
	tweakListOptionsFunc := func(options *v1.ListOptions) {
		if "" == options.LabelSelector && "" != opts.selector {
			options.LabelSelector = opts.selector
		}
	}
	// FIXME: use kubeMixClient to support memkube pods
	informerFactory := informers.NewFilteredSharedInformerFactory(c.kubeClientSetCache, time.Duration(0), opts.namespace, tweakListOptionsFunc)
	carbonInformerFactory := carboninformers.NewFilteredSharedInformerFactory(c.carbonClientSetCache, time.Duration(0), opts.namespace, tweakListOptionsFunc)

	// 注册informer到client-go
	carbonInformerFactory.Carbon().V1().RollingSets().Informer()
	carbonInformerFactory.Carbon().V1().ServicePublishers().Informer()
	carbonInformerFactory.Carbon().V1().ShardGroups().Informer()
	carbonInformerFactory.Carbon().V1().WorkerNodes().Informer()
	carbonInformerFactory.Carbon().V1().CarbonJobs().Informer()
	carbonInformerFactory.Carbon().V1().WorkerNodeEvictions().Informer()
	informerFactory.Core().V1().Pods().Informer()
	// 启动informer
	informerFactory.Start(stopCh)
	carbonInformerFactory.Start(stopCh)

	// resourcemanager赋值
	rollingSetInformer := carbonInformerFactory.Carbon().V1().RollingSets()
	servicePublisherInformer :=
		carbonInformerFactory.Carbon().V1().ServicePublishers()
	shardGroupInformer := carbonInformerFactory.Carbon().V1().ShardGroups()
	workerNodeInformer := carbonInformerFactory.Carbon().V1().WorkerNodes()
	podInformer := informerFactory.Core().V1().Pods()

	c.workerLister = workerNodeInformer.Lister()
	c.workerListerSynced = workerNodeInformer.Informer().HasSynced
	c.serviceLister = servicePublisherInformer.Lister()
	c.serviceListerSynced = servicePublisherInformer.Informer().HasSynced
	c.rollingSetLister = rollingSetInformer.Lister()
	c.rollingSetSynced = rollingSetInformer.Informer().HasSynced
	c.groupLister = shardGroupInformer.Lister()
	c.groupSynced = shardGroupInformer.Informer().HasSynced
	c.podLister = podInformer.Lister()
	c.podListerSynced = podInformer.Informer().HasSynced
	// 等待informer缓存完毕
	glog.Info("Wait cache synced...")
	for {
		if ok := cache.WaitForCacheSync(stopCh,
			c.workerListerSynced,
			c.serviceListerSynced,
			c.rollingSetSynced,
			c.groupSynced,
			c.podListerSynced,
		); !ok {
			time.Sleep(time.Second)
			continue
		}
		break
	}
}

func (c *defaultResourceManager) getRollingsetByGroupIDs(
	appName string, groupIDs []string, single bool) (
	[]carbonv1.RollingSet, error) {
	var rollingSets = make([]carbonv1.RollingSet, 0)
	var err error
	if 0 == len(groupIDs) {
		rollingSets, err = c.listRollingSet(appName, "", single)
		if nil != err {
			glog.Error(err)
			return nil, err
		}
	} else {
		rollingSets, err = c.listRollingSets(appName, groupIDs, single)
		if nil != err {
			glog.Error(err)
			return nil, err
		}
	}
	if glog.V(4) {
		glog.Infof("getRollingsetByGroupIDs, appName:%s, groupIDs:%v, rollingSets:%v", appName, groupIDs, len(rollingSets))
	}
	return filterDupRollingsets(rollingSets, single), nil
}

func filterDupRollingsets(srcs []carbonv1.RollingSet, single bool) []carbonv1.RollingSet {
	var results = make([]carbonv1.RollingSet, 0, len(srcs))
	for i := range srcs {
		if !single && srcs[i].OwnerReferences == nil && len(srcs[i].Spec.GangVersionPlan) == 0 {
			continue
		}
		//TODO 这段代码有隐患，重构admin链路需要考虑
		// if single && srcs[i].OwnerReferences != nil && srcs[i].OwnerReferences[0].Kind != "Admin" {
		// 	continue
		// }
		exist := false
		for j := range results {
			if srcs[i].Name == results[j].Name {
				exist = true
				if results[j].CreationTimestamp.Time.Unix() >
					srcs[i].CreationTimestamp.Time.Unix() {
					results[j] = srcs[i]
				}
				break
			}
		}
		if !exist {
			results = append(results, srcs[i])
		}
	}
	return results
}

func (c *defaultResourceManager) createNamespace(namespace string) error {
	_, err := c.kubeClientSet.CoreV1().
		Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if nil != err {
		if !errors.IsNotFound(err) {
			glog.Error(err)
			return err
		}
		var kubeNamespace corev1.Namespace
		kubeNamespace.Name = namespace
		kubeNamespace.Labels = map[string]string{"alibabacloud.com/owner": "hippo"}
		_, err := c.kubeClientSet.CoreV1().Namespaces().Create(context.Background(), &kubeNamespace, metav1.CreateOptions{})
		glog.Infof("create namespace :%s ,%v", utils.ObjJSON(kubeNamespace), err)
		if nil != err && !errors.IsAlreadyExists(err) {
			glog.Error(err)
			return err
		}
	}
	return nil
}

func (c *defaultResourceManager) checkNamespace(appName string, namespaces []string) ([]string, error) {
	var namespaceChanged = true
	namespace, err := c.getTargetNamespace(appName)
	if nil != err || "" == namespace {
		glog.Errorf("Failed to get namespace %s,%v", appName, err)
		return nil, err
	}
	if 0 == len(namespaces) {
		namespaces = []string{namespace}
		namespaceChanged = false
		return namespaces, nil
	} else if 1 == len(namespaces) && namespaces[0] == namespace {
		return namespaces, nil
	}

	if namespaceChanged {
		glog.Warningf("Found changed namespace app: %s", appName)
	}
	return namespaces, nil
}

func (c *defaultResourceManager) syncServices(namespace string,
	oldServices []carbonv1.ServicePublisher, services []*carbonv1.ServicePublisher) error {
	start := time.Now()
	for i := range oldServices {
		shouldDelete := true
		for j := range services {
			if oldServices[i].Name == services[j].Name {
				shouldDelete = false
			}
		}
		if shouldDelete {
			err := c.carbonClientSet.CarbonV1().ServicePublishers(namespace).
				Delete(context.Background(), oldServices[i].Name, metav1.DeleteOptions{})
			glog.Infof("delete services :%s , %s , %v",
				namespace, oldServices[i].Name, err)
			if nil != err && !errors.IsNotFound(err) {
				return err
			}
		}
	}

	for i := range services {
		var oldService *carbonv1.ServicePublisher
		shouldCreate := true
		for j := range oldServices {
			if oldServices[j].Name == services[i].Name {
				shouldCreate = false
				oldService = &oldServices[j]
				break
			}
		}
		if shouldCreate && !services[i].Spec.SoftDelete {
			_, err := c.carbonClientSet.CarbonV1().
				ServicePublishers(namespace).Create(context.Background(), services[i], metav1.CreateOptions{})
			glog.Infof("create services :%s , %s , %v",
				namespace, utils.ObjJSON(services[i]), err)
			if nil != err && !errors.IsAlreadyExists(err) {
				glog.Error(err)
				return err
			}
		}
		if !shouldCreate && nil != oldService && services[i].Spec.SoftDelete && carbonv1.ServiceSkyline == services[i].Spec.Type {
			oldService.Spec.SoftDelete = true
			_, err := c.carbonClientSet.CarbonV1().
				ServicePublishers(namespace).Update(context.Background(), oldService, metav1.UpdateOptions{})
			glog.Infof("update services :%s , %s , %v",
				namespace, utils.ObjJSON(services[i]), err)
			if nil != err && !errors.IsNotFound(err) {
				glog.Error(err)
				return err
			}
		}
	}
	if glog.V(4) {
		glog.Infof("complete set %s services after %d ms",
			namespace, time.Now().Sub(start).Nanoseconds()/1000/1000)
	}
	return nil
}

func (c *defaultResourceManager) getRollingsetServices(
	appName, namespace, rollingsetName string) (
	[]carbonv1.ServicePublisher, error) {
	return c.getServices("", namespace, "", rollingsetName)
}

func (c *defaultResourceManager) getServiceByGroupname(
	appName, namespace, groupName string) (
	[]carbonv1.ServicePublisher, error) {
	return c.getServices("", namespace, groupName, "")
}

func (c *defaultResourceManager) getServices(
	appName, namespace, sg, rs string) (
	[]carbonv1.ServicePublisher, error) {
	selector, err := newC2ObjectSelector(appName, sg, rs)
	if nil != err {
		glog.Warningf("create service lister err: %v", err)
		return nil, err
	}
	services, err := c.carbonClientSet.CarbonV1().
		ServicePublishers(namespace).
		List(context.Background(), metav1.ListOptions{LabelSelector: selector.String()})
	glog.V(4).Infof("getServices selector %s, %s, services: %s, err: %v", namespace, selector.String(), utils.ObjJSON(services), err)
	if nil != err && !errors.IsNotFound(err) {
		return nil, err
	}
	return services.Items, err
}

func (c *defaultResourceManager) syncRollingset(
	namespace string, rollingset *carbonv1.RollingSet) error {
	var err error
	for i := 0; i < 3; i++ {
		var oldRollingset *carbonv1.RollingSet
		oldRollingset, err = c.carbonClientSet.CarbonV1().
			RollingSets(namespace).Get(context.Background(), rollingset.Name, metav1.GetOptions{})
		if nil != err {
			if errors.IsNotFound(err) {
				rollingset.Spec.Selector = new(metav1.LabelSelector)
				carbonv1.AddRsSelectorHashKey(rollingset.Spec.Selector, rollingset.Name)
				_, err := c.carbonClientSet.CarbonV1().
					RollingSets(namespace).Create(context.Background(), rollingset, metav1.CreateOptions{})
				glog.Infof("create rollingset :%s , %s , %v",
					namespace, utils.ObjJSON(rollingset), err)
				return err
			}
			return err
		}
		err = checkNameConflict(oldRollingset, rollingset, len(rollingset.Spec.GangVersionPlan) != 0)
		if nil != err {
			glog.Errorf("check name conficts :%v", err)
			rollingset.SetName(rollingset.GetName() + "-i")
			continue
		}
		rollingset.Spec.ScaleSchedulePlan = oldRollingset.Spec.ScaleSchedulePlan
		rollingset.Spec.ScaleConfig = oldRollingset.Spec.ScaleConfig
		rollingset.Spec.Capacity = oldRollingset.Spec.Capacity
		rollingset.Spec.Version = oldRollingset.Spec.Version
		rollingset.Spec.ResVersion = oldRollingset.Spec.ResVersion
		rollingset.Spec.Selector = oldRollingset.Spec.Selector
		rollingset.Spec.CheckSum = oldRollingset.Spec.CheckSum
		rollingset.Spec.InstanceID = oldRollingset.Spec.InstanceID
		carbonv1.AddRsUniqueLabelHashKey(rollingset.Labels, rollingset.Name)
		labels := rollingset.Labels
		annotations := rollingset.Annotations
		rollingset.ObjectMeta = oldRollingset.ObjectMeta
		rollingset.Labels = labels
		rollingset.Annotations = annotations
		carbonv1.SyncCrdTime(&oldRollingset.ObjectMeta, &rollingset.ObjectMeta)
		c.syncSpread(oldRollingset, rollingset)
		fixRsReplicasFromSpread(rollingset)
		if !reflect.DeepEqual(rollingset.Spec, oldRollingset.Spec) ||
			!reflect.DeepEqual(rollingset.ObjectMeta.Labels, oldRollingset.ObjectMeta.Labels) ||
			!reflect.DeepEqual(rollingset.ObjectMeta.Annotations, oldRollingset.ObjectMeta.Annotations) {
			util.UpdateCrdTime(&rollingset.ObjectMeta, time.Now())
			_, err = c.carbonClientSet.CarbonV1().RollingSets(namespace).Update(context.Background(), rollingset, metav1.UpdateOptions{})
			if glog.V(4) {
				glog.Infof("update rollingset old: %s, new: %s, oldmetas: %s, newmetas: %s, error: %v", utils.ObjJSON(oldRollingset.Spec),
					utils.ObjJSON(rollingset.Spec), utils.ObjJSON(oldRollingset.ObjectMeta), utils.ObjJSON(rollingset.ObjectMeta), err)
			} else {
				glog.Infof("update rollingset : %s, error: %v", oldRollingset.Name, err)
			}
			if nil != err && errors.IsConflict(err) {
				continue
			}
		}
		break
	}
	glog.Infof("sync rollingset :%s,%s,%v", namespace, rollingset.Name, err)
	return err
}

func (c *defaultResourceManager) syncSpread(old, new *carbonv1.RollingSet) {
	if v, ok := old.Annotations[carbonv1.AnnotationC2Spread]; ok {
		if new.Annotations == nil {
			new.Annotations = map[string]string{}
		}
		new.Annotations[carbonv1.AnnotationC2Spread] = v
	}
}

func (c *defaultResourceManager) listRollingsetBySelector(namespace string, selector labels.Selector) ([]carbonv1.RollingSet, error) {
	list, err := c.carbonClientSet.CarbonV1().RollingSets(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}
	return list.Items, err
}

func (c *defaultResourceManager) getShardGroupName(appName, namespace, groupID string) (string, error) {
	return c.nameAllocator.allocateShardGroupName(appName, namespace, groupID)
}

func (c *defaultResourceManager) getRollingSetName(appName, namespace, groupID string) (string, error) {
	return c.nameAllocator.allocateRollingSetName(appName, namespace, groupID)
}

func (c *defaultResourceManager) getGangRollingSetName(appName, namespace, groupID string) (string, error) {
	return c.nameAllocator.allocateGangRollingSetName(appName, namespace, groupID)
}

func (c *defaultResourceManager) syncShardGroup(
	namespace string, shardGroup *carbonv1.ShardGroup) error {
	var err error
	for i := 0; i < 3; i++ {
		var oldShardGroup *carbonv1.ShardGroup
		oldShardGroup, err = c.carbonClientSet.CarbonV1().
			ShardGroups(namespace).Get(context.Background(), shardGroup.Name, metav1.GetOptions{})
		if nil != err {
			if errors.IsNotFound(err) {
				_, err := c.carbonClientSet.CarbonV1().
					ShardGroups(namespace).Create(context.Background(), shardGroup, metav1.CreateOptions{})
				glog.Infof("create shardGroup :%s , %s , %v",
					namespace, utils.ObjJSON(shardGroup), err)
				return err
			}
			return err
		}
		err = checkNameConflict(oldShardGroup, shardGroup, false)
		if nil != err {
			glog.Errorf("chech name conficts :%v", err)
			return err
		}
		shardGroup.Spec.RollingVersion = oldShardGroup.Spec.RollingVersion
		shardGroup.Spec.ScaleStrategy = oldShardGroup.Spec.ScaleStrategy
		shardGroup.Spec.ScaleConfig = oldShardGroup.Spec.ScaleConfig
		shardGroup.Spec.Selector = oldShardGroup.Spec.Selector
		carbonv1.AddGroupUniqueLabelHashKey(shardGroup.Labels, shardGroup.Name)
		labels := shardGroup.Labels
		annotations := shardGroup.Annotations
		shardGroup.ObjectMeta = oldShardGroup.ObjectMeta
		shardGroup.Labels = labels
		shardGroup.Annotations = annotations
		if shardGroup.Annotations != nil && oldShardGroup.Annotations != nil {
			shardGroup.Annotations["updateCrdTime"] = oldShardGroup.ObjectMeta.Annotations["updateCrdTime"]
		}
		for k := range shardGroup.Spec.ShardTemplates {
			spec := shardGroup.Spec.ShardTemplates[k]
			shardGroup.Spec.ShardTemplates[k] = spec
		}
		shardGroup.Spec.UpdatePlanTimestamp = oldShardGroup.Spec.UpdatePlanTimestamp
		if !reflect.DeepEqual(shardGroup.Spec, oldShardGroup.Spec) ||
			!reflect.DeepEqual(shardGroup.ObjectMeta.Labels, oldShardGroup.ObjectMeta.Labels) ||
			!reflect.DeepEqual(shardGroup.ObjectMeta.Annotations, oldShardGroup.ObjectMeta.Annotations) {
			util.UpdateCrdTime(&shardGroup.ObjectMeta, time.Now())
			shardGroup.Spec.UpdatePlanTimestamp = time.Now().UnixNano()
			_, err = c.carbonClientSet.CarbonV1().
				ShardGroups(namespace).Update(context.Background(), shardGroup, metav1.UpdateOptions{})
			if glog.V(4) {
				glog.Infof("update shardgroup old : %s "+
					" new : %s  oldmetas : %s newmetas : %s, error: %v",
					utils.ObjJSON(oldShardGroup.Spec),
					utils.ObjJSON(shardGroup.Spec),
					utils.ObjJSON(oldShardGroup.ObjectMeta),
					utils.ObjJSON(shardGroup.ObjectMeta), err)
			} else {
				glog.Infof("update shardgroup : %s, error: %v ", shardGroup.Name, err)
			}
			if nil != err && errors.IsConflict(err) {
				continue
			}
		}
		break
	}
	glog.Infof("sync shardgroup :%s,%s,%v", namespace, shardGroup.Name, err)
	return err
}

func (c *defaultResourceManager) syncWorkerNode(
	namespace string, workerNode *carbonv1.WorkerNode, immutableNode bool) error {
	if workerNode == nil {
		return fmt.Errorf("workernode shall not be nil on syncWorkerNode")
	}
	oldWorkerNode, err := c.carbonClientSet.CarbonV1().
		WorkerNodes(namespace).Get(context.Background(), workerNode.GetName(), metav1.GetOptions{})
	if nil != err {
		if errors.IsNotFound(err) {
			glog.Infof("create workernode %s", utils.ObjJSON(workerNode))
			_, err := c.carbonClientSet.CarbonV1().
				WorkerNodes(namespace).Create(context.Background(), workerNode, metav1.CreateOptions{})
			glog.Infof("create worker node :%s, %s, %v",
				namespace, utils.ObjJSON(workerNode), err)
			return err
		}
		return err
	}
	if immutableNode {
		glog.Infof("skip to update node by immutable flag: %s", workerNode.GetName())
		return nil
	}
	err = checkNameConflict(oldWorkerNode, workerNode, false)
	if nil != err {
		glog.Errorf("check name conficts failed: %v, %s, %s", err, utils.ObjJSON(oldWorkerNode), utils.ObjJSON(workerNode))
		return err
	}
	// copy info might be changed by controller
	labels := workerNode.Labels
	workerNode.ObjectMeta = oldWorkerNode.ObjectMeta
	workerNode.Spec.Version = oldWorkerNode.Spec.Version
	workerNode.Spec.ResVersion = oldWorkerNode.Spec.ResVersion
	workerNode.Spec.Selector = oldWorkerNode.Spec.Selector
	workerNode.Labels = labels
	if nil != err {
		glog.Error(err)
		return err
	}

	// Update only change spec, apiserver will overwrite change of status
	if !reflect.DeepEqual(workerNode.Spec, oldWorkerNode.Spec) ||
		!reflect.DeepEqual(workerNode.ObjectMeta.Labels, oldWorkerNode.ObjectMeta.Labels) {
		glog.Infof("update workernode before: %s, after: %s", utils.ObjJSON(oldWorkerNode), utils.ObjJSON(workerNode))
		if _, err := c.carbonClientSet.CarbonV1().
			WorkerNodes(namespace).Update(context.Background(), workerNode, metav1.UpdateOptions{}); err != nil {
			err := fmt.Errorf("sync workernode failed, workernode: %s, err: %s",
				utils.ObjJSON(workerNode), err)
			return err
		}
	}
	glog.Infof("sync workernode success, workernode: %s, %s",
		namespace, workerNode.Name)
	return nil
}

func (c *defaultResourceManager) listWorkerNode(
	namespace string, selector labels.Selector) (
	[]carbonv1.WorkerNode, error) {
	if c.memObjMap["workernodes"] != true { //走memkube rpc client端没有watch机制，本地cache不保留数据，需要走remote调用c2， 走carbonClientSet
		workerPtrs, err := c.workerLister.
			WorkerNodes(namespace).List(selector)
		if err != nil {
			glog.Warningf("ListWorker failure, err: %v", err)
		}
		var workers = make([]carbonv1.WorkerNode, len(workerPtrs))
		for i := range workers {
			workers[i] = *workerPtrs[i]
		}
		return workers, err
	}
	workers, err := c.carbonClientSet.CarbonV1().
		WorkerNodes(namespace).
		List(context.Background(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		glog.Errorf("List workernode error, selector: %s, err: %v", selector.String(), err)
		return nil, err
	}
	glog.V(4).Infof("List workernode, namespace: %s, result.size: %d", namespace, len(workers.Items))
	return workers.Items, nil
}

func (c *defaultResourceManager) listPod(
	namespace string, selector labels.Selector) ([]corev1.Pod, error) {
	if !c.memObjMap["pods"] {
		podPtrs, err := c.podLister.Pods(namespace).List(selector)
		if err != nil {
			glog.Errorf("ListPod failure, err: %v", err)
		}
		var pods = make([]corev1.Pod, len(podPtrs))
		for i := range pods {
			pods[i] = *podPtrs[i]
		}
		return pods, err
	}
	pods, err := c.kubeClientSet.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		glog.Errorf("List pod error, selector: %s, err: %v", selector.String(), err)
		return nil, err
	}
	glog.V(4).Infof("List pod result size: %d", len(pods.Items))
	return pods.Items, nil
}

func (c *defaultResourceManager) listGroup(
	appName, groupID string) ([]carbonv1.ShardGroup, error) {
	selector, err := newHippoKeySelector(c.cluster, appName, groupID, "", nil, nil)
	if nil != err {
		return nil, err
	}

	groupPtrs, err := c.groupLister.List(selector)
	if err != nil {
		glog.Warningf("ListPod failure, err: %v", err)
	}
	var groups = make([]carbonv1.ShardGroup, len(groupPtrs))
	for i := range groups {
		groups[i] = *groupPtrs[i]
	}
	return groups, err
}

func (c *defaultResourceManager) listRollingSets(appName string, groupIDs []string, single bool) (
	[]carbonv1.RollingSet, error) {
	var roleNames = []string{}
	if single {
		for i := range groupIDs {
			groupID := groupIDs[i]
			var roleName string
			if groupID == carbonv1.HippoAdminRoleTag || groupID == carbonv1.C2AdminRoleTag {
				roleName = groupID
			} else {
				roleName = utils.AppendString(groupID, ".", groupID)
			}
			roleNames = append(roleNames, roleName)
		}
		groupIDs = nil
	}
	selector, err := newHippoKeySelector(c.cluster, appName, "", "", groupIDs, roleNames)
	if nil != err {
		glog.Errorf("new label selector error: %v", err)
		return nil, err
	}
	if glog.V(4) {
		glog.Infof("listRollingSets newHippoKeySelector:%s", selector.String())
	}
	rollingsetPtrs, err := c.rollingSetLister.List(selector)
	if err != nil {
		glog.Warningf("ListRollingset failure, err: %v", err)
		return nil, err
	}
	var rollingsets = make([]carbonv1.RollingSet, len(rollingsetPtrs))
	for i := range rollingsets {
		rollingsets[i] = *rollingsetPtrs[i]
	}
	return rollingsets, err
}

func (c *defaultResourceManager) listZoneRollingSet(appName, groupID, zoneName string) ([]*carbonv1.RollingSet, error) {
	labelSelector := metav1.SetAsLabelSelector(labels.Set{})
	carbonv1.SetAppSelector(labelSelector, appName)
	carbonv1.SetGroupSelector(labelSelector, groupID)
	carbonv1.SetZoneSelector(labelSelector, zoneName)
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return nil, err
	}
	return c.rollingSetLister.List(selector)
}

func (c *defaultResourceManager) listRollingSet(
	appName, groupID string, single bool) (
	[]carbonv1.RollingSet, error) {
	var roleName string
	if "" != groupID && single {
		if groupID == typespec.AppMasterTag {
			roleName = typespec.AppMasterTag
		} else {
			roleName = utils.AppendString(groupID, ".", groupID)
		}
		groupID = ""
	}
	selector, err := newHippoKeySelector(c.cluster, appName, groupID, roleName, nil, nil)
	if nil != err {
		glog.Errorf("new label selector error: %v", err)
		return nil, err
	}
	if glog.V(4) {
		glog.Infof("listRollingSet newHippoKeySelector:%s", selector.String())
	}
	rollingsetPtrs, err := c.rollingSetLister.List(selector)
	if err != nil {
		glog.Warningf("ListRollingset failure, err: %v", err)
		return nil, err
	}
	var rollingsets = make([]carbonv1.RollingSet, len(rollingsetPtrs))
	for i := range rollingsets {
		rollingsets[i] = *rollingsetPtrs[i]
	}
	return rollingsets, err
}

func (c *defaultResourceManager) getTargetNamespace(appName string) (string, error) {
	return c.nsRouter.GetNamespace(appName)
}

func (c *defaultResourceManager) getWorkerNode(
	namespace, name string) (*carbonv1.WorkerNode, error) {
	if c.memObjMap["workernodes"] != true {
		workerNode, err := c.workerLister.WorkerNodes(namespace).Get(name)
		if err != nil {
			glog.Warningf("getWorkerNode %s, %s failed, err: %s",
				namespace, name, err)
		}
		return workerNode, err
	}
	workerNode, err := c.carbonClientSet.CarbonV1().
		WorkerNodes(namespace).Get(context.Background(), name, metav1.GetOptions{})
	return workerNode, err
}

func (c *defaultResourceManager) deleteService(
	namespace, name string) error {
	err := c.carbonClientSet.CarbonV1().
		ServicePublishers(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	glog.Infof("delete services :%s , %s , %v", namespace, name, err)
	if nil != err && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (c *defaultResourceManager) getRollingset(namespace, name string) (*carbonv1.RollingSet, error) {
	return c.carbonClientSet.CarbonV1().RollingSets(namespace).Get(context.Background(), name, metav1.GetOptions{})
}

func (c *defaultResourceManager) deleteRollingSet(
	namespace, name string) error {
	err := c.carbonClientSet.CarbonV1().
		RollingSets(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	glog.Infof("delete rollingsets :%s , %s , %v", namespace, name, err)
	if nil != err && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func (c *defaultResourceManager) createRollingset(namespace string, rollingset *carbonv1.RollingSet) error {
	_, err := c.carbonClientSet.CarbonV1().RollingSets(namespace).Create(context.Background(), rollingset, metav1.CreateOptions{})
	glog.Infof("create rollingset:%s, err:%v", utils.ObjJSON(rollingset), err)
	return err
}

func (c *defaultResourceManager) updateRollingset(namespace string, rollingset *carbonv1.RollingSet) error {
	_, err := c.carbonClientSet.CarbonV1().RollingSets(namespace).Update(context.Background(), rollingset, metav1.UpdateOptions{})
	glog.Infof("update rollingset:%s, err:%v", utils.ObjJSON(rollingset), err)
	return err
}

func (c *defaultResourceManager) deleteShardGroup(
	namespace, name string) error {
	err := c.carbonClientSet.CarbonV1().
		ShardGroups(namespace).Delete(context.Background(), name, metav1.DeleteOptions{})
	glog.Infof("delete shardgroups :%s , %s , %v", namespace, name, err)
	if nil != err && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func checkNameConflict(old, new metav1.Object, gang bool) error {
	var err error
	if carbonv1.GetAppName(old) != carbonv1.GetAppName(new) {
		err = fmt.Errorf("app name conflict :%s,%s,%s", carbonv1.GetAppName(old), carbonv1.GetAppName(new), old.GetName())
	}
	if carbonv1.GetGroupName(old) != "" && carbonv1.GetGroupName(old) != carbonv1.GetGroupName(new) {
		err = fmt.Errorf("group name conflict :%s,%s,%s", carbonv1.GetGroupName(old), carbonv1.GetGroupName(new), old.GetName())
	}
	if carbonv1.GetRoleName(old) != carbonv1.GetRoleName(new) && !gang {
		err = fmt.Errorf("role name conflict :%s,%s,%s", carbonv1.GetRoleName(old), carbonv1.GetRoleName(new), old.GetName())
	}
	return err
}

func newHippoKeySelector(cluster, app, group, role string, groupsIDs []string, roleNames []string) (labels.Selector, error) {
	labelSelector := metav1.SetAsLabelSelector(labels.Set{})
	carbonv1.SetClusterSelector(labelSelector, cluster)
	carbonv1.SetAppSelector(labelSelector, app)
	carbonv1.SetGroupSelector(labelSelector, group)
	carbonv1.SetRoleSelector(labelSelector, role)
	carbonv1.SetGroupsSelector(labelSelector, groupsIDs...)
	carbonv1.SetRolesSelector(labelSelector, roleNames...)
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	return selector, err
}

func newC2ObjectSelector(appName, shardgroup, rs string) (labels.Selector, error) {
	labelSelector := metav1.SetAsLabelSelector(labels.Set{})
	carbonv1.AddGroupSelectorHashKey(labelSelector, shardgroup)
	carbonv1.AddRsSelectorHashKey(labelSelector, rs)
	carbonv1.AddSelectorHashKey(labelSelector, carbonv1.LabelKeyAppName, appName)

	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	return selector, err
}

func (c *defaultResourceManager) getWorkerEvictionAPIs() apiset.WorkerEvictionAPIs {
	return apiset.NewWorkerEvictionAPIs(c.carbonClientSet)
}

func (c *defaultResourceManager) getCarbonJobAPIs() apiset.CarbonJobAPIs {
	return apiset.NewCarbonJobAPIs(c.carbonClientSet)
}
