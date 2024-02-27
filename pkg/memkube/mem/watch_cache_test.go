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
	"strconv"
	"testing"
	"time"

	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	assert "github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func TestCacheWatcher(t *testing.T) {
	assert := assert.New(t)
	stopCh := make(chan struct{})
	defer close(stopCh)
	budget := newTimeBudget(stopCh)
	name := "w1"
	node := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Age: 11,
	}
	var ver uint64 = 101
	SetMetaResourceVersion(node, ver)
	node2 := node.DeepCopyObject()
	SetMetaResourceVersion(node2, ver+1)
	removed := false
	timeout := int64(1)
	watcher := newCacheWatcher(0, false, "", "", ver, 1, SelectionPredicate{}, timeout, func(bool) { removed = true })
	result := watcher.ResultChan()
	// new events
	watcher.add(&watchItem{version: ver + 2, event: watch.Event{Type: watch.Deleted, Object: node2}}, budget)
	evt := <-result
	assert.Equal(watch.Deleted, evt.Type)
	// timeout with expired error
	evt = <-result
	assert.Equal(watch.Error, evt.Type)
	status, ok := evt.Object.(*metav1.Status)
	assert.True(ok)
	assert.Equal(metav1.StatusReasonExpired, status.Reason)
	// stop
	watcher.Stop()
	assert.True(removed)

	// verify `local` watch mode
	watcher = newCacheWatcher(0, true, "", "", ver, 1, SelectionPredicate{}, timeout, func(bool) { removed = true })
	watcher.add(&watchItem{version: ver + 2, event: watch.Event{Type: watch.Deleted, Object: node2}}, budget)
	result = watcher.ResultChan()
	evt = <-result
	// no init events received
	assert.Equal(watch.Deleted, evt.Type)
	// no timeout
	timer := time.NewTimer(time.Duration(timeout*2) * time.Second)
loop:
	for {
		select {
		case <-timer.C:
			break loop
		}
	}
	// no expired error event received
	assert.Equal(0, len(result))
	watcher.Stop()
}

func TestCacheWatcherManagement(t *testing.T) {
	assert := assert.New(t)
	wb := NewCacheWatchBroadcaster(10, "")
	defer wb.Shutdown()
	wi, err := wb.Watch(&WatchOptions{}, SelectionPredicate{})
	assert.Nil(err)
	ch := wi.ResultChan()

	node := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "w1",
		},
		Age: 11,
	}
	node.SetResourceVersion("1")
	wb.Action(watch.Event{Type: watch.Added, Object: node})
	// wait the events distributed
	time.Sleep(time.Millisecond * 5)
	wi.Stop()
	assert.Equal(0, len(wb.(*watchBroadcaster).watchers))
	// a new event
	node.SetResourceVersion("2")
	wb.Action(watch.Event{Type: watch.Modified, Object: node})
	// not received because Closed.
	assert.Equal(1, len(ch))

	// another watcher without Stop explicitly
	_, err = wb.Watch(&WatchOptions{}, SelectionPredicate{})
	assert.Nil(err)
}

func TestCacheWatcherSendTimeout(t *testing.T) {
	assert := assert.New(t)
	wb := NewCacheWatchBroadcaster(10, "")
	defer wb.Shutdown()
	_, err := wb.Watch(&WatchOptions{}, SelectionPredicate{})
	assert.Nil(err)
	w, err := wb.Watch(&WatchOptions{}, SelectionPredicate{})
	assert.Nil(err)
	go func() {
		for _ = range w.ResultChan() {
		}
	}()
	node := &spec.WorkerNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: "w1",
		},
		Age: 11,
	}
	assert.Equal(2, len(wb.(*watchBroadcaster).watchers))
	// greater than 10, block sending
	for i := 1; i < 100; i++ {
		node.SetResourceVersion(strconv.Itoa(i))
		wb.Action(watch.Event{Type: watch.Modified, Object: node})
	}
	// wait to send event
	time.Sleep(5 * time.Millisecond)
	// removed by timeout. 1 slow watcher may cause remove all other normal watchers
	assert.True(2 > len(wb.(*watchBroadcaster).watchers))
}
