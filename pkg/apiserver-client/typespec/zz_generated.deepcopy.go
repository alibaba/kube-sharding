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
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/carbon"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/hippo"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BrokenRecoverQuotaConfig) DeepCopyInto(out *BrokenRecoverQuotaConfig) {
	*out = *in
	if in.MaxFailedCount != nil {
		in, out := &in.MaxFailedCount, &out.MaxFailedCount
		*out = new(int32)
		**out = **in
	}
	if in.TimeWindow != nil {
		in, out := &in.TimeWindow, &out.TimeWindow
		*out = new(int32)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BrokenRecoverQuotaConfig.
func (in *BrokenRecoverQuotaConfig) DeepCopy() *BrokenRecoverQuotaConfig {
	if in == nil {
		return nil
	}
	out := new(BrokenRecoverQuotaConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ConstraintConfig) DeepCopyInto(out *ConstraintConfig) {
	*out = *in
	if in.Level != nil {
		in, out := &in.Level, &out.Level
		*out = new(int32)
		**out = **in
	}
	if in.Strictly != nil {
		in, out := &in.Strictly, &out.Strictly
		*out = new(bool)
		**out = **in
	}
	if in.UseHostWorkDir != nil {
		in, out := &in.UseHostWorkDir, &out.UseHostWorkDir
		*out = new(bool)
		**out = **in
	}
	if in.MaxInstancePerHost != nil {
		in, out := &in.MaxInstancePerHost, &out.MaxInstancePerHost
		*out = new(int32)
		**out = **in
	}
	if in.MaxInstancePerFrame != nil {
		in, out := &in.MaxInstancePerFrame, &out.MaxInstancePerFrame
		*out = new(int32)
		**out = **in
	}
	if in.MaxInstancePerRack != nil {
		in, out := &in.MaxInstancePerRack, &out.MaxInstancePerRack
		*out = new(int32)
		**out = **in
	}
	if in.MaxInstancePerASW != nil {
		in, out := &in.MaxInstancePerASW, &out.MaxInstancePerASW
		*out = new(int32)
		**out = **in
	}
	if in.MaxInstancePerPSW != nil {
		in, out := &in.MaxInstancePerPSW, &out.MaxInstancePerPSW
		*out = new(int32)
		**out = **in
	}
	if in.SpecifiedIps != nil {
		in, out := &in.SpecifiedIps, &out.SpecifiedIps
		*out = new([]string)
		if **in != nil {
			in, out := *in, *out
			*out = make([]string, len(*in))
			copy(*out, *in)
		}
	}
	if in.ProhibitedIps != nil {
		in, out := &in.ProhibitedIps, &out.ProhibitedIps
		*out = new([]string)
		if **in != nil {
			in, out := *in, *out
			*out = make([]string, len(*in))
			copy(*out, *in)
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ConstraintConfig.
func (in *ConstraintConfig) DeepCopy() *ConstraintConfig {
	if in == nil {
		return nil
	}
	out := new(ConstraintConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GlobalPlan) DeepCopyInto(out *GlobalPlan) {
	*out = *in
	if in.Count != nil {
		in, out := &in.Count, &out.Count
		*out = new(int32)
		**out = **in
	}
	if in.LatestVersionRatio != nil {
		in, out := &in.LatestVersionRatio, &out.LatestVersionRatio
		*out = new(int32)
		**out = **in
	}
	if in.HealthCheckerConfig != nil {
		in, out := &in.HealthCheckerConfig, &out.HealthCheckerConfig
		*out = new(HealthCheckerConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.ServiceConfigs != nil {
		in, out := &in.ServiceConfigs, &out.ServiceConfigs
		*out = make([]ServiceConfig, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.BrokenRecoverQuotaConfig != nil {
		in, out := &in.BrokenRecoverQuotaConfig, &out.BrokenRecoverQuotaConfig
		*out = new(BrokenRecoverQuotaConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.Properties != nil {
		in, out := &in.Properties, &out.Properties
		*out = new(map[string]string)
		if **in != nil {
			in, out := *in, *out
			*out = make(map[string]string, len(*in))
			for key, val := range *in {
				(*out)[key] = val
			}
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GlobalPlan.
func (in *GlobalPlan) DeepCopy() *GlobalPlan {
	if in == nil {
		return nil
	}
	out := new(GlobalPlan)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GroupPlan) DeepCopyInto(out *GroupPlan) {
	*out = *in
	if in.MinHealthCapacity != nil {
		in, out := &in.MinHealthCapacity, &out.MinHealthCapacity
		*out = new(int32)
		**out = **in
	}
	if in.ExtraRatio != nil {
		in, out := &in.ExtraRatio, &out.ExtraRatio
		*out = new(int32)
		**out = **in
	}
	if in.RolePlans != nil {
		in, out := &in.RolePlans, &out.RolePlans
		*out = make(map[string]*RolePlan, len(*in))
		for key, val := range *in {
			var outVal *RolePlan
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = new(RolePlan)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
	if in.SchedulerConfig != nil {
		in, out := &in.SchedulerConfig, &out.SchedulerConfig
		*out = new(GroupSchedulerConfig)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GroupPlan.
func (in *GroupPlan) DeepCopy() *GroupPlan {
	if in == nil {
		return nil
	}
	out := new(GroupPlan)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *GroupSchedulerConfig) DeepCopyInto(out *GroupSchedulerConfig) {
	*out = *in
	if in.Name != nil {
		in, out := &in.Name, &out.Name
		*out = new(string)
		**out = **in
	}
	if in.ConfigStr != nil {
		in, out := &in.ConfigStr, &out.ConfigStr
		*out = new(string)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new GroupSchedulerConfig.
func (in *GroupSchedulerConfig) DeepCopy() *GroupSchedulerConfig {
	if in == nil {
		return nil
	}
	out := new(GroupSchedulerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HealthCheckerConfig) DeepCopyInto(out *HealthCheckerConfig) {
	*out = *in
	if in.Name != nil {
		in, out := &in.Name, &out.Name
		*out = new(string)
		**out = **in
	}
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HealthCheckerConfig.
func (in *HealthCheckerConfig) DeepCopy() *HealthCheckerConfig {
	if in == nil {
		return nil
	}
	out := new(HealthCheckerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *LaunchPlan) DeepCopyInto(out *LaunchPlan) {
	*out = *in
	if in.PackageInfos != nil {
		in, out := &in.PackageInfos, &out.PackageInfos
		*out = make([]PackageInfo, len(*in))
		copy(*out, *in)
	}
	if in.ProcessInfos != nil {
		in, out := &in.ProcessInfos, &out.ProcessInfos
		*out = make([]ProcessInfo, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.DataInfos != nil {
		in, out := &in.DataInfos, &out.DataInfos
		*out = new([]interface{})
		if **in != nil {
			in, out := *in, *out
			*out = make([]interface{}, len(*in))
			for i := range *in {
				if (*in)[i] != nil {
					(*out)[i] = (*in)[i]
				}
			}
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new LaunchPlan.
func (in *LaunchPlan) DeepCopy() *LaunchPlan {
	if in == nil {
		return nil
	}
	out := new(LaunchPlan)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PackageInfo) DeepCopyInto(out *PackageInfo) {
	*out = *in
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PackageInfo.
func (in *PackageInfo) DeepCopy() *PackageInfo {
	if in == nil {
		return nil
	}
	out := new(PackageInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ProcessInfo) DeepCopyInto(out *ProcessInfo) {
	*out = *in
	if in.IsDaemon != nil {
		in, out := &in.IsDaemon, &out.IsDaemon
		*out = new(bool)
		**out = **in
	}
	if in.Name != nil {
		in, out := &in.Name, &out.Name
		*out = new(string)
		**out = **in
	}
	if in.Cmd != nil {
		in, out := &in.Cmd, &out.Cmd
		*out = new(string)
		**out = **in
	}
	if in.Args != nil {
		in, out := &in.Args, &out.Args
		*out = new([][]string)
		if **in != nil {
			in, out := *in, *out
			*out = make([][]string, len(*in))
			for i := range *in {
				if (*in)[i] != nil {
					in, out := &(*in)[i], &(*out)[i]
					*out = make([]string, len(*in))
					copy(*out, *in)
				}
			}
		}
	}
	if in.Envs != nil {
		in, out := &in.Envs, &out.Envs
		*out = new([][]string)
		if **in != nil {
			in, out := *in, *out
			*out = make([][]string, len(*in))
			for i := range *in {
				if (*in)[i] != nil {
					in, out := &(*in)[i], &(*out)[i]
					*out = make([]string, len(*in))
					copy(*out, *in)
				}
			}
		}
	}
	if in.OtherInfos != nil {
		in, out := &in.OtherInfos, &out.OtherInfos
		*out = new([][]string)
		if **in != nil {
			in, out := *in, *out
			*out = make([][]string, len(*in))
			for i := range *in {
				if (*in)[i] != nil {
					in, out := &(*in)[i], &(*out)[i]
					*out = make([]string, len(*in))
					copy(*out, *in)
				}
			}
		}
	}
	if in.InstanceID != nil {
		in, out := &in.InstanceID, &out.InstanceID
		*out = new(int64)
		**out = **in
	}
	if in.StopTimeout != nil {
		in, out := &in.StopTimeout, &out.StopTimeout
		*out = new(int)
		**out = **in
	}
	if in.RestartInterval != nil {
		in, out := &in.RestartInterval, &out.RestartInterval
		*out = new(int64)
		**out = **in
	}
	if in.RestartCountLimit != nil {
		in, out := &in.RestartCountLimit, &out.RestartCountLimit
		*out = new(int)
		**out = **in
	}
	if in.ProcStopSig != nil {
		in, out := &in.ProcStopSig, &out.ProcStopSig
		*out = new(int32)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ProcessInfo.
func (in *ProcessInfo) DeepCopy() *ProcessInfo {
	if in == nil {
		return nil
	}
	out := new(ProcessInfo)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ResourcePlan) DeepCopyInto(out *ResourcePlan) {
	*out = *in
	if in.Resources != nil {
		in, out := &in.Resources, &out.Resources
		*out = make([]carbon.SlotResource, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	if in.Declarations != nil {
		in, out := &in.Declarations, &out.Declarations
		*out = new([]hippo.Resource)
		if **in != nil {
			in, out := *in, *out
			*out = make([]hippo.Resource, len(*in))
			for i := range *in {
				(*in)[i].DeepCopyInto(&(*out)[i])
			}
		}
	}
	if in.AllocateMode != nil {
		in, out := &in.AllocateMode, &out.AllocateMode
		*out = new(string)
		**out = **in
	}
	if in.Queue != nil {
		in, out := &in.Queue, &out.Queue
		*out = new(string)
		**out = **in
	}
	if in.Priority != nil {
		in, out := &in.Priority, &out.Priority
		*out = new(carbon.CarbonPriority)
		(*in).DeepCopyInto(*out)
	}
	if in.Group != nil {
		in, out := &in.Group, &out.Group
		*out = new(string)
		**out = **in
	}
	if in.Constraints != nil {
		in, out := &in.Constraints, &out.Constraints
		*out = new(ConstraintConfig)
		(*in).DeepCopyInto(*out)
	}
	if in.ContainerConfigs != nil {
		in, out := &in.ContainerConfigs, &out.ContainerConfigs
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	if in.MetaTags != nil {
		in, out := &in.MetaTags, &out.MetaTags
		*out = new(map[string]string)
		if **in != nil {
			in, out := *in, *out
			*out = make(map[string]string, len(*in))
			for key, val := range *in {
				(*out)[key] = val
			}
		}
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ResourcePlan.
func (in *ResourcePlan) DeepCopy() *ResourcePlan {
	if in == nil {
		return nil
	}
	out := new(ResourcePlan)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RolePlan) DeepCopyInto(out *RolePlan) {
	*out = *in
	in.Global.DeepCopyInto(&out.Global)
	if in.RoleSchedulerConfig != nil {
		in, out := &in.RoleSchedulerConfig, &out.RoleSchedulerConfig
		*out = new(RoleSchedulerConfig)
		(*in).DeepCopyInto(*out)
	}
	in.Version.DeepCopyInto(&out.Version)
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RolePlan.
func (in *RolePlan) DeepCopy() *RolePlan {
	if in == nil {
		return nil
	}
	out := new(RolePlan)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RoleSchedulerConfig) DeepCopyInto(out *RoleSchedulerConfig) {
	*out = *in
	if in.MinHealthCapacity != nil {
		in, out := &in.MinHealthCapacity, &out.MinHealthCapacity
		*out = new(int32)
		**out = **in
	}
	if in.ExtraRatio != nil {
		in, out := &in.ExtraRatio, &out.ExtraRatio
		*out = new(int32)
		**out = **in
	}
	if in.SchedulerConfig != nil {
		in, out := &in.SchedulerConfig, &out.SchedulerConfig
		*out = new(GroupSchedulerConfig)
		(*in).DeepCopyInto(*out)
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RoleSchedulerConfig.
func (in *RoleSchedulerConfig) DeepCopy() *RoleSchedulerConfig {
	if in == nil {
		return nil
	}
	out := new(RoleSchedulerConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceConfig) DeepCopyInto(out *ServiceConfig) {
	*out = *in
	if in.Masked != nil {
		in, out := &in.Masked, &out.Masked
		*out = new(bool)
		**out = **in
	}
	if in.MetaStr != nil {
		in, out := &in.MetaStr, &out.MetaStr
		*out = new(string)
		**out = **in
	}
	if in.DeleteDelay != nil {
		in, out := &in.DeleteDelay, &out.DeleteDelay
		*out = new(int)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceConfig.
func (in *ServiceConfig) DeepCopy() *ServiceConfig {
	if in == nil {
		return nil
	}
	out := new(ServiceConfig)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *VersionedPlan) DeepCopyInto(out *VersionedPlan) {
	*out = *in
	in.ResourcePlan.DeepCopyInto(&out.ResourcePlan)
	in.LaunchPlan.DeepCopyInto(&out.LaunchPlan)
	if in.Signature != nil {
		in, out := &in.Signature, &out.Signature
		*out = new(string)
		**out = **in
	}
	if in.UserDefVersion != nil {
		in, out := &in.UserDefVersion, &out.UserDefVersion
		*out = new(string)
		**out = **in
	}
	if in.CustomInfo != nil {
		in, out := &in.CustomInfo, &out.CustomInfo
		*out = new(string)
		**out = **in
	}
	if in.Online != nil {
		in, out := &in.Online, &out.Online
		*out = new(bool)
		**out = **in
	}
	if in.NotMatchTimeout != nil {
		in, out := &in.NotMatchTimeout, &out.NotMatchTimeout
		*out = new(int64)
		**out = **in
	}
	if in.NotReadyTimeout != nil {
		in, out := &in.NotReadyTimeout, &out.NotReadyTimeout
		*out = new(int64)
		**out = **in
	}
	if in.UpdatingGracefully != nil {
		in, out := &in.UpdatingGracefully, &out.UpdatingGracefully
		*out = new(bool)
		**out = **in
	}
	if in.RestartAfterResourceChange != nil {
		in, out := &in.RestartAfterResourceChange, &out.RestartAfterResourceChange
		*out = new(bool)
		**out = **in
	}
	if in.Preload != nil {
		in, out := &in.Preload, &out.Preload
		*out = new(bool)
		**out = **in
	}
	return
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new VersionedPlan.
func (in *VersionedPlan) DeepCopy() *VersionedPlan {
	if in == nil {
		return nil
	}
	out := new(VersionedPlan)
	in.DeepCopyInto(out)
	return out
}
