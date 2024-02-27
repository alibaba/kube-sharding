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

package utils

import (
	"encoding/json"
	"time"
)

// DirtyObject tracks its state which expected to persist. When the state changed, it's dirty,
// and after it's committed, it's not dirty.
type DirtyObject struct {
	UpdateAt int64
	CommitAt int64
	parent   *DirtyObject
}

// NewDirtyObject creates a new dirty object.
func NewDirtyObject() *DirtyObject {
	return &DirtyObject{}
}

// SetDirty marks this object dirty.
func (d *DirtyObject) SetDirty() {
	if d.parent != nil {
		d.GetRoot().SetDirty()
	} else {
		d.UpdateAt = time.Now().UnixNano() / 1000
	}
}

// IsDirty check if this object is dirty.
func (d *DirtyObject) IsDirty() bool {
	if d.parent != nil {
		return d.GetRoot().IsDirty()
	}
	return d.UpdateAt != d.CommitAt
}

// TransactDirty transact commit if you want to persist dirty object state into db or somewhere...
func (d *DirtyObject) TransactDirty(fn func() error) error {
	if d.parent != nil {
		return d.GetRoot().TransactDirty(fn)
	}
	old := d.CommitAt
	d.CommitAt = d.UpdateAt
	if err := fn(); err != nil {
		d.CommitAt = old
		return err
	}
	return nil
}

// LinkDirty links 2 dirty objects.
func (d *DirtyObject) LinkDirty(child *DirtyObject) {
	child.parent = d
}

// Stringify for logging.
func (d *DirtyObject) Stringify() string {
	if d.parent != nil {
		return d.GetRoot().Stringify()
	}
	b, _ := json.Marshal(d)
	return string(b)
}

// GetRoot get the root dirty object.
func (d *DirtyObject) GetRoot() *DirtyObject {
	var o *DirtyObject
	for o = d; o.parent != nil; o = o.parent {
	}
	return o
}
