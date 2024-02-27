package hippo

import (
	"encoding/json"
)

// This file patches some types in *.pb.go to support marshal enum to string.

func (x AppSummary_Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}

func (x *Resource_Type) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}

func (x ResourceRequest_SpreadLevel) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}

func (x *FieldInfo_Field) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}

func (x *ResourceRequest_AllocateMode) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}

func (x *ApplicationDescription_Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}

func (x *ApplicationDescription_ExclusiveMode) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}

func (x *PackageInfo_PackageType) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}

func (x *ResourceRequest_CpusetMode) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}

func (x *ProcessStatus_Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}

func (x *PackageStatus_Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}

func (x *SlaveStatus_Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}

func (x ErrorCode) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}
