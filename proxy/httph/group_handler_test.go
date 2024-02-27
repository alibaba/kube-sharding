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
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/sdk"

	"github.com/alibaba/kube-sharding/common"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec/carbon"
	"github.com/alibaba/kube-sharding/proxy/service"
	"github.com/emicklei/go-restful"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	glog "k8s.io/klog"
)

func TestGroupHandlerGetOptions(t *testing.T) {
	assert := assert.New(t)
	groupHandler := &GroupHandler{}

	// default
	httpReq, _ := http.NewRequest("GET", "", nil)
	opts, err := groupHandler.getSchOptions(restful.NewRequest(httpReq))
	assert.Nil(err)
	assert.Equal(service.SchTypeGroup, opts.SchType)

	// old version
	httpReq, _ = http.NewRequest("GET", "", nil)
	httpReq.Header.Add("single", "1")
	opts, err = groupHandler.getSchOptions(restful.NewRequest(httpReq))
	assert.Nil(err)
	assert.Equal(service.SchTypeRole, opts.SchType)

	// full new version
	httpReq, _ = http.NewRequest("GET", "", nil)
	httpReq.Header.Add("schOptions", "schType=node;appChecksum=123")
	opts, err = groupHandler.getSchOptions(restful.NewRequest(httpReq))
	assert.Nil(err)
	assert.Equal(service.SchTypeNode, opts.SchType)
	assert.Equal("123", opts.AppChecksum)

	// case-insensitive
	httpReq, _ = http.NewRequest("GET", "", nil)
	httpReq.Header.Add("schoptions", "schtype=group;appchecksum=123")
	opts, err = groupHandler.getSchOptions(restful.NewRequest(httpReq))
	assert.Nil(err)
	assert.Equal(service.SchTypeGroup, opts.SchType)
	assert.Equal("123", opts.AppChecksum)
}

func newTestGroupHandler(t *testing.T) (*GroupHandler, *gomock.Controller, *service.MockGroupService) {
	ctrl := gomock.NewController(t)
	groupService := service.NewMockGroupService(ctrl)
	handler, _ := NewGroupHandler(groupService)
	// reset default container
	restful.DefaultContainer = restful.NewContainer()
	restful.Add(handler.NewWebService())
	return handler, ctrl, groupService
}

func TestGroupHandler_getGroup(t *testing.T) {
	assert := assert.New(t)
	_, ctrl, groupService := newTestGroupHandler(t)
	defer ctrl.Finish()
	app := "app1"

	var resp carbon.Response
	gids := []string{"g1"}
	b, _ := json.Marshal(gids)
	groupService.EXPECT().GetGroup(app, service.SchTypeGroup, gids).Times(2).Return(nil, nil)
	_, err := common.TestCallHTTPJson("POST", fmt.Sprintf("/app/%s/group/status", app), b, nil, &resp)
	assert.Nil(err)
	assert.False(resp.HasError())

	_, err = common.TestCallHTTPPB("POST", fmt.Sprintf("/app/%s/group/status", app), b, nil, &resp)
	assert.Nil(err)
	assert.False(resp.HasError())
}

func TestGroupHandler_deleteGroup(t *testing.T) {
	assert := assert.New(t)
	_, ctrl, groupService := newTestGroupHandler(t)
	defer ctrl.Finish()
	app := "app1"
	gid := "g1"

	var resp sdk.APIResponse
	groupService.EXPECT().DeleteGroup(app, service.SchTypeGroup, gid).Times(1).Return(nil)
	_, err := common.TestCallHTTPJson("DELETE", fmt.Sprintf("/app/%s/group/%s", app, gid), nil, nil, &resp)
	assert.Nil(err)
	assert.False(resp.HasError())
}

// matcher for gomock
type jsonMatcher struct {
	b string
}

func newJSONMatcher(b []byte, obj interface{}) *jsonMatcher {
	// format to go json
	json.Unmarshal(b, obj)
	b, _ = json.Marshal(obj)
	return &jsonMatcher{string(b)}
}

func (m *jsonMatcher) String() string {
	return m.b
}

func (m *jsonMatcher) Matches(x interface{}) bool {
	b, _ := json.Marshal(x)
	if string(b) != m.b {
		glog.Infof("expect:%s, actual:%s", m.b, string(b))
	}
	return string(b) == m.b
}

func Test_validateGroupPlans(t *testing.T) {
	type args struct {
		app     string
		plans   map[string]*typespec.GroupPlan
		schType string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "normal",
			args: args{
				app: "test",
				plans: map[string]*typespec.GroupPlan{
					"aa": nil,
					"bb": nil,
				},
			},
			wantErr: false,
		},
		{
			name: "abnormal",
			args: args{
				app: "test",
				plans: map[string]*typespec.GroupPlan{
					"aa": nil,
					"bb": nil,
					"cc": nil,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		carbonv1.MaxGroupsPerApp = 2
		t.Run(tt.name, func(t *testing.T) {
			if err := validateGroupPlans(tt.args.app, tt.args.plans, tt.args.schType); (err != nil) != tt.wantErr {
				t.Errorf("validateGroupPlans() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
