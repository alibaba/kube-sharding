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

package rpc

// NodeInfo contains ip and port
type NodeInfo struct {
	IP   string
	Port string
}

func (n *NodeInfo) String() string {
	return n.IP + ":" + n.Port
}

// EndpointResolver resolve server address.
type EndpointResolver interface {
	// RegisterCalcShardIDFunc  register calc shardID func
	RegisterCalcShardIDFunc(f CalcShardIDFunc)
	Resolve(namespace string, obj interface{}) (*NodeInfo, error)
	ResolveAll(namespace string) ([]*NodeInfo, error)
}

// CalcShardIDFunc calc shardID func
type CalcShardIDFunc func(obj interface{}) (string, error)

// FakeResolver resolve server address.
type FakeResolver struct {
}

// Resolve resolve config
func (f *FakeResolver) Resolve(namespace string, obj interface{}) (*NodeInfo, error) {
	return &NodeInfo{}, nil
}

// ResolveAll resolve config
func (f *FakeResolver) ResolveAll(namespace string) ([]*NodeInfo, error) {
	return []*NodeInfo{}, nil
}

// RegisterCalcShardIDFunc  register calc shardID func
func (f *FakeResolver) RegisterCalcShardIDFunc(cf CalcShardIDFunc) {
}
