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

package healthcheck

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/alibaba/kube-sharding/common"
	mock "github.com/alibaba/kube-sharding/controller/healthcheck/mock"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func newResponse(statusCode int, status string) *http.Response {
	response := new(http.Response)
	response.Status = "200 ok"
	if statusCode != 200 {
		response.Status = status + " error"
	}
	response.StatusCode = statusCode
	return response
}

func NewWorkernode(healthStatus carbonv1.HealthStatus, lostCount int, lastLostTime, lastTransitionTime int64) *carbonv1.WorkerNode {
	workerNode := carbonv1.WorkerNode{}
	workerNode.Status = carbonv1.WorkerNodeStatus{}
	workerNode.Status.IP = "10.10.2.122"
	workerNode.Status.HealthStatus = healthStatus
	workerNode.Status.Version = "oldversion"
	workerNode.Status.HealthCondition = carbonv1.HealthCondition{}
	workerNode.Status.HealthCondition.Type = carbonv1.Lv7Health
	workerNode.Status.HealthCondition.Status = healthStatus
	workerNode.Status.HealthCondition.LostCount = int32(lostCount)
	workerNode.Status.HealthCondition.LastLostTime = lastLostTime
	workerNode.Status.HealthCondition.Message = "200 ok"
	second := time.Now().Unix()
	//	time.Now().Unix

	time := time.Unix(second, 0)
	//	workerNode.Status.HealthCondition.LastTransitionTime = v1.Time.Unix(v1.Time.Unix.Now())
	workerNode.Status.HealthCondition.LastTransitionTime = metav1.NewTime(time)
	return &workerNode
}

func NewConfig() *carbonv1.HealthCheckerConfig {
	config := carbonv1.HealthCheckerConfig{}
	config.Lv7Config = &carbonv1.Lv7HealthCheckerConfig{}
	config.Lv7Config.LostCountThreshold = 5
	config.Lv7Config.LostTimeout = 10 //10s

	config.Lv7Config.Path = "test"
	config.Lv7Config.Port = intstr.FromInt(8080)
	return &config
}
func Test_transfer_alive(t *testing.T) {
	type args struct {
		response   *http.Response
		err        error
		curTime    int64
		workernode *carbonv1.WorkerNode
		config     *carbonv1.HealthCheckerConfig
	}
	curTime := time.Now().Unix()
	baseWorkernode := NewWorkernode(carbonv1.HealthAlive, 5, 0, 0)
	tests := []struct {
		name string
		args args

		want1          *carbonv1.HealthCondition
		status         carbonv1.HealthStatus
		lastTransition int64
		wantReason     string
	}{
		{
			name: "test5-1 err",
			args: args{response: nil,
				err:        errors.New("test error"),
				curTime:    curTime,
				workernode: NewWorkernode(carbonv1.HealthLost, 6, 0, 0),
				config:     NewConfig()},

			//	want1:          &baseWorkernode.Status.HealthCondition,
			status:         carbonv1.HealthDead,
			lastTransition: curTime,
			wantReason:     "test error",
		},
		{
			name: "test5-2 err",
			args: args{response: nil,
				err:        errors.New("test error"),
				curTime:    curTime,
				workernode: NewWorkernode(carbonv1.HealthAlive, 3, 0, 0),
				config:     NewConfig()},

			//	want1:          &baseWorkernode.Status.HealthCondition,
			status:         carbonv1.HealthAlive,
			lastTransition: curTime,
			wantReason:     "test error",
		},
		{
			name: "test5-3 err",
			args: args{response: nil,
				err:        errors.New("test error"),
				curTime:    curTime,
				workernode: NewWorkernode(carbonv1.HealthAlive, 0, 0, 0),
				config:     NewConfig()},

			//	want1:          &baseWorkernode.Status.HealthCondition,
			status:         carbonv1.HealthAlive,
			lastTransition: curTime,
			wantReason:     "test error",
		},
		{
			name: "test5-4 err",
			args: args{response: nil,
				err:        errors.New("test error"),
				curTime:    curTime,
				workernode: NewWorkernode(carbonv1.HealthAlive, 5, 0, 0),
				config:     NewConfig()},

			//	want1:          &baseWorkernode.Status.HealthCondition,
			status:         carbonv1.HealthLost,
			lastTransition: curTime,
			wantReason:     "test error",
		},
		{
			name: "test4",
			args: args{response: newResponse(403, "403"),
				curTime:    curTime,
				workernode: NewWorkernode(carbonv1.HealthAlive, 10, 0, 0),
				config:     NewConfig()},

			//	want1:          &baseWorkernode.Status.HealthCondition,
			status:         carbonv1.HealthLost,
			lastTransition: curTime,
			wantReason:     "",
		},
		{
			name: "test3",
			args: args{response: newResponse(403, "403"),
				curTime:    curTime,
				workernode: NewWorkernode(carbonv1.HealthAlive, 5, 0, 0),
				config:     NewConfig()},

			//	want1:          &baseWorkernode.Status.HealthCondition,
			status:         carbonv1.HealthLost,
			lastTransition: curTime,
			wantReason:     "",
		},
		{
			name: "test2",
			args: args{response: newResponse(403, "403"),
				curTime:    curTime,
				workernode: NewWorkernode(carbonv1.HealthAlive, 0, 0, 0),
				config:     NewConfig()},

			//	want1:          &baseWorkernode.Status.HealthCondition,
			status:         carbonv1.HealthAlive,
			lastTransition: baseWorkernode.Status.HealthCondition.LastTransitionTime.Unix(),
			wantReason:     "",
		},
		{
			name: "test1",
			args: args{response: newResponse(200, "200 ok"),
				curTime:    curTime,
				workernode: baseWorkernode,
				config:     NewConfig()},
			want1:          &NewWorkernode(carbonv1.HealthAlive, 0, 0, 0).Status.HealthCondition,
			status:         carbonv1.HealthAlive,
			lastTransition: baseWorkernode.Status.HealthCondition.LastTransitionTime.Unix(),
			wantReason:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got1 := transfer(tt.args.response, tt.args.err, tt.args.workernode, tt.args.config)

			if tt.want1 != nil && !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("transfer() got1 = %v, want %v", got1, tt.want1)
			}
			if got1.Status != tt.status {
				t.Errorf("transfer() got1 = %v, want %v", got1.Status, tt.status)
			}
			if baseWorkernode.Status.HealthCondition.LastTransitionTime.Unix() != tt.lastTransition {
				t.Errorf("transfer() got1 = %v, want %v", baseWorkernode.Status.HealthCondition.LastTransitionTime.Unix(), tt.lastTransition)
			}
			if got1.Reason != tt.wantReason {
				t.Errorf("transfer() got1.Reason = %v, wantReason %v", got1.Reason, tt.wantReason)
			}
		})
	}
}
func Test_transfer_unknow(t *testing.T) {
	type args struct {
		response   *http.Response
		err        error
		curTime    int64
		workernode *carbonv1.WorkerNode
		config     *carbonv1.HealthCheckerConfig
	}
	curTime := time.Now().Unix()
	baseWorkernode := NewWorkernode(carbonv1.HealthUnKnown, 5, 0, 0)
	tests := []struct {
		name string
		args args

		want1          *carbonv1.HealthCondition
		status         carbonv1.HealthStatus
		lastTransition int64
	}{
		{
			name: "test4",
			args: args{response: newResponse(403, "403"),
				curTime:    curTime,
				workernode: NewWorkernode(carbonv1.HealthUnKnown, 10, 0, 0),
				config:     NewConfig()},

			//	want1:          &baseWorkernode.Status.HealthCondition,
			status:         carbonv1.HealthLost,
			lastTransition: curTime,
		},
		{
			name: "test3",
			args: args{response: newResponse(403, "403"),
				curTime:    curTime,
				workernode: NewWorkernode(carbonv1.HealthUnKnown, 5, 0, 0),
				config:     NewConfig()},

			//	want1:          &baseWorkernode.Status.HealthCondition,
			status:         carbonv1.HealthLost,
			lastTransition: curTime,
		},
		{
			name: "test2",
			args: args{response: newResponse(403, "403"),
				curTime:    curTime,
				workernode: NewWorkernode(carbonv1.HealthUnKnown, 0, 0, 0),
				config:     NewConfig()},

			//	want1:          &baseWorkernode.Status.HealthCondition,
			status:         carbonv1.HealthUnKnown,
			lastTransition: baseWorkernode.Status.HealthCondition.LastTransitionTime.Unix(),
		},
		{
			name: "test1",
			args: args{response: newResponse(200, "200 ok"),
				curTime:    curTime,
				workernode: baseWorkernode,
				config:     NewConfig()},

			//	want1:          &baseWorkernode.Status.HealthCondition,
			status:         carbonv1.HealthAlive,
			lastTransition: curTime,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got1 := transfer(tt.args.response, tt.args.err, tt.args.workernode, tt.args.config)

			if tt.want1 != nil && !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("transfer() got1 = %v, want %v", got1, tt.want1)
			}
			if got1.Status != tt.status {
				t.Errorf("transfer() got1 = %v, want %v", got1.Status, tt.status)
			}
			if baseWorkernode.Status.HealthCondition.LastTransitionTime.Unix() != tt.lastTransition {
				t.Errorf("transfer() got1 = %v, want %v", baseWorkernode.Status.HealthCondition.LastTransitionTime.Unix(), tt.lastTransition)
			}
		})
	}
}

func Test_transfer_lost(t *testing.T) {
	type args struct {
		response   *http.Response
		err        error
		curTime    int64
		workernode *carbonv1.WorkerNode
		config     *carbonv1.HealthCheckerConfig
	}
	curTime := time.Now().Unix()
	baseWorkernode := NewWorkernode(carbonv1.HealthLost, 5, curTime-5, 0)
	tests := []struct {
		name string
		args args

		want1          *carbonv1.HealthCondition
		status         carbonv1.HealthStatus
		lastTransition int64
	}{
		{
			name: "test4",
			args: args{response: newResponse(403, "403"),
				curTime:    curTime,
				workernode: NewWorkernode(carbonv1.HealthLost, 0, curTime-100, 0),
				config:     NewConfig()},

			//	want1:          &baseWorkernode.Status.HealthCondition,
			status:         carbonv1.HealthDead,
			lastTransition: curTime,
		},
		{
			name: "test3",
			args: args{response: newResponse(403, "403"),
				curTime:    curTime,
				workernode: baseWorkernode,
				config:     NewConfig()},

			want1:          &baseWorkernode.Status.HealthCondition,
			status:         carbonv1.HealthLost,
			lastTransition: curTime,
		},
		{
			name: "test2",
			args: args{response: newResponse(403, "403"),
				curTime:    curTime,
				workernode: baseWorkernode,
				config:     NewConfig()},

			want1:          &baseWorkernode.Status.HealthCondition,
			status:         carbonv1.HealthLost,
			lastTransition: baseWorkernode.Status.HealthCondition.LastTransitionTime.Unix(),
		},
		{
			name: "test1",
			args: args{response: newResponse(200, "200 ok"),
				curTime:    curTime,
				workernode: baseWorkernode,
				config:     NewConfig()},
			//	want1:          &baseWorkernode.Status.HealthCondition,
			status:         carbonv1.HealthAlive,
			lastTransition: curTime,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got1 := transfer(tt.args.response, tt.args.err, tt.args.workernode, tt.args.config)

			if tt.want1 != nil && !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("transfer() got1 = %v, want %v", got1, tt.want1)
			}
			if got1.Status != tt.status {
				t.Errorf("transfer() got1 = %v, want %v", got1.Status, tt.status)
			}
			if baseWorkernode.Status.HealthCondition.LastTransitionTime.Unix() != tt.lastTransition {
				t.Errorf("transfer() got1 = %v, want %v", baseWorkernode.Status.HealthCondition.LastTransitionTime.Unix(), tt.lastTransition)
			}
		})
	}
}

func Test_transfer_dead(t *testing.T) {
	type args struct {
		response   *http.Response
		err        error
		curTime    int64
		workernode *carbonv1.WorkerNode
		config     *carbonv1.HealthCheckerConfig
	}
	curTime := time.Now().Unix()
	baseWorkernode := NewWorkernode(carbonv1.HealthDead, 5, curTime-5, 0)
	tests := []struct {
		name           string
		args           args
		want1          *carbonv1.HealthCondition
		status         carbonv1.HealthStatus
		lastTransition int64
	}{

		{
			name: "test2",
			args: args{response: newResponse(403, "403"),
				curTime:    curTime,
				workernode: baseWorkernode,
				config:     NewConfig()},

			want1:          &baseWorkernode.Status.HealthCondition,
			status:         carbonv1.HealthDead,
			lastTransition: baseWorkernode.Status.HealthCondition.LastTransitionTime.Unix(),
		},
		{
			name: "test1",
			args: args{response: newResponse(200, "200 ok"),
				curTime:    curTime,
				workernode: baseWorkernode,
				config:     NewConfig()},

			//	want1:          &baseWorkernode.Status.HealthCondition,
			status:         carbonv1.HealthAlive,
			lastTransition: curTime,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got1 := transfer(tt.args.response, tt.args.err, tt.args.workernode, tt.args.config)

			if tt.want1 != nil && !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("transfer() got1 = %v, want %v", got1, tt.want1)
			}
			if got1.Status != tt.status {
				t.Errorf("transfer() got1 = %v, want %v", got1.Status, tt.status)
			}
			if baseWorkernode.Status.HealthCondition.LastTransitionTime.Unix() != tt.lastTransition {
				t.Errorf("transfer() got1 = %v, want %v", baseWorkernode.Status.HealthCondition.LastTransitionTime.Unix(), tt.lastTransition)
			}
		})
	}
}

func TestLv7HealthChecker_excuteHTTPGetQuery(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters

		assert.Equal(t, req.URL.String(), "/healthcheck")
		assert.Equal(t, req.Method, "GET")
		assert.Equal(t, req.Header.Get("Content-Type"), utils.HTTPContentTypeJSON)
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(req.Body)
		assert.Equal(t, err, nil)
		//	assert.Equal(t, "test", buf.String())

		// Send response to be tested
		rw.Write([]byte(`{result:OK}`))
	}))
	defer server.Close()
	t.Log(server)

	type fields struct {
		httpClient *common.HTTPClient
		helper     Helper
		rollingSet *carbonv1.RollingSet
	}
	type args struct {
		url string
	}
	helper := &Task{httpClient: common.NewWithHTTPClient(server.Client())}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       *http.Response
		wantErr    bool
		wantResult string
	}{
		{
			name:   "test1",
			fields: fields{helper: helper, rollingSet: newRollingSet("")},
			args:   args{url: server.URL + "/healthcheck"},

			wantErr:    false,
			wantResult: "{result:OK}",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Lv7HealthChecker{
				helper:     tt.fields.helper,
				rollingSet: tt.fields.rollingSet,
			}
			got, err := l.excuteHTTPGetQuery(nil, tt.args.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("Lv7HealthChecker.excuteHttpQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// if !reflect.DeepEqual(got, tt.want) {
			// 	t.Errorf("Lv7HealthChecker.excuteHttpQuery() = %v, want %v", got, tt.want)
			// }

			if got.Status != "200 OK" {
				t.Errorf("Lv7HealthChecker.excuteHttpQuery() = %v, want %v", got.Status, "200 OK")

			}

			if got.StatusCode != 200 {
				t.Errorf("Lv7HealthChecker.excuteHttpQuery() = %v, want %v", got.StatusCode, 200)

			}
			result, _ := utils.ParseRespToString(got, err)
			if result != tt.wantResult {
				t.Errorf("Lv7HealthChecker.excuteHttpQuery() = %v, want %v", got.Status, tt.wantResult)

			}
		})
	}
}

func NewWorkernodeWithCheck(healthStatus carbonv1.HealthStatus, allocStatus carbonv1.AllocStatus, phase carbonv1.WorkerPhase, lostCount int32) *carbonv1.WorkerNode {
	workerNode := carbonv1.WorkerNode{}
	workerNode.Status = carbonv1.WorkerNodeStatus{}
	workerNode.Status.IP = "127.0.0.1"
	workerNode.Status.HealthStatus = healthStatus
	workerNode.Status.HealthCondition = carbonv1.HealthCondition{}
	workerNode.Status.HealthCondition.Status = healthStatus
	workerNode.Status.HealthCondition.LostCount = lostCount
	workerNode.Status.HealthCondition.Version = "testversion"
	workerNode.Status.AllocStatus = allocStatus
	workerNode.Status.Phase = phase
	workerNode.Status.Version = "testversion"
	workerNode.Status.ProcessReady = true
	second := time.Now().Unix()
	//	time.Now().Unix

	time := time.Unix(second, 0)
	//	workerNode.Status.HealthCondition.LastTransitionTime = v1.Time.Unix(v1.Time.Unix.Now())
	workerNode.Status.HealthCondition.LastTransitionTime = metav1.NewTime(time)
	workerNode.Status.HealthCondition.LastLostTime = second
	return &workerNode
}

func NewConfigWithServer(server *httptest.Server, path string) *carbonv1.HealthCheckerConfig {
	config := carbonv1.HealthCheckerConfig{}
	config.Lv7Config = &carbonv1.Lv7HealthCheckerConfig{}
	config.Lv7Config.LostCountThreshold = 5
	config.Lv7Config.LostTimeout = 10 //10s

	config.Lv7Config.Path = path
	// http://ipaddr:port
	url := server.URL
	port := strings.Split(url, ":")[2]

	config.Lv7Config.Port = intstr.FromString(port)
	return &config
}

func TestLv7HealthChecker_doCheck_alive(t *testing.T) {
	type fields struct {
		helper Helper
	}
	type args struct {
		workernode *carbonv1.WorkerNode
		config     *carbonv1.HealthCheckerConfig
	}

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters

		assert.Equal(t, req.URL.String(), "/healthcheck")
		assert.Equal(t, req.Method, "GET")
		assert.Equal(t, req.Header.Get("Content-Type"), utils.HTTPContentTypeJSON)
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(req.Body)
		assert.Equal(t, err, nil)
		//	assert.Equal(t, "test", buf.String())

		// Send response to be tested
		rw.Write([]byte(`{result:OK}`))
	}))
	defer server.Close()
	t.Log(server)
	httpClient := common.NewWithHTTPClient(server.Client())

	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockHelper := mock.NewMockHelper(ctl)
	mockHelper.EXPECT().Sync(gomock.Any(), gomock.Any()).Return(true).MaxTimes(10)
	mockHelper.EXPECT().GetHTTPClient().Return(httpClient).MaxTimes(10)
	mockHelper.EXPECT().GetWorkerNode(gomock.Any()).Return(nil, nil).MaxTimes(10)

	tests := []struct {
		name      string
		fields    fields
		args      args
		status    carbonv1.HealthStatus
		version   string
		lostCount int32
	}{
		{
			name:   "test1",
			fields: fields{helper: mockHelper},
			args: args{workernode: NewWorkernodeWithCheck(carbonv1.HealthAlive, carbonv1.WorkerAssigned, carbonv1.Running, 0),
				config: NewConfigWithServer(server, "healthcheck")},
			status:    carbonv1.HealthAlive,
			version:   "testversion",
			lostCount: 0,
		},
		{
			name:   "test2",
			fields: fields{helper: mockHelper},
			args: args{workernode: NewWorkernodeWithCheck(carbonv1.HealthLost, carbonv1.WorkerAssigned, carbonv1.Running, 2),
				config: NewConfigWithServer(server, "healthcheck")},
			status:    carbonv1.HealthAlive,
			version:   "testversion",
			lostCount: 0,
		},
		{
			name:   "test3",
			fields: fields{helper: mockHelper},
			args: args{workernode: NewWorkernodeWithCheck(carbonv1.HealthLost, carbonv1.WorkerOfflining, carbonv1.Running, 2),
				config: NewConfigWithServer(server, "healthcheck")},
			status:    carbonv1.HealthAlive,
			version:   "testversion",
			lostCount: 0,
		},
		{
			name:   "test4",
			fields: fields{helper: mockHelper},
			args: args{workernode: NewWorkernodeWithCheck(carbonv1.HealthLost, carbonv1.WorkerReleased, carbonv1.Running, 0),
				config: NewConfigWithServer(server, "healthcheck")},
			status:    carbonv1.HealthAlive,
			version:   "testversion",
			lostCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Lv7HealthChecker{
				helper: tt.fields.helper,
				rollingSet: &carbonv1.RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{},
					},
				},
			}
			got := l.doCheck(tt.args.workernode, tt.args.config, nil)
			if got.Status != tt.status {
				t.Errorf("lv7HealthChecker.docheck() = %v, want %v", got, tt.status)
			}
			if got.Version != tt.version {
				t.Errorf("lv7HealthChecker.docheck() = %v, want %v", got.Version, tt.version)
			}
			if got.LostCount != tt.lostCount {
				t.Errorf("lv7HealthChecker.docheck() = %v, want %v", got.LostCount, tt.lostCount)
			}
		})
	}
}

func TestLv7HealthChecker_doCheck_dead(t *testing.T) {
	type fields struct {
		helper Helper
	}
	type args struct {
		workernode *carbonv1.WorkerNode
		config     *carbonv1.HealthCheckerConfig
	}

	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters

		assert.Equal(t, req.URL.String(), "/healthcheck")
		assert.Equal(t, req.Method, "GET")
		assert.Equal(t, req.Header.Get("Content-Type"), utils.HTTPContentTypeJSON)
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(req.Body)
		assert.Equal(t, err, nil)
		//	assert.Equal(t, "test", buf.String())

		// Send response to be tested
		rw.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()
	t.Log(server)
	httpClient := common.NewWithHTTPClient(server.Client())

	ctl := gomock.NewController(t)
	defer ctl.Finish()
	mockHelper := mock.NewMockHelper(ctl)
	mockHelper.EXPECT().Sync(gomock.Any(), gomock.Any()).Return(true).MaxTimes(10)
	mockHelper.EXPECT().GetHTTPClient().Return(httpClient).MaxTimes(10)
	mockHelper.EXPECT().GetWorkerNode(gomock.Any()).Return(nil, nil).MaxTimes(10)

	tests := []struct {
		name      string
		fields    fields
		args      args
		status    carbonv1.HealthStatus
		version   string
		lostCount int32
	}{
		{
			name:   "test1",
			fields: fields{helper: mockHelper},
			args: args{workernode: NewWorkernodeWithCheck(carbonv1.HealthAlive, carbonv1.WorkerAssigned, carbonv1.Running, 0),
				config: NewConfigWithServer(server, "healthcheck")},
			status:    carbonv1.HealthAlive,
			version:   "testversion",
			lostCount: 1,
		},
		{
			name:   "test2",
			fields: fields{helper: mockHelper},
			args: args{workernode: NewWorkernodeWithCheck(carbonv1.HealthAlive, carbonv1.WorkerAssigned, carbonv1.Running, 2),
				config: NewConfigWithServer(server, "healthcheck")},
			status:    carbonv1.HealthAlive,
			version:   "testversion",
			lostCount: 3,
		},
		{
			name:   "test3",
			fields: fields{helper: mockHelper},
			args: args{workernode: NewWorkernodeWithCheck(carbonv1.HealthAlive, carbonv1.WorkerOfflining, carbonv1.Running, 5),
				config: NewConfigWithServer(server, "healthcheck")},
			status:    carbonv1.HealthLost,
			version:   "testversion",
			lostCount: 6,
		},
		{
			name:   "test4",
			fields: fields{helper: mockHelper},
			args: args{workernode: NewWorkernodeWithCheck(carbonv1.HealthLost, carbonv1.WorkerAssigned, carbonv1.Running, 7),
				config: NewConfigWithServer(server, "healthcheck")},
			status:    carbonv1.HealthLost,
			version:   "testversion",
			lostCount: 7,
		},
		{
			name:   "test5",
			fields: fields{helper: mockHelper},
			args: args{workernode: NewWorkernodeWithCheck(carbonv1.HealthDead, carbonv1.WorkerAssigned, carbonv1.Running, 7),
				config: NewConfigWithServer(server, "healthcheck")},
			status:    carbonv1.HealthDead,
			version:   "testversion",
			lostCount: 7,
		},
		{
			name:   "test update version",
			fields: fields{helper: mockHelper},
			args: args{workernode: func() *carbonv1.WorkerNode {
				w := NewWorkernodeWithCheck(carbonv1.HealthAlive, carbonv1.WorkerAssigned, carbonv1.Running, 0)
				w.Status.Version = "newversion"
				return w
			}(),
				config: NewConfigWithServer(server, "healthcheck")},
			status:    carbonv1.HealthUnKnown,
			version:   "newversion",
			lostCount: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &Lv7HealthChecker{
				helper: tt.fields.helper,
				rollingSet: &carbonv1.RollingSet{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{},
					},
				},
			}
			got := l.doCheck(tt.args.workernode, tt.args.config, nil)
			if got.Status != tt.status {
				t.Errorf("lv7HealthChecker.docheck() = %v, want %v", got.Status, tt.status)
			}
			if got.Version != tt.version {
				t.Errorf("lv7HealthChecker.docheck() = %v, want %v", got.Version, tt.version)
			}
			if got.LostCount != tt.lostCount {
				t.Errorf("lv7HealthChecker.docheck() = %v, want %v", got.LostCount, tt.lostCount)
			}
		})
	}
}
