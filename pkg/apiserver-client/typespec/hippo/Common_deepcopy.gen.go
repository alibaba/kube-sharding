// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: Common.proto

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

// DeepCopyInto supports using ErrorInfo within kubernetes types, where deepcopy-gen is used.
func (in *ErrorInfo) DeepCopyInto(out *ErrorInfo) {
	p := proto.Clone(in).(*ErrorInfo)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ErrorInfo. Required by controller-gen.
func (in *ErrorInfo) DeepCopy() *ErrorInfo {
	if in == nil {
		return nil
	}
	out := new(ErrorInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new ErrorInfo. Required by controller-gen.
func (in *ErrorInfo) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlotId within kubernetes types, where deepcopy-gen is used.
func (in *SlotId) DeepCopyInto(out *SlotId) {
	p := proto.Clone(in).(*SlotId)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlotId. Required by controller-gen.
func (in *SlotId) DeepCopy() *SlotId {
	if in == nil {
		return nil
	}
	out := new(SlotId)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlotId. Required by controller-gen.
func (in *SlotId) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using GenericResponse within kubernetes types, where deepcopy-gen is used.
func (in *GenericResponse) DeepCopyInto(out *GenericResponse) {
	p := proto.Clone(in).(*GenericResponse)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GenericResponse. Required by controller-gen.
func (in *GenericResponse) DeepCopy() *GenericResponse {
	if in == nil {
		return nil
	}
	out := new(GenericResponse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new GenericResponse. Required by controller-gen.
func (in *GenericResponse) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using PreferenceDescription within kubernetes types, where deepcopy-gen is used.
func (in *PreferenceDescription) DeepCopyInto(out *PreferenceDescription) {
	p := proto.Clone(in).(*PreferenceDescription)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PreferenceDescription. Required by controller-gen.
func (in *PreferenceDescription) DeepCopy() *PreferenceDescription {
	if in == nil {
		return nil
	}
	out := new(PreferenceDescription)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new PreferenceDescription. Required by controller-gen.
func (in *PreferenceDescription) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using Resource within kubernetes types, where deepcopy-gen is used.
func (in *Resource) DeepCopyInto(out *Resource) {
	p := proto.Clone(in).(*Resource)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Resource. Required by controller-gen.
func (in *Resource) DeepCopy() *Resource {
	if in == nil {
		return nil
	}
	out := new(Resource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new Resource. Required by controller-gen.
func (in *Resource) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using Priority within kubernetes types, where deepcopy-gen is used.
func (in *Priority) DeepCopyInto(out *Priority) {
	p := proto.Clone(in).(*Priority)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Priority. Required by controller-gen.
func (in *Priority) DeepCopy() *Priority {
	if in == nil {
		return nil
	}
	out := new(Priority)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new Priority. Required by controller-gen.
func (in *Priority) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlotResource within kubernetes types, where deepcopy-gen is used.
func (in *SlotResource) DeepCopyInto(out *SlotResource) {
	p := proto.Clone(in).(*SlotResource)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlotResource. Required by controller-gen.
func (in *SlotResource) DeepCopy() *SlotResource {
	if in == nil {
		return nil
	}
	out := new(SlotResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlotResource. Required by controller-gen.
func (in *SlotResource) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlaveResource within kubernetes types, where deepcopy-gen is used.
func (in *SlaveResource) DeepCopyInto(out *SlaveResource) {
	p := proto.Clone(in).(*SlaveResource)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlaveResource. Required by controller-gen.
func (in *SlaveResource) DeepCopy() *SlaveResource {
	if in == nil {
		return nil
	}
	out := new(SlaveResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlaveResource. Required by controller-gen.
func (in *SlaveResource) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using GrantedResource within kubernetes types, where deepcopy-gen is used.
func (in *GrantedResource) DeepCopyInto(out *GrantedResource) {
	p := proto.Clone(in).(*GrantedResource)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GrantedResource. Required by controller-gen.
func (in *GrantedResource) DeepCopy() *GrantedResource {
	if in == nil {
		return nil
	}
	out := new(GrantedResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new GrantedResource. Required by controller-gen.
func (in *GrantedResource) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlaveRuntimeConfig within kubernetes types, where deepcopy-gen is used.
func (in *SlaveRuntimeConfig) DeepCopyInto(out *SlaveRuntimeConfig) {
	p := proto.Clone(in).(*SlaveRuntimeConfig)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlaveRuntimeConfig. Required by controller-gen.
func (in *SlaveRuntimeConfig) DeepCopy() *SlaveRuntimeConfig {
	if in == nil {
		return nil
	}
	out := new(SlaveRuntimeConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlaveRuntimeConfig. Required by controller-gen.
func (in *SlaveRuntimeConfig) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlaveDescription within kubernetes types, where deepcopy-gen is used.
func (in *SlaveDescription) DeepCopyInto(out *SlaveDescription) {
	p := proto.Clone(in).(*SlaveDescription)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlaveDescription. Required by controller-gen.
func (in *SlaveDescription) DeepCopy() *SlaveDescription {
	if in == nil {
		return nil
	}
	out := new(SlaveDescription)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlaveDescription. Required by controller-gen.
func (in *SlaveDescription) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlaveSchedulability within kubernetes types, where deepcopy-gen is used.
func (in *SlaveSchedulability) DeepCopyInto(out *SlaveSchedulability) {
	p := proto.Clone(in).(*SlaveSchedulability)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlaveSchedulability. Required by controller-gen.
func (in *SlaveSchedulability) DeepCopy() *SlaveSchedulability {
	if in == nil {
		return nil
	}
	out := new(SlaveSchedulability)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlaveSchedulability. Required by controller-gen.
func (in *SlaveSchedulability) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlotPreference within kubernetes types, where deepcopy-gen is used.
func (in *SlotPreference) DeepCopyInto(out *SlotPreference) {
	p := proto.Clone(in).(*SlotPreference)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlotPreference. Required by controller-gen.
func (in *SlotPreference) DeepCopy() *SlotPreference {
	if in == nil {
		return nil
	}
	out := new(SlotPreference)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlotPreference. Required by controller-gen.
func (in *SlotPreference) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlotPreferenceConfig within kubernetes types, where deepcopy-gen is used.
func (in *SlotPreferenceConfig) DeepCopyInto(out *SlotPreferenceConfig) {
	p := proto.Clone(in).(*SlotPreferenceConfig)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlotPreferenceConfig. Required by controller-gen.
func (in *SlotPreferenceConfig) DeepCopy() *SlotPreferenceConfig {
	if in == nil {
		return nil
	}
	out := new(SlotPreferenceConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlotPreferenceConfig. Required by controller-gen.
func (in *SlotPreferenceConfig) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using Visibility within kubernetes types, where deepcopy-gen is used.
func (in *Visibility) DeepCopyInto(out *Visibility) {
	p := proto.Clone(in).(*Visibility)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Visibility. Required by controller-gen.
func (in *Visibility) DeepCopy() *Visibility {
	if in == nil {
		return nil
	}
	out := new(Visibility)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new Visibility. Required by controller-gen.
func (in *Visibility) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using ApplicationSlotConfig within kubernetes types, where deepcopy-gen is used.
func (in *ApplicationSlotConfig) DeepCopyInto(out *ApplicationSlotConfig) {
	p := proto.Clone(in).(*ApplicationSlotConfig)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationSlotConfig. Required by controller-gen.
func (in *ApplicationSlotConfig) DeepCopy() *ApplicationSlotConfig {
	if in == nil {
		return nil
	}
	out := new(ApplicationSlotConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationSlotConfig. Required by controller-gen.
func (in *ApplicationSlotConfig) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using ResourceRequest within kubernetes types, where deepcopy-gen is used.
func (in *ResourceRequest) DeepCopyInto(out *ResourceRequest) {
	p := proto.Clone(in).(*ResourceRequest)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceRequest. Required by controller-gen.
func (in *ResourceRequest) DeepCopy() *ResourceRequest {
	if in == nil {
		return nil
	}
	out := new(ResourceRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new ResourceRequest. Required by controller-gen.
func (in *ResourceRequest) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using ApplicationDescription within kubernetes types, where deepcopy-gen is used.
func (in *ApplicationDescription) DeepCopyInto(out *ApplicationDescription) {
	p := proto.Clone(in).(*ApplicationDescription)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationDescription. Required by controller-gen.
func (in *ApplicationDescription) DeepCopy() *ApplicationDescription {
	if in == nil {
		return nil
	}
	out := new(ApplicationDescription)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new ApplicationDescription. Required by controller-gen.
func (in *ApplicationDescription) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using QueueResource within kubernetes types, where deepcopy-gen is used.
func (in *QueueResource) DeepCopyInto(out *QueueResource) {
	p := proto.Clone(in).(*QueueResource)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new QueueResource. Required by controller-gen.
func (in *QueueResource) DeepCopy() *QueueResource {
	if in == nil {
		return nil
	}
	out := new(QueueResource)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new QueueResource. Required by controller-gen.
func (in *QueueResource) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using QueueDescription within kubernetes types, where deepcopy-gen is used.
func (in *QueueDescription) DeepCopyInto(out *QueueDescription) {
	p := proto.Clone(in).(*QueueDescription)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new QueueDescription. Required by controller-gen.
func (in *QueueDescription) DeepCopy() *QueueDescription {
	if in == nil {
		return nil
	}
	out := new(QueueDescription)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new QueueDescription. Required by controller-gen.
func (in *QueueDescription) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using PodTraceInfo within kubernetes types, where deepcopy-gen is used.
func (in *PodTraceInfo) DeepCopyInto(out *PodTraceInfo) {
	p := proto.Clone(in).(*PodTraceInfo)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodTraceInfo. Required by controller-gen.
func (in *PodTraceInfo) DeepCopy() *PodTraceInfo {
	if in == nil {
		return nil
	}
	out := new(PodTraceInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new PodTraceInfo. Required by controller-gen.
func (in *PodTraceInfo) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using ResourceResponse within kubernetes types, where deepcopy-gen is used.
func (in *ResourceResponse) DeepCopyInto(out *ResourceResponse) {
	p := proto.Clone(in).(*ResourceResponse)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourceResponse. Required by controller-gen.
func (in *ResourceResponse) DeepCopy() *ResourceResponse {
	if in == nil {
		return nil
	}
	out := new(ResourceResponse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new ResourceResponse. Required by controller-gen.
func (in *ResourceResponse) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlaveStatus within kubernetes types, where deepcopy-gen is used.
func (in *SlaveStatus) DeepCopyInto(out *SlaveStatus) {
	p := proto.Clone(in).(*SlaveStatus)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlaveStatus. Required by controller-gen.
func (in *SlaveStatus) DeepCopy() *SlaveStatus {
	if in == nil {
		return nil
	}
	out := new(SlaveStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlaveStatus. Required by controller-gen.
func (in *SlaveStatus) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlaveHealth within kubernetes types, where deepcopy-gen is used.
func (in *SlaveHealth) DeepCopyInto(out *SlaveHealth) {
	p := proto.Clone(in).(*SlaveHealth)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlaveHealth. Required by controller-gen.
func (in *SlaveHealth) DeepCopy() *SlaveHealth {
	if in == nil {
		return nil
	}
	out := new(SlaveHealth)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlaveHealth. Required by controller-gen.
func (in *SlaveHealth) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SystemStatus within kubernetes types, where deepcopy-gen is used.
func (in *SystemStatus) DeepCopyInto(out *SystemStatus) {
	p := proto.Clone(in).(*SystemStatus)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SystemStatus. Required by controller-gen.
func (in *SystemStatus) DeepCopy() *SystemStatus {
	if in == nil {
		return nil
	}
	out := new(SystemStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SystemStatus. Required by controller-gen.
func (in *SystemStatus) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using ProcessStatus within kubernetes types, where deepcopy-gen is used.
func (in *ProcessStatus) DeepCopyInto(out *ProcessStatus) {
	p := proto.Clone(in).(*ProcessStatus)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProcessStatus. Required by controller-gen.
func (in *ProcessStatus) DeepCopy() *ProcessStatus {
	if in == nil {
		return nil
	}
	out := new(ProcessStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new ProcessStatus. Required by controller-gen.
func (in *ProcessStatus) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using PackageDetail within kubernetes types, where deepcopy-gen is used.
func (in *PackageDetail) DeepCopyInto(out *PackageDetail) {
	p := proto.Clone(in).(*PackageDetail)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PackageDetail. Required by controller-gen.
func (in *PackageDetail) DeepCopy() *PackageDetail {
	if in == nil {
		return nil
	}
	out := new(PackageDetail)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new PackageDetail. Required by controller-gen.
func (in *PackageDetail) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlotPayload within kubernetes types, where deepcopy-gen is used.
func (in *SlotPayload) DeepCopyInto(out *SlotPayload) {
	p := proto.Clone(in).(*SlotPayload)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlotPayload. Required by controller-gen.
func (in *SlotPayload) DeepCopy() *SlotPayload {
	if in == nil {
		return nil
	}
	out := new(SlotPayload)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlotPayload. Required by controller-gen.
func (in *SlotPayload) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using AssignedSlot within kubernetes types, where deepcopy-gen is used.
func (in *AssignedSlot) DeepCopyInto(out *AssignedSlot) {
	p := proto.Clone(in).(*AssignedSlot)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AssignedSlot. Required by controller-gen.
func (in *AssignedSlot) DeepCopy() *AssignedSlot {
	if in == nil {
		return nil
	}
	out := new(AssignedSlot)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new AssignedSlot. Required by controller-gen.
func (in *AssignedSlot) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using HistoryInfo within kubernetes types, where deepcopy-gen is used.
func (in *HistoryInfo) DeepCopyInto(out *HistoryInfo) {
	p := proto.Clone(in).(*HistoryInfo)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HistoryInfo. Required by controller-gen.
func (in *HistoryInfo) DeepCopy() *HistoryInfo {
	if in == nil {
		return nil
	}
	out := new(HistoryInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new HistoryInfo. Required by controller-gen.
func (in *HistoryInfo) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using HistoryApp within kubernetes types, where deepcopy-gen is used.
func (in *HistoryApp) DeepCopyInto(out *HistoryApp) {
	p := proto.Clone(in).(*HistoryApp)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HistoryApp. Required by controller-gen.
func (in *HistoryApp) DeepCopy() *HistoryApp {
	if in == nil {
		return nil
	}
	out := new(HistoryApp)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new HistoryApp. Required by controller-gen.
func (in *HistoryApp) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using HistoryTag within kubernetes types, where deepcopy-gen is used.
func (in *HistoryTag) DeepCopyInto(out *HistoryTag) {
	p := proto.Clone(in).(*HistoryTag)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HistoryTag. Required by controller-gen.
func (in *HistoryTag) DeepCopy() *HistoryTag {
	if in == nil {
		return nil
	}
	out := new(HistoryTag)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new HistoryTag. Required by controller-gen.
func (in *HistoryTag) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using HistoryPod within kubernetes types, where deepcopy-gen is used.
func (in *HistoryPod) DeepCopyInto(out *HistoryPod) {
	p := proto.Clone(in).(*HistoryPod)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HistoryPod. Required by controller-gen.
func (in *HistoryPod) DeepCopy() *HistoryPod {
	if in == nil {
		return nil
	}
	out := new(HistoryPod)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new HistoryPod. Required by controller-gen.
func (in *HistoryPod) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using ProcessLaunchContext within kubernetes types, where deepcopy-gen is used.
func (in *ProcessLaunchContext) DeepCopyInto(out *ProcessLaunchContext) {
	p := proto.Clone(in).(*ProcessLaunchContext)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProcessLaunchContext. Required by controller-gen.
func (in *ProcessLaunchContext) DeepCopy() *ProcessLaunchContext {
	if in == nil {
		return nil
	}
	out := new(ProcessLaunchContext)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new ProcessLaunchContext. Required by controller-gen.
func (in *ProcessLaunchContext) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using ProcessLaunchExtraInfo within kubernetes types, where deepcopy-gen is used.
func (in *ProcessLaunchExtraInfo) DeepCopyInto(out *ProcessLaunchExtraInfo) {
	p := proto.Clone(in).(*ProcessLaunchExtraInfo)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProcessLaunchExtraInfo. Required by controller-gen.
func (in *ProcessLaunchExtraInfo) DeepCopy() *ProcessLaunchExtraInfo {
	if in == nil {
		return nil
	}
	out := new(ProcessLaunchExtraInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new ProcessLaunchExtraInfo. Required by controller-gen.
func (in *ProcessLaunchExtraInfo) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using PackageInfo within kubernetes types, where deepcopy-gen is used.
func (in *PackageInfo) DeepCopyInto(out *PackageInfo) {
	p := proto.Clone(in).(*PackageInfo)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PackageInfo. Required by controller-gen.
func (in *PackageInfo) DeepCopy() *PackageInfo {
	if in == nil {
		return nil
	}
	out := new(PackageInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new PackageInfo. Required by controller-gen.
func (in *PackageInfo) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using TrackData within kubernetes types, where deepcopy-gen is used.
func (in *TrackData) DeepCopyInto(out *TrackData) {
	p := proto.Clone(in).(*TrackData)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new TrackData. Required by controller-gen.
func (in *TrackData) DeepCopy() *TrackData {
	if in == nil {
		return nil
	}
	out := new(TrackData)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new TrackData. Required by controller-gen.
func (in *TrackData) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using ProcessInfo within kubernetes types, where deepcopy-gen is used.
func (in *ProcessInfo) DeepCopyInto(out *ProcessInfo) {
	p := proto.Clone(in).(*ProcessInfo)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProcessInfo. Required by controller-gen.
func (in *ProcessInfo) DeepCopy() *ProcessInfo {
	if in == nil {
		return nil
	}
	out := new(ProcessInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new ProcessInfo. Required by controller-gen.
func (in *ProcessInfo) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using Parameter within kubernetes types, where deepcopy-gen is used.
func (in *Parameter) DeepCopyInto(out *Parameter) {
	p := proto.Clone(in).(*Parameter)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Parameter. Required by controller-gen.
func (in *Parameter) DeepCopy() *Parameter {
	if in == nil {
		return nil
	}
	out := new(Parameter)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new Parameter. Required by controller-gen.
func (in *Parameter) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using DataInfo within kubernetes types, where deepcopy-gen is used.
func (in *DataInfo) DeepCopyInto(out *DataInfo) {
	p := proto.Clone(in).(*DataInfo)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataInfo. Required by controller-gen.
func (in *DataInfo) DeepCopy() *DataInfo {
	if in == nil {
		return nil
	}
	out := new(DataInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new DataInfo. Required by controller-gen.
func (in *DataInfo) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using DataStatus within kubernetes types, where deepcopy-gen is used.
func (in *DataStatus) DeepCopyInto(out *DataStatus) {
	p := proto.Clone(in).(*DataStatus)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataStatus. Required by controller-gen.
func (in *DataStatus) DeepCopy() *DataStatus {
	if in == nil {
		return nil
	}
	out := new(DataStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new DataStatus. Required by controller-gen.
func (in *DataStatus) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using PackageStatus within kubernetes types, where deepcopy-gen is used.
func (in *PackageStatus) DeepCopyInto(out *PackageStatus) {
	p := proto.Clone(in).(*PackageStatus)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PackageStatus. Required by controller-gen.
func (in *PackageStatus) DeepCopy() *PackageStatus {
	if in == nil {
		return nil
	}
	out := new(PackageStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new PackageStatus. Required by controller-gen.
func (in *PackageStatus) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using LoadInfo within kubernetes types, where deepcopy-gen is used.
func (in *LoadInfo) DeepCopyInto(out *LoadInfo) {
	p := proto.Clone(in).(*LoadInfo)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LoadInfo. Required by controller-gen.
func (in *LoadInfo) DeepCopy() *LoadInfo {
	if in == nil {
		return nil
	}
	out := new(LoadInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new LoadInfo. Required by controller-gen.
func (in *LoadInfo) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using ContainerLoadInfo within kubernetes types, where deepcopy-gen is used.
func (in *ContainerLoadInfo) DeepCopyInto(out *ContainerLoadInfo) {
	p := proto.Clone(in).(*ContainerLoadInfo)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ContainerLoadInfo. Required by controller-gen.
func (in *ContainerLoadInfo) DeepCopy() *ContainerLoadInfo {
	if in == nil {
		return nil
	}
	out := new(ContainerLoadInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new ContainerLoadInfo. Required by controller-gen.
func (in *ContainerLoadInfo) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlotLoadInfo within kubernetes types, where deepcopy-gen is used.
func (in *SlotLoadInfo) DeepCopyInto(out *SlotLoadInfo) {
	p := proto.Clone(in).(*SlotLoadInfo)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlotLoadInfo. Required by controller-gen.
func (in *SlotLoadInfo) DeepCopy() *SlotLoadInfo {
	if in == nil {
		return nil
	}
	out := new(SlotLoadInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlotLoadInfo. Required by controller-gen.
func (in *SlotLoadInfo) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlaveLoadInfo within kubernetes types, where deepcopy-gen is used.
func (in *SlaveLoadInfo) DeepCopyInto(out *SlaveLoadInfo) {
	p := proto.Clone(in).(*SlaveLoadInfo)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlaveLoadInfo. Required by controller-gen.
func (in *SlaveLoadInfo) DeepCopy() *SlaveLoadInfo {
	if in == nil {
		return nil
	}
	out := new(SlaveLoadInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlaveLoadInfo. Required by controller-gen.
func (in *SlaveLoadInfo) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SlaveNodeLoadInfo within kubernetes types, where deepcopy-gen is used.
func (in *SlaveNodeLoadInfo) DeepCopyInto(out *SlaveNodeLoadInfo) {
	p := proto.Clone(in).(*SlaveNodeLoadInfo)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SlaveNodeLoadInfo. Required by controller-gen.
func (in *SlaveNodeLoadInfo) DeepCopy() *SlaveNodeLoadInfo {
	if in == nil {
		return nil
	}
	out := new(SlaveNodeLoadInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SlaveNodeLoadInfo. Required by controller-gen.
func (in *SlaveNodeLoadInfo) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SetLoggerLevelRequest within kubernetes types, where deepcopy-gen is used.
func (in *SetLoggerLevelRequest) DeepCopyInto(out *SetLoggerLevelRequest) {
	p := proto.Clone(in).(*SetLoggerLevelRequest)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SetLoggerLevelRequest. Required by controller-gen.
func (in *SetLoggerLevelRequest) DeepCopy() *SetLoggerLevelRequest {
	if in == nil {
		return nil
	}
	out := new(SetLoggerLevelRequest)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SetLoggerLevelRequest. Required by controller-gen.
func (in *SetLoggerLevelRequest) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using SetLoggerLevelResponse within kubernetes types, where deepcopy-gen is used.
func (in *SetLoggerLevelResponse) DeepCopyInto(out *SetLoggerLevelResponse) {
	p := proto.Clone(in).(*SetLoggerLevelResponse)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SetLoggerLevelResponse. Required by controller-gen.
func (in *SetLoggerLevelResponse) DeepCopy() *SetLoggerLevelResponse {
	if in == nil {
		return nil
	}
	out := new(SetLoggerLevelResponse)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new SetLoggerLevelResponse. Required by controller-gen.
func (in *SetLoggerLevelResponse) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using ClusterDescription within kubernetes types, where deepcopy-gen is used.
func (in *ClusterDescription) DeepCopyInto(out *ClusterDescription) {
	p := proto.Clone(in).(*ClusterDescription)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterDescription. Required by controller-gen.
func (in *ClusterDescription) DeepCopy() *ClusterDescription {
	if in == nil {
		return nil
	}
	out := new(ClusterDescription)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new ClusterDescription. Required by controller-gen.
func (in *ClusterDescription) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}

// DeepCopyInto supports using DiskInfo within kubernetes types, where deepcopy-gen is used.
func (in *DiskInfo) DeepCopyInto(out *DiskInfo) {
	p := proto.Clone(in).(*DiskInfo)
	*out = *p
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DiskInfo. Required by controller-gen.
func (in *DiskInfo) DeepCopy() *DiskInfo {
	if in == nil {
		return nil
	}
	out := new(DiskInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInterface is an autogenerated deepcopy function, copying the receiver, creating a new DiskInfo. Required by controller-gen.
func (in *DiskInfo) DeepCopyInterface() interface{} {
	return in.DeepCopy()
}
