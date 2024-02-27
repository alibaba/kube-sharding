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

package resolver

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	glog "k8s.io/klog"

	"github.com/alibaba/kube-sharding/pkg/memkube/rpc"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	annotationsKey          = "control-plane.alpha.kubernetes.io/leader"
	configMapNodeKey        = "holderIdentity"
	renewTimeKey            = "renewTime"
	leaseDurationSecondsKey = "leaseDurationSeconds"
	renewTimeFormat         = "2016-01-02T15:04:05Z"
	noShardKey              = "0"
)

// NamespaceNodes namespace node list
type NamespaceNodes map[string]*rpc.NodeInfo

// Resolver  Configmap resolver
type Resolver struct {
	baseResolver
	mutex          *sync.RWMutex
	idFilterPrefix string

	//维护全量配置，key为namespace.name 节点的唯一标示
	remoteNodeInfos map[string]*remoteNodeInfo
	//第一层key namespace， 第二层key shards，value为remoteNodeInfos中的key
	nodeInfosIndex map[string]NamespaceNodes
}

// NewResolver  create Resolver and start
func NewResolver(sharedIndexInformers []cache.SharedIndexInformer, idFilterPrefix string) *Resolver {
	resolver := &Resolver{mutex: new(sync.RWMutex),
		remoteNodeInfos: make(map[string]*remoteNodeInfo), nodeInfosIndex: make(map[string]NamespaceNodes), idFilterPrefix: idFilterPrefix}
	for i := range sharedIndexInformers {
		sharedIndexInformer := sharedIndexInformers[i]
		sharedIndexInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc: resolver.addNode,
			UpdateFunc: func(old, new interface{}) {
				newObj := new.(metav1.Object)
				oldObj := old.(metav1.Object)
				if newObj.GetResourceVersion() == oldObj.GetResourceVersion() {
					return
				}
				resolver.updateNode(new)
			},
			DeleteFunc: resolver.deleteNode,
		})
	}
	return resolver
}

func (c *Resolver) deleteNode(obj interface{}) {
	if object, ok := obj.(metav1.Object); ok {
		key := object.GetNamespace() + "/" + object.GetName()
		c.mutex.Lock()
		defer c.mutex.Unlock()
		if node, ok := c.remoteNodeInfos[key]; ok && node != nil {
			glog.Infof("Remove node, key:%s", key)
			err := node.Close()
			if err != nil {
				glog.Warningf("Close node failed, key:%s err:%v", key, err)
				return
			}
			delete(c.remoteNodeInfos, key)
			c.updateNodeInfosIndexUnLock()
		}
	}
}

func (c *Resolver) updateNode(obj interface{}) {
	if object, ok := obj.(metav1.Object); ok {
		key := object.GetNamespace() + "/" + object.GetName()
		remoteNodeInfo := c.getRemoteNodeInfo(key, object)
		if remoteNodeInfo == nil {
			return
		}
		if !c.checkNode(remoteNodeInfo, key) {
			return
		}

		glog.Infof("Add/Update node, key:%s remoteNodeInfo:%s", key, utils.ObjJSON(remoteNodeInfo))
		c.mutex.Lock()
		defer c.mutex.Unlock()
		c.remoteNodeInfos[key] = remoteNodeInfo
		c.updateNodeInfosIndexUnLock()
	}
}

func (c *Resolver) addNode(obj interface{}) {
	c.updateNode(obj)
}

func (c *Resolver) checkNode(latestNode *remoteNodeInfo, key string) bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if oldRemoteNodeInfo, ok := c.remoteNodeInfos[key]; ok {
		if latestNode.HolderIdentity == oldRemoteNodeInfo.HolderIdentity {
			return false
		}
	}
	return true
}

func (c *Resolver) getRemoteNodeInfo(key string, object metav1.Object) *remoteNodeInfo {
	leaderElectionRecord := &resourcelock.LeaderElectionRecord{}
	var leaderInfoStr string
	switch t := object.(type) {
	case *coordinationv1.Lease:
		lease := object.(*coordinationv1.Lease)
		leaderElectionRecord = resourcelock.LeaseSpecToLeaderElectionRecord(&lease.Spec)
		leaderInfoStr = utils.ObjJSON(lease)
	case *corev1.ConfigMap:
		leaderInfoStr = object.GetAnnotations()[annotationsKey]
		if leaderInfoStr == "" {
			return nil
		}
		err := json.Unmarshal([]byte(leaderInfoStr), &leaderElectionRecord)
		if err != nil {
			glog.Warningf("Node invaild, key:%s  err:%s", key, err)
			return nil
		}
		if leaderElectionRecord.HolderIdentity == "" {
			glog.Warningf("Node configMapNodeKey invaild, key:%s  %s", key, leaderInfoStr)
			return nil
		}
	default:
		fmt.Printf("Unexpected type %T\n", t)
	}
	if !strings.HasPrefix(leaderElectionRecord.HolderIdentity, c.idFilterPrefix) {
		return nil
	}

	if leaderElectionRecord.LeaseDurationSeconds == 0 {
		glog.Warningf("LeaseDurationSeconds invaild, key:%s  %s", key, leaderInfoStr)
		return nil
	}
	if c.isExpired(leaderElectionRecord.RenewTime.Time, leaderElectionRecord.LeaseDurationSeconds) {
		glog.Warningf("Node expired, key:%s  %s", key, leaderInfoStr)
		return nil
	}
	values := strings.Split(leaderElectionRecord.HolderIdentity, "_")
	ip := values[1]
	port := values[2]
	namespace := object.GetNamespace()
	var shardIDs []string
	if len(values) == 5 {
		shardIDStr := values[4]
		shardIDs = strings.Split(shardIDStr, ",")
	}
	remoteNodeInfo := &remoteNodeInfo{NodeID: key, IP: ip, Port: port,
		ShardIDs: shardIDs, Namespace: namespace, CreateTime: time.Now(),
		RenewTime: leaderElectionRecord.RenewTime.Time, LeaseDurationSeconds: leaderElectionRecord.LeaseDurationSeconds, HolderIdentity: leaderElectionRecord.HolderIdentity}
	return remoteNodeInfo
}

func (c *Resolver) updateNodeInfosIndexUnLock() error {
	c.clearExpireNodeUnLock()
	glog.Infof("Update nodeInfo index start remoteNodeInfos:%s", utils.ObjJSON(c.remoteNodeInfos))
	tmpNodeInfosIndex := make(map[string]NamespaceNodes)
	for _, remoteNodeInfo := range c.remoteNodeInfos {
		namespace := remoteNodeInfo.Namespace
		shardIDs := remoteNodeInfo.ShardIDs
		var shardsNodeInfo map[string]*rpc.NodeInfo
		var ok bool
		if shardsNodeInfo, ok = tmpNodeInfosIndex[namespace]; !ok {
			shardsNodeInfo = make(map[string]*rpc.NodeInfo)
			tmpNodeInfosIndex[namespace] = shardsNodeInfo
		}
		if len(shardIDs) == 0 {
			shardsNodeInfo[noShardKey] = &rpc.NodeInfo{IP: remoteNodeInfo.IP, Port: remoteNodeInfo.Port}
		} else {
			for _, shardID := range shardIDs {
				shardsNodeInfo[shardID] = &rpc.NodeInfo{IP: remoteNodeInfo.IP, Port: remoteNodeInfo.Port}
			}
		}
	}
	c.nodeInfosIndex = tmpNodeInfosIndex
	glog.Infof("Update nodeInfo index end updateNodeInfosIndex:%s", utils.ObjJSON(c.nodeInfosIndex))
	return nil
}

func (c *Resolver) clearExpireNodeUnLock() {
	for nodeK, nodeV := range c.remoteNodeInfos {
		if c.isExpired(nodeV.RenewTime, nodeV.LeaseDurationSeconds) {
			glog.Infof("Clear expire node %s", nodeK)
			delete(c.remoteNodeInfos, nodeK)
		}
	}
}

func (c *Resolver) isExpired(renewTime time.Time, leaseDurationSeconds int) bool {
	now := time.Now()
	sub := now.Sub(renewTime)
	if int(sub.Seconds()) > leaseDurationSeconds {
		return true
	}
	return false
}

// Resolve resolve config
func (c *Resolver) Resolve(namespace string, obj interface{}) (*rpc.NodeInfo, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	// no shard
	if len(c.nodeInfosIndex[namespace]) == 1 {
		for _, nodeV := range c.nodeInfosIndex[namespace] {
			return nodeV, nil
		}
	}
	shardID, err := c.calcShardIDFunc(obj)
	if err != nil {
		return nil, err
	}
	return c.nodeInfosIndex[namespace][shardID], nil
}

// ResolveAll namespace all node
func (c *Resolver) ResolveAll(namespace string) ([]*rpc.NodeInfo, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	result := mapToNodeSlice(c.nodeInfosIndex[namespace])
	glog.V(4).Infof("ResolveAll namespace:%s, result:%s", namespace, utils.ObjJSON(result))
	return result, nil
}

func mapToNodeSlice(m map[string]*rpc.NodeInfo) []*rpc.NodeInfo {
	s := make([]*rpc.NodeInfo, 0, len(m))
	nodekeys := make(map[string]bool, 0)
	for _, v := range m {
		key := v.IP + "/" + v.Port
		// node exist
		if _, ok := nodekeys[key]; ok {
			continue
		}
		s = append(s, v)
		nodekeys[key] = true
	}
	return s
}

type remoteNodeInfo struct {
	NodeID               string
	IP                   string
	Port                 string
	Namespace            string
	ShardIDs             []string
	RenewTime            time.Time
	LeaseDurationSeconds int

	CreateTime time.Time

	HolderIdentity string
}

// Close remote node close
func (r *remoteNodeInfo) Close() error {
	return nil
}
