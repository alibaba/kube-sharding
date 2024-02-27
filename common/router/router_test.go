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

package router

import (
	"fmt"
	"testing"

	"github.com/alibaba/kube-sharding/common"

	"github.com/alibaba/kube-sharding/common/config"
	"github.com/alibaba/kube-sharding/pkg/apiserver-client/typespec"
	"github.com/golang/mock/gomock"
	assert "github.com/stretchr/testify/require"
)

type matcher struct {
	out routeConfig
}

func (m *matcher) Matches(x interface{}) bool {
	cp := x.(*routeConfig)
	*cp = m.out
	return true
}

func (m *matcher) String() string {
	return fmt.Sprintf("%+v", m.out)
}

func Test_simpleRouter_GetNamespace(t *testing.T) {
	assert := assert.New(t)
	router := NewSimpleRouter()
	ns, err := router.GetNamespace("app_1")
	assert.Nil(err)
	assert.Equal("app-1", ns)
	ns, _ = router.GetNamespace("_App_1_")
	assert.Equal("app-1", ns)
}

func Test_routeConfigMatch(t *testing.T) {
	assert := assert.New(t)
	conf := newRouteConfig()
	conf.Specifics = map[string][]string{
		"n1": {"app1"},
		"n2": {"app2", "^app3"},
		"n3": {"fiber.*"},
		"n4": {"drogo.*"},
		"n5": {"drogo_share2"},
		"n6": {"drogo_sh.*"},
	}
	conf.init()
	ns, _ := conf.matchSpecifies("app1")
	assert.Equal("n1", ns)
	ns, _ = conf.matchSpecifies("app2")
	assert.Equal("n2", ns)
	ns, _ = conf.matchSpecifies("app3")
	assert.Equal("n2", ns)
	ns, _ = conf.matchSpecifies("app345")
	assert.Equal("n2", ns)
	ns, _ = conf.matchSpecifies("fiber2_3")
	assert.Equal("n3", ns)
	ns, _ = conf.matchSpecifies("drogo_share2")
	assert.Equal("n5", ns)
	ns, _ = conf.matchSpecifies("drogo_share")
	assert.Equal("n6", ns)
	ns, _ = conf.matchSpecifies("drogo")
	assert.Equal("n4", ns)
}

func Test_configGetFiberId(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	// no config
	configer := config.NewMockConfiger(ctrl)
	configer.EXPECT().Get(FiberStageRouteConfigKey, gomock.Any()).Return(nil).Times(1)
	configer.EXPECT().Get(FiberRouteConfigKey, gomock.Any()).Return(nil).Times(1)

	r := &configRouter{loader: configer}
	val, err := r.GetFiberId("app", "az", "region", "stage", "unit")
	assert.Equal(t, err, NoDefaultError)
	assert.Equal(t, val, "")

	// routed by specifics
	configer0 := config.NewMockConfiger(ctrl)
	conf0 := newRouteConfig()
	conf0.Specifics = map[string][]string{
		"fiber-serverless-pre": {"PRE_PUBLISH"},
	}
	configer0.EXPECT().Get(FiberStageRouteConfigKey, &matcher{*conf0}).Return(nil).Times(1)

	r = &configRouter{loader: configer0}
	val, err = r.GetFiberId("app", "az", "region", "PRE_PUBLISH", "unit")
	assert.Nil(t, err)
	assert.Equal(t, val, "fiber-serverless-pre")

	configer1 := config.NewMockConfiger(ctrl)
	conf1 := newRouteConfig()
	conf1.Defaults = []string{
		"fiber-serverless",
	}
	configer1.EXPECT().Get(FiberStageRouteConfigKey, gomock.Any()).Return(nil).Times(1)
	configer1.EXPECT().Get(FiberRouteConfigKey, &matcher{*conf1}).Return(nil).Times(1)

	r = &configRouter{loader: configer1}
	val, err = r.GetFiberId("app", "az", "region", "PRE_PUBLISH", "unit")
	assert.Nil(t, err)
	assert.Equal(t, val, "fiber-serverless")

	configer2 := config.NewMockConfiger(ctrl)
	conf2 := newRouteConfig()
	conf2.Specifics = map[string][]string{
		"fiber-serverless-pre": {"PRE_PUBLISH"},
	}
	conf3 := newRouteConfig()
	conf3.Defaults = []string{
		"fiber-serverless",
	}
	configer2.EXPECT().Get(FiberStageRouteConfigKey, &matcher{*conf2}).Return(nil).Times(1)
	configer2.EXPECT().Get(FiberRouteConfigKey, &matcher{*conf3}).Return(nil).Times(1)

	r = &configRouter{loader: configer2}
	val, err = r.GetFiberId("app", "az", "region", "PUBLISH", "unit")
	assert.Nil(t, err)
	assert.Equal(t, val, "fiber-serverless")

}

func Test_RouteByConfVal(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockConfiger := config.NewMockConfiger(ctrl)
	common.SetGlobalConfiger(mockConfiger, "")
	mockConfiger.EXPECT().GetString(FiberStageRouteConfigKey).Return("{\"specifics\":{\"fiber-serverless-pre\":[\"PRE_PUBLISH\"]},\"defaults\":[]}", nil).AnyTimes()
	mockConfiger.EXPECT().GetString(FiberRouteConfigKey).Return("{\"defaults\":[\"fiber-serverless2\"]}", nil).AnyTimes()
	router := NewGlobalConfigRouter()
	id, err := router.GetFiberId("app", "az", "region", "PRE_PUBLISH", "unit")
	assert.Nil(t, err)
	assert.Equal(t, "fiber-serverless-pre", id)
	id, err = router.GetFiberId("app", "az", "region", "PUBLISH", "unit")
	assert.Nil(t, err)
	assert.Equal(t, "fiber-serverless2", id)
}

func Test_configRouter_GetNamespace(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type fields struct {
		loader config.Configer
	}
	type args struct {
		appName   string
		groupPlan *typespec.GroupPlan
	}
	// no config
	configer := config.NewMockConfiger(ctrl)
	configer.EXPECT().Get(RouteConfigKey, gomock.Any()).Return(nil).Times(1)

	// routed by specifics
	configer0 := config.NewMockConfiger(ctrl)
	conf0 := newRouteConfig()
	conf0.Specifics = map[string][]string{
		"ns1": {"app"},
	}
	configer0.EXPECT().Get(RouteConfigKey, &matcher{*conf0}).Return(nil).Times(1)

	// routed by defaults
	configer1 := config.NewMockConfiger(ctrl)
	conf1 := newRouteConfig()
	conf1.Defaults = []string{"ns1"}
	configer1.EXPECT().Get(RouteConfigKey, &matcher{*conf1}).Return(nil).Times(1)

	// wrong config
	configer2 := config.NewMockConfiger(ctrl)
	conf2 := newRouteConfig()
	conf2.Specifics = map[string][]string{
		"ns1": {"app"},
		"ns2": {"app"},
	}
	configer2.EXPECT().Get(RouteConfigKey, &matcher{*conf2}).Return(nil).Times(1)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "no config",
			fields: fields{
				loader: configer,
			},
			args: args{
				appName: "app",
			},
			wantErr: true,
		},
		{
			name: "specifiy config",
			fields: fields{
				loader: configer0,
			},
			args: args{
				appName: "app",
			},
			want: "ns1",
		},
		{
			name: "default config",
			fields: fields{
				loader: configer1,
			},
			args: args{
				appName: "app",
			},
			want: "ns1",
		},
		{
			name: "conflict config",
			fields: fields{
				loader: configer2,
			},
			args: args{
				appName: "app",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &configRouter{
				loader: tt.fields.loader,
			}
			got, err := r.GetNamespace(tt.args.appName)
			if (err != nil) != tt.wantErr {
				t.Errorf("configRouter.GetNamespace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("configRouter.GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}
