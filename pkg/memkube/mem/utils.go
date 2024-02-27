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

package mem

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"unsafe"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	glog "k8s.io/klog"
)

const (
	// MemObjectVersionAnnoKey the label key name for object version
	MemObjectVersionAnnoKey = "memkube/object-version"
	// MemObjectVersionAnnoValue version
	MemObjectVersionAnnoValue = "2.0.0"
	// MemObjectOwnerRefsKey the annotation key to hold mem object refs
	MemObjectOwnerRefsKey = "memkube/owner-refs"
)

var (
	// ErrInvalidOwnerRefs invalid owner refs
	ErrInvalidOwnerRefs = errors.New("invalid mem-obj owner refs")
)

// MetaName return the name of a runtime object.
func MetaName(obj runtime.Object) string {
	name, _ := MetaNameKey(obj)
	return name
}

// MetaNameKey used in cache.Indexer.
func MetaNameKey(obj interface{}) (string, error) {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return "", err
	}
	return accessor.GetName(), nil
}

// IsUIDEqual check if two objects has equal uid.
func IsUIDEqual(obj0, obj1 runtime.Object) (bool, error) {
	var eq bool
	err := MetaAccessors(func(accessor ...metav1.Object) error {
		eq = accessor[0].GetUID() == accessor[1].GetUID()
		return nil
	}, obj0, obj1)
	return eq, err
}

// MetaResourceVersion get resource version for an object.
func MetaResourceVersion(obj runtime.Object) (uint64, error) {
	accessor, err := meta.CommonAccessor(obj)
	if err != nil {
		return 0, err
	}
	ver := accessor.GetResourceVersion()
	return ParseResourceVersion(ver)
}

// ParseResourceVersion parse resource version
func ParseResourceVersion(rv string) (uint64, error) {
	return strconv.ParseUint(rv, 10, 64)
}

// MetaAccessors gets meta accessors for objects and call operation function.
func MetaAccessors(fn func(accessor ...metav1.Object) error, objs ...runtime.Object) error {
	accessors := make([]metav1.Object, 0)
	for i := range objs {
		accessor, err := meta.Accessor(objs[i])
		if err != nil {
			return err
		}
		accessors = append(accessors, accessor)
	}
	return fn(accessors...)
}

// Some fields can't be updated
func mergeReadOnlyFields(src, dst runtime.Object) error {
	return MetaAccessors(func(accessors ...metav1.Object) error {
		src, dst := accessors[0], accessors[1]
		if tm := src.GetDeletionTimestamp(); tm != nil {
			dst.SetDeletionTimestamp(tm)
		}
		return nil
	}, src, dst)
}

// SetMetaResourceVersion set the resource version
func SetMetaResourceVersion(obj runtime.Object, ver uint64) (uint64, error) {
	accessor, err := meta.CommonAccessor(obj)
	if err != nil {
		return 0, err
	}
	accessor.SetResourceVersion(strconv.FormatUint(ver, 10))
	return ver, nil
}

// TweakObjectMetas tweats some required properties for an object.
func TweakObjectMetas(obj runtime.Object, ver uint64) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	accessor.SetResourceVersion(strconv.FormatUint(ver, 10))
	uid := accessor.GetUID()
	if uid == "" {
		accessor.SetUID(types.UID(uuid.NewUUID()))
	}
	accessor.SetCreationTimestamp(metav1.Now())
	return Memorize(obj)
}

// TweakTypeMetas tweats object type metas.
func TweakTypeMetas(gvk schema.GroupVersionKind, obj runtime.Object) error {
	accessor, err := meta.TypeAccessor(obj)
	if err != nil {
		return err
	}
	accessor.SetKind(gvk.Kind)
	accessor.SetAPIVersion(gvk.GroupVersion().String())
	return nil
}

// Memorize make a normal kubenetes object into memory object.
func Memorize(obj runtime.Object) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	annos := accessor.GetAnnotations()
	if annos == nil {
		annos = make(map[string]string)
	}
	annos[MemObjectVersionAnnoKey] = MemObjectVersionAnnoValue
	accessor.SetAnnotations(annos)
	return nil
}

// Unmemorize strips memory object as a normal kubenetes object.
func Unmemorize(obj runtime.Object) error {
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}

	annos := accessor.GetAnnotations()
	if annos != nil {
		if _, ok := annos[MemObjectVersionAnnoKey]; ok {
			delete(annos, MemObjectVersionAnnoKey)
		}
		accessor.SetAnnotations(annos)
	}
	return nil
}

// SetOwnerReferences set owner references to object.
//   - If the owner is a k8s object, it's the real owner, set the owner to `OwnerReferences`
//   - If the owner is a mem object, then get the k8s owner of it (from `OwnerReferences`), and set the real k8s owner
//     to ownee's OwnerReferences, and set the direct mem owner to ownee's Annotations.
func SetOwnerReferences(owner runtime.Object, ownerGvk schema.GroupVersionKind, ownee metav1.Object) error {
	refs, err := GetOwneeReferences(owner, ownerGvk)
	if err != nil {
		return err
	}
	objMeta, err := meta.Accessor(owner)
	if err != nil {
		return err
	}
	isMem := IsMemObject(objMeta)
	// First set the real owner refs, the refs maybe constructed from `owner` if the owner is k8s object;
	// or the owner's owner if the owner is a mem object.
	// FIXME: If rollback a mem obj to kube obj, i don't known if the owner ref in the child kube object set by this function,
	// which cause the dirty owner ref in it.
	ownee.SetOwnerReferences(MergeOwnerReferences(refs, filterMemOwnerReference(ownee.GetOwnerReferences(), isMem, string(objMeta.GetUID()))))
	if !isMem {
		// ensure no mem owner refs in annotation
		annos := ownee.GetAnnotations()
		if annos != nil && annos[MemObjectOwnerRefsKey] != "" {
			delete(annos, MemObjectOwnerRefsKey)
			ownee.SetAnnotations(annos)
		}
		return nil
	}
	// Otherwise, the `owner` here is direct mem object owner, set it in the annotations for ownee.
	// We have to store the refs in the annotations because apiserver removes invalid owner refs.
	memRef := *metav1.NewControllerRef(objMeta, ownerGvk)
	b, err := json.Marshal(memRef)
	if err != nil {
		return err
	}
	annos := ownee.GetAnnotations()
	if annos == nil {
		annos = make(map[string]string)
	}
	annos[MemObjectOwnerRefsKey] = string(b)
	ownee.SetAnnotations(annos)
	return nil
}

// Remove the mem object in owner references.
func filterMemOwnerReference(refs []metav1.OwnerReference, isMem bool, uid string) []metav1.OwnerReference {
	if !isMem {
		return refs
	}
	for i := range refs {
		if string(refs[i].UID) == uid {
			return append(refs[:i], refs[i+1:]...)
		}
	}
	return refs
}

// MergeOwnerReferences merge owner refs
func MergeOwnerReferences(src []metav1.OwnerReference, dst []metav1.OwnerReference) []metav1.OwnerReference {
	idx := make(map[string]metav1.OwnerReference)
	for i := range dst {
		idx[string(dst[i].UID)] = dst[i]
	}
	for i := range src {
		if _, ok := idx[string(src[i].UID)]; !ok {
			dst = append(dst, src[i])
		}
	}
	return dst
}

// GetOwneeReferences constructs owner references for an object. If the owner is mem-object, return its owner refs.
func GetOwneeReferences(owner runtime.Object, gvk schema.GroupVersionKind) ([]metav1.OwnerReference, error) {
	objMeta, err := meta.Accessor(owner)
	if err != nil {
		return nil, err
	}
	if !IsMemObject(objMeta) {
		// K8S object can be real owner refs.
		ctlRef := *metav1.NewControllerRef(objMeta, gvk)
		return []metav1.OwnerReference{ctlRef}, nil
	}
	// A mem object's real owner refs are k8s object.
	refs := objMeta.GetOwnerReferences()
	if len(refs) != 1 {
		return nil, ErrInvalidOwnerRefs
	}
	ref := refs[0]
	ref.Controller = utils.BoolPtr(false)
	return []metav1.OwnerReference{ref}, nil
}

// GetControllerOf returns a pointer to a copy of the controllerRef if controllee has a controller
func GetControllerOf(controllee metav1.Object) (*metav1.OwnerReference, error) {
	annos := controllee.GetAnnotations()
	if annos == nil || annos[MemObjectOwnerRefsKey] == "" {
		return metav1.GetControllerOf(controllee), nil
	}
	refstr := annos[MemObjectOwnerRefsKey]
	var ref metav1.OwnerReference
	if err := json.Unmarshal([]byte(refstr), &ref); err != nil {
		return nil, fmt.Errorf("unmarshal owner refs error: %v", err)
	}
	return &ref, nil
}

// IsMemObject checks if an object is mem-kube object.
func IsMemObject(obj metav1.Object) bool {
	annos := obj.GetAnnotations()
	if annos == nil {
		return false
	}
	if _, ok := annos[MemObjectVersionAnnoKey]; ok {
		return true
	}
	return false
}

// Kind gets kind string for object.
func Kind(obj runtime.Object) string {
	t := reflect.TypeOf(obj).Elem()
	return t.Name()
}

// Stringify stringify a go object
func Stringify(o interface{}) string {
	b, err := json.Marshal(o)
	if err != nil {
		glog.Errorf("Stringify object error: %v", err)
		return ""
	}
	return *(*string)(unsafe.Pointer(&b))
}

// Describe describes a runtime.Object, only used to log.
func Describe(o runtime.Object) string {
	if o == nil {
		return "<nil>"
	}
	s := struct {
		Name      string
		Kind      string
		UID       types.UID
		Namespace string

		ResourceVersion   string
		CreationTimestamp metav1.Time
		DeletionTimestamp *metav1.Time
	}{
		Kind: Kind(o),
	}
	MetaAccessors(func(accessor ...metav1.Object) error {
		a := accessor[0]
		s.Name = a.GetName()
		s.Namespace = a.GetNamespace()
		s.UID = a.GetUID()
		s.ResourceVersion = a.GetResourceVersion()
		s.CreationTimestamp = a.GetCreationTimestamp()
		s.DeletionTimestamp = a.GetDeletionTimestamp()
		return nil
	}, o)
	b, _ := json.Marshal(&s)
	return string(b)
}
