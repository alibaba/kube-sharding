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

package config

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	assert "github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

type sampleConf struct {
	Name string `json:"name"`
}

func Test_kubeConfig(t *testing.T) {
	assert := assert.New(t)
	ns := "ns"
	fakekubeclient := fake.NewSimpleClientset([]runtime.Object{}...)
	factory := NewKubeConfigerFactory(fakekubeclient, ns, map[string]string{})
	name := "c1"
	configer := factory.Configer(name)
	var conf sampleConf
	// not exists
	key := "k1"
	assert.NotNil(configer.Get(key, &conf))
	var configMap = corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Data: map[string]string{},
	}
	assert.Nil(factory.informer().Informer().GetIndexer().Add(&configMap))
	// not found the key
	err := configer.Get(key, &conf)
	assert.NotNil(err)
	fmt.Printf("get key error: %s\n", err.Error())
	assert.True(strings.HasPrefix(err.Error(), "not found config"))
	// success
	bs, _ := json.Marshal(&sampleConf{Name: "alice"})
	configMap.Data = map[string]string{
		key: string(bs),
	}
	assert.Nil(factory.informer().Informer().GetIndexer().Update(&configMap))
	assert.Nil(configer.Get(key, &conf))
	assert.Equal("alice", conf.Name)
	val, err := configer.GetString(key)
	assert.Nil(err)
	assert.Equal("{\"name\":\"alice\"}", val)
	val, err = configer.GetString("not exist")
	assert.Equal(err, NotFoundError)
	assert.Equal(val, "")
}
