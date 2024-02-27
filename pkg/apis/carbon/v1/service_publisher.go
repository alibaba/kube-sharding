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

package v1

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	glog "k8s.io/klog"
)

// InitServiceFalseState
const (
	InitServiceFalseState = "init with false"
)

const (
	ReasonNotExist          = "Service Not Exist Node"
	ReasonServiceCheckFalse = "Service Check False"
	ReasonQueryError        = "Query NameService With Error"
	ReasonWarmUP            = "Node In Warmup"

	NodeHasPublished = "Node Published"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServicePublisher is a specification for a Publisher for a registry resource
type ServicePublisher struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ServicePublisherSpec   `json:"spec"`
	Status ServicePublisherStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ServicePublisherList is a list of Replica ServicePublisher
type ServicePublisherList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []ServicePublisher `json:"items"`
}

// ServicePublisherSpec is the Spec for a ServicePublisher resource
type ServicePublisherSpec struct {
	// Selector is a label query over pods that should match the replica count.
	// Label keys and values that must match in order to be controlled by this replica set.
	// It must match the pod template's labels.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	Selector map[string]string `json:"selector"`

	ServiceName  string          `json:"serviceName"`
	MetaStr      string          `json:"metaStr"`
	Type         RegistryType    `json:"registryType"`
	RegistryConf json.RawMessage `json:"registryConf"`

	SoftDelete bool `json:"softDelete"`
	// NewVersionRatio means the traffic ratio of new version replicas of a rollingset.
	// +optional
	NewVersionRatio int32 `json:"newVersionRatio,omitempty"`

	// HasBackup means if this service can cut .
	// +optional
	HasBackup bool `json:"hasBackup,omitempty"`
	// Cluster used for .
	// +optional
	Cluster string `json:"cluster"`
	// Disable means cut this service, of if HasBackup is true, Disable can be true.
	// +optional
	Disable bool `json:"disable,omitempty"`
	// Mirror means is this service a mirror of other cluster,
	// if mirror is true ,this publisher only used for cut service of other cluster.
	// +optional
	Mirror bool `json:"mirror,omitempty"`
}

// SkylinePublisherConf is the config of skyline publisher
type SkylinePublisherConf struct {
	AppName          string `json:"appName"`
	Host             string `json:"host"`
	Group            string `json:"group"`
	BuffGroup        string `json:"buffGroup"`
	AppUseType       string `json:"appUseType"`
	ReturnAppUseType string `json:"returnAppUseType"`
	Key              string `json:"key"`
	Secret           string `json:"secret"`
}

type VpcSlbPublisherConf struct {
	SlbApiEndpoint  string `json:"slbApiEndpoint"`
	EcsApiEndpoint  string `json:"ecsApiEndpoint"`
	LoadBalancerId  string `json:"lbId"`            // SLB 实例ID   lb-uf64ua6l5z4h4a2ts7yrr
	VpcId           string `json:"vpcId"`           // VPC 的ID   vpc-uf65qwgtx7npywl2kiwk5
	RegionId        string `json:"regionId"`        // 地域Id cn-shanghai
	VSwitchId       string `json:"vswitchId"`       // 这个参数是用来获取eni的时候，一个过滤条件，非必填
	ResourceGroupId string `json:"resourceGroupId"` // 这个参数同上
	VServerGroupId  string `json:"vServerGroupId"`  // 后端虚拟组ID
	ServicePort     int32  `json:"servicePort"`     // 后端server的端口号
	ListenerPort    int32  `json:"listenerPort"`    // 监听的端口号   这个是checkHealth 那个接口用到的
}

// AppsStatus means a scene plan status on a workerNode
type AppNodeStatus struct {
	AppId         string `json:"appId"`
	StartedAt     uint64 `json:"startedAt"`
	PlanStatus    int32  `json:"planStatus"`
	ServiceStatus int32  `json:"serviceStatus"`
	NeedBackup    bool   `json:"needBackup"`
}

// GrayPublisher means publish gray traffic
type GrayPublisher struct {
	// Selector is a label query over pods that should match the replica count.
	// Label keys and values that must match in order to be controlled by this replica set.
	// It must match the pod template's labels.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	Selector *metav1.LabelSelector `json:"selector"`

	TrafficRatio int32 `json:"trafficRatio"`
}

// IPNode means a node of service
type IPNode struct {
	IP            string `json:"ip"`
	Score         int64  `json:"score"`
	Health        bool   `json:"health"`
	Checked       bool
	Warmup        bool
	InWarmup      bool
	StartWarmup   bool
	Lost          bool
	Failed        bool
	Port          []int32
	Weight        float32
	Valid         bool
	NewAdded      bool
	HealthMeta    map[string]string
	BizMeta       map[string]string
	ServiceMeta   string
	ServiceDetail string
}

// ServicePublisherStatus is the Status for a ServicePublisher resource
type ServicePublisherStatus struct {
	// IpList means the published ips of this service
	IPList  []IPNode `json:"ipList,omitempty"`
	IPs     []string `json:"ips,omitempty"`
	Message string   `json:"message"`
	Reason  string   `json:"reason"`
	//LastUpdateStatusTime 最近一次更新时间
	LastUpdateStatusTime metav1.Time `json:"lastUpdateStatusTime,omitempty"`
}

// SyncWorkerServiceHealthStatus get service health status and sync to worker status
func SyncWorkerServiceHealthStatus(worker *WorkerNode) ServiceStatus {
	if nil == worker {
		return ServiceUnKnown
	}
	worker.Status.ServiceStatus = GetServiceStatusFromCondition(worker)
	return worker.Status.ServiceStatus
}

// SetWorkerServiceHealthCondition set or add a service condition
func SetWorkerServiceHealthCondition(worker *WorkerNode, service *ServicePublisher,
	serviceDetail string, status v1.ConditionStatus, score int64, reason, message string, inWarmup, startWarmup bool) ServiceStatus {
	if nil == worker {
		return ServiceUnKnown
	}
	var deleteCount = 0
	// ServiceGeneral 类型，连续两次查不到才能删除，避免同步问题
	if service.Spec.Type == ServiceGeneral {
		deleteCount = 1
	}
	if status == v1.ConditionTrue || reason == ReasonServiceCheckFalse {
		message = NodeHasPublished
	}

	for j := range worker.Status.ServiceConditions {
		if worker.Status.ServiceConditions[j].ServiceName == service.Name || worker.Status.ServiceConditions[j].Name == service.Name {
			worker.Status.ServiceConditions[j].ServiceName = service.Spec.ServiceName
			worker.Status.ServiceConditions[j].Name = service.Name
			if worker.Status.ServiceConditions[j].Status != status {
				worker.Status.ServiceConditions[j].LastTransitionTime = metav1.NewTime(time.Now())
			}
			worker.Status.ServiceConditions[j].Status = status
			worker.Status.ServiceConditions[j].Score = score
			worker.Status.ServiceConditions[j].Reason = reason
			if strings.Contains(worker.Status.ServiceConditions[j].Message, NodeHasPublished) && status != v1.ConditionTrue && !strings.Contains(message, NodeHasPublished) {
				if message == "" {
					message = NodeHasPublished
				} else {
					message = NodeHasPublished + " and query with error :  " + message
				}
			}
			worker.Status.ServiceConditions[j].Message = message
			worker.Status.ServiceConditions[j].InWarmup = inWarmup
			worker.Status.ServiceConditions[j].StartWarmup = startWarmup
			worker.Status.ServiceConditions[j].DeleteCount = deleteCount
			worker.Status.ServiceConditions[j].Version = worker.Status.Version
			worker.Status.ServiceConditions[j].ServiceDetail = serviceDetail
			return SyncWorkerServiceHealthStatus(worker)
		}
	}

	var condition = ServiceCondition{
		LastTransitionTime: metav1.NewTime(time.Now()),
		Type:               service.Spec.Type,
		ServiceName:        service.Spec.ServiceName,
		Name:               service.Name,
		Status:             status,
		Score:              score,
		Reason:             reason,
		Message:            message,
		InWarmup:           inWarmup,
		StartWarmup:        startWarmup,
		DeleteCount:        deleteCount,
		Version:            worker.Status.Version,
	}
	worker.Status.ServiceConditions = append(worker.Status.ServiceConditions, condition)
	return SyncWorkerServiceHealthStatus(worker)
}

// DeleteWorkerServiceHealthCondition delete a service condition
func DeleteWorkerServiceHealthCondition(worker *WorkerNode, name string, serviceErr error) ServiceStatus {
	if nil == worker {
		return ServiceUnKnown
	}
	var newCondition = make([]ServiceCondition, 0, len(worker.Status.ServiceConditions))
	for i := range worker.Status.ServiceConditions {
		if worker.Status.ServiceConditions[i].ServiceName == name || worker.Status.ServiceConditions[i].Name == name {
			if glog.V(4) {
				glog.Infof("delete service :%s, %s, %s", worker.Name, name, utils.ObjJSON(worker))
			}
			var unpublished = true
			if serviceErr != nil && strings.Contains(worker.Status.ServiceConditions[i].Message, NodeHasPublished) {
				worker.Status.ServiceConditions[i].Reason = ReasonQueryError
				unpublished = false
			}
			if worker.Status.ServiceConditions[i].DeleteCount <= 0 && unpublished {
				continue
			}
			worker.Status.ServiceConditions[i].DeleteCount--
		}
		newCondition = append(newCondition, worker.Status.ServiceConditions[i])
	}
	worker.Status.ServiceConditions = newCondition
	return SyncWorkerServiceHealthStatus(worker)
}

// IsWorkerInWarmup IsWorkerInWarmup
func IsWorkerInWarmup(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}
	var inWarmup int
	var couldWarmup int

	for j := range worker.Status.ServiceConditions {
		if worker.Status.ServiceConditions[j].Type == ServiceCM2 {
			couldWarmup++
			if worker.Status.ServiceConditions[j].InWarmup {
				inWarmup++
			}
		}
	}
	if inWarmup == 0 {
		return false
	}
	return inWarmup == couldWarmup
}

// GetServiceStatusFromCondition get service health status
func GetServiceStatusFromCondition(worker *WorkerNode) ServiceStatus {
	if nil == worker {
		return ServiceUnKnown
	}

	var okService = 0
	var okGracefulService = 0
	for j := range worker.Status.ServiceConditions {
		if worker.Status.ServiceConditions[j].Status == v1.ConditionTrue {
			okService++
			if isGracefullyServiceType(worker.Status.ServiceConditions[j].Type) {
				okGracefulService++
			}
		}
	}

	if len(worker.Status.ServiceConditions) != 0 && okService == 0 {
		return ServiceUnAvailable
	}
	if IsWorkerServiceOffline(worker) && okGracefulService == 0 {
		return ServiceUnAvailable
	}
	if len(worker.Status.ServiceConditions) != okService {
		return ServicePartAvailable
	}

	return ServiceAvailable
}

// IsWorkerUnserviceable IsWorkerUnserviceable
func IsWorkerUnserviceable(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}

	var gracefulService = 0
	var okGracefulService = 0
	var unavailableService = 0
	for j := range worker.Status.ServiceConditions {
		if isGracefullyServiceType(worker.Status.ServiceConditions[j].Type) {
			gracefulService++
			if worker.Status.ServiceConditions[j].Status == v1.ConditionTrue {
				okGracefulService++
			} else if worker.Status.ServiceConditions[j].Reason == ReasonServiceCheckFalse ||
				worker.Status.ServiceConditions[j].Reason == ReasonNotExist {
				unavailableService++
			}
		}
	}

	if gracefulService != 0 && okGracefulService == 0 && unavailableService == gracefulService {
		return true
	}
	return false
}

// IsWorkerUnpublished IsWorkerUnpublished
func IsWorkerUnpublished(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}

	var gracefulService = 0
	var okGracefulService = 0
	var unpublished = 0
	for j := range worker.Status.ServiceConditions {
		if isGracefullyServiceType(worker.Status.ServiceConditions[j].Type) {
			gracefulService++
			if worker.Status.ServiceConditions[j].Status == v1.ConditionTrue {
				okGracefulService++
			} else if worker.Status.ServiceConditions[j].Reason == ReasonNotExist || worker.Status.ServiceConditions[j].Reason == InitServiceFalseState ||
				(worker.Status.ServiceConditions[j].Message != "" && !strings.Contains(worker.Status.ServiceConditions[j].Message, NodeHasPublished)) || // 未发布过的
				(worker.Status.ServiceConditions[j].Type != ServiceGeneral && worker.Status.ServiceConditions[j].DeleteCount < -3600) { // 删除了很久的有问题的service
				unpublished++
			}
		}
	}

	if okGracefulService == 0 && unpublished == gracefulService {
		return true
	}
	return false
}

// IsWorkerOfflined means is service status is UNAVAILABLE and has no conditions
func IsWorkerOfflined(worker *WorkerNode) bool {
	if nil == worker {
		return false
	}

	var gracefulService = 0
	for j := range worker.Status.ServiceConditions {
		if isGracefullyServiceType(worker.Status.ServiceConditions[j].Type) {
			gracefulService++
		}
	}
	return IsWorkerUnAvaliable(worker) && (IsWorkerUnpublished(worker) || gracefulService == 0)
}

// NeedUpdateGracefully is this service need updated gracefully
func NeedUpdateGracefully(service *ServicePublisher) bool {
	if nil == service {
		return true
	}
	return isGracefullyServiceType(service.Spec.Type)
}

func isGracefullyServiceType(registryType RegistryType) bool {
	if ServiceSkyline == registryType {
		return false
	}
	return true
}

// IfWorkerServiceMetasRecovered IfWorkerServiceMetasRecovered
func IfWorkerServiceMetasRecovered(worker *WorkerNode) bool {
	if worker == nil || IsWorkerNotPublishToCM2(worker) {
		return true
	}
	return true
}

// IsWorkerNotPublishToCM2 IsWorkerNotPublishToCM2
func IsWorkerNotPublishToCM2(worker *WorkerNode) bool {
	if worker == nil || len(worker.Status.ServiceConditions) == 0 {
		return true
	}
	var cm2Published = false
	var cm2ServiceCount = 0
	for i := range worker.Status.ServiceConditions {
		if worker.Status.ServiceConditions[i].Type == ServiceCM2 {
			cm2ServiceCount++
			if worker.Status.ServiceConditions[i].Status == v1.ConditionTrue {
				cm2Published = true
				break
			}
			if worker.Status.ServiceConditions[i].Status == v1.ConditionFalse &&
				worker.Status.ServiceConditions[i].Reason != InitServiceFalseState &&
				worker.Status.ServiceConditions[i].Reason != ReasonNotExist ||
				strings.Contains(worker.Status.ServiceConditions[i].Message, NodeHasPublished) {
				cm2Published = true
				break
			}
		}
	}
	if cm2ServiceCount == 0 {
		return true
	}
	return !cm2Published
}

func NeedReclaimByService(worker *WorkerNode) bool {
	if worker.Status.BadReason == BadReasonServiceReclaim {
		return true
	}
	for i := range worker.Status.ServiceConditions {
		condition := worker.Status.ServiceConditions[i]
		if condition.Type != ServiceGeneral || condition.ServiceDetail == "" {
			continue
		}
		var status []AppNodeStatus
		_ = json.Unmarshal([]byte(condition.ServiceDetail), &status)
		for j := range status {
			if status[j].NeedBackup {
				return true
			}
		}
	}
	return false
}
