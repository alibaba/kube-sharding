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
	"github.com/alibaba/kube-sharding/pkg/memkube/rpc"
)

// BaseResolver base
type baseResolver struct {
	calcShardIDFunc rpc.CalcShardIDFunc
}

// RegisterCalcShardIDFunc  register calc shardID func
func (b *baseResolver) RegisterCalcShardIDFunc(f rpc.CalcShardIDFunc) {
	b.calcShardIDFunc = f
}

// StaticResolver static router
type StaticResolver struct {
	baseResolver
	//staticNodeInfos map[string]*remoteNodeInfo
	nodeInfosIndex map[string]NamespaceNodes
}

// SetIndex sets resolve static config
func (s *StaticResolver) SetIndex(nodeInfosIndex map[string]NamespaceNodes) {
	s.nodeInfosIndex = nodeInfosIndex
}

// Resolve resolve config
func (s *StaticResolver) Resolve(namespace string, obj interface{}) (*rpc.NodeInfo, error) {
	if s.nodeInfosIndex[namespace] != nil {
		for _, v := range s.nodeInfosIndex[namespace] {
			return v, nil
		}
	}
	return nil, nil
}

// ResolveAll namespace all node
func (s *StaticResolver) ResolveAll(namespace string) ([]*rpc.NodeInfo, error) {
	list := make([]*rpc.NodeInfo, 0)
	for _, v := range s.nodeInfosIndex[namespace] {
		list = append(list, v)
	}
	return list, nil
}
