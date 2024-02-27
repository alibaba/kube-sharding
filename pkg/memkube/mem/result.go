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
	"errors"
	"math"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
)

var (
	// ErrWaitTimeout wait operation timeout
	ErrWaitTimeout = errors.New("wait the operation result timeout")
)

const (
	maxVersion = math.MaxUint64
)

// OpResult represents store operation results.
type OpResult interface {
	Wait(timeout time.Duration) error
}

// WriteResult is write operation result.
type WriteResult interface {
	OpResult
	Object() runtime.Object
}

// ReadResult for read operations.
type ReadResult interface {
	OpResult
	List() []runtime.Object
}

type opResult struct {
	interval time.Duration
	condFunc func() (bool, error)
}

func (r *opResult) Wait(timeout time.Duration) error {
	err := wait.PollImmediate(r.interval, timeout, r.condFunc)
	if err == wait.ErrWaitTimeout {
		return ErrWaitTimeout
	}
	return err
}

type writeResult struct {
	opResult
	pending *pendingObject
}

func (r *writeResult) Object() runtime.Object {
	return r.pending.GetCurrent()
}

func newWriteResult(gr schema.GroupResource, interval time.Duration, pending *pendingObject, ver uint64) WriteResult {
	return &writeResult{
		opResult: opResult{
			interval: interval,
			// If the object committed, which means the internal object resource version must be greater than the expected version.
			condFunc: func() (bool, error) {
				if ver == maxVersion { // to deletion
					return !pending.IsPending(), nil
				}
				if pending.Deleted() {
					// Delete the object while we're still wait some update requests committed.
					return false, kerrors.NewNotFound(gr, pending.name)
				}
				obj := pending.GetCurrent()
				if obj == nil { // the add operation not completed
					return false, nil
				}
				curVer, err := MetaResourceVersion(obj)
				if err != nil {
					return false, err
				}
				return curVer >= ver, nil
			},
		},
		pending: pending,
	}
}

type readResult struct {
	opResult
	pendings []*pendingObject
}

func newReadResult(interval time.Duration, pendings []*pendingObject) ReadResult {
	r := &readResult{
		opResult: opResult{
			interval: interval,
			condFunc: func() (bool, error) {
				for _, pending := range pendings {
					if nil == pending.GetCurrent() {
						return false, nil
					}
				}
				return true, nil

			},
		},
		pendings: pendings,
	}
	return r
}

func (r *readResult) List() []runtime.Object {
	objs := make([]runtime.Object, 0)
	for _, pending := range r.pendings {
		objs = append(objs, pending.GetCurrent())
	}
	return objs
}
