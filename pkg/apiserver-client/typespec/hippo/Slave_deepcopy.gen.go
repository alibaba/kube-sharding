// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: Slave.proto

package hippo

import (
	fmt "fmt"
	proto "github.com/gogo/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// DeepCopyInto supports using Metric within kubernetes types, where deepcopy-gen is used.
func (in *Metric) DeepCopyInto(out *Metric) {
	p := proto.Clone(in).(*Metric)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Metric. Required by controller-gen.
func (in *Metric) DeepCopy() *Metric {
	if in == nil {
		return nil
	}
	out := new(Metric)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new Metric. Required by controller-gen.
func (in *Metric) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SocketInfo within kubernetes types, where deepcopy-gen is used.
func (in *SocketInfo) DeepCopyInto(out *SocketInfo) {
	p := proto.Clone(in).(*SocketInfo)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SocketInfo. Required by controller-gen.
func (in *SocketInfo) DeepCopy() *SocketInfo {
	if in == nil {
		return nil
	}
	out := new(SocketInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SocketInfo. Required by controller-gen.
func (in *SocketInfo) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SocketNodePolicy within kubernetes types, where deepcopy-gen is used.
func (in *SocketNodePolicy) DeepCopyInto(out *SocketNodePolicy) {
	p := proto.Clone(in).(*SocketNodePolicy)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SocketNodePolicy. Required by controller-gen.
func (in *SocketNodePolicy) DeepCopy() *SocketNodePolicy {
	if in == nil {
		return nil
	}
	out := new(SocketNodePolicy)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SocketNodePolicy. Required by controller-gen.
func (in *SocketNodePolicy) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlotInfo within kubernetes types, where deepcopy-gen is used.
func (in *SlotInfo) DeepCopyInto(out *SlotInfo) {
	p := proto.Clone(in).(*SlotInfo)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlotInfo. Required by controller-gen.
func (in *SlotInfo) DeepCopy() *SlotInfo {
	if in == nil {
		return nil
	}
	out := new(SlotInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlotInfo. Required by controller-gen.
func (in *SlotInfo) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlaveInfo within kubernetes types, where deepcopy-gen is used.
func (in *SlaveInfo) DeepCopyInto(out *SlaveInfo) {
	p := proto.Clone(in).(*SlaveInfo)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlaveInfo. Required by controller-gen.
func (in *SlaveInfo) DeepCopy() *SlaveInfo {
	if in == nil {
		return nil
	}
	out := new(SlaveInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlaveInfo. Required by controller-gen.
func (in *SlaveInfo) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlotDetail within kubernetes types, where deepcopy-gen is used.
func (in *SlotDetail) DeepCopyInto(out *SlotDetail) {
	p := proto.Clone(in).(*SlotDetail)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlotDetail. Required by controller-gen.
func (in *SlotDetail) DeepCopy() *SlotDetail {
	if in == nil {
		return nil
	}
	out := new(SlotDetail)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlotDetail. Required by controller-gen.
func (in *SlotDetail) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlaveDetail within kubernetes types, where deepcopy-gen is used.
func (in *SlaveDetail) DeepCopyInto(out *SlaveDetail) {
	p := proto.Clone(in).(*SlaveDetail)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlaveDetail. Required by controller-gen.
func (in *SlaveDetail) DeepCopy() *SlaveDetail {
	if in == nil {
		return nil
	}
	out := new(SlaveDetail)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlaveDetail. Required by controller-gen.
func (in *SlaveDetail) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using NameAmount within kubernetes types, where deepcopy-gen is used.
func (in *NameAmount) DeepCopyInto(out *NameAmount) {
	p := proto.Clone(in).(*NameAmount)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NameAmount. Required by controller-gen.
func (in *NameAmount) DeepCopy() *NameAmount {
	if in == nil {
		return nil
	}
	out := new(NameAmount)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new NameAmount. Required by controller-gen.
func (in *NameAmount) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlotAssignment within kubernetes types, where deepcopy-gen is used.
func (in *SlotAssignment) DeepCopyInto(out *SlotAssignment) {
	p := proto.Clone(in).(*SlotAssignment)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlotAssignment. Required by controller-gen.
func (in *SlotAssignment) DeepCopy() *SlotAssignment {
	if in == nil {
		return nil
	}
	out := new(SlotAssignment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlotAssignment. Required by controller-gen.
func (in *SlotAssignment) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlaveAssignment within kubernetes types, where deepcopy-gen is used.
func (in *SlaveAssignment) DeepCopyInto(out *SlaveAssignment) {
	p := proto.Clone(in).(*SlaveAssignment)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlaveAssignment. Required by controller-gen.
func (in *SlaveAssignment) DeepCopy() *SlaveAssignment {
	if in == nil {
		return nil
	}
	out := new(SlaveAssignment)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlaveAssignment. Required by controller-gen.
func (in *SlaveAssignment) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SystemConfig within kubernetes types, where deepcopy-gen is used.
func (in *SystemConfig) DeepCopyInto(out *SystemConfig) {
	p := proto.Clone(in).(*SystemConfig)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SystemConfig. Required by controller-gen.
func (in *SystemConfig) DeepCopy() *SystemConfig {
	if in == nil {
		return nil
	}
	out := new(SystemConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SystemConfig. Required by controller-gen.
func (in *SystemConfig) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SystemInfo within kubernetes types, where deepcopy-gen is used.
func (in *SystemInfo) DeepCopyInto(out *SystemInfo) {
	p := proto.Clone(in).(*SystemInfo)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SystemInfo. Required by controller-gen.
func (in *SystemInfo) DeepCopy() *SystemInfo {
	if in == nil {
		return nil
	}
	out := new(SystemInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SystemInfo. Required by controller-gen.
func (in *SystemInfo) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using ProcessLaunchRequest within kubernetes types, where deepcopy-gen is used.
func (in *ProcessLaunchRequest) DeepCopyInto(out *ProcessLaunchRequest) {
	p := proto.Clone(in).(*ProcessLaunchRequest)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProcessLaunchRequest. Required by controller-gen.
func (in *ProcessLaunchRequest) DeepCopy() *ProcessLaunchRequest {
	if in == nil {
		return nil
	}
	out := new(ProcessLaunchRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new ProcessLaunchRequest. Required by controller-gen.
func (in *ProcessLaunchRequest) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using ProcessLaunchResponse within kubernetes types, where deepcopy-gen is used.
func (in *ProcessLaunchResponse) DeepCopyInto(out *ProcessLaunchResponse) {
	p := proto.Clone(in).(*ProcessLaunchResponse)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProcessLaunchResponse. Required by controller-gen.
func (in *ProcessLaunchResponse) DeepCopy() *ProcessLaunchResponse {
	if in == nil {
		return nil
	}
	out := new(ProcessLaunchResponse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new ProcessLaunchResponse. Required by controller-gen.
func (in *ProcessLaunchResponse) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using ProcessTerminateRequest within kubernetes types, where deepcopy-gen is used.
func (in *ProcessTerminateRequest) DeepCopyInto(out *ProcessTerminateRequest) {
	p := proto.Clone(in).(*ProcessTerminateRequest)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProcessTerminateRequest. Required by controller-gen.
func (in *ProcessTerminateRequest) DeepCopy() *ProcessTerminateRequest {
	if in == nil {
		return nil
	}
	out := new(ProcessTerminateRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new ProcessTerminateRequest. Required by controller-gen.
func (in *ProcessTerminateRequest) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using ProcessTerminateResponse within kubernetes types, where deepcopy-gen is used.
func (in *ProcessTerminateResponse) DeepCopyInto(out *ProcessTerminateResponse) {
	p := proto.Clone(in).(*ProcessTerminateResponse)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProcessTerminateResponse. Required by controller-gen.
func (in *ProcessTerminateResponse) DeepCopy() *ProcessTerminateResponse {
	if in == nil {
		return nil
	}
	out := new(ProcessTerminateResponse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new ProcessTerminateResponse. Required by controller-gen.
func (in *ProcessTerminateResponse) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using StopRequest within kubernetes types, where deepcopy-gen is used.
func (in *StopRequest) DeepCopyInto(out *StopRequest) {
	p := proto.Clone(in).(*StopRequest)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StopRequest. Required by controller-gen.
func (in *StopRequest) DeepCopy() *StopRequest {
	if in == nil {
		return nil
	}
	out := new(StopRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new StopRequest. Required by controller-gen.
func (in *StopRequest) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using ResetSlotRequest within kubernetes types, where deepcopy-gen is used.
func (in *ResetSlotRequest) DeepCopyInto(out *ResetSlotRequest) {
	p := proto.Clone(in).(*ResetSlotRequest)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResetSlotRequest. Required by controller-gen.
func (in *ResetSlotRequest) DeepCopy() *ResetSlotRequest {
	if in == nil {
		return nil
	}
	out := new(ResetSlotRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new ResetSlotRequest. Required by controller-gen.
func (in *ResetSlotRequest) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using KillSlotRequest within kubernetes types, where deepcopy-gen is used.
func (in *KillSlotRequest) DeepCopyInto(out *KillSlotRequest) {
	p := proto.Clone(in).(*KillSlotRequest)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new KillSlotRequest. Required by controller-gen.
func (in *KillSlotRequest) DeepCopy() *KillSlotRequest {
	if in == nil {
		return nil
	}
	out := new(KillSlotRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new KillSlotRequest. Required by controller-gen.
func (in *KillSlotRequest) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using GetSlotDetailsRequest within kubernetes types, where deepcopy-gen is used.
func (in *GetSlotDetailsRequest) DeepCopyInto(out *GetSlotDetailsRequest) {
	p := proto.Clone(in).(*GetSlotDetailsRequest)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GetSlotDetailsRequest. Required by controller-gen.
func (in *GetSlotDetailsRequest) DeepCopy() *GetSlotDetailsRequest {
	if in == nil {
		return nil
	}
	out := new(GetSlotDetailsRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new GetSlotDetailsRequest. Required by controller-gen.
func (in *GetSlotDetailsRequest) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using GetSlotDetailsResponse within kubernetes types, where deepcopy-gen is used.
func (in *GetSlotDetailsResponse) DeepCopyInto(out *GetSlotDetailsResponse) {
	p := proto.Clone(in).(*GetSlotDetailsResponse)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GetSlotDetailsResponse. Required by controller-gen.
func (in *GetSlotDetailsResponse) DeepCopy() *GetSlotDetailsResponse {
	if in == nil {
		return nil
	}
	out := new(GetSlotDetailsResponse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new GetSlotDetailsResponse. Required by controller-gen.
func (in *GetSlotDetailsResponse) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using GetSlotDetailRequest within kubernetes types, where deepcopy-gen is used.
func (in *GetSlotDetailRequest) DeepCopyInto(out *GetSlotDetailRequest) {
	p := proto.Clone(in).(*GetSlotDetailRequest)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GetSlotDetailRequest. Required by controller-gen.
func (in *GetSlotDetailRequest) DeepCopy() *GetSlotDetailRequest {
	if in == nil {
		return nil
	}
	out := new(GetSlotDetailRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new GetSlotDetailRequest. Required by controller-gen.
func (in *GetSlotDetailRequest) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using GetSlotDetailResponse within kubernetes types, where deepcopy-gen is used.
func (in *GetSlotDetailResponse) DeepCopyInto(out *GetSlotDetailResponse) {
	p := proto.Clone(in).(*GetSlotDetailResponse)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GetSlotDetailResponse. Required by controller-gen.
func (in *GetSlotDetailResponse) DeepCopy() *GetSlotDetailResponse {
	if in == nil {
		return nil
	}
	out := new(GetSlotDetailResponse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new GetSlotDetailResponse. Required by controller-gen.
func (in *GetSlotDetailResponse) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using RebindAllSlotCpusetRequest within kubernetes types, where deepcopy-gen is used.
func (in *RebindAllSlotCpusetRequest) DeepCopyInto(out *RebindAllSlotCpusetRequest) {
	p := proto.Clone(in).(*RebindAllSlotCpusetRequest)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RebindAllSlotCpusetRequest. Required by controller-gen.
func (in *RebindAllSlotCpusetRequest) DeepCopy() *RebindAllSlotCpusetRequest {
	if in == nil {
		return nil
	}
	out := new(RebindAllSlotCpusetRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new RebindAllSlotCpusetRequest. Required by controller-gen.
func (in *RebindAllSlotCpusetRequest) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using RebindAllSlotCpusetResponse within kubernetes types, where deepcopy-gen is used.
func (in *RebindAllSlotCpusetResponse) DeepCopyInto(out *RebindAllSlotCpusetResponse) {
	p := proto.Clone(in).(*RebindAllSlotCpusetResponse)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RebindAllSlotCpusetResponse. Required by controller-gen.
func (in *RebindAllSlotCpusetResponse) DeepCopy() *RebindAllSlotCpusetResponse {
	if in == nil {
		return nil
	}
	out := new(RebindAllSlotCpusetResponse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new RebindAllSlotCpusetResponse. Required by controller-gen.
func (in *RebindAllSlotCpusetResponse) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlaveNodeLoadInfoRequest within kubernetes types, where deepcopy-gen is used.
func (in *SlaveNodeLoadInfoRequest) DeepCopyInto(out *SlaveNodeLoadInfoRequest) {
	p := proto.Clone(in).(*SlaveNodeLoadInfoRequest)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlaveNodeLoadInfoRequest. Required by controller-gen.
func (in *SlaveNodeLoadInfoRequest) DeepCopy() *SlaveNodeLoadInfoRequest {
	if in == nil {
		return nil
	}
	out := new(SlaveNodeLoadInfoRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlaveNodeLoadInfoRequest. Required by controller-gen.
func (in *SlaveNodeLoadInfoRequest) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlaveNodeLoadInfoResponse within kubernetes types, where deepcopy-gen is used.
func (in *SlaveNodeLoadInfoResponse) DeepCopyInto(out *SlaveNodeLoadInfoResponse) {
	p := proto.Clone(in).(*SlaveNodeLoadInfoResponse)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlaveNodeLoadInfoResponse. Required by controller-gen.
func (in *SlaveNodeLoadInfoResponse) DeepCopy() *SlaveNodeLoadInfoResponse {
	if in == nil {
		return nil
	}
	out := new(SlaveNodeLoadInfoResponse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlaveNodeLoadInfoResponse. Required by controller-gen.
func (in *SlaveNodeLoadInfoResponse) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlaveHealthInfoRequest within kubernetes types, where deepcopy-gen is used.
func (in *SlaveHealthInfoRequest) DeepCopyInto(out *SlaveHealthInfoRequest) {
	p := proto.Clone(in).(*SlaveHealthInfoRequest)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlaveHealthInfoRequest. Required by controller-gen.
func (in *SlaveHealthInfoRequest) DeepCopy() *SlaveHealthInfoRequest {
	if in == nil {
		return nil
	}
	out := new(SlaveHealthInfoRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlaveHealthInfoRequest. Required by controller-gen.
func (in *SlaveHealthInfoRequest) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlaveHealthInfoResponse within kubernetes types, where deepcopy-gen is used.
func (in *SlaveHealthInfoResponse) DeepCopyInto(out *SlaveHealthInfoResponse) {
	p := proto.Clone(in).(*SlaveHealthInfoResponse)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlaveHealthInfoResponse. Required by controller-gen.
func (in *SlaveHealthInfoResponse) DeepCopy() *SlaveHealthInfoResponse {
	if in == nil {
		return nil
	}
	out := new(SlaveHealthInfoResponse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlaveHealthInfoResponse. Required by controller-gen.
func (in *SlaveHealthInfoResponse) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using HeartbeatRequest within kubernetes types, where deepcopy-gen is used.
func (in *HeartbeatRequest) DeepCopyInto(out *HeartbeatRequest) {
	p := proto.Clone(in).(*HeartbeatRequest)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HeartbeatRequest. Required by controller-gen.
func (in *HeartbeatRequest) DeepCopy() *HeartbeatRequest {
	if in == nil {
		return nil
	}
	out := new(HeartbeatRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new HeartbeatRequest. Required by controller-gen.
func (in *HeartbeatRequest) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using HeartbeatResponse within kubernetes types, where deepcopy-gen is used.
func (in *HeartbeatResponse) DeepCopyInto(out *HeartbeatResponse) {
	p := proto.Clone(in).(*HeartbeatResponse)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HeartbeatResponse. Required by controller-gen.
func (in *HeartbeatResponse) DeepCopy() *HeartbeatResponse {
	if in == nil {
		return nil
	}
	out := new(HeartbeatResponse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new HeartbeatResponse. Required by controller-gen.
func (in *HeartbeatResponse) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using UpdatePackageVersionRequest within kubernetes types, where deepcopy-gen is used.
func (in *UpdatePackageVersionRequest) DeepCopyInto(out *UpdatePackageVersionRequest) {
	p := proto.Clone(in).(*UpdatePackageVersionRequest)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UpdatePackageVersionRequest. Required by controller-gen.
func (in *UpdatePackageVersionRequest) DeepCopy() *UpdatePackageVersionRequest {
	if in == nil {
		return nil
	}
	out := new(UpdatePackageVersionRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new UpdatePackageVersionRequest. Required by controller-gen.
func (in *UpdatePackageVersionRequest) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using UpdatePackageVersionResponse within kubernetes types, where deepcopy-gen is used.
func (in *UpdatePackageVersionResponse) DeepCopyInto(out *UpdatePackageVersionResponse) {
	p := proto.Clone(in).(*UpdatePackageVersionResponse)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new UpdatePackageVersionResponse. Required by controller-gen.
func (in *UpdatePackageVersionResponse) DeepCopy() *UpdatePackageVersionResponse {
	if in == nil {
		return nil
	}
	out := new(UpdatePackageVersionResponse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new UpdatePackageVersionResponse. Required by controller-gen.
func (in *UpdatePackageVersionResponse) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using CpuAllocatorDebugInfo within kubernetes types, where deepcopy-gen is used.
func (in *CpuAllocatorDebugInfo) DeepCopyInto(out *CpuAllocatorDebugInfo) {
	p := proto.Clone(in).(*CpuAllocatorDebugInfo)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CpuAllocatorDebugInfo. Required by controller-gen.
func (in *CpuAllocatorDebugInfo) DeepCopy() *CpuAllocatorDebugInfo {
	if in == nil {
		return nil
	}
	out := new(CpuAllocatorDebugInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new CpuAllocatorDebugInfo. Required by controller-gen.
func (in *CpuAllocatorDebugInfo) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using CpuAllocatorDetail within kubernetes types, where deepcopy-gen is used.
func (in *CpuAllocatorDetail) DeepCopyInto(out *CpuAllocatorDetail) {
	p := proto.Clone(in).(*CpuAllocatorDetail)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CpuAllocatorDetail. Required by controller-gen.
func (in *CpuAllocatorDetail) DeepCopy() *CpuAllocatorDetail {
	if in == nil {
		return nil
	}
	out := new(CpuAllocatorDetail)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new CpuAllocatorDetail. Required by controller-gen.
func (in *CpuAllocatorDetail) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using CpuAllocatorDebugInfoRequest within kubernetes types, where deepcopy-gen is used.
func (in *CpuAllocatorDebugInfoRequest) DeepCopyInto(out *CpuAllocatorDebugInfoRequest) {
	p := proto.Clone(in).(*CpuAllocatorDebugInfoRequest)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CpuAllocatorDebugInfoRequest. Required by controller-gen.
func (in *CpuAllocatorDebugInfoRequest) DeepCopy() *CpuAllocatorDebugInfoRequest {
	if in == nil {
		return nil
	}
	out := new(CpuAllocatorDebugInfoRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new CpuAllocatorDebugInfoRequest. Required by controller-gen.
func (in *CpuAllocatorDebugInfoRequest) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using CpuAllocatorDebugInfoResponse within kubernetes types, where deepcopy-gen is used.
func (in *CpuAllocatorDebugInfoResponse) DeepCopyInto(out *CpuAllocatorDebugInfoResponse) {
	p := proto.Clone(in).(*CpuAllocatorDebugInfoResponse)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CpuAllocatorDebugInfoResponse. Required by controller-gen.
func (in *CpuAllocatorDebugInfoResponse) DeepCopy() *CpuAllocatorDebugInfoResponse {
	if in == nil {
		return nil
	}
	out := new(CpuAllocatorDebugInfoResponse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new CpuAllocatorDebugInfoResponse. Required by controller-gen.
func (in *CpuAllocatorDebugInfoResponse) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlotResourceTunning within kubernetes types, where deepcopy-gen is used.
func (in *SlotResourceTunning) DeepCopyInto(out *SlotResourceTunning) {
	p := proto.Clone(in).(*SlotResourceTunning)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlotResourceTunning. Required by controller-gen.
func (in *SlotResourceTunning) DeepCopy() *SlotResourceTunning {
	if in == nil {
		return nil
	}
	out := new(SlotResourceTunning)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlotResourceTunning. Required by controller-gen.
func (in *SlotResourceTunning) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlotResourceTunningRequest within kubernetes types, where deepcopy-gen is used.
func (in *SlotResourceTunningRequest) DeepCopyInto(out *SlotResourceTunningRequest) {
	p := proto.Clone(in).(*SlotResourceTunningRequest)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlotResourceTunningRequest. Required by controller-gen.
func (in *SlotResourceTunningRequest) DeepCopy() *SlotResourceTunningRequest {
	if in == nil {
		return nil
	}
	out := new(SlotResourceTunningRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlotResourceTunningRequest. Required by controller-gen.
func (in *SlotResourceTunningRequest) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlotResourceTunningResponse within kubernetes types, where deepcopy-gen is used.
func (in *SlotResourceTunningResponse) DeepCopyInto(out *SlotResourceTunningResponse) {
	p := proto.Clone(in).(*SlotResourceTunningResponse)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlotResourceTunningResponse. Required by controller-gen.
func (in *SlotResourceTunningResponse) DeepCopy() *SlotResourceTunningResponse {
	if in == nil {
		return nil
	}
	out := new(SlotResourceTunningResponse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlotResourceTunningResponse. Required by controller-gen.
func (in *SlotResourceTunningResponse) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using OutlierEvent within kubernetes types, where deepcopy-gen is used.
func (in *OutlierEvent) DeepCopyInto(out *OutlierEvent) {
	p := proto.Clone(in).(*OutlierEvent)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OutlierEvent. Required by controller-gen.
func (in *OutlierEvent) DeepCopy() *OutlierEvent {
	if in == nil {
		return nil
	}
	out := new(OutlierEvent)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new OutlierEvent. Required by controller-gen.
func (in *OutlierEvent) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using OutlierEventRequest within kubernetes types, where deepcopy-gen is used.
func (in *OutlierEventRequest) DeepCopyInto(out *OutlierEventRequest) {
	p := proto.Clone(in).(*OutlierEventRequest)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OutlierEventRequest. Required by controller-gen.
func (in *OutlierEventRequest) DeepCopy() *OutlierEventRequest {
	if in == nil {
		return nil
	}
	out := new(OutlierEventRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new OutlierEventRequest. Required by controller-gen.
func (in *OutlierEventRequest) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using OutlierEventResponse within kubernetes types, where deepcopy-gen is used.
func (in *OutlierEventResponse) DeepCopyInto(out *OutlierEventResponse) {
	p := proto.Clone(in).(*OutlierEventResponse)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new OutlierEventResponse. Required by controller-gen.
func (in *OutlierEventResponse) DeepCopy() *OutlierEventResponse {
	if in == nil {
		return nil
	}
	out := new(OutlierEventResponse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new OutlierEventResponse. Required by controller-gen.
func (in *OutlierEventResponse) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}