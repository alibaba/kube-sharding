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

package typespec

import (
	"fmt"
	"log"
	"reflect"
	"strconv"
	"strings"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/hippo"
)

// Common text
const (
	AppMasterTag = "internal_appmaster_resource_tag"
)

const (
	slotResourcePath      = "slotResource/resources"
	slaveResourcePath     = "slaveResource/resources"
	slaveUsedResourcePath = "usedSlaveResource/resources"
	requestResourcePath   = "options/0/resources"
)

// IsMasterTag is the tag is am
func IsMasterTag(tag string) bool {
	return tag == AppMasterTag
}

// GetRecourceFromResources GetRecourceFromResources
func GetRecourceFromResources(resourceName string, resourceType string, resources []*hippo.Resource) (int32, error) {
	for _, resource := range resources {
		if nil != resource.Type && resourceType == resource.GetType().String() &&
			nil != resource.Name && strings.HasPrefix(*resource.Name, resourceName) && nil != resource.Amount {
			return *resource.Amount, nil
		}
	}
	return 0, fmt.Errorf("can't find resource :%s,%s", resourceName, resourceType)
}

// GetResourceSlice GetResourceSlice
func GetResourceSlice(src interface{}, resourcePath string) ([]*hippo.Resource, error) {
	result, err := utils.GetFieldValue(src, resourcePath)
	if nil != err {
		return nil, err
	}
	var resources []*hippo.Resource
	err = utils.JSONDeepCopy(result, &resources)
	return resources, err
}

// GetSlotResources GetSlotResources
func GetSlotResources(src interface{}) ([]*hippo.Resource, error) {
	return GetResourceSlice(src, slotResourcePath)
}

// GetSlaveResources GetSlaveResources
func GetSlaveResources(src interface{}) ([]*hippo.Resource, error) {
	return GetResourceSlice(src, slaveResourcePath)
}

// GetSlaveUsedResources GetSlaveUsedResources
func GetSlaveUsedResources(src interface{}) ([]*hippo.Resource, error) {
	return GetResourceSlice(src, slaveUsedResourcePath)
}

// GetRequestResources GetRequestResources
func GetRequestResources(src interface{}) ([]*hippo.Resource, error) {
	return GetResourceSlice(src, requestResourcePath)
}

// GetWorkDirTag GetWorkDirTag
func GetWorkDirTag(req *hippo.ResourceRequest) string {
	if nil != req.MetaTags {
		for _, para := range req.MetaTags {
			if nil != para.Key && "META_TAG_WORKDIR" == *para.Key &&
				nil != para.Value {
				return *para.Value
			}
		}
	}
	return *req.Tag
}

// GetWorkDirTagFromSlot GetWorkDirTagFromSlot
func GetWorkDirTagFromSlot(slot *hippo.SlotStatus) string {
	for _, para := range slot.GetMetaTags() {
		if "META_TAG_WORKDIR" == para.GetKey() {
			return para.GetValue()
		}
	}
	return slot.GetResourceTag()
}

// GetResource GetResource
func GetResource(src interface{}, resourcePath string, resourceName string, resourceType string) (int32, error) {
	resources, err := GetResourceSlice(src, resourcePath)
	if nil == err {
		return GetRecourceFromResources(resourceName, resourceType, resources)
	}
	return 0, err
}

// GetSlotResource GetSlotResource
func GetSlotResource(src interface{}, resourceName string, resourceType string) (int32, error) {
	return GetResource(src, slotResourcePath, resourceName, resourceType)
}

// GetSlotMem GetSlotMem
func GetSlotMem(src interface{}) (int32, error) {
	return GetSlotResource(src, "mem", "SCALAR")
}

// GetSlotCPU GetSlotCPU
func GetSlotCPU(src interface{}) (int32, error) {
	return GetSlotResource(src, "cpu", "SCALAR")
}

// GetSlotIP GetSlotIP
func GetSlotIP(src interface{}) (int32, error) {
	return GetSlotResource(src, "ip", "SCALAR")
}

// GetSlotGpu GetSlotGpu
func GetSlotGpu(src interface{}) (int32, error) {
	return GetSlotResource(src, "gpu", "SCALAR")
}

// GetSlotFpga GetSlotFpga
func GetSlotFpga(src interface{}) (int32, error) {
	return GetSlotResource(src, "fpga", "SCALAR")
}

// GetSlotDisk GetSlotDisk
func GetSlotDisk(src interface{}) (int32, error) {
	disksize, _ := GetSlotResource(src, "disk_size", "SCALAR")
	disksize1, _ := GetSlotResource(src, "disk_size_1", "SCALAR")
	disksize9999, _ := GetSlotResource(src, "disk_size_9999", "SCALAR")
	return disksize + disksize1 + disksize9999, nil
}

// GetSlotRes GetSlotRes
func GetSlotRes(src interface{}) (int32, int32, int32, int32, int32, int32) {
	cpu, _ := GetSlotCPU(src)
	mem, _ := GetSlotMem(src)
	disk, _ := GetSlotDisk(src)
	ip, _ := GetSlotIP(src)
	gpu, _ := GetSlotGpu(src)
	fpga, _ := GetSlotFpga(src)
	return cpu, mem, disk, ip, gpu, fpga
}

// GetSlaveResource GetSlaveResource
func GetSlaveResource(src interface{}, resourceName string, resourceType string) (int32, error) {
	return GetResource(src, slaveResourcePath, resourceName, resourceType)
}

// GetSlaveMem GetSlaveMem
func GetSlaveMem(src interface{}) (int32, error) {
	return GetSlaveResource(src, "mem", "SCALAR")
}

// GetSlaveCPU GetSlaveCpu
func GetSlaveCPU(src interface{}) (int32, error) {
	return GetSlaveResource(src, "cpu", "SCALAR")
}

// GetSlaveDisk GetSlaveDisk
func GetSlaveDisk(src interface{}) (int32, error) {
	return GetSlaveResource(src, "disk_size", "SCALAR")
}

// GetTotalSlaveDisk GetTotalSlaveDisk
func GetTotalSlaveDisk(src interface{}) (int32, error) {
	size9999, _ := GetSlaveResource(src, "disk_size_9999", "SCALAR")
	size1, _ := GetSlaveResource(src, "disk_size_1", "SCALAR")
	size, _ := GetSlaveResource(src, "disk_size", "SCALAR")
	return size + size9999 + size1, nil
}

// GetFreeCPU GetFreeCPU
func GetFreeCPU(src interface{}) (int32, error) {
	cpu, err := GetSlaveCPU(src)
	if nil != err {
		return 0, err
	}
	usedcpu, err := GetSlaveUsedCPU(src)
	if nil != err {
		return 0, err
	}
	return cpu - usedcpu, nil
}

// GetFreeMem GetFreeMem
func GetFreeMem(src interface{}) (int32, error) {
	mem, err := GetSlaveMem(src)
	if nil != err {
		return 0, err
	}
	usedmem, err := GetSlaveUsedMem(src)
	if nil != err {
		return 0, err
	}
	return mem - usedmem, nil
}

// GetFreeDISK GetFreeDISK
func GetFreeDISK(src interface{}) (int32, error) {
	disk, err := GetTotalSlaveDisk(src)
	if nil != err {
		return 0, err
	}
	useddisk, err := GetSlaveUsedDisk(src)
	if nil != err {
		return 0, err
	}
	return disk - useddisk, nil
}

// GetFreeResource GetFreeResource
func GetFreeResource(src interface{}) (int32, int32, int32) {
	freeCPU, _ := GetFreeCPU(src)
	freeMem, _ := GetFreeMem(src)
	freeDisk, _ := GetFreeDISK(src)
	return freeCPU, freeMem, freeDisk
}

// GetSlaveUsedResource GetSlaveUsedResource
func GetSlaveUsedResource(src interface{}, resourceName string, resourceType string) (int32, error) {
	return GetResource(src, slaveUsedResourcePath, resourceName, resourceType)
}

// GetSlaveUsedMem GetSlaveUsedMem
func GetSlaveUsedMem(src interface{}) (int32, error) {
	return GetSlaveUsedResource(src, "mem", "SCALAR")
}

// GetSlaveUsedCPU GetSlaveUsedCPU
func GetSlaveUsedCPU(src interface{}) (int32, error) {
	return GetSlaveUsedResource(src, "cpu", "SCALAR")
}

// GetSlaveUsedDisk GetSlaveUsedDisk
func GetSlaveUsedDisk(src interface{}) (int32, error) {
	return GetSlaveUsedResource(src, "disk_size", "SCALAR")
}

// GetRequestResource GetRequestResource
func GetRequestResource(src interface{}, resourceName string, resourceType string) (int32, error) {
	return GetResource(src, requestResourcePath, resourceName, resourceType)
}

// GetRequestMem GetRequestMem
func GetRequestMem(src interface{}) (int32, error) {
	return GetRequestResource(src, "mem", "SCALAR")
}

// GetRequestCPU GetRequestCPU
func GetRequestCPU(src interface{}) (int32, error) {
	return GetRequestResource(src, "cpu", "SCALAR")
}

// GetRequestIP GetRequestIP
func GetRequestIP(src interface{}) (int32, error) {
	return GetRequestResource(src, "ip", "SCALAR")
}

// GetRequestDisk GetRequestDisk
func GetRequestDisk(src interface{}) (int32, error) {
	disksize, _ := GetRequestResource(src, "disk_size", "SCALAR")
	disksize1, _ := GetRequestResource(src, "disk_size_1", "SCALAR")
	disksize9999, _ := GetRequestResource(src, "disk_size_9999", "SCALAR")
	return disksize + disksize1 + disksize9999, nil
}

// GetRequestGpu GetRequestGpu
func GetRequestGpu(src interface{}) (int32, error) {
	return GetRequestResource(src, "gpu", "SCALAR")
}

// GetRequestFpga GetRequestFpga
func GetRequestFpga(src interface{}) (int32, error) {
	return GetRequestResource(src, "fpga", "SCALAR")
}

// GetRequestRes GetRequestRes
func GetRequestRes(src interface{}) (int32, int32, int32, int32, int32, int32) {
	cpu, _ := GetRequestCPU(src)
	mem, _ := GetRequestMem(src)
	disk, _ := GetRequestDisk(src)
	ip, _ := GetRequestIP(src)
	gpu, _ := GetRequestGpu(src)
	fpga, _ := GetRequestFpga(src)
	return cpu, mem, disk, ip, gpu, fpga
}

// GetAppstatusCPU GetAppstatusCPU
func GetAppstatusCPU(app *hippo.AppStatusResponse) int32 {
	if nil == app || nil == app.LastAllocateRequest || nil == app.LastAllocateRequest.Require {
		return 0
	}
	var appCPU int32
	for _, request := range app.LastAllocateRequest.Require {
		if nil != request.Priority && nil != request.Priority.MajorPriority {
			if *request.Priority.MajorPriority > 32 {
				continue
			}
		}
		cpu, _ := GetRequestCPU(request)
		cpu *= *request.Count
		appCPU += cpu
	}
	return appCPU
}

// Flater Flater
type Flater interface {
	Flat(prefix string) map[string]interface{}
}

// FlatSlaves FlatSlaves
func FlatSlaves(slaveNodes []*hippo.SlaveNodeStatus) ([]map[string]interface{}, []map[string]interface{}) {
	var slaveResults = make([]map[string]interface{}, 0)
	var slotsResults = make([]map[string]interface{}, 0)

	for _, slaveNode := range slaveNodes {
		slaveResult, slotsResult := FlatSlave(slaveNode)
		if nil == slaveResult || nil == slotsResult {
			continue
		}
		slaveResults = append(slaveResults, slaveResult)
		slotsResults = append(slotsResults, slotsResult...)
	}

	return slaveResults, slotsResults
}

// FlatSlave FlatSlave
func FlatSlave(slaveNode *hippo.SlaveNodeStatus) (map[string]interface{}, []map[string]interface{}) {
	var slaveResult = make(map[string]interface{})
	slaveFlat := FlatStruct("slave", slaveNode)
	if nil != slaveNode.HeartbeatStatus && nil != slaveNode.HeartbeatStatus.Status {
		slaveFlat["slave_status"] = *slaveNode.HeartbeatStatus.Status
	}

	if nil != slaveNode.Health && nil != slaveNode.Health.IsHealthy {
		slaveFlat["slave_health"] = *slaveNode.Health.IsHealthy
	} else {
		slaveFlat["slave_health"] = true
	}

	mergeMap(slaveResult, slaveFlat)
	slaveResource, err := GetSlaveResources(slaveNode)
	if nil == err && nil != slaveResource {
		flatSlaveResource := FlatResource("slave", slaveResource)
		mergeMap(slaveResult, flatSlaveResource)
	} else {
		log.Print(err, *slaveNode.Address)
		return nil, nil
	}
	slaveUsedResource, err := GetSlaveUsedResources(slaveNode)
	if nil == err {
		flatSlaveResource := FlatResource("slave_used", slaveUsedResource)
		mergeMap(slaveResult, flatSlaveResource)
	}

	if nil != slaveNode.Visibilities {
		flatSlaveVisibility := FlatVisility("slave_visibility", slaveNode.Visibilities)
		mergeMap(slaveResult, flatSlaveVisibility)
	}

	var slotsResult = make([]map[string]interface{}, len(slaveNode.Slots))
	for _, slot := range slaveNode.Slots {
		if nil != slot.Priority && nil != slot.Priority.MajorPriority && *slot.Priority.MajorPriority > 32 {
			//continue
		}

		var slotResult = make(map[string]interface{})
		flatSlot := FlatStruct("slot", slot)
		if nil != slot.Priority && nil != slot.Priority.MajorPriority {
			flatSlot["slot_priority"] = *slot.Priority.MajorPriority
		}
		if nil != slot.Cpusets && "" != *slot.Cpusets {
			cpus := strings.Split(*slot.Cpusets, ",")
			flatSlot["slot_cpu_set"] = len(cpus)
		}
		mergeMap(slotResult, flatSlot)
		slotResource, err := GetSlotResources(slot)
		if nil == err {
			flatSlotResource := FlatResource("slot", slotResource)
			mergeMap(slotResult, flatSlotResource)
			if v, ok := flatSlotResource["slot_exclusive"]; ok {
				slaveResult["slave_exclusive"] = appendValue(slaveResult["slave_exclusive"], v.(string))
			}
			if flatSlotResource["slot_ip"] != 0 {
				slaveResult["slave_used_ip"] = slaveResult["slave_used_ip"].(int) + 1
			}
			if flatSlotResource["slot_fpga"] != 0 {
				slaveResult["slave_used_fpga"] = slaveResult["slave_used_fpga"].(int) + 1
			}
			if flatSlotResource["slot_gpu"] != 0 {
				slaveResult["slave_used_gpu"] = slaveResult["slave_used_gpu"].(int) + 1
			}
		} else {
			log.Print(err)
		}
		slaveResult["slave_tags"] = appendValue(slaveResult["slave_tags"], *slot.ApplicationId+"."+*slot.ResourceTag)
		slaveResult["slave_free_ip"] = slaveResult["slave_ip"].(int) - slaveResult["slave_used_ip"].(int)
		slaveResult["slave_free_fpga"] = slaveResult["slave_fpga"].(int) - slaveResult["slave_used_fpga"].(int)
		slaveResult["slave_free_gpu"] = slaveResult["slave_gpu"].(int) - slaveResult["slave_used_gpu"].(int)
		mergeMap(slotResult, slaveResult)
		slotsResult = append(slotsResult, slotResult)
	}
	return slaveResult, slotsResult
}

func stringptr(str string) *string {
	return &str
}

// FlatResource FlatResource
func FlatResource(prefix string, resources []*hippo.Resource) map[string]interface{} {
	if nil == resources {
		return nil
	}
	var result = make(map[string]interface{})
	var ip = 0
	var fpga = 0
	var gpu = 0
	for i := range resources {
		name := resources[i].GetName()
		if strings.Contains(name, "ip_") {
			ip++
		}
		if resources[i].GetType() == hippo.Resource_SCALAR {
			name := resources[i].GetName()
			if name == "disk_size_9999" {
				name = "disk"
			} else if strings.HasPrefix(name, "disk_size") {
				name = "disk"
			} else if name == "/dev/nvidia0" {
				name = "nvidia"
				gpu++
			} else if name == "/dev/nvidia1" {
				name = "nvidia"
				gpu++
			} else if strings.HasPrefix(name, "ip_") {
				continue
			} else if name == "/dev/xdma0" {
				name = "xdma"
				fpga++
			} else if name == "ip" {
				ip++
			} else if name == "fpga" && prefix == "slot" {
				resources[i].Type = (*hippo.Resource_Type)(utils.Int32Ptr(int32(hippo.Resource_EXCLUSIVE)))
				resources = append(resources, resources[i])
				fpga++
			}

			key := joinKey(prefix, name)
			result[key] = resources[i].GetAmount()
		} else {
			if "" == resources[i].GetType().String() {
				continue
			}
			key := joinKey(prefix, resources[i].GetType().String())
			result[key] = appendValue(result[key], resources[i].GetName())
		}
	}
	key := joinKey(prefix, "ip")
	result[key] = ip
	fpgakey := joinKey(prefix, "fpga")
	result[fpgakey] = fpga
	gpukey := joinKey(prefix, "gpu")
	result[gpukey] = gpu
	return result
}

// FlatStruct FlatStruct
func FlatStruct(prefix string, st interface{}) map[string]interface{} {
	var result = make(map[string]interface{})
	fromv := utils.Indirect(reflect.ValueOf(st))

	for i := 0; i < fromv.Type().NumField(); i++ {
		sf := fromv.Type().Field(i)
		if utils.IsZero(fromv.Field(i)) {
			continue
		}
		if !utils.IsSimpleType(utils.Indirect(fromv.Field(i)).Type()) {
			continue
		}
		tags := sf.Tag.Get("json")
		name := sf.Name
		if tags == "" || tags == "-" {
		} else {
			tagspl := strings.SplitN(tags, ",", 2)
			if len(tagspl) > 0 {
				name = tagspl[0]
			}
		}
		key := joinKey(prefix, name)
		result[key] = utils.Indirect(fromv.Field(i)).Interface()
	}
	return result
}

// FlatMap FlatMap
func FlatMap(prefix string, m map[string]interface{}) map[string]interface{} {
	var result = make(map[string]interface{})
	for k, v := range m {
		sf := utils.Indirect(reflect.ValueOf(v))
		if !utils.IsSimpleType(sf.Type()) {
			continue
		}
		key := joinKey(prefix, k)
		result[key] = sf.Interface()
	}
	return result
}

// FlatVisility FlatVisility
func FlatVisility(prefix string, vis []*hippo.Visibility) map[string]interface{} {
	var result = make(map[string]interface{})
	for i := range vis {
		key := joinKey(prefix, vis[i].GetOp().String())
		pattern := vis[i].GetPattern() + ";"
		if vis[i].GetScope() == hippo.Visibility_GLOBAL {
			pattern = hippo.Visibility_GLOBAL.String()
		}
		result[key] = appendValue(result[key], pattern)
	}
	return result
}

func stringEqualIgnoreCase(s1, s2 string) bool {
	return (strings.ToLower(s1) == strings.ToLower(s2))
}

func joinKey(s ...string) string {
	sep := "_"
	s1 := make([]string, 0)
	for i := range s {
		if "" != s[i] {
			s[i] = strings.ToLower(s[i])
			s1 = append(s1, s[i])
		}
	}
	if 1 == len(s1) {
		return s1[0]
	}

	return strings.Join(s1, sep)
}

func joinValue(s ...string) string {
	sep := ";"
	if 1 == len(s) {
		return s[0]
	}
	return strings.Join(s, sep)
}

func appendValue(s1 interface{}, appends ...string) string {
	s := ""
	if nil != s1 {
		s = s1.(string)
	}
	if s == "" {
		return joinValue(appends...)
	}
	appends = append(appends, s)
	return joinValue(appends...)
}

func mergeMap(dst, src map[string]interface{}) {
	if nil == dst {
		return
	}
	for k, v := range src {
		dst[k] = v
	}
}

// FlatAppStatus FlatAppStatus
func FlatAppStatus(app *hippo.AppStatusResponse) (map[string]map[string]interface{}, []map[string]interface{}) {
	var mapResults = make(map[string]map[string]interface{})
	var sliceResults = make([]map[string]interface{}, 0)
	if nil == app.LastAllocateRequest || nil == app.LastAllocateResponse {
		return nil, nil
	}
	for _, request := range app.LastAllocateRequest.Require {
		if *request.Tag == "__internal_appmaster_resource_tag__" {
			continue
		}
		var flatTag = make(map[string]interface{})
		flatRequest := FlatStruct("req", request)
		mergeMap(flatTag, flatRequest)
		resources, err := GetRequestResources(request)
		if nil == err {
			flatResource := FlatResource("req", resources)
			mergeMap(flatTag, flatResource)
		}
		if nil != request.Priority && nil != request.Priority.MajorPriority {
			flatTag["req_priority"] = int32(*request.Priority.MajorPriority)
		} else {
			flatTag["req_priority"] = -1
		}
		if nil != request.MetaTags {
			for _, configs := range request.ContainerConfigs {
				kvs := strings.Split(configs, "=")
				if len(kvs) == 2 && kvs[0] == "CPU_RESERVED_TO_SHARE_NUM" {
					sharedcpu, _ := strconv.Atoi(kvs[1])
					flatTag["req_share_cpu"] = sharedcpu
				}
			}
		}
		workdirtag := GetWorkDirTag(request)
		tag := request.GetTag()
		flatTag["req_applicationid"] = app.LastAllocateRequest.ApplicationId
		flatTag["req_id"] = app.LastAllocateRequest.GetApplicationId() + "." + tag
		flatTag["req_workdirtag"] = workdirtag
		if 0 == int(*request.Count) {
			flatTag["req_gap"] = 0
			flatTag["req_allocated"] = 0
		} else {
			for _, response := range app.LastAllocateResponse {
				if *response.ResourceTag == *request.Tag {
					flatTag["req_allocated"] = len(response.AssignedSlots)
					flatTag["req_gap"] = int(*request.Count) - len(response.AssignedSlots)
				}
			}
		}
		mapResults[app.LastAllocateRequest.GetApplicationId()+"."+tag] = flatTag
		sliceResults = append(sliceResults, flatTag)
	}

	return mapResults, sliceResults
}

// GetRequestFromAppstatus GetRequestFromAppstatus
func GetRequestFromAppstatus(app *hippo.AppStatusResponse, tag string) *hippo.ResourceRequest {
	if nil == app || nil == app.LastAllocateRequest || nil == app.LastAllocateRequest.Require {
		return nil
	}
	for _, request := range app.LastAllocateRequest.Require {
		if *request.Tag == tag || strings.HasPrefix(*request.Tag, tag) {
			return request
		}
	}
	return nil
}

// GetResponseFromAppstatus GetResponseFromAppstatus
func GetResponseFromAppstatus(app *hippo.AppStatusResponse, tag string) *hippo.ResourceResponse {
	if nil == app || nil == app.LastAllocateResponse {
		return nil
	}
	for _, response := range app.LastAllocateResponse {
		if *response.ResourceTag == tag || strings.HasPrefix(*response.ResourceTag, tag) {
			return response
		}
	}
	return nil
}

// FlatedSlave FlatedSlave
type FlatedSlave struct {
	SlaveSlaveid           int64  `json:"slave_slaveid"`
	SlaveAddress           string `json:"slave_address"`
	SlaveSlavehttpport     int32  `json:"slave_slavehttpport"`
	SlaveRestfulhttpport   int32  `json:"slave_restfulhttpport"`
	SlaveState             string `json:"slave_state"`
	SlaveLastheartbeattime int64  `json:"slave_lastheartbeattime"`
	SlaveTags              string `json:"slave_tags" sql:"type:text"`
	SlaveIsoffline         bool   `json:"slave_isoffline"`
	SlaveStatus            string `json:"slave_status"`

	SlaveResourcegroup string `json:"slave_resourcegroup"`
	SlaveQueue         string `json:"slave_queue"`
	SlaveText          string `json:"slave_text"`
	SlaveExcludeText   string `json:"slave_exclude_text"`
	SlaveExclusive     string `json:"slave_exclusive" sql:"type:text"`
	SlaveVisibilityIn  string `json:"slave_visibility_in" sql:"type:text"`

	SlaveCPU       int32 `json:"slave_cpu"`
	SlaveMem       int32 `json:"slave_mem"`
	SlaveDisk      int32 `json:"slave_disk"`
	SlaveUsedCPU   int32 `json:"slave_used_cpu"`
	SlaveUsedMem   int32 `json:"slave_used_mem"`
	SlaveUsedDisk  int32 `json:"slave_used_disk"`
	SlaveSlotcount int32 `json:"slave_slotcount"`
	SlaveIP        int32 `json:"slave_ip"`
	SlaveFreeIP    int32 `json:"slave_free_ip"`

	SlaveBinaryversion string `json:"slave_binaryversion"`

	ClusterID string `json:"cluster_id"`
	UpdateAt  int64  `json:"update_at"`
}

// FlatedApp FlatedApp
type FlatedApp struct {
	ReqID            string `json:"req_id"`
	ReqApplicationid string `json:"req_applicationid"`
	ReqTag           string `json:"req_tag"`

	ReqGroupid     string `json:"req_groupid"`
	ReqText        string `json:"req_text"`
	ReqExclusive   string `json:"req_exclusive"`
	ReqExcludeText string `json:"req_exclude_text"`
	ReqQueue       string `json:"req_queue"`
	ReqPriority    int32  `json:"req_priority"`

	ReqCpusetmode string `json:"req_cpusetmode"`
	ReqCPU        int32  `json:"req_cpu"`
	ReqMem        int32  `json:"req_mem"`
	ReqDisk       int32  `json:"req_disk"`
	ReqGpu        int32  `json:"req_gpu"`
	ReqCount      int32  `json:"req_count"`
	ReqGap        int    `json:"req_gap"`
	ReqAllocated  int    `json:"req_allocated"`
	ReqShareCPU   int    `json:"req_share_cpu"`

	ReqWorkdirtag    string `json:"req_workdirtag"`
	ReqAllocatemode  string `json:"req_allocatemode"`
	ReqRequirementid string `json:"req_requirementid"`

	ClusterID string `json:"cluster_id"`
	UpdateAt  int64  `json:"update_at"`
}

// FlatedSlot FlatedSlot
type FlatedSlot struct {
	SlotSlotid        int32  `json:"slot_slotid"`
	SlotResourcetag   string `json:"slot_resourcetag"`
	SlotApplicationid string `json:"slot_applicationid"`
	SlotRequirementid string `json:"slot_requirementid"`
	SlotIsstopping    bool   `json:"slot_isstopping"`

	SlotGroupid     string `json:"slot_groupid"`
	SlotText        string `json:"slot_text"`
	SlotExcludeText string `json:"slot_exclude_text"`
	SlotExclusive   string `json:"slot_exclusive"`

	SlotCPU      int32  `json:"slot_cpu"`
	SlotCPUSet   int32  `json:"slot_cpu_set"`
	SlotMem      int32  `json:"slot_mem"`
	SlotDisk     int32  `json:"slot_disk"`
	SlotNvidia0  int32  `json:"slot_nvidia0"`
	SlotPriority uint32 `json:"slot_priority"`

	SlaveAddress         string `json:"slave_address"`
	SlaveRestfulhttpport int32  `json:"slave_restfulhttpport"`
	SlaveIsoffline       bool   `json:"slave_isoffline"`
	SlaveStatus          string `json:"slave_status"`

	ClusterID string `json:"cluster_id"`
	UpdateAt  int64  `json:"update_at"`
}

// FlatedSummary FlatedSummary
type FlatedSummary struct {
	SlotSlotid        int32  `json:"slot_slotid"`
	SlotResourcetag   string `json:"slot_resourcetag"`
	SlotApplicationid string `json:"slot_applicationid"`
	SlotGroupid       string `json:"slot_groupid"`

	SlotCPU      int32  `json:"slot_cpu"`
	SlotCPUSet   int32  `json:"slot_cpu_set"`
	SlotMem      int32  `json:"slot_mem"`
	SlotDisk     int32  `json:"slot_disk"`
	SlotPriority uint32 `json:"slot_priority"`

	ReqCPU        int32  `json:"req_cpu"`
	ReqCount      int32  `json:"req_count"`
	ReqAllocated  int    `json:"req_allocated"`
	ReqWorkdirtag string `json:"req_workdirtag"`

	SlaveAddress         string `json:"slave_address"`
	SlaveRestfulhttpport int32  `json:"slave_restfulhttpport"`
	SlaveIsoffline       bool   `json:"slave_isoffline"`
	SlaveStatus          string `json:"slave_status"`

	SlaveCPU      int32 `json:"slave_cpu"`
	SlaveMem      int32 `json:"slave_mem"`
	SlaveDisk     int32 `json:"slave_disk"`
	SlaveUsedCPU  int32 `json:"slave_used_cpu"`
	SlaveUsedMem  int32 `json:"slave_used_mem"`
	SlaveUsedDisk int32 `json:"slave_used_disk"`

	ClusterID string `json:"cluster_id"`
	UpdateAt  int64  `json:"update_at"`
}
