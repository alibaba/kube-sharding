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
	"sync"
	"testing"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/stretchr/testify/assert"
	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigmapResolver_getRemoteNodeInfo(t *testing.T) {

	var lease coordinationv1.Lease
	lease.Namespace = "test"
	c := &Resolver{mutex: new(sync.RWMutex), idFilterPrefix: "memkube:register:v1"}
	configMap := corev1.ConfigMap{}
	configMap.Namespace = "test"
	configMap.Annotations = make(map[string]string)
	//empty
	configMap.Annotations[annotationsKey] = ""
	node := c.getRemoteNodeInfo("test", &configMap)
	assert.Nil(t, node)

	node = c.getRemoteNodeInfo("test", &lease)
	assert.Nil(t, node)

	configMap.Annotations[annotationsKey] = "{\"leaseDurationSeconds\":15,\"acquireTime\":\"2020-05-28T08:24:23Z\",\"renewTime\":\"2020-06-01T02:49:56Z\",\"leaderTransitions\":39}"
	node = c.getRemoteNodeInfo("test", &configMap)
	assert.Nil(t, node)

	renewTime := time.Now()
	acquireTime := time.Now().Add(-20000 * time.Second)
	lease.Spec.RenewTime = &metav1.MicroTime{Time: renewTime}
	lease.Spec.AcquireTime = &metav1.MicroTime{Time: acquireTime}
	lease.Spec.LeaseTransitions = utils.Int32Ptr(39)
	lease.Spec.LeaseDurationSeconds = utils.Int32Ptr(15)
	lease.Spec.HolderIdentity = utils.StringPtr("")
	node = c.getRemoteNodeInfo("test", &lease)
	assert.Nil(t, node)

	renewTime = time.Now()
	acquireTime = time.Now().Add(-20000 * time.Second)
	lease.Spec.RenewTime = &metav1.MicroTime{Time: renewTime}
	lease.Spec.AcquireTime = &metav1.MicroTime{Time: acquireTime}
	lease.Spec.LeaseTransitions = utils.Int32Ptr(39)
	lease.Spec.LeaseDurationSeconds = utils.Int32Ptr(15)
	node = c.getRemoteNodeInfo("test", &lease)
	assert.Nil(t, node)

	configMap.Annotations[annotationsKey] = "{\"holderIdentity\":\"vsearchlochost010103163124.et2_c96ebb1dedd0239443c2096e6289fd83\",\"leaseDurationSeconds\":15,\"acquireTime\":\"2020-05-28T08:24:23Z\",\"renewTime\":\"2020-06-01T02:49:56Z\",\"leaderTransitions\":39}"
	node = c.getRemoteNodeInfo("test", &configMap)
	assert.Nil(t, node)

	renewTime = time.Now()
	acquireTime = time.Now().Add(-20000 * time.Second)
	lease.Spec.RenewTime = &metav1.MicroTime{Time: renewTime}
	lease.Spec.AcquireTime = &metav1.MicroTime{Time: acquireTime}
	lease.Spec.LeaseTransitions = utils.Int32Ptr(39)
	lease.Spec.LeaseDurationSeconds = utils.Int32Ptr(15)
	lease.Spec.HolderIdentity = utils.StringPtr("vsearchlochost010103163124.et2_c96ebb1dedd0239443c2096e6289fd83")
	node = c.getRemoteNodeInfo("test", &lease)
	assert.Nil(t, node)

	//expire
	configMap.Annotations[annotationsKey] = "{\"holderIdentity\":\"memkube:register:v1_127.0.0.1_10111_test_1,2,3\",\"leaseDurationSeconds\":15,\"acquireTime\":\"2020-05-28T08:24:23Z\",\"renewTime\":\"2020-06-01T02:49:56Z\",\"leaderTransitions\":39}"
	node = c.getRemoteNodeInfo("test", &configMap)
	assert.Nil(t, node)

	renewTime = time.Now().Add(-200 * time.Second)
	acquireTime = time.Now().Add(-2000 * time.Second)
	lease.Spec.RenewTime = &metav1.MicroTime{Time: renewTime}
	lease.Spec.AcquireTime = &metav1.MicroTime{Time: acquireTime}
	lease.Spec.LeaseTransitions = utils.Int32Ptr(39)
	lease.Spec.LeaseDurationSeconds = utils.Int32Ptr(15)
	lease.Spec.HolderIdentity = utils.StringPtr("memkube:register:v1_127.0.0.1_10111_test_1,2,3")
	node = c.getRemoteNodeInfo("test", &lease)
	assert.Nil(t, node)

	reTime := time.Now().Add(-200 * time.Second).Format(time.RFC3339) //"2020-06-01T03:47:56Z"
	configMap.Annotations[annotationsKey] = "{\"holderIdentity\":\"memkube:register:v1_127.0.0.1_10111_test_1\",\"leaseDurationSeconds\":100,\"acquireTime\":\"2020-05-28T08:24:23Z\",\"renewTime\":\"" + reTime + "\",\"leaderTransitions\":39}"
	node = c.getRemoteNodeInfo("test", &configMap)
	assert.Nil(t, node)

	renewTime = time.Now().Add(-200 * time.Second)
	acquireTime = time.Now().Add(-2000 * time.Second)
	lease.Spec.RenewTime = &metav1.MicroTime{Time: renewTime}
	lease.Spec.AcquireTime = &metav1.MicroTime{Time: acquireTime}
	lease.Spec.LeaseTransitions = utils.Int32Ptr(39)
	lease.Spec.LeaseDurationSeconds = utils.Int32Ptr(100)
	lease.Spec.HolderIdentity = utils.StringPtr("memkube:register:v1_127.0.0.1_10111_test_1")
	configMap.Annotations[annotationsKey] = "{\"holderIdentity\":\"memkube:register:v1_127.0.0.1_10111_test_1\",\"leaseDurationSeconds\":100,\"acquireTime\":\"2020-05-28T08:24:23Z\",\"renewTime\":\"" + reTime + "\",\"leaderTransitions\":39}"
	node = c.getRemoteNodeInfo("test", &lease)
	assert.Nil(t, node)

	//normal
	reTime = time.Now().Format(time.RFC3339)

	configMap.Annotations[annotationsKey] = "{\"holderIdentity\":\"memkube:register:v1_127.0.0.1_10111_test_1,2,3\",\"leaseDurationSeconds\":15,\"acquireTime\":\"2020-05-28T08:24:23Z\",\"renewTime\":\"" + reTime + "\",\"leaderTransitions\":39}"
	node = c.getRemoteNodeInfo("test", &configMap)
	assert.NotNil(t, node)
	assert.Equal(t, 15, node.LeaseDurationSeconds)
	assert.Equal(t, "127.0.0.1", node.IP)
	assert.Equal(t, "10111", node.Port)
	assert.Equal(t, "test", node.Namespace)
	assert.Equal(t, []string{"1", "2", "3"}, node.ShardIDs)

	renewTime = time.Now()
	acquireTime = time.Now().Add(-20000 * time.Second)
	lease.Spec.RenewTime = &metav1.MicroTime{Time: renewTime}
	lease.Spec.AcquireTime = &metav1.MicroTime{Time: acquireTime}
	lease.Spec.LeaseTransitions = utils.Int32Ptr(39)
	lease.Spec.LeaseDurationSeconds = utils.Int32Ptr(15)
	lease.Spec.HolderIdentity = utils.StringPtr("memkube:register:v1_127.0.0.1_10111_test_1,2,3")
	node = c.getRemoteNodeInfo("test", &lease)
	assert.NotNil(t, node)
	assert.Equal(t, 15, node.LeaseDurationSeconds)
	assert.Equal(t, "127.0.0.1", node.IP)
	assert.Equal(t, "10111", node.Port)
	assert.Equal(t, "test", node.Namespace)
	assert.Equal(t, []string{"1", "2", "3"}, node.ShardIDs)

	reTime = time.Now().Add(-50 * time.Second).Format(time.RFC3339) //"2020-06-01T03:47:56Z"

	configMap.Annotations[annotationsKey] = "{\"holderIdentity\":\"memkube:register:v1_127.0.0.1_10111_test_1\",\"leaseDurationSeconds\":100,\"acquireTime\":\"2020-05-28T08:24:23Z\",\"renewTime\":\"" + reTime + "\",\"leaderTransitions\":39}"
	node = c.getRemoteNodeInfo("test", &configMap)
	assert.NotNil(t, node)
	assert.Equal(t, 100, node.LeaseDurationSeconds)
	assert.Equal(t, "127.0.0.1", node.IP)
	assert.Equal(t, "10111", node.Port)
	assert.Equal(t, "test", node.Namespace)
	assert.Equal(t, []string{"1"}, node.ShardIDs)

	renewTime = time.Now().Add(-50 * time.Second)
	acquireTime = time.Now().Add(-20000 * time.Second)
	lease.Spec.RenewTime = &metav1.MicroTime{Time: renewTime}
	lease.Spec.AcquireTime = &metav1.MicroTime{Time: acquireTime}
	lease.Spec.LeaseTransitions = utils.Int32Ptr(39)
	lease.Spec.LeaseDurationSeconds = utils.Int32Ptr(100)
	lease.Spec.HolderIdentity = utils.StringPtr("memkube:register:v1_127.0.0.1_10111_test_1")
	node = c.getRemoteNodeInfo("test", &lease)
	assert.NotNil(t, node)
	assert.Equal(t, 100, node.LeaseDurationSeconds)
	assert.Equal(t, "127.0.0.1", node.IP)
	assert.Equal(t, "10111", node.Port)
	assert.Equal(t, "test", node.Namespace)
	assert.Equal(t, []string{"1"}, node.ShardIDs)
}

func TestConfigmapResolver_handler_empty(t *testing.T) {
	namespace := "test"
	c := &Resolver{mutex: new(sync.RWMutex)}
	nodes, err := c.ResolveAll(namespace)
	assert.Equal(t, 0, len(nodes))
	assert.NoError(t, err)
}

func TestConfigmapResolver_addNode_noShard(t *testing.T) {
	namespace := "test"
	reTime := time.Now().Format(time.RFC3339)
	c := &Resolver{mutex: new(sync.RWMutex), remoteNodeInfos: make(map[string]*remoteNodeInfo),
		nodeInfosIndex: make(map[string]NamespaceNodes), idFilterPrefix: "memkube:register:v1"}
	c.RegisterCalcShardIDFunc(func(obj interface{}) (string, error) {
		return obj.(string), nil
	})
	configMap := corev1.ConfigMap{}
	configMap.Kind = "configMap"
	configMap.Name = "test-controller-0"
	configMap.Namespace = namespace
	configMap.Annotations = make(map[string]string)
	configMap.Annotations[annotationsKey] = "{\"holderIdentity\":\"memkube:register:v1_127.0.0.1_10111_test\",\"leaseDurationSeconds\":15,\"acquireTime\":\"2020-05-28T08:24:23Z\",\"renewTime\":\"" + reTime + "\",\"leaderTransitions\":39}"
	c.addNode(&configMap)
	nodes, err := c.ResolveAll(namespace)
	assert.Equal(t, 1, len(nodes))
	assert.Equal(t, "10111", nodes[0].Port)
	assert.NoError(t, err)

	node, err := c.Resolve(namespace, "1")
	assert.NoError(t, err)
	assert.NotNil(t, node)
	node, err = c.Resolve(namespace, "0")
	assert.NoError(t, err)
	assert.NotNil(t, node)
}

func TestConfigmapResolver_updateNode(t *testing.T) {
	namespace := "test"
	reTime := time.Now().Format(time.RFC3339)
	c := &Resolver{mutex: new(sync.RWMutex), remoteNodeInfos: make(map[string]*remoteNodeInfo),
		nodeInfosIndex: make(map[string]NamespaceNodes), idFilterPrefix: "memkube:register:v1"}
	c.RegisterCalcShardIDFunc(func(obj interface{}) (string, error) {
		return obj.(string), nil
	})
	//var configMap metav1.Object
	var configMap corev1.ConfigMap
	configMap = corev1.ConfigMap{}
	configMap.Kind = "configMap"
	configMap.Name = "test-controller-0"
	configMap.Namespace = namespace
	configMap.Annotations = make(map[string]string)
	//configMap.GetAnnotations()
	configMap.Annotations[annotationsKey] = "{\"holderIdentity\":\"memkube:register:v1_127.0.0.1_10111_test_1,2,3\",\"leaseDurationSeconds\":15,\"acquireTime\":\"2020-05-28T08:24:23Z\",\"renewTime\":\"" + reTime + "\",\"leaderTransitions\":39}"
	c.addNode(&configMap)
	nodes, err := c.ResolveAll(namespace)
	assert.Equal(t, 1, len(nodes))
	assert.NoError(t, err)

	c.addNode(&configMap)
	nodes, err = c.ResolveAll(namespace)
	assert.Equal(t, 1, len(nodes))
	assert.NoError(t, err)
	//recover
	configMap.Annotations[annotationsKey] = "{\"holderIdentity\":\"memkube:register:v1_127.0.0.1_20222_test_4,5,6\",\"leaseDurationSeconds\":15,\"acquireTime\":\"2020-05-28T08:24:23Z\",\"renewTime\":\"" + reTime + "\",\"leaderTransitions\":39}"
	c.addNode(&configMap)
	nodes, err = c.ResolveAll(namespace)
	assert.Equal(t, 1, len(nodes))
	assert.Equal(t, "20222", nodes[0].Port)
	assert.NoError(t, err)

	node, err := c.Resolve(namespace, "4")
	assert.NoError(t, err)
	assert.Equal(t, "20222", node.Port)

	node, err = c.Resolve(namespace, "1")
	assert.NoError(t, err)
	assert.Nil(t, node)
}

func TestConfigmapResolver_addNode(t *testing.T) {
	namespace := "test"
	reTime := time.Now().Format(time.RFC3339)
	c := &Resolver{mutex: new(sync.RWMutex), remoteNodeInfos: make(map[string]*remoteNodeInfo),
		nodeInfosIndex: make(map[string]NamespaceNodes), idFilterPrefix: "memkube:register:v1"}
	c.RegisterCalcShardIDFunc(func(obj interface{}) (string, error) {
		return obj.(string), nil
	})
	configMap := corev1.ConfigMap{}
	configMap.Kind = "configMap"
	configMap.Name = "test-controller-0"
	configMap.Namespace = namespace
	configMap.Annotations = make(map[string]string)
	configMap.Annotations[annotationsKey] = "{\"holderIdentity\":\"memkube:register:v1_127.0.0.1_10111_test_1,2,3\",\"leaseDurationSeconds\":15,\"acquireTime\":\"2020-05-28T08:24:23Z\",\"renewTime\":\"" + reTime + "\",\"leaderTransitions\":39}"
	c.addNode(&configMap)

	//add
	configMap.Name = "test-controller-1"
	configMap.Annotations[annotationsKey] = "{\"holderIdentity\":\"memkube:register:v1_127.0.0.1_20222_test_4,5,6\",\"leaseDurationSeconds\":15,\"acquireTime\":\"2020-05-28T08:24:23Z\",\"renewTime\":\"" + reTime + "\",\"leaderTransitions\":39}"
	c.addNode(&configMap)
	nodes, err := c.ResolveAll(namespace)
	assert.Equal(t, 2, len(nodes))
	assert.NoError(t, err)

	//expired
	reTime = time.Now().Add(-5000 * time.Second).Format(time.RFC3339)
	configMap.Name = "test-controller-2"
	configMap.Annotations[annotationsKey] = "{\"holderIdentity\":\"memkube:register:v1_127.0.0.1_30333_test_7\",\"leaseDurationSeconds\":15,\"acquireTime\":\"2020-05-28T08:24:23Z\",\"renewTime\":\"" + reTime + "\",\"leaderTransitions\":39}"
	c.addNode(&configMap)
	nodes, err = c.ResolveAll(namespace)
	assert.Equal(t, 2, len(nodes))
	assert.NoError(t, err)

	//invaild
	configMap.Name = "test-controller-3"
	configMap.Annotations[annotationsKey] = "{\"holderIdentity\":\"kube:register:v1_127.0.0.1_20222_test_4,5,6\",\"leaseDurationSeconds\":15,\"acquireTime\":\"2020-05-28T08:24:23Z\",\"renewTime\":\"" + reTime + "\",\"leaderTransitions\":39}"
	c.addNode(&configMap)
	nodes, err = c.ResolveAll(namespace)
	assert.Equal(t, 2, len(nodes))
	assert.NoError(t, err)

	reTime = time.Now().Format(time.RFC3339)
	configMap.Name = "test-controller-4"
	configMap.Namespace = "test2"
	configMap.Annotations[annotationsKey] = "{\"holderIdentity\":\"memkube:register:v1_127.0.0.1_40444_test2_0\",\"leaseDurationSeconds\":15,\"acquireTime\":\"2020-05-28T08:24:23Z\",\"renewTime\":\"" + reTime + "\",\"leaderTransitions\":39}"
	c.addNode(&configMap)
	nodes, err = c.ResolveAll(namespace)
	assert.Equal(t, 2, len(nodes))
	assert.NotEqual(t, "30333", nodes[1].Port)
	assert.NotEqual(t, "40444", nodes[1].Port)
	assert.NoError(t, err)
	nodes, err = c.ResolveAll("test2")
	assert.Equal(t, 1, len(nodes))
	assert.Equal(t, "40444", nodes[0].Port)
	assert.NoError(t, err)
}

func TestConfigmapResolver_deleteNode(t *testing.T) {
	namespace := "test"
	reTime := time.Now().Format(time.RFC3339)
	c := &Resolver{mutex: new(sync.RWMutex), remoteNodeInfos: make(map[string]*remoteNodeInfo),
		nodeInfosIndex: make(map[string]NamespaceNodes), idFilterPrefix: "memkube:register:v1"}
	c.RegisterCalcShardIDFunc(func(obj interface{}) (string, error) {
		return obj.(string), nil
	})
	configMap := corev1.ConfigMap{}
	configMap.Kind = "configMap"
	configMap.Name = "test-controller-0"
	configMap.Namespace = namespace
	configMap.Annotations = make(map[string]string)
	configMap.Annotations[annotationsKey] = "{\"holderIdentity\":\"memkube:register:v1_127.0.0.1_10111_test_1,2,3\",\"leaseDurationSeconds\":15,\"acquireTime\":\"2020-05-28T08:24:23Z\",\"renewTime\":\"" + reTime + "\",\"leaderTransitions\":39}"
	c.addNode(&configMap)

	//add
	configMap.Name = "test-controller-1"
	configMap.Annotations[annotationsKey] = "{\"holderIdentity\":\"memkube:register:v1_127.0.0.1_20222_test_4,5,6\",\"leaseDurationSeconds\":15,\"acquireTime\":\"2020-05-28T08:24:23Z\",\"renewTime\":\"" + reTime + "\",\"leaderTransitions\":39}"
	c.addNode(&configMap)
	nodes, err := c.ResolveAll(namespace)
	assert.Equal(t, 2, len(nodes))
	assert.NoError(t, err)

	configMap.Name = "test-controller-1"
	c.deleteNode(&configMap)
	nodes, err = c.ResolveAll(namespace)
	assert.Equal(t, 1, len(nodes))
	assert.Equal(t, "10111", nodes[0].Port)
	assert.NoError(t, err)

	configMap.Name = "test-controller-2"
	c.deleteNode(&configMap)
	nodes, err = c.ResolveAll(namespace)
	assert.Equal(t, 1, len(nodes))
	assert.NoError(t, err)
	node, err := c.Resolve(namespace, "1")
	assert.NoError(t, err)
	assert.Equal(t, "10111", node.Port)

	configMap.Name = "test-controller-0"
	c.deleteNode(&configMap)
	nodes, err = c.ResolveAll(namespace)
	assert.Equal(t, 0, len(nodes))
	assert.NoError(t, err)
}

func TestConfigmapResolver_updateNodeInfosIndexUnLock_expire(t *testing.T) {

	c := &Resolver{}
	err := c.updateNodeInfosIndexUnLock()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(c.nodeInfosIndex))
	assert.Equal(t, 0, len(c.nodeInfosIndex))

	remoteNodeInfos := make(map[string]*remoteNodeInfo)
	remoteNodeInfo1 := remoteNodeInfo{NodeID: "node1", IP: "127.0.0.1", Port: "1011", RenewTime: time.Now(), LeaseDurationSeconds: 20, Namespace: "ns1"}
	remoteNodeInfo2 := remoteNodeInfo{NodeID: "node2", IP: "127.0.0.1", Port: "2022", RenewTime: time.Now(), LeaseDurationSeconds: 20, Namespace: "ns2"}
	remoteNodeInfos["node1"] = &remoteNodeInfo1
	remoteNodeInfos["node2"] = &remoteNodeInfo2

	c.remoteNodeInfos = remoteNodeInfos
	err = c.updateNodeInfosIndexUnLock()
	assert.NoError(t, err)
	assert.Equal(t, 2, len(c.nodeInfosIndex))
	assert.Equal(t, 1, len(c.nodeInfosIndex["ns1"]))
	assert.Equal(t, 1, len(c.nodeInfosIndex["ns2"]))
	assert.Equal(t, "1011", c.nodeInfosIndex["ns1"]["0"].Port)
	assert.Equal(t, "2022", c.nodeInfosIndex["ns2"]["0"].Port)

	remoteNodeInfo1 = remoteNodeInfo{NodeID: "node1", IP: "127.0.0.1", Port: "1011", RenewTime: time.Now(), LeaseDurationSeconds: 20, Namespace: "ns1"}
	remoteNodeInfo2 = remoteNodeInfo{NodeID: "node2", IP: "127.0.0.1", Port: "2022", RenewTime: time.Now(), LeaseDurationSeconds: 20, Namespace: "ns1"}
	remoteNodeInfos["node1"] = &remoteNodeInfo1
	remoteNodeInfos["node2"] = &remoteNodeInfo2

	c.remoteNodeInfos = remoteNodeInfos
	err = c.updateNodeInfosIndexUnLock()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(c.nodeInfosIndex))
	assert.Equal(t, 1, len(c.nodeInfosIndex["ns1"]))
	assert.Equal(t, 0, len(c.nodeInfosIndex["ns2"]))

	//expired
	reTime := time.Now().Add(-200 * time.Second)
	remoteNodeInfo1 = remoteNodeInfo{NodeID: "node1", IP: "127.0.0.1", Port: "1011", RenewTime: reTime, LeaseDurationSeconds: 20, Namespace: "ns1"}
	remoteNodeInfo2 = remoteNodeInfo{NodeID: "node2", IP: "127.0.0.1", Port: "2022", RenewTime: time.Now(), LeaseDurationSeconds: 20, Namespace: "ns2"}
	remoteNodeInfos["node1"] = &remoteNodeInfo1
	remoteNodeInfos["node2"] = &remoteNodeInfo2

	c.remoteNodeInfos = remoteNodeInfos
	err = c.updateNodeInfosIndexUnLock()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(c.nodeInfosIndex))
	assert.Equal(t, 0, len(c.nodeInfosIndex["ns1"]))
	assert.Equal(t, 1, len(c.nodeInfosIndex["ns2"]))
	assert.Equal(t, "2022", c.nodeInfosIndex["ns2"]["0"].Port)
}
