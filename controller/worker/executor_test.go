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

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_updateExecutor_shouldReclaim(t *testing.T) {
	type fields struct {
		baseExecutor baseExecutor
	}
	type args struct {
		before *corev1.Pod
		after  *corev1.Pod
		worker *carbonv1.WorkerNode
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "pod version change",
			args: args{
				before: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"app.hippo.io/app-name":  "app",
							"app.hippo.io/role-name": "role",
						},
					},
				},
				after: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app.hippo.io/app-name":    "app",
							"app.hippo.io/role-name":   "role",
							"app.hippo.io/pod-version": "v3.0",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "not change",
			args: args{
				before: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app.hippo.io/app-name":  "app",
							"app.hippo.io/role-name": "role",
						},
					},
				},
				after: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app.hippo.io/app-name":  "app",
							"app.hippo.io/role-name": "role",
						},
					},
				},
			},
			want: false,
		},
		{
			name: "not change",
			args: args{
				before: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"app.hippo.io/app-name":  "app",
							"app.hippo.io/role-name": "role",
						},
					},
				},
				after: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app.hippo.io/app-name":  "app",
							"app.hippo.io/role-name": "role",
						},
					},
				},
			},
			want: false,
		},
		{
			name: "app change",
			args: args{
				before: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app.hippo.io/app-name":  "app-1",
							"app.hippo.io/role-name": "role",
						},
					},
				},
				after: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app.hippo.io/app-name":  "app",
							"app.hippo.io/role-name": "role",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "role change",
			args: args{
				before: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app.hippo.io/app-name":  "app",
							"app.hippo.io/role-name": "role-1",
						},
					},
				},
				after: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app.hippo.io/app-name":  "app",
							"app.hippo.io/role-name": "role",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "pod version change",
			args: args{
				before: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"app.hippo.io/app-name":  "app",
							"app.hippo.io/role-name": "role",
						},
					},
				},
				after: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app.hippo.io/app-name":    "app",
							"app.hippo.io/role-name":   "role",
							"app.hippo.io/pod-version": "v2.0",
						},
					},
				},
			},
			want: false,
		},
		{
			name: "3.0 pod update not alloc resource",
			args: args{
				before: &corev1.Pod{
					Spec: corev1.PodSpec{
						SchedulerName: carbonv1.SchedulerNameAsi,
					},
				},
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						ResVersion: "a",
					},
				},
				after: &corev1.Pod{},
			},
			want: true,
		},
		{
			name: "3.0 pod update alloc resource",
			args: args{
				before: &corev1.Pod{
					Spec: corev1.PodSpec{
						NodeName:      "a",
						SchedulerName: carbonv1.SchedulerNameAsi,
					},
				},
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						ResVersion: "a",
					},
				},
				after: &corev1.Pod{},
			},
			want: false,
		},
		{
			name: "group change",
			args: args{
				before: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"sigma.ali/instance-group":        "app",
							"sigma.alibaba-inc.com/app-unit":  "role",
							"sigma.alibaba-inc.com/app-stage": "PUBLISH",
						},
					},
				},
				after: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"sigma.ali/instance-group":        "app",
							"sigma.alibaba-inc.com/app-unit":  "role",
							"sigma.alibaba-inc.com/app-stage": "PRE_PUBLISH",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "group change",
			args: args{
				before: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"sigma.ali/instance-group":        "app",
							"sigma.alibaba-inc.com/app-unit":  "role",
							"sigma.alibaba-inc.com/app-stage": "PUBLISH",
						},
					},
				},
				after: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"sigma.ali/instance-group":        "app",
							"sigma.alibaba-inc.com/app-unit":  "role1",
							"sigma.alibaba-inc.com/app-stage": "PUBLISH",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "group change",
			args: args{
				before: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"sigma.ali/instance-group":        "app",
							"sigma.alibaba-inc.com/app-unit":  "role",
							"sigma.alibaba-inc.com/app-stage": "PUBLISH",
						},
					},
				},
				after: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"sigma.ali/instance-group":        "app1",
							"sigma.alibaba-inc.com/app-unit":  "role",
							"sigma.alibaba-inc.com/app-stage": "PUBLISH",
						},
					},
				},
			},
			want: true,
		},
		{
			name: "befor is empty",
			args: args{
				before: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"sigma.ali/instance-group":        "",
							"sigma.alibaba-inc.com/app-unit":  "",
							"sigma.alibaba-inc.com/app-stage": "",
						},
					},
				},
				after: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"sigma.ali/instance-group":        "app",
							"sigma.alibaba-inc.com/app-unit":  "role",
							"sigma.alibaba-inc.com/app-stage": "PRE_PUBLISH",
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &updateExecutor{
				baseExecutor: tt.fields.baseExecutor,
			}
			if got := e.shouldReclaim(tt.args.worker, tt.args.before, tt.args.after); got != tt.want {
				t.Errorf("updateExecutor.isPodAppRoleNameChanged() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_updateExecutor_isPodInplaceUpdateForbidden(t *testing.T) {
	type fields struct {
		baseExecutor baseExecutor
	}
	type args struct {
		before *corev1.Pod
		after  *corev1.Pod
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "update forbidden",
			args: args{
				before: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"alibabacloud.com/inplace-update-forbidden": "true",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "c1",
								Image: "m1",
							},
						},
					},
				},
				after: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"alibabacloud.com/inplace-update-forbidden": "true",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "c1",
								Image: "m2",
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "update not forbidden",
			args: args{
				before: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "c1",
								Image: "m1",
							},
						},
					},
				},
				after: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "c1",
								Image: "m2",
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "update not forbidden",
			args: args{
				before: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"alibabacloud.com/inplace-update-forbidden": "true",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "c1",
								Image: "m1",
							},
						},
					},
				},
				after: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"alibabacloud.com/inplace-update-forbidden": "true",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "c2",
								Image: "m2",
							},
						},
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &updateExecutor{
				baseExecutor: tt.fields.baseExecutor,
			}
			if got := e.isPodInplaceUpdateForbidden(tt.args.before, tt.args.after); got != tt.want {
				t.Errorf("updateExecutor.isPodInplaceUpdateForbidden() = %v, want %v", got, tt.want)
			}
		})
	}
}
