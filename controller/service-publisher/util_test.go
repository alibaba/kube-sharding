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

package publisher

import (
	"reflect"
	"testing"

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_getBizMetas(t *testing.T) {
	type args struct {
		worker *carbonv1.WorkerNode
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "normal",
			args: args{
				worker: &carbonv1.WorkerNode{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app.hippo.io/app-name":     "app",
							"app.hippo.io/cluster-name": "cluster",
							"app.hippo.io/group-name":   "group",
							"app.hippo.io/role-name":    "group.role",
							"app.c2.io/biz-paltform":    "platform",
							"app.c2.io/biz-paltform1":   "platform1",
						},
					},
				},
			},
			want: map[string]string{
				"HIPPO_CLUSTER": "cluster",
				"HIPPO_APP":     "app",
				"HIPPO_ROLE":    "group.role",
				"C2_GROUP":      "group",
				"C2_ROLE":       "role",
				"paltform":      "platform",
				"paltform1":     "platform1",
			},
		},
		{
			name: "nil",
			args: args{
				worker: &carbonv1.WorkerNode{
					ObjectMeta: metav1.ObjectMeta{},
				},
			},
			want: map[string]string{},
		},
		{
			name: "template",
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								Template: func() *carbonv1.HippoPodTemplate {
									var pod = corev1.Pod{
										ObjectMeta: metav1.ObjectMeta{
											Labels: map[string]string{
												"app.hippo.io/app-name":     "app",
												"app.hippo.io/cluster-name": "cluster",
												"app.hippo.io/group-name":   "group",
												"app.hippo.io/role-name":    "group.role",
												"app.c2.io/biz-paltform":    "platform",
												"app.c2.io/biz-paltform1":   "platform1",
											},
										},
									}
									template := carbonv1.GetHippoPodTemplateFromPod(pod)
									return &template
								}(),
							},
						},
					},
				},
			},
			want: map[string]string{
				"HIPPO_CLUSTER": "cluster",
				"HIPPO_APP":     "app",
				"HIPPO_ROLE":    "group.role",
				"C2_GROUP":      "group",
				"C2_ROLE":       "role",
				"paltform":      "platform",
				"paltform1":     "platform1",
			},
		},
		{
			name: "template annotatios",
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								Template: func() *carbonv1.HippoPodTemplate {
									var pod = corev1.Pod{
										ObjectMeta: metav1.ObjectMeta{
											Annotations: map[string]string{
												"app.hippo.io/app-name":             "app",
												"app.hippo.io/cluster-name":         "cluster",
												"app.hippo.io/group-name":           "group",
												"app.hippo.io/role-name":            "group.role",
												"app.c2.io/biz-cm2-metas-paltform":  "platform",
												"app.c2.io/biz-cm2-metas-paltform1": "platform1",
											},
										},
									}
									template := carbonv1.GetHippoPodTemplateFromPod(pod)
									return &template
								}(),
							},
						},
					},
				},
			},
			want: map[string]string{
				"HIPPO_APP":  "app",
				"HIPPO_ROLE": "group.role",
				"C2_GROUP":   "group",
				"C2_ROLE":    "role",
				"paltform":   "platform",
				"paltform1":  "platform1",
			},
		},
		{
			name: "warmup",
			args: args{
				worker: &carbonv1.WorkerNode{
					Spec: carbonv1.WorkerNodeSpec{
						VersionPlan: carbonv1.VersionPlan{
							SignedVersionPlan: carbonv1.SignedVersionPlan{
								Template: func() *carbonv1.HippoPodTemplate {
									var pod = corev1.Pod{
										ObjectMeta: metav1.ObjectMeta{
											Labels: map[string]string{
												"app.hippo.io/app-name":     "app",
												"app.hippo.io/cluster-name": "cluster",
												"app.hippo.io/group-name":   "group",
												"app.hippo.io/role-name":    "group.role",
												"app.c2.io/biz-paltform":    "platform",
												"app.c2.io/biz-paltform1":   "platform1",
											},
										},
									}
									template := carbonv1.GetHippoPodTemplateFromPod(pod)
									return &template
								}(),
								Cm2TopoInfo: "cm2:topo",
							},
						},
					},
					Status: carbonv1.WorkerNodeStatus{
						Warmup: true,
					},
				},
			},
			want: map[string]string{
				"HIPPO_CLUSTER": "cluster",
				"HIPPO_APP":     "app",
				"HIPPO_ROLE":    "group.role",
				"C2_GROUP":      "group",
				"C2_ROLE":       "role",
				"paltform":      "platform",
				"paltform1":     "platform1",
				"target_weight": "-100",
				"topo_info":     "cm2:topo",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getBizMetas(tt.args.worker); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getBizMetas() = %v, want %v", got, tt.want)
			}
		})
	}
}
