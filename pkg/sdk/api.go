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

package sdk

import (
	v1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
)

type RollingsetAPI interface {
	New(app string, rs *v1.RollingSet) error
	Update(app, name string, rs *v1.RollingSet) error
	Delete(app, name string) error
	Get(app, name string) (*v1.RollingSet, error)
	SyncSubrs(namespace, rollingset string, subrss []v1.Subrs, subrsEnable string) error
	GetVerbose(app, name string) (*v1.RollingsetVerbose, error)
	Patch(namespace, rollingset string, patch *v1.RollingsetPatch) error
}

type ReplicaNodeAPI interface {
	GetStatus(app, rollingset, name string) (*v1.ReplicaStatus, error)
	ReclaimWorkerNode(app, rollingset, name, workerNodeId string) error
}

type CloneSetAPI interface {
	Patch(namespace, cloneSet string, data map[string]interface{}) error
}

type APIResponse struct {
	Code    int         `json:"code"`
	SubCode int         `json:"subCode"`
	Msg     string      `json:"msg"`
	Data    interface{} `json:"data,omitempty"`
}

func (r *APIResponse) Success() bool {
	return !r.HasError()
}

func (r *APIResponse) HasError() bool {
	return r.Code != 0 || r.SubCode != 0
}

func (r *APIResponse) SetCode(code int) {
	r.Code = code
}

func NewResponse(data interface{}, err error, suberr error) *APIResponse {
	resp := &APIResponse{Code: 0, Data: data}
	if err != nil {
		resp.Code = -1
		resp.Msg = err.Error()
	}
	if suberr != nil {
		resp.SubCode = -1
		resp.Msg += suberr.Error()
	}
	if err == nil && suberr == nil {
		resp.Msg = "success"
	}
	return resp
}
