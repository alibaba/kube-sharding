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

	"github.com/alibaba/kube-sharding/pkg/memkube/testutils/carbon/spec"
	assert "github.com/stretchr/testify/require"
)

func TestSchemeNew(t *testing.T) {
	assert := assert.New(t)
	scheme := NewResourceScheme(spec.Scheme)
	obj, err := scheme.New(spec.SchemeGroupVersion.WithResource("workernodes"))
	assert.Nil(err)
	_, ok := obj.(*spec.WorkerNode)
	assert.True(ok)
	// singular also works
	obj, err = scheme.New(spec.SchemeGroupVersion.WithResource("workernode"))
	assert.Nil(err)
	_, ok = obj.(*spec.WorkerNode)
	assert.True(ok)
	obj, err = scheme.NewList(spec.SchemeGroupVersion.WithResource("workernodes"))
	assert.Nil(err)
	_, ok = obj.(*spec.WorkerNodeList)
	assert.True(ok)

	_, err = scheme.New(spec.SchemeGroupVersion.WithResource("unknown"))
	assert.NotNil(err)
}
