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
	"net/http"
	"sync"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	glog "k8s.io/klog"
)

// Most copied from k8s.io/apiserver/pkg/storage/cacher.go

// NOTE: memkube can't implement a real watch cache, because it's watch version seq can't be guaranteed.
// So this file was refactored already to remove the `watch cache`.
// see also see http://github.com/alibaba/kube-sharding/issues/108763

const (
	cacheCapacity = 10000
)

var timerPool sync.Pool
var neverTimeoutWatch <-chan time.Time = make(chan time.Time)

type watchItem struct {
	version uint64
	event   watch.Event
}

// SelectionPredicate filter events
type SelectionPredicate struct {
	Label labels.Selector
}

// Match check if this obj matches
func (s *SelectionPredicate) Match(obj runtime.Object) (bool, error) {
	// TODO
	return true, nil
}

// WatchBroadcaster broadcasts watch events
type WatchBroadcaster interface {
	Watch(opts *WatchOptions, pred SelectionPredicate) (watch.Interface, error)
	Action(event watch.Event) error
	Shutdown()
}

// broadcast watch events
type watchBroadcaster struct {
	sync.RWMutex
	watchers   map[int]*cacheWatcher
	watcherIdx int
	incoming   chan watch.Event
	kind       string

	dispatchTimeoutBudget *timeBudget
}

// NewCacheWatchBroadcaster creates a cached watch broadcaster
func NewCacheWatchBroadcaster(cap int, kind string) WatchBroadcaster {
	stopCh := make(chan struct{})
	wb := &watchBroadcaster{
		watchers:   make(map[int]*cacheWatcher),
		watcherIdx: 0,
		incoming:   make(chan watch.Event, 100),
		kind:       kind,

		dispatchTimeoutBudget: newTimeBudget(stopCh),
	}
	go wb.process(stopCh)
	return wb
}

func forgetWatcher(wb *watchBroadcaster, idx int) func(bool) {
	return func(lock bool) {
		if lock {
			wb.Lock()
			defer wb.Unlock()
		}
		wb.removeWatcher(idx)
	}
}

func (wb *watchBroadcaster) Watch(opts *WatchOptions, pred SelectionPredicate) (watch.Interface, error) {
	ver := uint64(0)
	chanSize := 10
	wb.Lock()
	defer wb.Unlock()
	id := wb.watcherIdx
	timeout := int64(0)
	if opts.TimeoutSeconds != nil {
		timeout = *opts.TimeoutSeconds
	}
	watcher := newCacheWatcher(id, opts.LocalWatch != nil && *opts.LocalWatch, opts.Source, wb.kind, ver, chanSize, pred, timeout,
		forgetWatcher(wb, id))
	glog.Infof("Add watcher with id %d with opts: %+v, timeout %v", id, *opts, timeout)
	wb.watchers[id] = watcher
	wb.watcherIdx++
	return watcher, nil
}

func (wb *watchBroadcaster) Action(event watch.Event) error {
	wb.incoming <- event
	return nil
}

func (wb *watchBroadcaster) Shutdown() {
	wb.Lock()
	defer wb.Unlock()
	close(wb.incoming)
	for _, w := range wb.watchers {
		w.dostop()
	}
}

func (wb *watchBroadcaster) process(stopCh chan struct{}) {
	n := len(wb.incoming)
	glog.V(5).Infof("watchBroadcaster is to broadcast %d events", n)
	for event := range wb.incoming {
		wb.distribute(event)
	}
	close(stopCh)
}

func (wb *watchBroadcaster) distribute(event watch.Event) {
	ver, err := MetaResourceVersion(event.Object)
	if err != nil {
		glog.Errorf("Get event object resource version failed: %v", err)
		return
	}
	wb.Lock()
	defer wb.Unlock()
	item := &watchItem{version: ver, event: event}
	for _, w := range wb.watchers {
		w.add(item, wb.dispatchTimeoutBudget)
	}
}

func (wb *watchBroadcaster) removeWatcher(id int) {
	if _, ok := wb.watchers[id]; ok {
		delete(wb.watchers, id)
		glog.Infof("Remove watcher id %d", id)
	}
}

// cacherWatcher implements watch.Interface
type cacheWatcher struct {
	sync.Mutex
	id      int
	input   chan *watchItem
	result  chan watch.Event
	done    chan struct{}
	stopped bool
	local   bool
	forget  func(bool)
	pred    SelectionPredicate
	timeout time.Duration
	source  string
	kind    string
}

func newCacheWatcher(id int, local bool, source string, kind string, resourceVersion uint64, chanSize int, pred SelectionPredicate, timeout int64,
	forget func(bool)) *cacheWatcher {
	if local {
		source = "<local>"
	}
	watcher := &cacheWatcher{
		id:     id,
		input:  make(chan *watchItem, chanSize),
		result: make(chan watch.Event, chanSize),
		done:   make(chan struct{}),
		forget: forget,
		pred:   pred,
		local:  local,
		source: source,
		kind:   kind,
	}
	go watcher.process(resourceVersion, timeout)
	return watcher
}

func (c *cacheWatcher) ResultChan() <-chan watch.Event {
	return c.result
}

func (c *cacheWatcher) Stop() {
	c.forget(true)
	c.dostop()
}

func (c *cacheWatcher) innerStop() {
	c.forget(false)
	c.dostop()
}

func (c *cacheWatcher) add(event *watchItem, budget *timeBudget) {
	if c.stopped {
		return
	}
	if !c.local {
		c.addWithTimeout(event, budget)
		return
	}
	// For local mode, don't timeout, otherwise events lost which cause client-go cache is old forever
	// see issue http://github.com/alibaba/kube-sharding/issues/110314 for more details
	// NOTE: If no timeout here, deadlock may happens if the watcher call `Stop` before consume all events
	c.input <- event
}

func (c *cacheWatcher) addWithTimeout(event *watchItem, budget *timeBudget) {
	startTime := time.Now()
	defer Metrics.Get(MetricWatchSendTime).Elapsed(startTime, c.kind, c.source)
	select {
	case c.input <- event:
		return
	default:
	}

	timeout := budget.takeAvailable()

	t, ok := timerPool.Get().(*time.Timer)
	if ok {
		t.Reset(timeout)
	} else {
		t = time.NewTimer(timeout)
	}
	defer timerPool.Put(t)

	select {
	case c.input <- event:
		stopped := t.Stop()
		if !stopped {
			// Consume triggered (but not yet received) timer event
			// so that future reuse does not get a spurious timeout.
			<-t.C
		}
	case <-t.C:
		glog.Warningf("Send event to watcher timeout: %d", c.id)
		// This means that we couldn't send event to that watcher.
		// Since we don't want to block on it infinitely,
		// we simply terminate it.
		c.innerStop()
	}

	budget.returnUnused(timeout - time.Since(startTime))
}

func (c *cacheWatcher) process(resourceVersion uint64, timeout int64) {
	timeoutCh := neverTimeoutWatch
	if !c.local && timeout > 0 {
		t := time.NewTimer(time.Duration(timeout) * time.Second)
		timeoutCh = t.C
	}

	defer close(c.result)
	for {
		select {
		case <-timeoutCh:
			glog.Infof("Stop watcher by timeout, id: %d", c.id)
			// Send a watch error to client, to make client re-list.
			// see issue #108763 (http://github.com/alibaba/kube-sharding/issues/108763)
			status := kerrors.NewResourceExpired("memkube force client to re-list").Status()
			evt := watch.Event{
				Type:   watch.Error,
				Object: &status,
			}
			c.sendEvent(&watchItem{event: evt})
			c.Stop()
			return
		case event, ok := <-c.input:
			if !ok {
				return
			}
			// only send events newer than resourceVersion
			if c.local || event.version > resourceVersion {
				c.sendEvent(event)
			}
		}
	}
}

// TODO: filter by pred
func (c *cacheWatcher) sendEvent(event *watchItem) {
	// to avoid send on already closed channel c.result ?
	select {
	case <-c.done:
		return
	default:
	}

	select {
	case c.result <- event.event:
	// we don't want to block infinitely on putting to c.result, when c.done is already closed.
	case <-c.done:
	}
}

func (c *cacheWatcher) dostop() {
	c.Lock()
	defer c.Unlock()
	if !c.stopped {
		c.stopped = true
		close(c.done)
		close(c.input)
	}
}

// cacherWatch implements watch.Interface to return a single error
type errWatcher struct {
	result chan watch.Event
}

var _ watch.Interface = &errWatcher{}

func newErrWatcher(err error) *errWatcher {
	// Create an error event
	errEvent := watch.Event{Type: watch.Error}
	switch err := err.(type) {
	case runtime.Object:
		errEvent.Object = err
	case *kerrors.StatusError:
		errEvent.Object = &err.ErrStatus
	default:
		errEvent.Object = &metav1.Status{
			Status:  metav1.StatusFailure,
			Message: err.Error(),
			Reason:  metav1.StatusReasonInternalError,
			Code:    http.StatusInternalServerError,
		}
	}

	// Create a watcher with room for a single event, populate it, and close the channel
	watcher := &errWatcher{result: make(chan watch.Event, 1)}
	watcher.result <- errEvent
	close(watcher.result)

	return watcher
}

// Implements watch.Interface.
func (c *errWatcher) ResultChan() <-chan watch.Event {
	return c.result
}

// Implements watch.Interface.
func (c *errWatcher) Stop() {
	// no-op
}
