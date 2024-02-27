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

package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"reflect"

	"github.com/alibaba/kube-sharding/common/config"
	"github.com/spf13/pflag"
	glog "k8s.io/klog"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	clientset "k8s.io/client-go/kubernetes"
)

var (
	configer           config.Configer
	defaultNamespace   = ""
	globalKubeConfName = "hippo-c2-system/c2-global-config"
)

// all config keys
var (
	ConfStarAgentSidecar = "staragent-sidecar"
	ConfInterposeObject  = "interpose"
	ConfSkylineAPI       = "skyline-api"
	ConfSkylineDefault   = "skyline-default"
	// in corev1.Container format
	ConfSkylineInitContainer  = "skyline-container-template"
	ConfFiberBoxIdMapping     = "fiber-box-id-mapping"
	ConfFiberBox              = "fiber-box-config"
	ConfPoolMetaCopy          = "pool-meta-copy"
	ConfFiberZkRoot           = "fiber-zk-root"
	ConfServerlessTweaks      = "serverless-tweaks"
	ConfRollingsetTweaks      = "rollingset-tweaks"
	ConfProxyCopyLabels       = "proxy-copy-labels"
	ConfVipserverEnvInterpose = "vipserver-env-interpose"
	ConfNonVersionedKeys      = "non-versioned-keys"
)

// InitFlags is for explicitly initializing the flags.
func InitFlags(flagset *pflag.FlagSet) {
	flagset.StringVar(&globalKubeConfName, "global-conf", globalKubeConfName, "global configmap conf name ($namespace/$name)")
}

// InitGlobalKubeConfiger initializes the configer
func InitGlobalKubeConfiger(namespace string, kubeClient clientset.Interface, stopCh <-chan struct{}) error {
	conf := strings.Split(globalKubeConfName, "/")
	if len(conf) != 2 || conf[0] == "" || conf[1] == "" {
		return fmt.Errorf("invalid config, it should be $namespace/$config")
	}
	glog.Infof("Init global config: %s", globalKubeConfName)
	factory := config.NewKubeConfigerFactory(kubeClient, conf[0], nil)
	factory.Start(stopCh)
	SetGlobalConfiger(factory.Configer(conf[1]), namespace)
	return nil
}

// SetGlobalConfiger for test only
func SetGlobalConfiger(c config.Configer, ns string) {
	configer = c
	defaultNamespace = ns
}

// GetGlobalConfigVal get config value, `namespace` is optional
func GetGlobalConfigVal(confKey, namespace string) (string, error) {
	if configer == nil {
		return "", nil
	}
	if namespace == "" && defaultNamespace != "" {
		namespace = defaultNamespace
	}
	if namespace != "" {
		val, err := configer.GetString(confKey + "_" + namespace)
		if (err != nil && err != config.NotFoundError && !errors.IsNotFound(err)) || val != "" {
			return val, err
		}
	}
	val, err := configer.GetString(confKey)
	if err != nil && (err == config.NotFoundError || errors.IsNotFound(err)) {
		return "", nil
	}
	return val, err
}

// GetGlobalConfigJSON get json config
func GetGlobalConfigJSON(confKey, namespace string, obj interface{}) error {
	s, err := GetGlobalConfigVal(confKey, namespace)
	if err != nil {
		return err
	}
	if s == "" {
		return fmt.Errorf("empty string for key: %s", confKey)
	}
	return json.Unmarshal([]byte(s), obj)
}

// GetGlobalConfigValForObj get interposed object config
func GetGlobalConfigValForObj(confKey string, kind string, object metav1.Object) (string, error) {
	val, err := GetGlobalConfigVal(confKey, "")
	if err != nil || val == "" {
		return val, err
	}
	vi := reflect.ValueOf(object)
	if vi.IsNil() {
		return val, err
	}
	var configs = []*Config{}
	err = json.Unmarshal([]byte(val), &configs)
	if err != nil {
		return val, err
	}
	var matchedConfig *Config
	for i := range configs {
		if configs[i].MatchRule.match(kind, object) {
			if matchedConfig != nil {
				if configs[i].MatchRule.betterThan(&matchedConfig.MatchRule) {
					matchedConfig = configs[i]
				}
			} else {
				matchedConfig = configs[i]
			}
		}
	}
	if matchedConfig != nil {
		return string(matchedConfig.Data), nil
	}
	return "", nil
}

type Config struct {
	MatchRule MatchRule       `json:"matchRule,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
}

type MatchRule struct {
	LabelSelector *metav1.LabelSelector `json:"labelSelector,omitempty"`
	Namespaces    []string              `json:"namespaces,omitempty"`
	Kind          string                `json:"kind,omitempty"`
}

func (m *MatchRule) betterThan(rule *MatchRule) bool {
	if m.Kind != "" && rule.Kind == "" {
		return true
	}
	if m.Kind == "" && rule.Kind != "" {
		return false
	}
	if len(m.Namespaces) > len(rule.Namespaces) {
		return true
	}
	if len(m.Namespaces) < len(rule.Namespaces) {
		return false
	}
	if m.LabelSelector != nil && rule.LabelSelector == nil {
		return true
	}
	if m.LabelSelector == nil && rule.LabelSelector != nil {
		return true
	}
	if m.LabelSelector != nil && rule.LabelSelector != nil {
		if len(m.LabelSelector.MatchExpressions) > len(rule.LabelSelector.MatchExpressions) ||
			len(m.LabelSelector.MatchLabels) > len(rule.LabelSelector.MatchLabels) {
			return true
		}
	}
	return false
}

func (m *MatchRule) match(kind string, object metav1.Object) bool {
	if m.Kind != "" {
		if m.Kind != kind {
			return false
		}
	}
	if len(m.Namespaces) != 0 {
		var matched bool
		for i := range m.Namespaces {
			if m.Namespaces[i] == object.GetNamespace() {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if m.LabelSelector != nil {
		selector, err := metav1.LabelSelectorAsSelector(m.LabelSelector)
		if err != nil {
			glog.Errorf("get selector error %s, %v", m.LabelSelector.String(), err)
			return false
		}
		if !selector.Matches(labels.Set(object.GetLabels())) {
			return false
		}
	}
	return true
}
