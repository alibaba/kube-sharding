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
	"testing"
	"time"

	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	assert "github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestReadResults(t *testing.T) {
	assert := assert.New(t)
	interval := time.Millisecond * 10

	node := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "n1",
		},
	}
	pending := newPendingObject(node, nil)
	read := newReadResult(interval, []*pendingObject{pending})
	assert.Equal(ErrWaitTimeout, read.Wait(interval*2))
	pending.Commit(false)
	assert.Nil(read.Wait(interval * 2))
	items := read.List()
	assert.Equal(1, len(items))
	assert.Equal(node.Name, MetaName(items[0]))
	// duplicated wait
	assert.Nil(read.Wait(interval * 2))
}

func TestWriteResults(t *testing.T) {
	assert := assert.New(t)
	interval := time.Millisecond * 10
	node := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "n1",
			ResourceVersion: "1",
		},
	}
	pending := newPendingObject(node, nil)
	write := newWriteResult(schema.GroupResource{}, interval, pending, 0)
	assert.Equal(ErrWaitTimeout, write.Wait(interval*2))
	pending.Commit(false)
	assert.Nil(write.Wait(interval * 2))

	// delete the object while waiting the update requests finished
	write = newWriteResult(schema.GroupResource{}, interval, pending, 1)
	pending.RequestDelete(NewMetaInfo())
	pending.Commit(true)
	assert.True(pending.Deleted())
	err := write.Wait(interval * 2)
	assert.True(errors.IsNotFound(err))
}
