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
	"fmt"
	"testing"

	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	assert "github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
)

func TestTweakObjectMetas(t *testing.T) {
	assert := assert.New(t)
	node := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "n1",
		},
	}
	assert.Nil(TweakObjectMetas(node, 1))
	accessor, err := meta.Accessor(node)
	assert.Nil(err)
	assert.True(accessor.GetUID() != "")
	assert.Equal("1", accessor.GetResourceVersion())
	assert.Equal(MemObjectVersionAnnoValue, accessor.GetAnnotations()[MemObjectVersionAnnoKey])
	assert.True(accessor.GetCreationTimestamp().Unix() > 0)

	// Type metas
	assert.Nil(TweakTypeMetas(spec.SchemeGroupVersion.WithKind(Kind(node)), node))
	taccessor, err := meta.TypeAccessor(node)
	assert.Nil(err)
	assert.True(taccessor.GetAPIVersion() != "")
	assert.Equal(Kind(node), taccessor.GetKind())
}

func TestOwnerReferences(t *testing.T) {
	assert := assert.New(t)
	kobj := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "n1",
			UID:  types.UID(uuid.NewUUID()),
		},
	}
	kind := Kind(kobj)
	// 1st level
	refs, err := GetOwneeReferences(kobj, spec.SchemeGroupVersion.WithKind(kind))
	assert.Nil(err)
	assert.Equal(1, len(refs))
	assert.Equal(kobj.UID, refs[0].UID)

	verifyKubeOwner := func(obj metav1.Object) {
		refs := obj.GetOwnerReferences()
		assert.Equal(1, len(refs))
		assert.Equal(kobj.UID, refs[0].UID)
	}
	verifyMemOwner := func(owner metav1.Object, ownee metav1.Object) {
		ref, err := GetControllerOf(ownee)
		assert.Nil(err)
		assert.Equal(owner.GetUID(), ref.UID)
	}

	// 2nd level: mem ref to k8s
	mobj := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "n2",
		},
	}
	assert.Nil(TweakObjectMetas(mobj, 1))
	assert.Nil(SetOwnerReferences(kobj, spec.SchemeGroupVersion.WithKind(kind), mobj))
	verifyKubeOwner(mobj)

	// 3rd level: mem ref to mem
	mobj1 := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "n3",
		},
	}
	assert.Nil(TweakObjectMetas(mobj1, 1))
	assert.Nil(SetOwnerReferences(mobj, spec.SchemeGroupVersion.WithKind(kind), mobj1))
	verifyKubeOwner(mobj1)
	verifyMemOwner(mobj, mobj1)

	// 4rd level: k8s ref to mem
	kobj1 := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "n4",
			UID:  types.UID(uuid.NewUUID()),
		},
	}
	assert.Nil(SetOwnerReferences(mobj1, spec.SchemeGroupVersion.WithKind(kind), kobj1))
	verifyKubeOwner(kobj1)
	verifyMemOwner(mobj1, kobj1)
}

// Filter invalid mem owner refs, and annotations
func TestOwnerReferencesFilter(t *testing.T) {
	assert := assert.New(t)
	kobj := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "n1",
			UID:  types.UID(uuid.NewUUID()),
		},
	}
	mobj := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "n2",
			UID:  types.UID(uuid.NewUUID()),
		},
	}
	kind := Kind(kobj)
	assert.Nil(TweakObjectMetas(mobj, 1)) // memorize the object
	assert.True(IsMemObject(mobj))
	// Set the owner for this mem obj
	assert.Nil(SetOwnerReferences(kobj, spec.SchemeGroupVersion.WithKind(kind), mobj))

	kobj2 := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "n3",
			UID:  types.UID(uuid.NewUUID()),
		},
	}
	// First, a kube obj refs to a kube obj (kobj -> kobj2)
	assert.Nil(SetOwnerReferences(kobj, spec.SchemeGroupVersion.WithKind(kind), kobj2))
	// And then make a mem obj refs to the kube obj (mobj -> kobj2)
	assert.Nil(SetOwnerReferences(mobj, spec.SchemeGroupVersion.WithKind(kind), kobj2))
	// Expect the kube obj has only 1 owner ref, and with annotations
	MetaAccessors(func(accessors ...metav1.Object) error {
		assert.Equal(1, len(accessors[0].GetOwnerReferences()))
		assert.True(accessors[0].GetAnnotations()[MemObjectOwnerRefsKey] != "")
		return nil
	}, kobj2)

	// Change the owner for kobj2 (kobj -> kobj2)
	assert.Nil(SetOwnerReferences(kobj, spec.SchemeGroupVersion.WithKind(kind), kobj2))
	// Expect no annotations and only 1 owner refs
	MetaAccessors(func(accessors ...metav1.Object) error {
		assert.Equal(1, len(accessors[0].GetOwnerReferences()))
		assert.True(accessors[0].GetAnnotations()[MemObjectOwnerRefsKey] == "")
		return nil
	}, kobj2)
}

func TestGetKind(t *testing.T) {
	assert := assert.New(t)
	obj := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "n1",
		},
	}
	assert.Equal("WorkerNode", Kind(obj))
	fmt.Printf("Describe: %s\n", Describe(obj))
}

func TestMergeOwnerReferences(t *testing.T) {
	assert := assert.New(t)
	src := []metav1.OwnerReference{
		{UID: "123"},
		{UID: "124"},
	}
	dst := []metav1.OwnerReference{
		{UID: "123"},
		{UID: "125"},
	}
	refs := MergeOwnerReferences(src, dst)
	assert.Equal(3, len(refs))
}

func TestDeferCall(t *testing.T) {
	fn := func(i int) {
		fmt.Printf("i %d\n", i)
	}
	a := 1
	defer fn(a)
	a = 2
	defer fn(a)
}
