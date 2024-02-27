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
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_escapeGroupID(t *testing.T) {
	type args struct {
		groupID string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test",
			args: args{
				groupID: "/root",
			},
			want: "root",
		},
		{
			name: "test",
			args: args{
				groupID: "_a_a_a_",
			},
			want: "a-a-a",
		},
		{
			name: "test",
			args: args{
				groupID: "_a-a-a_",
			},
			want: "a-a-a",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EscapeGroupID(tt.args.groupID); got != tt.want {
				t.Errorf("escapeGroupID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_podAllocator_getNotScheduleTime(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "test",
			args: args{},
			want: 0,
		},
		{
			name: "test",
			args: args{
				pod: &corev1.Pod{
					Spec: corev1.PodSpec{
						NodeName: "host1",
					},
				},
			},
			want: 0,
		},
		{
			name: "test",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.Time{
							Time: time.Now().Add(time.Second * -100),
						},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							corev1.PodCondition{
								Type: corev1.PodScheduled,
							},
						},
					},
				},
			},
			want: 0,
		},
		{
			name: "test",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						CreationTimestamp: metav1.Time{
							Time: time.Now().Add(time.Second * -100),
						},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{
							corev1.PodCondition{},
						},
					},
				},
			},
			want: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getNotScheduleTime(tt.args.pod); got != tt.want {
				t.Errorf("podAllocator.getNotScheduleTime() = %v, want %v", got, tt.want)
			}
		})
	}
}
