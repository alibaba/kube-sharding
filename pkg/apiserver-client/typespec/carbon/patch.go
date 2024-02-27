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

package carbon

import (
	"encoding/json"

	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/hippo"
)

// ProcessStatus ProcessStatus
type ProcessStatus hippo.ProcessStatus

// TmpProcessStatus TmpProcessStatus
type TmpProcessStatus ProcessStatus

// ProcessStatusStatus ProcessStatusStatus
type ProcessStatusStatus hippo.ProcessStatus_Status

// ProcessStatusJSONMarshaler ProcessStatusJSONMarshaler
type ProcessStatusJSONMarshaler struct {
	TmpProcessStatus
	Status *ProcessStatusStatus `json:"status,omitempty"`
}

// UnmarshalJSON UnmarshalJSON
func (a *ProcessStatus) UnmarshalJSON(b []byte) error {
	var marshaler ProcessStatusJSONMarshaler
	err := json.Unmarshal(b, &marshaler)
	if nil != err {
		return err
	}
	if nil != marshaler.Status {
		marshaler.TmpProcessStatus.Status = (*hippo.ProcessStatus_Status)(marshaler.Status)
	}
	*a = (ProcessStatus)(marshaler.TmpProcessStatus)
	return nil
}

// MarshalJSON MarshalJSON
func (a ProcessStatus) MarshalJSON() ([]byte, error) {
	var marshaler ProcessStatusJSONMarshaler
	marshaler.TmpProcessStatus = TmpProcessStatus(a)
	if nil != marshaler.TmpProcessStatus.Status {
		marshaler.Status = (*ProcessStatusStatus)(marshaler.TmpProcessStatus.Status)
	}
	return json.Marshal(marshaler)
}

func convertProcessStatus(status []*ProcessStatus) []*hippo.ProcessStatus {
	if nil == status {
		return nil
	}
	var hippoStatus = make([]*hippo.ProcessStatus, len(status))
	for i := range status {
		hippoStatus[i] = (*hippo.ProcessStatus)(status[i])
	}
	return hippoStatus
}

func convertHippoProcessStatus(status []*hippo.ProcessStatus) []*ProcessStatus {
	if nil == status {
		return nil
	}
	var hippoStatus = make([]*ProcessStatus, len(status))
	for i := range status {
		hippoStatus[i] = (*ProcessStatus)(status[i])
	}
	return hippoStatus
}

// TmpSlotInfo TmpSlotInfo
type TmpSlotInfo SlotInfo

// PackageStatus PackageStatus_Status
type PackageStatus hippo.PackageStatus_Status

// SlaveStatus SlaveStatus_Status
type SlaveStatus hippo.SlaveStatus_Status

// SlotInfoJSONMarshaler SlotInfoJSONMarshaler
type SlotInfoJSONMarshaler struct {
	TmpSlotInfo            `json:",inline"`
	PackageStatus          *PackageStatus   `json:"packageStatus,omitempty"`
	PreDeployPackageStatus *PackageStatus   `json:"preDeployPackageStatus,omitempty"`
	SlaveStatus            *SlaveStatus     `json:"slaveStatus,omitempty"`
	ProcessStatus          []*ProcessStatus `json:"processStatus,omitempty"`
}

// UnmarshalJSON UnmarshalJSON
func (a *SlotInfo) UnmarshalJSON(b []byte) error {
	var marshaler SlotInfoJSONMarshaler
	err := json.Unmarshal(b, &marshaler)
	if nil != err {
		return err
	}
	if nil != marshaler.PackageStatus {
		marshaler.TmpSlotInfo.PackageStatus = &hippo.PackageStatus{
			Status: (*hippo.PackageStatus_Status)(marshaler.PackageStatus),
		}
	}
	if nil != marshaler.PreDeployPackageStatus {
		marshaler.TmpSlotInfo.PreDeployPackageStatus = &hippo.PackageStatus{
			Status: (*hippo.PackageStatus_Status)(marshaler.PreDeployPackageStatus),
		}
	}
	if nil != marshaler.SlaveStatus {
		marshaler.TmpSlotInfo.SlaveStatus = &hippo.SlaveStatus{
			Status: (*hippo.SlaveStatus_Status)(marshaler.SlaveStatus),
		}
	}
	if nil != marshaler.ProcessStatus {
		marshaler.TmpSlotInfo.ProcessStatus = convertProcessStatus(marshaler.ProcessStatus)
	}
	*a = (SlotInfo)(marshaler.TmpSlotInfo)
	return nil
}

// MarshalJSON MarshalJSON
func (a SlotInfo) MarshalJSON() ([]byte, error) {
	var marshaler SlotInfoJSONMarshaler
	marshaler.TmpSlotInfo = TmpSlotInfo(a)
	if nil != marshaler.TmpSlotInfo.PackageStatus {
		marshaler.PackageStatus = (*PackageStatus)(marshaler.TmpSlotInfo.PackageStatus.Status)
	}
	if nil != marshaler.TmpSlotInfo.PreDeployPackageStatus {
		marshaler.PreDeployPackageStatus = (*PackageStatus)(marshaler.TmpSlotInfo.PreDeployPackageStatus.Status)
	}
	if nil != marshaler.TmpSlotInfo.SlaveStatus {
		marshaler.SlaveStatus = (*SlaveStatus)(marshaler.TmpSlotInfo.SlaveStatus.Status)
	}
	if nil != marshaler.TmpSlotInfo.ProcessStatus {
		marshaler.ProcessStatus = convertHippoProcessStatus(marshaler.TmpSlotInfo.ProcessStatus)
	}
	return json.Marshal(marshaler)
}

// HasError HasError
func (r *Response) HasError() bool {
	return r.GetCode() != 0 || r.GetSubCode() != 0
}

// MarshalJSON MarshalJSON
func (x *PackageStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal((*int32)(x))
}

// MarshalJSON MarshalJSON
func (x *SlaveStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal((*int32)(x))
}

// UnmarshalJSON UnmarshalJSON
func (x *PackageStatus) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*hippo.PackageStatus_Status)(x))
}

// UnmarshalJSON UnmarshalJSON
func (x *SlaveStatus) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*hippo.SlaveStatus_Status)(x))
}

// MarshalJSON MarshalJSON
func (x *ProcessStatusStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal((*int32)(x))
}

// UnmarshalJSON UnmarshalJSON
func (x *ProcessStatusStatus) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, (*hippo.ProcessStatus_Status)(x))
}
