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

package httph

import (
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/carbon"
)

const (
	any = "*/*"
)

func newPBResponse(data []*carbon.GroupStatus, err error, suberr error) *carbon.Response {
	resp := &carbon.Response{Code: utils.Int32Ptr(0), SubCode: utils.Int32Ptr(0), Data: data}
	var msg string
	if err != nil {
		resp.Code = utils.Int32Ptr(-1)
		msg = err.Error()
	}
	if suberr != nil {
		resp.SubCode = utils.Int32Ptr(-1)
		msg += suberr.Error()
	}
	if err == nil && suberr == nil {
		msg = "success"
	}
	resp.Msg = utils.StringPtr(msg)
	return resp
}
