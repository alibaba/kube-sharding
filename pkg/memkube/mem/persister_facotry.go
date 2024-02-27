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

	"github.com/alibaba/kube-sharding/pkg/ago-util/fstorage"
	"k8s.io/apimachinery/pkg/runtime/schema"
	glog "k8s.io/klog"
)

const (
	//TouchName new start touch file name
	TouchName = ".meta"
)

// NamespacePersisterFactory persister factory
type NamespacePersisterFactory struct {
	url       string
	backupURL string
	namespace string
}

// NewNamespacePersisterFactory new fac
func NewNamespacePersisterFactory(url, backupURL, namespace string) (*NamespacePersisterFactory, error) {
	return &NamespacePersisterFactory{url: url, backupURL: backupURL, namespace: namespace}, nil
}

// Check check url vaild
func (f *NamespacePersisterFactory) Check() error {
	path := fstorage.JoinPath(f.url, f.namespace)
	fsl, err := fstorage.Create(path)
	if err != nil {
		glog.Errorf("Create fstorage from path error: %s, %v", path, err)
		return err
	}
	defer fsl.Close()

	fs, err := fsl.New(TouchName)
	if err != nil {
		return err
	}
	defer fs.Close()

	exists, err := fs.Exists()
	if err != nil {
		glog.Errorf("Exists at namespace error: %s, %s, %v", f.url, f.namespace, err)
		return err
	}
	if !exists {
		return fmt.Errorf("start error url:%s/%s is not exist .meta", f.url, f.namespace)
	}
	return nil
}

// NewFsPersister new FsPersister
func (f *NamespacePersisterFactory) NewFsPersister(gvk schema.GroupVersionKind, matcher TraitsMatcher) (Persister, error) {
	return NewFsPersister(f.url, f.backupURL, f.namespace, gvk, matcher)
}
