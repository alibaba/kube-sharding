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

package rollalgorithm

import (
	"errors"
	"fmt"
	"sort"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	app "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	glog "k8s.io/klog"
	"k8s.io/utils/integer"
)

// Node represents an abstract node that can be rolled updated.
type Node interface {
	GetName() string
	GetVersion() string
	GetVersionPlan() interface{}
	GetScore() int
	GetDeletionCost() int
	IsReady() bool
	IsEmpty() bool
	IsRelease() bool
	SetVersion(string, interface{})
	Stop()
}

// GroupNode interface contains setter and getter for parameters controlled by a Group.
type GroupNode interface {
	GetGroupVersion() string
	GetDependencyReady() bool
	SetDependencyReady(bool)
}

// NodeUpdateScore interface contains score which represent the weight of node when upgrading.
type NodeUpdateScore interface {
	GetUpdateScore() int
}

// NodeSet is a set of Nodes
type NodeSet []Node

// RollingUpdater interface contains rolling update Shcedule interface.
type RollingUpdater interface {
	Schedule(*RollingUpdateArgs) (map[string]NodeSet, map[string]NodeSet, NodeSet, error)
}

// NodeSetUpdater implements a generic rolling update method.
type NodeSetUpdater struct {
	shardScheduler ShardScheduler
	diffLogger     *utils.DiffLogger
	debug          bool
}

// GroupNodeSetArgs contains parameters for NodeSet updates controlled by a Group.
type GroupNodeSetArgs struct {
	GroupVersionMap                map[string]string
	VersionHoldMatrixPercent       map[string]float64
	PaddedLatestVersionRatio       *int32
	VersionDependencyMatrixPercent map[string]float64
	LatestVersionCarryStrategy     CarryStrategyType
}

// RollingUpdateArgs contains common parameters for rolling updates.
type RollingUpdateArgs struct {
	Nodes      NodeSet
	CreateNode func() Node
	// Schedule Plan Params
	Replicas           *int32
	LatestVersionRatio *int32
	MaxUnavailable     *intstr.IntOrString
	MaxSurge           *intstr.IntOrString
	LatestVersion      string
	LatestVersionPlan  interface{}
	RollingStrategy    string
	Release            bool
	GroupNodeSetArgs
	ExtraNodes          NodeSet
	DisableReplaceEmpty bool
	NodeSetName         string
}

// NewDefaultNodeSetUpdater create a new default NodeSetUpdater
func NewDefaultNodeSetUpdater(diffLogger *utils.DiffLogger) RollingUpdater {
	return &NodeSetUpdater{
		shardScheduler: &InPlaceShardScheduler{
			diffLogger: diffLogger,
		},
		diffLogger: diffLogger,
	}
}

// NewNodeSetUpdater create a new NodeSetUpdater
func NewNodeSetUpdater(scheduler ShardScheduler, diffLogger *utils.DiffLogger) RollingUpdater {
	return &NodeSetUpdater{
		shardScheduler: scheduler,
		diffLogger:     diffLogger,
	}
}

// Schedule is the generic rolling update function.
func (n *NodeSetUpdater) Schedule(args *RollingUpdateArgs) (map[string]NodeSet, map[string]NodeSet, NodeSet, error) {
	err := n.checkRollingUpdateArgs(args)
	if err != nil {
		return nil, nil, nil, err
	}
	releaseNodes := n.collectReleaseNode(&args.Nodes, args.NodeSetName)
	targetMatrix, dependencyReadyMatrix, err := n.calcTargetMatrix(args)
	if err != nil {
		return nil, nil, nil, err
	}
	sort.Stable(sort.Reverse(args.Nodes))
	nodeSetVersionMap := n.initNodesVersionMap(args.Nodes)
	hasDependcy := n.hasDependency(dependencyReadyMatrix)
	if err = n.checkGroupNode(hasDependcy, nodeSetVersionMap); err != nil {
		return nil, nil, nil, err
	}
	createNodeNum, updateNodes, deleteNodes := n.rollingUpdateNodeSet(nodeSetVersionMap, targetMatrix, args)
	versionToPlan := n.initVersionToPlan(args)
	createNodes := n.syncCreateNodes(args, createNodeNum, versionToPlan, hasDependcy)
	n.syncUpdateNodes(updateNodes, versionToPlan, hasDependcy)
	n.syncDeleteNodes(deleteNodes)
	if hasDependcy {
		n.fixDependency(nodeSetVersionMap, updateNodes, dependencyReadyMatrix)
	}
	deleteNodes = append(deleteNodes, releaseNodes...)
	return createNodes, updateNodes, deleteNodes, nil
}

func (n *NodeSetUpdater) rollingUpdateNodeSet(versionNodeMap map[string]NodeSet, targetMatrix map[string]int32, args *RollingUpdateArgs) (map[string]int, map[string]NodeSet, NodeSet) {
	deleteNodes := make(NodeSet, 0)
	redundantNodes, activeNodes := n.collectRedundantNode(versionNodeMap, targetMatrix, args.NodeSetName)
	redundantNodes = append(redundantNodes, args.ExtraNodes...)
	createNodes, updateNodes := n.supplyNode(versionNodeMap, targetMatrix, &redundantNodes, args.NodeSetName)
	if !args.DisableReplaceEmpty {
		n.replaceEmptyNode(updateNodes, &deleteNodes, &activeNodes, &redundantNodes, args.NodeSetName)
	}
	deleteNodes = append(deleteNodes, redundantNodes...)
	return createNodes, updateNodes, deleteNodes
}

func (n *NodeSetUpdater) initNodesVersionMap(nodes NodeSet) map[string]NodeSet {
	nodesVersionMap := make(map[string]NodeSet)
	for _, node := range nodes {
		version := node.GetVersion()
		nodesVersionMap[version] = append(nodesVersionMap[version], node)
	}
	return nodesVersionMap
}

func (n *NodeSetUpdater) initVersionToPlan(args *RollingUpdateArgs) map[string]interface{} {
	versionToPlan := make(map[string]interface{})
	for _, node := range args.Nodes {
		version := node.GetVersion()
		plan := node.GetVersionPlan()
		versionToPlan[version] = plan
	}
	versionToPlan[args.LatestVersion] = args.LatestVersionPlan
	return versionToPlan
}

func (n *NodeSetUpdater) initShardParams(args *RollingUpdateArgs) *ShardScheduleParams {
	versionHoldMatrix := transferHoldMatrix(args.VersionHoldMatrixPercent, *args.Replicas)
	var params = &ShardScheduleParams{
		Plan: SchedulePlan{
			Replicas: args.Replicas,
			Strategy: app.DeploymentStrategy{
				Type: app.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &app.RollingUpdateDeployment{
					MaxUnavailable: args.MaxUnavailable,
					MaxSurge:       args.MaxSurge,
				},
			},
			PaddedLatestVersionRatio:       args.PaddedLatestVersionRatio,
			VersionHoldMatrix:              versionHoldMatrix,
			VersionHoldMatrixPercent:       args.VersionHoldMatrixPercent,
			LatestVersionRatio:             args.LatestVersionRatio,
			LatestVersionCarryStrategy:     args.LatestVersionCarryStrategy,
			VersionDependencyMatrixPercent: args.VersionDependencyMatrixPercent,
		},
		ShardRuntimeArgs: ShardRuntimeArgs{
			Releasing:                args.Release,
			LatestVersion:            args.LatestVersion,
			GroupVersionToVersionMap: args.GroupVersionMap,
			RollingStrategy:          args.RollingStrategy,
		},
	}
	return params
}

func (n *NodeSetUpdater) calcTargetMatrix(args *RollingUpdateArgs) (map[string]int32, map[string]int32, error) {
	params := n.initShardParams(args)
	params.initScheduleArgs(args)
	targetMatrix, dependencyReadyMatrix, err := n.shardScheduler.Schedule(args, params)
	n.diffLog(fmt.Sprintf("calcTargetMatrix %s: ", args.NodeSetName),
		fmt.Sprintf("targetMatrix: %v, dependencyReadyMatrix: %v", targetMatrix, dependencyReadyMatrix))
	return targetMatrix, dependencyReadyMatrix, err
}

func (n *NodeSetUpdater) collectReleaseNode(nodes *NodeSet, name string) NodeSet {
	releaseNodes := make(NodeSet, 0)
	afterRelease := make(NodeSet, 0)
	for _, node := range *nodes {
		if node.IsRelease() {
			releaseNodes = append(releaseNodes, node)
		} else {
			afterRelease = append(afterRelease, node)
		}
	}
	*nodes = afterRelease
	n.diffLog(fmt.Sprintf("collectReleaseNode %s: ", name), fmt.Sprintf("releteNodes: %v", releaseNodes))
	return releaseNodes
}

func (n *NodeSetUpdater) collectRedundantNode(versionNodeMap map[string]NodeSet, targetMatrix map[string]int32, name string) (redundantNodes, activeNodes NodeSet) {
	for version, versionNodes := range versionNodeMap {
		count := targetMatrix[version]
		if int(count) < len(versionNodes) {
			redundantNodes = append(redundantNodes, versionNodes[count:]...)
			activeNodes = append(activeNodes, versionNodes[:count]...)
			versionNodeMap[version] = versionNodes[:count]
		} else {
			activeNodes = append(activeNodes, versionNodes...)
		}
	}
	n.diffLog(fmt.Sprintf("collectRedundantNode %s: ", name), fmt.Sprintf("redundantNodes: %v, activeNodes: %v", redundantNodes, activeNodes))
	return
}

func (n *NodeSetUpdater) supplyNode(versionNodeMap map[string]NodeSet, targetMatrix map[string]int32, redundantNodes *NodeSet, name string) (map[string]int, map[string]NodeSet) {
	createNodes := make(map[string]int)
	updateNodes := make(map[string]NodeSet)
	sort.SliceStable(*redundantNodes, redundantNodes.lessForUpdate)
	// supplement nodes
	for version, count := range targetMatrix {
		if versionNodes, ok := versionNodeMap[version]; ok {
			count -= int32(len(versionNodes))
		}
		updateCount := integer.Int32Min(count, int32(len(*redundantNodes)))
		updateNodes[version] = append(updateNodes[version], (*redundantNodes)[:updateCount]...)
		*redundantNodes = (*redundantNodes)[updateCount:]
		count -= updateCount
		if count > 0 {
			createNodes[version] = int(count)
		}
	}
	n.diffLog(fmt.Sprintf("supplyNode %s: ", name),
		fmt.Sprintf("createNodes: %v, updateNodes: %v, redundantNodes: %v", createNodes, updateNodes, redundantNodes))
	return createNodes, updateNodes
}

func (n *NodeSetUpdater) replaceEmptyNode(updateNodes map[string]NodeSet, deleteNodes, activeNodes, redundantNodes *NodeSet, name string) {
	sort.Stable(activeNodes)
	for _, node := range *activeNodes {
		if len(*redundantNodes) <= 0 || (*redundantNodes)[0].IsEmpty() {
			break
		}
		if node.IsEmpty() {
			version := node.GetVersion()
			updateNodes[version] = append(updateNodes[version], (*redundantNodes)[0])
			*redundantNodes = (*redundantNodes)[1:]
			*deleteNodes = append(*deleteNodes, node)
		}
	}
	n.diffLog(fmt.Sprintf("replaceEmptyNode %s: ", name),
		fmt.Sprintf("redundantNodes: %v, updateNodes: %v, deleteNodes: %v", redundantNodes, updateNodes, deleteNodes))
}

func (n *NodeSetUpdater) hasDependency(dependency map[string]int32) bool {
	if nil == dependency || 0 == len(dependency) {
		return false
	}
	return true
}

func (n *NodeSetUpdater) syncCreateNodes(args *RollingUpdateArgs, createNodeNum map[string]int, versionToPlan map[string]interface{}, hasDependcy bool) map[string]NodeSet {
	createNodes := make(map[string]NodeSet)
	for version, num := range createNodeNum {
		for i := 0; i < num; i++ {
			node := args.CreateNode()
			node.SetVersion(version, versionToPlan[version])
			n.setGroupNodeDependencyReady(node, !hasDependcy)
			createNodes[version] = append(createNodes[version], node)
		}
	}
	return createNodes
}

func (n *NodeSetUpdater) syncUpdateNodes(updateNodes map[string]NodeSet, versionToPlan map[string]interface{}, hasDependcy bool) {
	for version, nodes := range updateNodes {
		for _, node := range nodes {
			node.SetVersion(version, versionToPlan[version])
			n.setGroupNodeDependencyReady(node, !hasDependcy)
		}
	}
}

func (n *NodeSetUpdater) syncDeleteNodes(deleteNodes NodeSet) {
	for _, node := range deleteNodes {
		node.Stop()
	}
}

func (n *NodeSetUpdater) fixDependency(nodeSetVersionMap, updateNodes map[string]NodeSet, dependencyReadyMatrix map[string]int32) {
	for version, dependencyReadyCount := range dependencyReadyMatrix {
		nowReady := 0
		for _, node := range nodeSetVersionMap[version] {
			gNode, _ := node.(GroupNode)
			if gNode.GetDependencyReady() {
				nowReady++
			}
		}
		if gap := int(dependencyReadyCount) - nowReady; gap > 0 {
			for _, node := range nodeSetVersionMap[version] {
				gNode, _ := node.(GroupNode)
				if !gNode.GetDependencyReady() {
					n.setGroupNodeDependencyReady(node, true)
					updateNodes[version] = append(updateNodes[version], node)
					gap--
				}
				if gap <= 0 {
					break
				}
			}
		}
	}
}

func (n *NodeSetUpdater) checkGroupNode(hasDependcy bool, nodeSetVersionMap map[string]NodeSet) error {
	if hasDependcy {
		for _, nodes := range nodeSetVersionMap {
			for _, node := range nodes {
				if _, ok := node.(GroupNode); !ok {
					return errors.New("Node not implement GroupNode")
				}
				return nil
			}
		}
	}
	return nil
}

func (n *NodeSetUpdater) checkRollingUpdateArgs(args *RollingUpdateArgs) error {
	if args == nil {
		return errors.New("nil RollingUpdateArgs")
	}
	if args.CreateNode == nil {
		return errors.New("nil RollingUpdateArgs CreateNode")
	}
	if args.Replicas != nil && *args.Replicas < 0 {
		return errors.New("illegal RollingUpdateArgs Replicas")
	}
	if args.LatestVersionRatio == nil {
		args.LatestVersionRatio = &DefaultLatestVersionRatio
	}
	if args.MaxUnavailable == nil {
		args.MaxUnavailable = &Default25IntOrString
	}
	if args.MaxSurge == nil {
		args.MaxSurge = &Default10IntOrString
	}
	return nil
}

func (n *NodeSetUpdater) setGroupNodeDependencyReady(node Node, dependencyReady bool) {
	groupNode, ok := node.(GroupNode)
	if ok {
		groupNode.SetDependencyReady(dependencyReady)
	}
}

// GetReplicas returns replicas.
// The implementation of Shard interface.
func (args *RollingUpdateArgs) GetReplicas() int32 {
	return *args.Replicas
}

// GetLatestVersionRatio returns LatestVersionRatio.
// The implementation of Shard interface.
func (args *RollingUpdateArgs) GetLatestVersionRatio(version string) int32 {
	if args.LatestVersion == version {
		return *args.LatestVersionRatio
	}
	return 0
}

// GetRawReplicas returns RawReplicas.
// The implementation of Shard interface.
func (args *RollingUpdateArgs) GetRawReplicas() int32 {
	return *args.Replicas
}

// GetDependencyLevel returns DependencyLevel.
// The implementation of Shard interface.
func (args *RollingUpdateArgs) GetDependencyLevel() int32 {
	return 0
}

// GetCPU returns CPU.
// The implementation of Shard interface.
func (args *RollingUpdateArgs) GetCPU() int32 {
	return 0
}

// GetName returns Name.
// The implementation of Shard interface.
func (args *RollingUpdateArgs) GetName() string {
	return ""
}

// GetCarbonRoleName returns CarbonRoleName.
// The implementation of Shard interface.
func (args *RollingUpdateArgs) GetCarbonRoleName() string {
	return ""
}

// GetVersionStatus returns VersionStatus.
// The implementation of Shard interface.
func (args *RollingUpdateArgs) GetVersionStatus() map[string]*VersionStatus {
	versionStatus := make(map[string]*VersionStatus)
	group := args.Nodes.isGroupNodeSet()
	for _, node := range args.Nodes {
		version := node.GetVersion()
		if group {
			groupnode := node.(GroupNode)
			version = groupnode.GetGroupVersion()
		}
		if _, ok := versionStatus[version]; !ok {
			versionStatus[version] = &VersionStatus{
				Version: version,
			}
		}
		versionStatus[version].Replicas++
		if node.IsReady() {
			versionStatus[version].ReadyReplicas++
		}
	}
	return versionStatus
}

// GetResourceVersionStatus returns ResourceVersionStatus.
// The implementation of Shard interface.
func (args *RollingUpdateArgs) GetResourceVersionStatus() map[string]*ResourceVersionStatus {
	return nil
}

func (a NodeSet) Len() int { return len(a) }
func (a NodeSet) Less(i, j int) bool {
	scoreI := a[i].GetScore()
	scoreJ := a[j].GetScore()
	iCost := a[i].GetDeletionCost()
	jCost := a[j].GetDeletionCost()
	if scoreI == scoreJ {
		if iCost == jCost {
			return a[i].GetName() > a[j].GetName()
		}
		return iCost < jCost
	}
	return scoreI < scoreJ
}
func (a NodeSet) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a NodeSet) String() string {
	names := make([]string, 0)
	for i := range a {
		names = append(names, a[i].GetName())
	}
	return fmt.Sprintf("NodeSet%v", names)
}

func (n *NodeSetUpdater) diffLog(key, msg string) {
	if nil != n.diffLogger {
		n.diffLogger.Log(key, msg)
	}
	if n.debug {
		glog.Infof("%s: %s", key, msg)
	}
}

func (a NodeSet) isGroupNodeSet() bool {
	for _, node := range a {
		_, ok := node.(GroupNode)
		return ok
	}
	return false
}

func (a NodeSet) lessForUpdate(i, j int) bool {
	n1, n2 := a[i], a[j]
	m1, ok1 := n1.(NodeUpdateScore)
	m2, ok2 := n2.(NodeUpdateScore)
	iCost := n1.GetDeletionCost()
	jCost := n2.GetDeletionCost()
	var iScore, jScore int
	if ok1 && ok2 {
		iScore, jScore = m1.GetUpdateScore(), m2.GetUpdateScore()
	} else {
		// if not define UpdateScore, compare which node is empty by default
		iEmpty := n1.IsEmpty()
		jEmpty := n2.IsEmpty()
		if iEmpty != jEmpty {
			return jEmpty
		}
		iScore, jScore = n1.GetScore(), n2.GetScore()
	}
	if iScore == jScore {
		if iCost == jCost {
			return n1.GetName() < n2.GetName()
		}
		return iCost > jCost
	}
	return iScore > jScore
}
