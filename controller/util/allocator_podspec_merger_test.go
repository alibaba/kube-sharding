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

package util

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	mapset "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_podSpecMerger_mergeKVs(t *testing.T) {
	type args struct {
		target  map[string]string
		latest  map[string]string
		current map[string]string
	}
	tests := []struct {
		name      string
		m         *PodSpecMerger
		args      args
		want      map[string]string
		wantError bool
	}{
		{
			name: "normal case",
			m:    &PodSpecMerger{},
			args: args{
				target:  map[string]string{"k1": "new_v1", "k2": "v2"},
				latest:  map[string]string{"k1": "v1"},
				current: map[string]string{"k1": "v1", "k888": "v888", "k999": "v999"},
			},
			want:      map[string]string{"k1": "new_v1", "k2": "v2", "k888": "v888", "k999": "v999"},
			wantError: false,
		},
		{
			name: "normal case 2, delete all keys declared by c2 ",
			m:    &PodSpecMerger{},
			args: args{
				target:  map[string]string{},
				latest:  map[string]string{"k1": "v1"},
				current: map[string]string{"k1": "v1", "k888": "v888", "k999": "v999"},
			},
			want:      map[string]string{"k888": "v888", "k999": "v999"},
			wantError: false,
		},
		{
			name: "normal case 3, update, add, and delete some keys declared by c2 ",
			m:    &PodSpecMerger{},
			args: args{
				target:  map[string]string{"k1": "new_v1", "k2": "new_v2", "k4": "k4"},
				latest:  map[string]string{"k1": "v1", "k2": "v2", "k3": "v3"},
				current: map[string]string{"k1": "v1", "k2": "v2", "k3": "v3", "k888": "v888", "k999": "v999"},
			},
			want:      map[string]string{"k1": "new_v1", "k2": "new_v2", "k4": "k4", "k888": "v888", "k999": "v999"},
			wantError: false,
		},
		{
			name: "normal case 4, nil lastest keys declared by c2 ",
			m:    &PodSpecMerger{},
			args: args{
				target:  map[string]string{"k1": "new_v1", "k2": "new_v2", "k4": "k4"},
				latest:  nil,
				current: map[string]string{"k888": "v888", "k999": "v999"},
			},
			want:      map[string]string{"k1": "new_v1", "k2": "new_v2", "k4": "k4", "k888": "v888", "k999": "v999"},
			wantError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &PodSpecMerger{}
			got := m.mergeKVs(tt.args.target, tt.args.current, mapToSet(tt.args.latest))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PodSpecMerger.mergeKVs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_podSpecMerger_mergeSlice(t *testing.T) {
	type args struct {
		target     interface{}
		current    interface{}
		lastKeys   []string
		getKeyFunc getKeyFunc
	}
	tests := []struct {
		name string
		m    *PodSpecMerger
		args args
		want []interface{}
	}{
		{
			name: "normal case",
			m:    &PodSpecMerger{},
			args: args{
				target:     []interface{}{testObj{id: "k1", value: "new_v1"}, testObj{id: "k2", value: "v2"}},
				current:    []interface{}{testObj{id: "k1", value: "v1"}, testObj{id: "k2", value: "v2"}, testObj{id: "k3", value: "v3"}, testObj{id: "k4", value: "v4"}},
				lastKeys:   []string{"k1", "k2", "k3"},
				getKeyFunc: func(v interface{}) string { return v.(testObj).id },
			},
			want: []interface{}{testObj{id: "k1", value: "new_v1"}, testObj{id: "k2", value: "v2"}, testObj{id: "k4", value: "v4"}},
		},
		{
			name: "normal case nil target",
			m:    &PodSpecMerger{},
			args: args{
				target:     nil,
				current:    []interface{}{testObj{id: "k1", value: "v1"}, testObj{id: "k2", value: "v2"}, testObj{id: "k3", value: "v3"}, testObj{id: "k4", value: "v4"}},
				lastKeys:   []string{"k1", "k2", "k3"},
				getKeyFunc: func(v interface{}) string { return v.(testObj).id },
			},
			want: []interface{}{testObj{id: "k4", value: "v4"}},
		},
		{
			name: "normal case, empty target",
			m:    &PodSpecMerger{},
			args: args{
				target:     []interface{}{},
				current:    []interface{}{testObj{id: "k1", value: "v1"}, testObj{id: "k2", value: "v2"}, testObj{id: "k3", value: "v3"}, testObj{id: "k4", value: "v4"}},
				lastKeys:   []string{"k1", "k2", "k3"},
				getKeyFunc: func(v interface{}) string { return v.(testObj).id },
			},
			want: []interface{}{testObj{id: "k4", value: "v4"}},
		},
		{
			name: "normal case, nil current",
			m:    &PodSpecMerger{},
			args: args{
				target:     []interface{}{testObj{id: "k1", value: "new_v1"}, testObj{id: "k2", value: "v2"}},
				current:    nil,
				lastKeys:   nil,
				getKeyFunc: func(v interface{}) string { return v.(testObj).id },
			},
			want: []interface{}{testObj{id: "k1", value: "new_v1"}, testObj{id: "k2", value: "v2"}},
		},
		{
			name: "normal case, nil taregt, current",
			m:    &PodSpecMerger{},
			args: args{
				target:     nil,
				current:    nil,
				lastKeys:   nil,
				getKeyFunc: func(v interface{}) string { return v.(testObj).id },
			},
			want: nil,
		},
		{
			name: "normal case, nil taregt, empty current",
			m:    &PodSpecMerger{},
			args: args{
				target:     nil,
				current:    []interface{}{},
				lastKeys:   []string{},
				getKeyFunc: func(v interface{}) string { return v.(testObj).id },
			},
			want: []interface{}{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &PodSpecMerger{}
			if got := m.mergeSlice(tt.args.target, tt.args.current, tt.args.lastKeys, tt.args.getKeyFunc); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PodSpecMerger.mergeSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mergeAnnotationsOfTargetPods(t *testing.T) {
	assert := assert.New(t)
	tPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}
	cPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}
	pM := &PodSpecMerger{}
	err := pM.MergeTargetPods(nil, nil, nil)
	assert.Nil(err)

	// normal case update, add and delete in Annotations field
	tPod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"k1": "new_v1", "k3": "v3"},
		},
	}
	cPod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"k1": "v1", "k2": "v2", "wh3": "wh_v3", carbonv1.AnnotationC2DeclaredKeys: `{"annotationKeys":["k1","k2"]}`},
		},
	}
	pM = &PodSpecMerger{}
	err = pM.MergeTargetPods(tPod, nil, cPod)
	assert.Nil(err)

	assertC2DelaredKeysEqual(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"k1": "new_v1", "wh3": "wh_v3", "k3": "v3", carbonv1.AnnotationC2DeclaredKeys: `{"annotationKeys":["app.c2.io/c2-declared-env-keys","k1","k3"]}`},
		},
	}, tPod, assert)
	assert.NotEqual(tPod, cPod)
}

func Test_mergeLabelsOfTargetPods(t *testing.T) {

}

func Test_mergeVolumeOfTargetPods(t *testing.T) {
	assert := assert.New(t)
	tPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}
	cPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
	}
	pM := &PodSpecMerger{}

	// normal case update, add and delete in Volume field
	c2Vol1 := corev1.Volume{
		Name: "c2_vol1",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "c2_vol1_path1",
			}},
	}
	c2Vol3 := corev1.Volume{
		Name: "c2_vol3",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "c2_vol1_path3",
			}},
	}
	tPod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"k1": "new_v1", "k3": "v3"},
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				c2Vol1,
				c2Vol3,
			},
			Containers: []corev1.Container{
				{
					VolumeMounts: []corev1.VolumeMount{
						{
							Name: "c2_vol3",
						}, {
							Name: "c2_vol1",
						},
					},
				},
			},
		},
	}

	webHookVol2 := corev1.Volume{
		Name: "webhook_vol_2",
	}
	c2OldVol1 := corev1.Volume{
		Name: "c2_vol1",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "c2_vol1_old_path1",
			}},
	}
	c2Vol4 := corev1.Volume{
		Name: "c2_vol4",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "c2_vol1_path4",
			}},
	}
	cPod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"k1": "v1", "k2": "v2", "wh3": "wh_v3", carbonv1.AnnotationC2DeclaredKeys: `{"annotationKeys":["k1","k2"],"volumeKeys":["c2_vol1","c2_vol4"]}`},
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				webHookVol2,
				c2OldVol1,
				c2Vol4,
			},
		},
	}
	err := pM.MergeTargetPods(tPod, nil, cPod)
	assert.Nil(err)
	assert.Equal(3, len(tPod.Spec.Volumes))
	assertC2DelaredKeysEqual(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{carbonv1.AnnotationC2DeclaredKeys: `{"annotationKeys":["app.c2.io/c2-declared-env-keys","k1","k3"],"volumeKeys":["c2_vol1","c2_vol3"]}`},
		},
	}, tPod, assert)

	assert.ElementsMatch([]corev1.Volume{c2Vol1, c2Vol3, webHookVol2}, tPod.Spec.Volumes)
	// volume updated
	for _, v := range tPod.Spec.Volumes {
		if v.Name == "c2_vol1" {
			assert.Equal(c2Vol1.HostPath.Path, v.HostPath.Path)
		}
	}
	// volume add
	assert.Subset(tPod.Spec.Volumes, []corev1.Volume{c2Vol3})
	// volume delete
	assert.NotSubset(tPod.Spec.Volumes, []corev1.Volume{c2Vol4})
	assert.NotEqual(tPod, cPod)
}

func Test_mergeVolumeMountOfContainersInTargetPods(t *testing.T) {
	assert := assert.New(t)
	pM := &PodSpecMerger{}

	// normal case update, add and delete in Env field
	targetVolMnts := []corev1.VolumeMount{{
		Name:      "vol_mnt_1",
		MountPath: "vol_mnt_path_1",
	}, {
		Name:      "vol_mnt_3",
		MountPath: "vol_mnt_path_3",
	}}

	webHookVolMnt := corev1.VolumeMount{
		Name:      "webhook_vol_mnt_1",
		MountPath: "webhook_vol_mnt_1_path",
	}
	currentVolMnts := []corev1.VolumeMount{{
		Name:      "vol_mnt_1",
		MountPath: "vol_mnt_path_1_old",
	}, {
		Name:      "vol_mnt_2",
		MountPath: "vol_mnt_path_2",
	}, webHookVolMnt}

	tPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:         "container1",
					VolumeMounts: targetVolMnts,
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "vol_mnt_1",
				},
				{
					Name: "vol_mnt_3",
				},
			},
		},
	}
	cPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{carbonv1.AnnotationC2DeclaredKeys: `{"volumeKeys":["vol_mnt_1", "vol_mnt_2"]}`},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:         "container1",
					VolumeMounts: currentVolMnts,
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "vol_mnt_1",
				},
				{
					Name: "vol_mnt_2",
				},
				{
					Name: "webhook_vol_mnt_1",
				},
			},
		},
	}
	err := pM.MergeTargetPods(tPod, nil, cPod)
	assert.Nil(err)
	assert.Equal(3, len(tPod.Spec.Containers[0].VolumeMounts))
	assertC2DelaredKeysEqual(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{carbonv1.AnnotationC2DeclaredKeys: `{"volumeKeys":["vol_mnt_1", "vol_mnt_3"],"annotationKeys":["app.c2.io/c2-declared-env-keys"]}`},
		},
	}, tPod, assert)

	assert.ElementsMatch([]corev1.VolumeMount{
		{Name: "vol_mnt_1", MountPath: "vol_mnt_path_1"},
		{Name: "vol_mnt_3", MountPath: "vol_mnt_path_3"},
		webHookVolMnt}, tPod.Spec.Containers[0].VolumeMounts)
	assert.NotEqual(tPod, cPod)
}

func Test_mergeVolumeMountOfContainersInTargetPodsMultiContainers(t *testing.T) {
	assert := assert.New(t)
	pM := &PodSpecMerger{}

	// normal case update, add and delete in Env field
	targetVolMnts := []corev1.VolumeMount{{
		Name:      "vol_mnt_1",
		MountPath: "vol_mnt_path_1",
	}, {
		Name:      "vol_mnt_3",
		MountPath: "vol_mnt_path_3",
	}}

	webHookVolMnt := corev1.VolumeMount{
		Name:      "webhook_vol_mnt_1",
		MountPath: "webhook_vol_mnt_1_path",
	}
	currentVolMnts := []corev1.VolumeMount{{
		Name:      "vol_mnt_1",
		MountPath: "vol_mnt_path_1_old",
	}, {
		Name:      "vol_mnt_2",
		MountPath: "vol_mnt_path_2",
	}, webHookVolMnt}

	tPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:         "container1",
					VolumeMounts: targetVolMnts,
				},
				{
					Name:         "container2",
					VolumeMounts: targetVolMnts,
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "vol_mnt_1",
				},
				{
					Name: "vol_mnt_3",
				},
			},
		},
	}
	cPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{carbonv1.AnnotationC2DeclaredKeys: `{"volumeKeys":["vol_mnt_1", "vol_mnt_2"]}`},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:         "container1",
					VolumeMounts: currentVolMnts,
				},
				{
					Name:         "container2",
					VolumeMounts: currentVolMnts,
				},
				{
					Name:         "container3",
					VolumeMounts: currentVolMnts,
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "vol_mnt_1",
				},
				{
					Name: "vol_mnt_2",
				},
				{
					Name: "webhook_vol_mnt_1",
				},
			},
		},
	}
	err := pM.MergeTargetPods(tPod, nil, cPod)
	assert.Nil(err)
	assert.Equal(3, len(tPod.Spec.Containers[0].VolumeMounts))
	assert.Equal(3, len(tPod.Spec.Containers[1].VolumeMounts))
	assertC2DelaredKeysEqual(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{carbonv1.AnnotationC2DeclaredKeys: `{"volumeKeys":["vol_mnt_1", "vol_mnt_3"],"annotationKeys":["app.c2.io/c2-declared-env-keys"]}`},
		},
	}, tPod, assert)

	assert.ElementsMatch([]corev1.VolumeMount{
		{Name: "vol_mnt_1", MountPath: "vol_mnt_path_1"},
		{Name: "vol_mnt_3", MountPath: "vol_mnt_path_3"},
		webHookVolMnt}, tPod.Spec.Containers[0].VolumeMounts)
	assert.NotEqual(tPod, cPod)
}

func Test_mergeEnvOfContainersInTargetPods(t *testing.T) {
	assert := assert.New(t)
	pM := &PodSpecMerger{}

	// normal case update, add and delete in Env field
	targetEnv := []corev1.EnvVar{{
		Name:  "env_1",
		Value: "env_value_1",
	}, {
		Name:  "env_3",
		Value: "env_value_3",
	}}

	webHookEnv := corev1.EnvVar{
		Name:  "webhook_env_1",
		Value: "webhook_value_1",
	}
	currentEnv := []corev1.EnvVar{{
		Name:  "env_1",
		Value: "env_value_1_old",
	}, {
		Name:  "env_2",
		Value: "env_value_2",
	}, webHookEnv}

	tPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container1",
					Env:  targetEnv,
				},
			},
		},
	}
	cPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{carbonv1.AnnotationC2DeclaredEnvKeys: `["env_1", "env_2"]`},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container1",
					Env:  currentEnv,
				},
			},
		},
	}
	err := pM.MergeEnvs(tPod, nil, cPod)
	assert.Nil(err)
	assert.Equal(3, len(tPod.Spec.Containers[0].Env))
	assertC2DelaredEnvKeysEqual(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{carbonv1.AnnotationC2DeclaredEnvKeys: `["env_1", "env_3"]`},
		},
	}, tPod, assert)

	assert.ElementsMatch([]corev1.EnvVar{{
		Name:  "env_1",
		Value: "env_value_1",
	}, {
		Name:  "env_3",
		Value: "env_value_3",
	}, webHookEnv}, tPod.Spec.Containers[0].Env)
	assert.NotEqual(tPod, cPod)

	// nil target
	err = pM.MergeEnvs(nil, nil, cPod)
	assert.Nil(err)
	err = pM.MergeEnvs(nil, nil, nil)
	assert.Nil(err)

	//empty c2 declared keys anno
	cPod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{carbonv1.AnnotationC2DeclaredKeys: ``},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container1",
					Env:  currentEnv,
				},
			},
		},
	}
	err = pM.MergeEnvs(tPod, nil, cPod)
	assert.Nil(err)

	tPod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container1",
					Env: []corev1.EnvVar{{
						Name:  "env_1",
						Value: "env_value_1",
					}},
				},
			},
		},
	}
	var podSpec = &corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name: "container1",
				Env: []corev1.EnvVar{{
					Name:  "env_1",
					Value: "env_value_1",
				}},
			},
		},
	}
	cPod = &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{carbonv1.AnnotationC2DeclaredEnvKeys: "[\"PATH\",\"HADOOP_HOME\",\"JAVA_HOME\",\"FSLIB_PANGU_APSARA_PARAMETERS\",\"LD_LIBRARY_PATH\",\"LD_PRELOAD\",\"PLATFORM\",\"RS_ALLOW_RELOAD_BY_CONFIG\",\"containerConfig_service\",\"containerConfig_tag_workspace\",\"containerConfig_tag_group_name\",\"containerConfig_tag_deployment_name\",\"containerConfig_tag_zone_name\",\"containerConfig_tag_zone_type\",\"containerConfig_tag_platform\",\"containerConfig_tag_env\",\"C2_GROUP\",\"C2_ROLE\"]"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container1",
					Env: []corev1.EnvVar{{
						Name:  "env_1",
						Value: "env_value_1_old",
					}, {
						Name:  "LD_PRELOAD",
						Value: "$(HIPPO_APP_INST_ROOT)/usr/local/alinpu/lib64/lib_ops_pub.so.1.12:$(HIPPO_APP_INST_ROOT)/usr/local/alinpu/lib64/lib_ops.so.1.12",
					}},
				},
			},
		},
	}
	pM = &PodSpecMerger{}
	err = pM.MergeEnvs(tPod, podSpec, cPod)

	tPod1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"app.c2.io/c2-declared-env-keys": "[\"env_1\"]"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container1",
					Env: []corev1.EnvVar{{
						Name:  "env_1",
						Value: "env_value_1",
					}},
				},
			},
		},
	}

	assert.Equal(tPod, tPod1)
}

func Test_mergeWorkingDirOfContainersInTargetPods(t *testing.T) {
	assert := assert.New(t)
	pM := &PodSpecMerger{}

	tPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container1",
				},
				{
					Name: "container2",
				},
			},
		},
	}
	cPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{carbonv1.AnnotationC2DeclaredKeys: `{"withWorkingDirContainers":["container1"]}`},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:       "container1",
					WorkingDir: "/home/admin/c2",
				},
				{
					Name:       "container2",
					WorkingDir: "/home/admin/c2",
				},
			},
		},
	}
	err := pM.MergeTargetPods(tPod, nil, cPod)
	assert.Nil(err)
	wantPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{carbonv1.AnnotationC2DeclaredKeys: "{\"annotationKeys\":[\"app.c2.io/c2-declared-env-keys\"],\"containerKeys\":[\"container1\",\"container2\"]}"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container1",
				},
				{
					Name:       "container2",
					WorkingDir: "/home/admin/c2",
				},
			},
		},
	}
	assert.Equal(utils.ObjJSON(tPod), utils.ObjJSON(wantPod))
}

func Test_mergeTolerationsOfTargetPods(t *testing.T) {

}

func assertC2DelaredKeysEqual(a, b *corev1.Pod, assert *assert.Assertions) {
	aKeys := carbonv1.KeysMayBeAdjust{}
	err := json.Unmarshal([]byte(a.Annotations[carbonv1.AnnotationC2DeclaredKeys]), &aKeys)
	assert.Nil(err)
	bKeys := carbonv1.KeysMayBeAdjust{}
	err = json.Unmarshal([]byte(b.Annotations[carbonv1.AnnotationC2DeclaredKeys]), &bKeys)
	assert.Nil(err)
	assert.ElementsMatch(aKeys.AnnotationKeys, bKeys.AnnotationKeys, "annotations keys not equal")
	assert.ElementsMatch(aKeys.VolumeKeys, bKeys.VolumeKeys, "VolumeKeys keys not equal")
	assert.ElementsMatch(aKeys.TolerationKeys, bKeys.TolerationKeys, "TolerationKeys keys not equal")
	assert.ElementsMatch(aKeys.LableKeys, bKeys.LableKeys, "LableKeys keys not equal")
}

func assertC2DelaredEnvKeysEqual(a, b *corev1.Pod, assert *assert.Assertions) {
	aKeys := []string{}
	err := json.Unmarshal([]byte(a.Annotations[carbonv1.AnnotationC2DeclaredEnvKeys]), &aKeys)
	assert.Nil(err)
	bKeys := []string{}
	err = json.Unmarshal([]byte(b.Annotations[carbonv1.AnnotationC2DeclaredEnvKeys]), &bKeys)
	assert.Nil(err)
	assert.ElementsMatch(aKeys, bKeys, "env keys not equal")
}

func Test_podSpecMerger_mergeResources(t *testing.T) {
	q100, q200 := resource.MustParse("100"), resource.MustParse("200")
	c2DecalredKeys := &carbonv1.KeysMayBeAdjust{
		ResKeys: []string{"CPU", "GPU"},
	}
	type fields struct {
		targetC2DeclaredKeys *carbonv1.KeysMayBeAdjust
	}
	type args struct {
		target   corev1.ResourceList
		current  corev1.ResourceList
		lastSets mapset.Set
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   corev1.ResourceList
	}{
		{
			name:   "normal case ( add, delete and update case)",
			fields: fields{c2DecalredKeys},
			args: args{
				target: corev1.ResourceList(
					map[corev1.ResourceName]resource.Quantity{
						"CPU":    q200,
						"NewGPU": q100,
					}),
				current: corev1.ResourceList(
					map[corev1.ResourceName]resource.Quantity{
						"CPU":         q100,
						"GPU":         q100,
						"WebHookDisk": q100,
					}),
				lastSets: sliceToSet(c2DecalredKeys.ResKeys),
			},
			want: corev1.ResourceList(
				map[corev1.ResourceName]resource.Quantity{
					"CPU":         q200,
					"NewGPU":      q100,
					"WebHookDisk": q100,
				}),
		},
		{
			name:   "empty target resource list",
			fields: fields{c2DecalredKeys},
			args: args{
				target: corev1.ResourceList(
					map[corev1.ResourceName]resource.Quantity{}),
				current: corev1.ResourceList(
					map[corev1.ResourceName]resource.Quantity{
						"CPU":         q100,
						"GPU":         q100,
						"WebHookDisk": q100,
					}),
				lastSets: sliceToSet(c2DecalredKeys.ResKeys),
			},
			want: corev1.ResourceList(
				map[corev1.ResourceName]resource.Quantity{
					"WebHookDisk": q100,
				}),
		},
		{
			name:   "empty current resource list",
			fields: fields{c2DecalredKeys},
			args: args{
				target: corev1.ResourceList(
					map[corev1.ResourceName]resource.Quantity{
						"CPU":    q200,
						"NewGPU": q100,
					}),
				current: corev1.ResourceList(
					map[corev1.ResourceName]resource.Quantity{}),
				lastSets: sliceToSet(c2DecalredKeys.ResKeys),
			},
			want: corev1.ResourceList(
				map[corev1.ResourceName]resource.Quantity{
					"CPU":    q200,
					"NewGPU": q100,
				}),
		},
		{
			name:   "empty target & current resource list",
			fields: fields{c2DecalredKeys},
			args: args{
				target:   corev1.ResourceList(map[corev1.ResourceName]resource.Quantity{}),
				current:  corev1.ResourceList(map[corev1.ResourceName]resource.Quantity{}),
				lastSets: sliceToSet(c2DecalredKeys.ResKeys),
			},
			want: corev1.ResourceList(map[corev1.ResourceName]resource.Quantity{}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &PodSpecMerger{
				targetC2DeclaredKeys: tt.fields.targetC2DeclaredKeys,
			}
			if got := m.mergeResources(tt.args.target, tt.args.current, tt.args.lastSets); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PodSpecMerger.mergeResources() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_mergeAutoGenVolumeMounts(t *testing.T) {
	assert := assert.New(t)
	pM := &PodSpecMerger{}

	// normal case update, add and delete in Env field
	targetVolMnts := []corev1.VolumeMount{{
		Name:      "autogen",
		MountPath: "/home/admin/dosa-serverless-runtime/logs",
		SubPath:   "vol_mnt_path_1",
	}, {
		Name:      "autogen",
		MountPath: "/home/admin/vmcommon",
		SubPath:   "vmcommon",
	}, {
		Name:      "autogen",
		MountPath: "/home/admin/top_vmcommon",
		SubPath:   "vmcommon",
	}}

	currentVolMnts := []corev1.VolumeMount{{
		Name:      "autogen",
		MountPath: "/home/admin/dosa-serverless-runtime/logs",
		SubPath:   "vol_mnt_path_1",
	}, {
		Name:      "autogen",
		MountPath: "/home/admin/logs",
		SubPath:   "vol_mnt_path_3",
	}, {
		Name:      "autogen",
		MountPath: "/home/admin/vmcommon",
		SubPath:   "vmcommon",
	}, {
		Name:      "autogen",
		MountPath: "/home/admin/top_vmcommon",
		SubPath:   "vmcommon",
	}}

	tPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:         "container1",
					VolumeMounts: targetVolMnts,
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "vol_mnt_path_1",
				},
			},
		},
	}
	path1Sign, _ := utils.SignatureShort("autogen_vol_mnt_path_1_/home/admin/dosa-serverless-runtime/logs")
	path3Sign, _ := utils.SignatureShort("autogen_vol_mnt_path_3_/home/admin/logs")
	cPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{carbonv1.AnnotationC2DeclaredKeys: `{"volumeKeys":["` + path1Sign + `", "` + path3Sign + `"]}`},
		},
		Spec: corev1.PodSpec{

			Containers: []corev1.Container{
				{
					Name:         "container1",
					VolumeMounts: currentVolMnts,
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "vol_mnt_path_1",
				}, {
					Name: "vol_mnt_path_3",
				}, {
					Name: "vmcommon",
				},
			},
		},
	}
	err := pM.MergeTargetPods(tPod, nil, cPod)
	assert.Nil(err)
	assert.Equal(3, len(tPod.Spec.Containers[0].VolumeMounts))
	assert.ElementsMatch([]corev1.VolumeMount{
		{Name: "autogen", MountPath: "/home/admin/dosa-serverless-runtime/logs", SubPath: "vol_mnt_path_1"},
		{Name: "autogen", MountPath: "/home/admin/top_vmcommon", SubPath: "vmcommon"},
		{Name: "autogen", MountPath: "/home/admin/vmcommon", SubPath: "vmcommon"},
	}, tPod.Spec.Containers[0].VolumeMounts)
	assert.NotEqual(tPod, cPod)
}

func Test_mergeContainersInTargetPods(t *testing.T) {
	assert := assert.New(t)
	pM := &PodSpecMerger{}

	tPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container1",
				},
				{
					Name: "container2",
				},
			},
		},
	}
	cPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{carbonv1.AnnotationC2DeclaredKeys: `{"withWorkingDirContainers":["container1"]}`},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:       "container1",
					WorkingDir: "/home/admin/c2",
				},
				{
					Name:       "container2",
					WorkingDir: "/home/admin/c2",
				},
				{
					Name:       "container3",
					WorkingDir: "/home/admin/c2",
				},
			},
			InitContainers: []corev1.Container{
				{
					Name:       "initContainer1",
					WorkingDir: "/home/admin/c2",
				},
			},
		},
	}
	err := pM.MergeTargetPods(tPod, nil, cPod)
	assert.Nil(err)
	wantPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{carbonv1.AnnotationC2DeclaredKeys: "{\"annotationKeys\":[\"app.c2.io/c2-declared-env-keys\"],\"containerKeys\":[\"container1\",\"container2\"]}"},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "container1",
				},
				{
					Name:       "container2",
					WorkingDir: "/home/admin/c2",
				},
				{
					Name:       "container3",
					WorkingDir: "/home/admin/c2",
				},
			},
			InitContainers: []corev1.Container{
				{
					Name:       "initContainer1",
					WorkingDir: "/home/admin/c2",
				},
			},
		},
	}
	assert.Equal(utils.ObjJSON(tPod), utils.ObjJSON(wantPod))
}

func Test_diff(t *testing.T) {
	type args struct {
		new mapset.Set
		old mapset.Set
	}
	tests := []struct {
		name         string
		args         args
		wantToAdd    mapset.Set
		wantToDelete mapset.Set
		wantToUpdate mapset.Set
	}{
		{
			name: "normal case",
			args: args{
				new: sliceToSet([]string{"a", "b", "c"}),
				old: sliceToSet([]string{"b", "c", "d"}),
			},
			wantToAdd:    sliceToSet([]string{"a"}),
			wantToDelete: sliceToSet([]string{"d"}),
			wantToUpdate: sliceToSet([]string{"b", "c"}),
		},
		{
			name: "normal case 2",
			args: args{
				new: sliceToSet([]string{"a", "b", "c"}),
				old: sliceToSet([]string{"a", "b", "c"}),
			},
			wantToAdd:    sliceToSet([]string{}),
			wantToDelete: sliceToSet([]string{}),
			wantToUpdate: sliceToSet([]string{"a", "b", "c"}),
		},
		{
			name: "nil new set",
			args: args{
				new: nil,
				old: sliceToSet([]string{"b", "c", "d"}),
			},
			wantToAdd:    nil,
			wantToDelete: sliceToSet([]string{"b", "c", "d"}),
			wantToUpdate: nil,
		},
		{
			name: "nil old set",
			args: args{
				new: sliceToSet([]string{"b", "c", "d"}),
				old: nil,
			},
			wantToAdd:    sliceToSet([]string{"b", "c", "d"}),
			wantToDelete: nil,
			wantToUpdate: nil,
		},
		{
			name: "nil new and old set",
			args: args{
				new: nil,
				old: nil,
			},
			wantToAdd:    nil,
			wantToDelete: nil,
			wantToUpdate: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotToAdd, gotToDelete, gotToUpdate := diff(tt.args.new, tt.args.old)
			if !reflect.DeepEqual(gotToAdd, tt.wantToAdd) {
				t.Errorf("diff() gotToAdd = %v, want %v", gotToAdd, tt.wantToAdd)
			}
			if !reflect.DeepEqual(gotToDelete, tt.wantToDelete) {
				t.Errorf("diff() gotToDelete = %v, want %v", gotToDelete, tt.wantToDelete)
			}
			if !reflect.DeepEqual(gotToUpdate, tt.wantToUpdate) {
				t.Errorf("diff() gotToUpdate = %v, want %v", gotToUpdate, tt.wantToUpdate)
			}
		})
	}
}

type testObj struct {
	id    string
	value string
}

func Test_stablePatchSlice(t *testing.T) {
	type args struct {
		from         interface{}
		to           interface{}
		toDeleteSets mapset.Set
		getKeyFunc   getKeyFunc
	}
	tests := []struct {
		name string
		args args
		want []interface{}
	}{
		{
			name: "normal case",
			args: args{
				from:         []interface{}{testObj{id: "k1", value: "new_v1"}, testObj{id: "k2", value: "v2"}},
				to:           []interface{}{testObj{id: "k1", value: "v1"}, testObj{id: "k2", value: "v2"}, testObj{id: "k3", value: "v3"}, testObj{id: "k4", value: "v4"}},
				toDeleteSets: sliceToSet([]string{"k3"}),
				getKeyFunc:   func(v interface{}) string { return v.(testObj).id },
			},
			want: []interface{}{testObj{id: "k1", value: "new_v1"}, testObj{id: "k2", value: "v2"}, testObj{id: "k4", value: "v4"}},
		},
		{
			name: "normal case : update, add & delete",
			args: args{
				from:         []interface{}{testObj{id: "k1", value: "new_v1"}, testObj{id: "k2", value: "v2"}, testObj{id: "k5", value: "v5"}},
				to:           []interface{}{testObj{id: "k1", value: "v1"}, testObj{id: "k2", value: "v2"}, testObj{id: "k3", value: "v3"}, testObj{id: "k4", value: "v4"}},
				toDeleteSets: sliceToSet([]string{"k3", "k9"}),
				getKeyFunc:   func(v interface{}) string { return v.(testObj).id },
			},
			want: []interface{}{testObj{id: "k1", value: "new_v1"}, testObj{id: "k2", value: "v2"}, testObj{id: "k4", value: "v4"}, testObj{id: "k5", value: "v5"}},
		},
		{
			name: "normal case nil from",
			args: args{
				from:         nil,
				to:           []interface{}{testObj{id: "k1", value: "v1"}, testObj{id: "k2", value: "v2"}, testObj{id: "k3", value: "v3"}, testObj{id: "k4", value: "v4"}},
				toDeleteSets: sliceToSet([]string{"k2", "k3", "k4", "k5"}),
				getKeyFunc:   func(v interface{}) string { return v.(testObj).id },
			},
			want: []interface{}{testObj{id: "k1", value: "v1"}},
		},
		{
			name: "normal case nil from and nil delete keys",
			args: args{
				from:         nil,
				to:           []interface{}{testObj{id: "k1", value: "v1"}, testObj{id: "k2", value: "v2"}, testObj{id: "k3", value: "v3"}, testObj{id: "k4", value: "v4"}},
				toDeleteSets: nil,
				getKeyFunc:   func(v interface{}) string { return v.(testObj).id },
			},
			want: []interface{}{testObj{id: "k1", value: "v1"}, testObj{id: "k2", value: "v2"}, testObj{id: "k3", value: "v3"}, testObj{id: "k4", value: "v4"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stablePatchSlice(tt.args.from, tt.args.to, tt.args.toDeleteSets, tt.args.getKeyFunc); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stablePatchSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_toMap(t *testing.T) {
	type args struct {
		arr interface{}
	}
	tests := []struct {
		name string
		args args
		want map[string]interface{}
	}{
		{
			name: "normal case",
			args: args{
				arr: map[string]string{"A": "B"},
			},
			want: map[string]interface{}{"A": "B"},
		},
		{
			name: "normal case 1",
			args: args{
				arr: map[string]test{"A": {v: "B"}},
			},
			want: map[string]interface{}{"A": test{v: "B"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toMap(tt.args.arr); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("toMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

type test struct {
	v string
}
