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

package mem

import "sync/atomic"

// MetaInfo tracks the meta info for the specified resource
type MetaInfo struct {
	// [1, N], the current max version
	VersionSeed uint64 `json:"versionSeed"`
}

// NewMetaInfo new a meta info
func NewMetaInfo() *MetaInfo {
	return &MetaInfo{VersionSeed: 0}
}

// Equal check if equals
func (mi *MetaInfo) Equal(other *MetaInfo) bool {
	return mi.VersionSeed == other.VersionSeed
}

// DeepCopy deep copy this
func (mi *MetaInfo) DeepCopy() *MetaInfo {
	dst := &MetaInfo{
		VersionSeed: atomic.LoadUint64(&mi.VersionSeed),
	}
	return dst
}

// IncVersionSeed inc the version seed
func (mi *MetaInfo) IncVersionSeed() uint64 {
	return atomic.AddUint64(&mi.VersionSeed, 1)
}

// CurrentVersion get current version
func (mi *MetaInfo) CurrentVersion() uint64 {
	return atomic.LoadUint64(&mi.VersionSeed)
}
