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

package publisher

import (
	"fmt"
	"time"

	"github.com/spf13/pflag"

	"github.com/alibaba/kube-sharding/common/metric"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	mapset "github.com/deckarep/golang-set"
	"github.com/prometheus/client_golang/prometheus"
	glog "k8s.io/klog"
)

// publisher used to publish or unpublish ip nodes to registry
type publisher interface {
	getNodes() ([]carbonv1.IPNode, error)
	updateNodes(ips []carbonv1.IPNode) error
	addNodes(ips []carbonv1.IPNode) error
	delNodes(ips []carbonv1.IPNode) error
	disableNodes(ips []carbonv1.IPNode) error
	enableNodes(ips []carbonv1.IPNode) error
	serviceValidate() (bool, error)
}

type releasingSetter interface {
	setReleasing(releasing bool)
}

type codeError interface {
	GetCode() int
}

// error code
const (
	ErrorCodeServerUnvailable = 503
)

func newPublisher(key string, spec carbonv1.ServicePublisherSpec) (publisher, error) {
	switch spec.Type {
	default:
		err := fmt.Errorf("not support this registry type :%s", spec.Type)
		return nil, err
	}
}

var (
	aliyunAccessKeyID     = ""
	aliyunAccessKeySecret = ""
)

// InitFlags is for explicitly initializing the flags.
func InitFlags(flagset *pflag.FlagSet) {
	flagset.StringVar(&aliyunAccessKeyID, "aliyun-access-key-id", aliyunAccessKeyID, "aliyun-access-key-id for slb publisher")
	flagset.StringVar(&aliyunAccessKeySecret, "aliyun-access-key-secret", aliyunAccessKeySecret, "aliyun-access-key-secret for slb publisher")
}

type abstractPublisher struct {
	key         string
	ptype       string
	serviceName string
	publisher
}

func newAbstractPublisher(key string, spec carbonv1.ServicePublisherSpec) (*abstractPublisher, error) {
	publisher, err := newPublisher(key, spec)
	if nil != err {
		return nil, err
	}
	var abPublisher = abstractPublisher{
		key:         key,
		ptype:       string(spec.Type),
		serviceName: spec.ServiceName,
		publisher:   publisher,
	}
	return &abPublisher, nil
}

func getLocalNodes(workers []*carbonv1.WorkerNode, service *carbonv1.ServicePublisher) ([]carbonv1.IPNode, []carbonv1.IPNode) {
	var publishNodes = make([]carbonv1.IPNode, 0, len(workers))
	var allNodes = make([]carbonv1.IPNode, 0, len(workers))
	needUpdateGracefully := carbonv1.NeedUpdateGracefully(service)
	for _, worker := range workers {
		if worker.Status.IP == "" {
			continue
		}
		var ipNode = carbonv1.IPNode{
			IP:         worker.Status.IP,
			HealthMeta: carbonv1.GetHealthMetas(worker),
			Checked:    carbonv1.IsHealthCheckDone(worker),
			BizMeta:    getBizMetas(worker),
			Warmup:     worker.Status.Warmup,
			Lost:       carbonv1.IsWorkerLost(worker),
			Failed:     carbonv1.IsWorkerFalied(worker),
		}
		// ServiceSkyline 不挂载 3.0
		if service != nil && service.Spec.Type == carbonv1.ServiceSkyline && carbonv1.IsV3(worker) {
			continue
		}
		allNodes = append(allNodes, ipNode)
		if needUpdateGracefully && carbonv1.IsWorkerServiceOffline(worker) {
			continue
		}
		if needUpdateGracefully && !carbonv1.IsWorkerOnline(worker) {
			continue
		}
		if carbonv1.IsStandbyWorker(worker) {
			continue
		}

		publishNodes = append(publishNodes, ipNode)
	}
	return publishNodes, allNodes
}

func (s *abstractPublisher) sync(publishedNodes, toPublishNodes, localNodes []carbonv1.IPNode, softDelete bool) ([]carbonv1.IPNode, error) {
	remoteNodes, err := s.getNodes()
	if err != nil {
		glog.Errorf("get nodes error %s,%s,%s,%v", s.key, s.ptype, s.serviceName, err)
		return nil, err
	}
	s.checkRemoteNodes(remoteNodes)
	needAddNodes, needDeleteNodes, needUpdateNodes := s.diff(publishedNodes, toPublishNodes, localNodes, remoteNodes)
	if glog.V(4) {
		glog.Infof("[%s]: localNodes:%s, remoteNodes:%s, needAddNodes:%s, needDeleteNodes:%s",
			s.serviceName, getIPs(localNodes), getIPs(remoteNodes), getIPs(needAddNodes), getIPs(needDeleteNodes))
	}
	if 0 == len(needAddNodes) && 0 == len(needDeleteNodes) && 0 == len(needUpdateNodes) {
		return s.intersect(remoteNodes, localNodes), nil
	}

	var newRemoteNodes = remoteNodes
	if nil == newRemoteNodes {
		newRemoteNodes = []carbonv1.IPNode{}
	}

	needAddNodes = s.filterLostFaildNodes(needAddNodes)
	if 0 != len(needAddNodes) && !softDelete {
		startTime := time.Now()
		err = s.addNodes(needAddNodes)
		recordSummaryMetric(s.ptype, s.key, addNodesHistogram, startTime, err)
		glog.Infof("[%s:%s] add nodes :%v,%v", s.key, s.ptype, getIPs(needAddNodes), err)
		if nil == err {
			for i := range needAddNodes {
				needAddNodes[i].NewAdded = true
			}
			newRemoteNodes = append(newRemoteNodes, needAddNodes...)
		} else {
			goto End
		}
	}

	needUpdateNodes = s.filterLostFaildNodes(needUpdateNodes)
	if 0 != len(needUpdateNodes) {
		startTime := time.Now()
		err = s.updateNodes(needUpdateNodes)
		recordSummaryMetric(s.ptype, s.key, updateNodesHistogram, startTime, err)
		if nil != err {
			goto End
		}
	}

	if 0 != len(needDeleteNodes) {
		startTime := time.Now()
		err = s.delNodes(needDeleteNodes)
		recordSummaryMetric(s.ptype, s.key, delNodesHistogram, startTime, err)
		glog.Infof("[%s:%s] delete nodes :%v,%v", s.key, s.ptype, getIPs(needDeleteNodes), err)
		if nil != err {
			goto End
		}
	}

End:
	return s.union(s.intersect(newRemoteNodes, localNodes), s.intersect(newRemoteNodes, publishedNodes)), nil
}

func (s *abstractPublisher) checkRemoteNodes(nodes []carbonv1.IPNode) error {
	var nodeSet = map[string]bool{}
	for i := range nodes {
		key := nodes[i].IP
		if nodeSet[key] {
			abnormalService.With(prometheus.Labels{
				"service_name": metric.EscapeLabelKey(s.key),
			}).Add(1)
		}
		nodeSet[key] = true
	}
	return nil
}

func (s *abstractPublisher) filterLostFaildNodes(nodes []carbonv1.IPNode) []carbonv1.IPNode {
	var newNodes = make([]carbonv1.IPNode, 0, len(nodes))
	for i := range nodes {
		node := nodes[i]
		if node.Lost || node.Failed {
			glog.V(4).Infof("%s is filted, lost: %v, failed: %v", node.IP, node.Lost, node.Failed)
			continue
		}
		newNodes = append(newNodes, node)
	}
	return newNodes
}

// toPublish 表示本轮调度所有需要挂载的节点
// publishedNodes 表示之前记录的, 已经成功挂载的节点
// localNodes 表示所有的节点, 包括摘掉的
// remoteNodes 表示远端服务上实际存在的节点
func (s *abstractPublisher) diff(publishedNodes, toPublish []carbonv1.IPNode, localNodes []carbonv1.IPNode, remoteNodes []carbonv1.IPNode) (
	needAddNodes, needDeleteNodes, needUpdateNodes []carbonv1.IPNode) {
	publishedSets := s.sliceToMap(publishedNodes)
	topublishSets := s.sliceToMap(toPublish)
	localSets := s.sliceToMap(localNodes)
	remoteSets := s.sliceToMap(remoteNodes)
	// 获取远端存在的本地节点, 也就是这个Publisher管理的节点中已经存在在远端的节点
	remoteSets = localSets.Intersect(remoteSets)
	// 获取记录了挂载, 但是实际不在当前的节点, 这个节点可能是之前没有删除成功就被释放掉了.
	garbageSets := publishedSets.Difference(localSets)
	// 需要添加的节点就是本地需要挂载但是远端不存在的节点
	needAdds := topublishSets.Difference(remoteSets)
	// 需要删除的节点就是这个Publisher管理的远端存在的节点中, 本轮调度不准备继续挂载的节点
	// 既然不准备继续挂载, 就要删掉
	needDeletes := remoteSets.Difference(topublishSets).Intersect(publishedSets)
	needDeletes = needDeletes.Union(garbageSets)
	needUpdates := topublishSets.Intersect(remoteSets)
	return s.setToSlice(needAdds, localNodes), s.setToSlice(needDeletes, remoteNodes), s.setToSlice(needUpdates, localNodes)
}

func (s *abstractPublisher) intersect(src1, src2 []carbonv1.IPNode) []carbonv1.IPNode {
	src1Sets := s.sliceToMap(src1)
	src2Sets := s.sliceToMap(src2)
	interSets := src1Sets.Intersect(src2Sets)
	return s.setToSlice(interSets, src1)
}

func (s *abstractPublisher) union(src1, src2 []carbonv1.IPNode) []carbonv1.IPNode {
	src1Sets := s.sliceToMap(src1)
	src2Sets := s.sliceToMap(src2)
	interSets := src1Sets.Union(src2Sets)
	return s.setToSliceBoth(interSets, src1, src2)
}

func (s *abstractPublisher) setToSliceBoth(set mapset.Set, nodes ...[]carbonv1.IPNode) []carbonv1.IPNode {
	var ipnodes = make([]carbonv1.IPNode, 0)
	var recoder = map[string]bool{}
	for i := range nodes {
		for j := range nodes[i] {
			if set.Contains(nodes[i][j].IP) {
				if !recoder[nodes[i][j].IP] {
					recoder[nodes[i][j].IP] = true
					ipnodes = append(ipnodes, nodes[i][j])
				}
			}
		}
	}

	return ipnodes
}

func (s *abstractPublisher) setToSlice(set mapset.Set, nodes []carbonv1.IPNode) []carbonv1.IPNode {
	var ipnodes = make([]carbonv1.IPNode, 0, len(nodes))
	for i := range nodes {
		if set.Contains(nodes[i].IP) {
			ipnodes = append(ipnodes, nodes[i])
		}
	}
	return ipnodes
}

func (s *abstractPublisher) sliceToMap(slices []carbonv1.IPNode) mapset.Set {
	sets := mapset.NewSet()
	if nil == slices {
		return sets
	}
	for i := range slices {
		sets.Add(slices[i].IP)
	}
	return sets
}
