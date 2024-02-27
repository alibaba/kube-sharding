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

package set

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntSet(t *testing.T) {
	set := NewIntSet()
	vals := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for i := range vals {
		set.Insert(vals[i])
		set.Insert(vals[i])
		set.Insert(vals[i])
		set.Insert(vals[i])
		set.Insert(vals[i])
	}

	assert.Equal(t, len(set), len(vals))
	for i := range vals {
		assert.Equal(t, set.Exist(vals[i]), true)
	}

	assert.Equal(t, set.Exist(11), false)
}

func TestInt64Set(t *testing.T) {
	set := NewInt64Set()
	vals := []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	for i := range vals {
		set.Insert(vals[i])
		set.Insert(vals[i])
		set.Insert(vals[i])
		set.Insert(vals[i])
		set.Insert(vals[i])
	}

	assert.Equal(t, len(set), len(vals))
	for i := range vals {
		assert.Equal(t, set.Exist(vals[i]), true)
	}

	assert.Equal(t, set.Exist(11), false)
}
