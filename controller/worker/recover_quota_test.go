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

package worker

import (
	"container/list"
	"testing"
	"time"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
)

func Test_recoverQuota_require(t *testing.T) {
	type fields struct {
		recovers *list.List
	}
	type args struct {
		maxFailedCount *int32
		timeWindow     *int32
	}
	quota := newRecoverQuota()
	tests := []struct {
		name   string
		fields fields
		args   args
		sleep  bool
		want   bool
	}{
		{
			name: "zeroquota",
			fields: fields{
				recovers: quota.recovers,
			},
			args: args{
				maxFailedCount: utils.Int32Ptr(0),
				timeWindow:     utils.Int32Ptr(0),
			},
			want: false,
		},
		{
			name: "nilquota",
			fields: fields{
				recovers: quota.recovers,
			},
			args: args{
				timeWindow: utils.Int32Ptr(0),
			},
			want: true,
		},
		{
			name: "nilwindow",
			fields: fields{
				recovers: quota.recovers,
			},
			args: args{
				maxFailedCount: utils.Int32Ptr(1),
			},
			want: true,
		},
		{
			name: "required",
			fields: fields{
				recovers: quota.recovers,
			},
			args: args{
				maxFailedCount: utils.Int32Ptr(1),
				timeWindow:     utils.Int32Ptr(10),
			},
			sleep: true,
			want:  true,
		},
		{
			name: "required",
			fields: fields{
				recovers: quota.recovers,
			},
			args: args{
				maxFailedCount: utils.Int32Ptr(1),
				timeWindow:     utils.Int32Ptr(1),
			},
			want: true,
		},
		{
			name: "notrequired",
			fields: fields{
				recovers: quota.recovers,
			},
			args: args{
				maxFailedCount: utils.Int32Ptr(1),
				timeWindow:     utils.Int32Ptr(10),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &timeRecoverQuota{
				recovers: tt.fields.recovers,
			}
			if got := r.require(tt.args.maxFailedCount, tt.args.timeWindow); got != tt.want {
				t.Errorf("recoverQuota.require() = %v, want %v", got, tt.want)
			}
			if tt.sleep {
				time.Sleep(2 * time.Second)
			}
		})
	}
}

func Test_recoverParallelQuota_require(t *testing.T) {
	quota := newParallelRecoverQuota()
	{
		if got := quota.require("worker-a", utils.Int32Ptr(0)); got != false {
			t.Errorf("recoverQuota.require() = %v, want %v", got, false)
		}
	}
	{
		if got := quota.require("worker-a", utils.Int32Ptr(2)); got != true {
			t.Errorf("recoverQuota.require() = %v, want %v", got, true)
		}
	}
	{
		if got := quota.require("worker-b", utils.Int32Ptr(2)); got != true {
			t.Errorf("recoverQuota.require() = %v, want %v", got, true)
		}
	}
	{
		if got := quota.require("worker-c", utils.Int32Ptr(2)); got != false {
			t.Errorf("recoverQuota.require() = %v, want %v", got, false)
		}
	}
	{
		if got := quota.require("worker-b", utils.Int32Ptr(2)); got != true {
			t.Errorf("recoverQuota.require() = %v, want %v", got, true)
		}
	}
	{
		quota.giveback("worker-b")
		if got := quota.require("worker-c", utils.Int32Ptr(2)); got != true {
			t.Errorf("recoverQuota.require() = %v, want %v", got, true)
		}
	}

}
