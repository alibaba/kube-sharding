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
	"errors"
	"reflect"
	"sort"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"

	mapset "github.com/deckarep/golang-set"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
)

// PodSpecMerger merge webhooks
type PodSpecMerger struct {
	targetC2DeclaredKeys *carbonv1.KeysMayBeAdjust
}

func (m *PodSpecMerger) getEnvKeysFromTarget(meta *metav1.ObjectMeta, podSpec *corev1.PodSpec) []string {
	var envKeys = make([]string, 0, 64)
	var envIndex = map[string]string{}
	for i := range podSpec.Containers {
		// collect env keys
		for j := range podSpec.Containers[i].Env {
			if _, ok := envIndex[podSpec.Containers[i].Env[j].Name]; ok {
				continue
			}
			envIndex[podSpec.Containers[i].Env[j].Name] = ""
			envKeys = append(envKeys, podSpec.Containers[i].Env[j].Name)
		}
	}
	return envKeys
}

func (m *PodSpecMerger) getVolMntDeclaredKeys(volumeKey string, containers []corev1.Container) string {
	for i := range containers {
		container := containers[i]
		for j := range container.VolumeMounts {
			volMnt := container.VolumeMounts[j]
			if volMnt.Name == carbonv1.UnifiedStorageInjectedVolumeName && volMnt.SubPath == volumeKey {
				sign, _ := utils.SignatureShort(volMnt.Name + "_" + volMnt.SubPath + "_" + volMnt.MountPath)
				return sign
			} else if volMnt.Name != carbonv1.UnifiedStorageInjectedVolumeName && volMnt.Name == volumeKey {
				return volumeKey
			}
		}
	}
	return ""
}

func (m *PodSpecMerger) getKeysFromTarget(meta *metav1.ObjectMeta, podSpec *corev1.PodSpec) *carbonv1.KeysMayBeAdjust {
	if meta == nil {
		return nil
	}
	if m.targetC2DeclaredKeys != nil {
		return m.targetC2DeclaredKeys
	}
	var declaredKeys = carbonv1.KeysMayBeAdjust{
		VolumeKeys:               make([]string, 0, 32),
		TolerationKeys:           make([]string, 0, 32),
		LableKeys:                make([]string, 0, 32),
		AnnotationKeys:           make([]string, 0, 32),
		ResKeys:                  make([]string, 0, 32),
		ContainerKeys:            make([]string, 0, len(podSpec.Containers)),
		WithWorkingDirContainers: make([]string, 0, 2),
	}
	if podSpec.InitContainers != nil {
		declaredKeys.InitContainerKeys = make([]string, 0, len(podSpec.InitContainers))
	}
	for k := range meta.Labels {
		declaredKeys.LableKeys = append(declaredKeys.LableKeys, k)
	}
	sort.Strings(declaredKeys.LableKeys)
	for k := range meta.Annotations {
		declaredKeys.AnnotationKeys = append(declaredKeys.AnnotationKeys, k)
	}
	if _, ok := meta.Annotations[carbonv1.AnnotationC2DeclaredEnvKeys]; !ok {
		declaredKeys.AnnotationKeys = append(declaredKeys.AnnotationKeys, carbonv1.AnnotationC2DeclaredEnvKeys)
	}
	sort.Strings(declaredKeys.AnnotationKeys)
	for i := range podSpec.Volumes {
		volMntKey := m.getVolMntDeclaredKeys(podSpec.Volumes[i].Name, podSpec.Containers)
		if volMntKey != "" {
			declaredKeys.VolumeKeys = append(declaredKeys.VolumeKeys, volMntKey)
		}
	}
	for i := range podSpec.Tolerations {
		key := m.getTolerationKey(podSpec.Tolerations[i])
		declaredKeys.TolerationKeys = append(declaredKeys.TolerationKeys, key)
	}

	var resIndex = map[string]string{}
	for i := range podSpec.Containers {
		// collect resource keys
		for resName := range podSpec.Containers[i].Resources.Limits {
			name := string(resName)
			if _, ok := resIndex[name]; ok {
				continue
			}
			resIndex[name] = ""
			declaredKeys.ResKeys = append(declaredKeys.ResKeys, name)
		}
		for resName := range podSpec.Containers[i].Resources.Requests {
			name := string(resName)
			if _, ok := resIndex[name]; ok {
				continue
			}
			resIndex[name] = ""
			declaredKeys.ResKeys = append(declaredKeys.ResKeys, name)
		}

		if podSpec.Containers[i].WorkingDir != "" {
			declaredKeys.WithWorkingDirContainers = append(declaredKeys.ResKeys, podSpec.Containers[i].Name)
		}
		declaredKeys.ContainerKeys = append(declaredKeys.ContainerKeys, podSpec.Containers[i].Name)
	}
	for i := range podSpec.InitContainers {
		declaredKeys.InitContainerKeys = append(declaredKeys.InitContainerKeys, podSpec.InitContainers[i].Name)
	}
	declaredKeys.SchedulerName = podSpec.SchedulerName
	sort.Strings(declaredKeys.ResKeys)
	m.targetC2DeclaredKeys = &declaredKeys
	return m.targetC2DeclaredKeys
}

func (m *PodSpecMerger) setKeysToPod(keys *carbonv1.KeysMayBeAdjust, pod *corev1.Pod) error {
	if nil == pod {
		return nil
	}
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	// update carbonv1.AnnotationC2DeclaredKeys annotation value
	b, err := json.Marshal(keys)
	if err != nil {
		return err
	}
	pod.Annotations[carbonv1.AnnotationC2DeclaredKeys] = string(b)
	return nil
}

func (m *PodSpecMerger) setEnvKeysToPod(keys []string, pod *corev1.Pod) error {
	if nil == pod {
		return nil
	}
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}
	// update carbonv1.AnnotationC2DeclaredKeys annotation value
	b, err := json.Marshal(keys)
	if err != nil {
		return err
	}
	pod.Annotations[carbonv1.AnnotationC2DeclaredEnvKeys] = string(b)
	return nil
}

// keys declared by c2, exclude carbonv1.AnnotationC2DeclaredKeys
func (m *PodSpecMerger) getC2DeclaredKeys(annotations map[string]string) (*carbonv1.KeysMayBeAdjust, error) {
	if nil == annotations {
		return nil, errors.New("getC2DeclaredKeys with nil annotations")
	}
	str, ok := annotations[carbonv1.AnnotationC2DeclaredKeys]
	if !ok || "" == str {
		return nil, errors.New("getC2DeclaredKeys with nil declared keys")
	}
	var keys carbonv1.KeysMayBeAdjust
	err := json.Unmarshal([]byte(str), &keys)
	return &keys, err
}

// keys declared by c2, exclude carbonv1.AnnotationC2DeclaredEnvKeys
func (m *PodSpecMerger) getC2DeclaredEnvKeys(annotations map[string]string) ([]string, error) {
	if nil == annotations {
		return nil, errors.New("getC2DeclaredEnvKeys with nil annotations")
	}
	str, ok := annotations[carbonv1.AnnotationC2DeclaredEnvKeys]
	if !ok || "" == str {
		return nil, errors.New("getC2DeclaredEnvKeys with nil declared keys")
	}
	var envKeys []string
	err := json.Unmarshal([]byte(str), &envKeys)
	return envKeys, err
}

func (m *PodSpecMerger) InitC2DelcaredKeysAnno(target *corev1.Pod, podSpec *corev1.PodSpec) error {
	if nil == target {
		return nil
	}
	if nil == podSpec {
		podSpec = &target.Spec
	}
	return m.setKeysToPod(m.getKeysFromTarget(&target.ObjectMeta, podSpec), target)
}

func (m *PodSpecMerger) InitC2DelcaredEnvKeysAnno(target *corev1.Pod, podSpec *corev1.PodSpec) error {
	if nil == target {
		return nil
	}
	if nil == podSpec {
		podSpec = &target.Spec
	}
	return m.setEnvKeysToPod(m.getEnvKeysFromTarget(&target.ObjectMeta, podSpec), target)
}

func (m *PodSpecMerger) MergeTargetPods(target *corev1.Pod, podSpec *corev1.PodSpec, current *corev1.Pod) error {
	if nil == current || nil == target {
		return nil
	}
	if err := m.InitC2DelcaredKeysAnno(target, podSpec); err != nil {
		return err
	}
	lastC2DeclaredKeys, err := m.getC2DeclaredKeys(current.Annotations)
	if nil != err {
		return nil
	}

	labels := m.mergeKVs(target.Labels, current.Labels, sliceToSet(lastC2DeclaredKeys.LableKeys))
	if len(labels) > 0 {
		target.Labels = labels
	}
	annos := m.mergeKVs(target.Annotations, current.Annotations, sliceToSet(lastC2DeclaredKeys.AnnotationKeys))
	if len(annos) > 0 {
		target.Annotations = annos
	}

	vols := make([]corev1.Volume, 0)
	for _, vol := range m.mergeSlice(target.Spec.Volumes, current.Spec.Volumes, lastC2DeclaredKeys.VolumeKeys,
		func(v interface{}) string { return v.(corev1.Volume).Name }) {
		vols = append(vols, vol.(corev1.Volume))
	}
	if len(vols) > 0 {
		target.Spec.Volumes = vols
	}

	tols := make([]corev1.Toleration, 0)
	for _, tol := range m.mergeSlice(target.Spec.Tolerations, current.Spec.Tolerations, lastC2DeclaredKeys.TolerationKeys,
		func(t interface{}) string { return m.getTolerationKey(t.(corev1.Toleration)) }) {
		tols = append(tols, tol.(corev1.Toleration))
	}
	if len(tols) > 0 {
		target.Spec.Tolerations = tols
	}

	for i := range target.Spec.Containers {
		for j := range current.Spec.Containers {
			if target.Spec.Containers[i].Name == current.Spec.Containers[j].Name {
				target.Spec.Containers[i].TerminationMessagePath = current.Spec.Containers[j].TerminationMessagePath
				target.Spec.Containers[i].TerminationMessagePolicy = current.Spec.Containers[j].TerminationMessagePolicy
				target.Spec.Containers[i].ImagePullPolicy = current.Spec.Containers[j].ImagePullPolicy
				if "" == target.Spec.Containers[i].WorkingDir && "" != current.Spec.Containers[j].WorkingDir {
					var workingDirSeted = false
					for k := range lastC2DeclaredKeys.WithWorkingDirContainers {
						if target.Spec.Containers[i].Name == lastC2DeclaredKeys.WithWorkingDirContainers[k] {
							workingDirSeted = true
							break
						}
					}
					if !workingDirSeted {
						target.Spec.Containers[i].WorkingDir = current.Spec.Containers[j].WorkingDir
					}
				}
				volMnts := make([]corev1.VolumeMount, 0)
				for _, vMnt := range m.mergeSlice(target.Spec.Containers[i].VolumeMounts, current.Spec.Containers[j].VolumeMounts, lastC2DeclaredKeys.VolumeKeys,
					// 使用云盘后，name会被修改为autogen，之前的name会写入subPath #issue 115767
					func(v interface{}) string {
						mount := v.(corev1.VolumeMount)
						if mount.Name == carbonv1.UnifiedStorageInjectedVolumeName && mount.SubPath != "" {
							hash, _ := utils.SignatureShort(mount.Name + "_" + mount.SubPath + "_" + mount.MountPath)
							return hash
						}
						return mount.Name
					}) {
					volMnts = append(volMnts, vMnt.(corev1.VolumeMount))
				}
				if len(volMnts) > 0 {
					target.Spec.Containers[i].VolumeMounts = volMnts
				}
				if carbonv1.GetPodVersion(target) == carbonv1.GetPodVersion(current) {
					limits := m.mergeResources(target.Spec.Containers[i].Resources.Limits, current.Spec.Containers[j].Resources.Limits, sliceToSet(lastC2DeclaredKeys.ResKeys))
					target.Spec.Containers[i].Resources.Limits = limits

					requests := m.mergeResources(target.Spec.Containers[i].Resources.Requests, current.Spec.Containers[j].Resources.Requests, sliceToSet(lastC2DeclaredKeys.ResKeys))
					target.Spec.Containers[i].Resources.Requests = requests
				}
				break
			}
		}
	}

	containers := make([]corev1.Container, 0)
	for _, container := range m.mergeSlice(target.Spec.Containers, current.Spec.Containers, lastC2DeclaredKeys.ContainerKeys,
		func(v interface{}) string { return v.(corev1.Container).Name }) {
		containers = append(containers, container.(corev1.Container))
	}
	target.Spec.Containers = containers

	if current.Spec.InitContainers != nil {
		initContainers := make([]corev1.Container, 0)
		for _, initContainer := range m.mergeSlice(target.Spec.InitContainers, current.Spec.InitContainers, lastC2DeclaredKeys.InitContainerKeys,
			func(v interface{}) string { return v.(corev1.Container).Name }) {
			initContainers = append(initContainers, initContainer.(corev1.Container))
		}
		target.Spec.InitContainers = initContainers
	}
	return nil
}

func (m *PodSpecMerger) MergeEnvs(target *corev1.Pod, podSpec *corev1.PodSpec, current *corev1.Pod) error {
	if nil == current || nil == target {
		return nil
	}
	if err := m.InitC2DelcaredEnvKeysAnno(target, podSpec); err != nil {
		return err
	}
	envKeys, err := m.getC2DeclaredEnvKeys(current.Annotations)
	if nil != err {
		return nil
	}
	for i := range target.Spec.Containers {
		for j := range current.Spec.Containers {
			if target.Spec.Containers[i].Name == current.Spec.Containers[j].Name {
				env := make([]corev1.EnvVar, 0)
				for _, e := range m.mergeSlice(target.Spec.Containers[i].Env, current.Spec.Containers[j].Env, envKeys,
					func(e interface{}) string { return e.(corev1.EnvVar).Name }) {
					env = append(env, e.(corev1.EnvVar))
				}
				if len(env) > 0 {
					target.Spec.Containers[i].Env = env
				}
				break
			}
		}
	}
	return nil
}

// MergeTolerations MergeTolerations
func (m *PodSpecMerger) MergeTolerations(target *corev1.Pod, podSpec *corev1.PodSpec, current *corev1.Pod) error {
	if nil == current || nil == target {
		return nil
	}
	spec := podSpec.DeepCopy()
	lastC2DeclaredKeys, err := m.getC2DeclaredKeys(current.Annotations)
	if nil != err {
		return err
	}

	tols := make([]corev1.Toleration, 0)
	for _, tol := range m.mergeSlice(target.Spec.Tolerations, current.Spec.Tolerations, lastC2DeclaredKeys.TolerationKeys,
		func(t interface{}) string { return m.getTolerationKey(t.(corev1.Toleration)) }) {
		tols = append(tols, tol.(corev1.Toleration))
	}
	if len(tols) > 0 {
		target.Spec.Tolerations = tols
	}
	lastC2DeclaredKeys.TolerationKeys = []string{}
	for i := range spec.Tolerations {
		key := m.getTolerationKey(spec.Tolerations[i])
		lastC2DeclaredKeys.TolerationKeys = append(lastC2DeclaredKeys.TolerationKeys, key)
	}
	m.setKeysToPod(lastC2DeclaredKeys, target)
	return nil
}

// target: only containing kvs declared by c2
// last: last c2 targets
// current: kvs declared by c2, webhook and others
func (m *PodSpecMerger) mergeKVs(target, current map[string]string, lastSets mapset.Set) map[string]string {
	r := m.mergeGenericKVs(target, current, lastSets)
	r1 := make(map[string]string)
	for k, v := range r {
		r1[k] = v.(string)
	}
	return r1
}

func (m *PodSpecMerger) mergeResources(target, current corev1.ResourceList, lastSets mapset.Set) corev1.ResourceList {
	t := make(map[string]interface{})
	c := make(map[string]interface{})
	for k, v := range target {
		t[string(k)] = v
	}
	for k, v := range current {
		c[string(k)] = v
	}
	ret := make(map[corev1.ResourceName]resource.Quantity)
	for k, v := range m.mergeGenericKVs(t, c, lastSets) {
		ret[corev1.ResourceName(k)] = v.(resource.Quantity)
	}
	return ret
}

func (m *PodSpecMerger) mergeGenericKVs(target, current interface{}, lastSets mapset.Set) map[string]interface{} {
	targetKV := toMap(target)
	currentKV := toMap(current)
	targetKeys := make(map[string]string)
	for k := range targetKV {
		targetKeys[k] = ""
	}
	targetSets := mapToSet(targetKeys)
	_, toDelete, _ := diff(targetSets, lastSets)
	r := make(map[string]interface{})
	for k, v := range currentKV {
		if toDelete != nil && toDelete.Contains(k) {
			continue
		}
		r[k] = v
	}
	for k, newV := range targetKV {
		r[k] = newV
	}
	return r
}

// return nil iff target and current is nil
func (m *PodSpecMerger) mergeSlice(target, current interface{}, lastC2DeclaredKeys []string, getKeyFunc getKeyFunc) []interface{} {
	targetSlice := toSlice(target)
	targetSets := mapset.NewSet()
	for i := range targetSlice {
		targetSets.Add(getKeyFunc(targetSlice[i]))
	}
	lastSets := sliceToSet(lastC2DeclaredKeys)
	_, toDelete, _ := diff(targetSets, lastSets)
	return stablePatchSlice(target, current, toDelete, getKeyFunc)
}

func (m *PodSpecMerger) getTolerationKey(tol corev1.Toleration) string {
	key := tol.Key
	if tol.Key != tol.Value {
		key = tol.Key + tol.Value
	}
	return key
}

func sliceToSet(keys []string) mapset.Set {
	sets := mapset.NewSet()
	if nil == keys {
		return sets
	}
	for i := range keys {
		sets.Add(keys[i])
	}
	return sets
}

func mapToSet(kv map[string]string) mapset.Set {
	sets := mapset.NewSet()
	if nil == kv {
		return sets
	}
	for k := range kv {
		sets.Add(k)
	}
	return sets
}

func diff(new, old mapset.Set) (toAdd, toDelete, toUpdate mapset.Set) {
	if new == nil {
		return nil, old, nil
	}
	if old == nil {
		return new, nil, nil
	}
	toDelete = old.Difference(new)
	toAdd = new.Difference(old)
	toUpdate = new.Intersect(old)
	return
}

func indirect(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Ptr && v.Kind() != reflect.Interface {
		return v
	}
	return v.Elem()
}

func toSlice(arr interface{}) []interface{} {
	var v reflect.Value = indirect(reflect.ValueOf(arr))
	if v.Kind() != reflect.Slice {
		return nil
	}
	l := v.Len()
	ret := make([]interface{}, l)
	for i := 0; i < l; i++ {
		ret[i] = v.Index(i).Interface()
	}
	return ret
}

func toMap(arr interface{}) map[string]interface{} {
	var v reflect.Value = indirect(reflect.ValueOf(arr))
	if v.Kind() != reflect.Map {
		return nil
	}
	ret := make(map[string]interface{})
	for _, key := range v.MapKeys() {
		k := key.Interface()
		str, ok := k.(string)
		if !ok {
			return nil
		}
		ret[str] = v.MapIndex(key).Interface()
	}
	return ret
}

type getKeyFunc func(interface{}) string

func stablePatchSlice(from, to interface{}, toDeleteSets mapset.Set, getKeyFunc getKeyFunc) []interface{} {
	fromSlice := toSlice(from)
	toSlice := toSlice(to)
	if 0 == len(fromSlice) && (toDeleteSets == nil || 0 == toDeleteSets.Cardinality()) {
		return toSlice
	}
	var resultSlice = []interface{}{}
	for i := range toSlice {
		to := toSlice[i]
		if toDeleteSets != nil && toDeleteSets.Contains(getKeyFunc(to)) {
			continue
		}
		resultSlice = append(resultSlice, to)
	}
	for i := range fromSlice {
		from := fromSlice[i]
		var exist = false
		fromKey := getKeyFunc(from)
		for j := range resultSlice {
			if fromKey == getKeyFunc(resultSlice[j]) {
				exist = true
				resultSlice[j] = from
				break
			}
		}
		if !exist {
			resultSlice = append(resultSlice, from)
		}
	}
	return resultSlice
}
