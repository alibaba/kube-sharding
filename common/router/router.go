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

package router

import (
	"errors"
	"fmt"
	"hash/fnv"
	"regexp"
	"sort"
	"strings"

	"github.com/alibaba/kube-sharding/common"

	"github.com/alibaba/kube-sharding/common/config"
	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	carbonv1 "github.com/alibaba/kube-sharding/pkg/apis/carbon/v1"
	glog "k8s.io/klog"
)

// Router routes a GroupPlan request from app to a kubernetes namespace.
type Router interface {
	GetNamespace(appName string) (string, error)
	GetFiberId(app, az, region, stage, unit string) (string, error)
}

var NoDefaultError = errors.New("no default route configed")

type simpleRouter struct{}

// NewSimpleRouter creates a simple router.
func NewSimpleRouter() Router {
	return &simpleRouter{}
}

func (r *simpleRouter) GetFiberId(app, az, region, stage, unit string) (string, error) {
	return app, nil
}

func (r *simpleRouter) GetNamespace(appName string) (string, error) {
	namespace := r.escapeName(appName)
	if len(namespace) > carbonv1.MaxNamespaceLength {
		namespace, _ = utils.SignatureWithMD5(namespace)
	}
	return namespace, nil
}

var namespaceReg = regexp.MustCompile(`[^a-z0-9-]`)

func (r *simpleRouter) escapeName(namespace string) string {
	namespace = strings.ToLower(namespace)
	result := namespaceReg.ReplaceAllString(namespace, "-")
	for strings.HasPrefix(result, "-") {
		result = strings.TrimPrefix(result, "-")
	}
	for strings.HasSuffix(result, "-") {
		result = strings.TrimSuffix(result, "-")
	}
	return result
}

type routeConfig struct {
	// {namespace, []appNames}
	Specifics map[string][]string `json:"specifics,omitempty"`
	// default namespace list
	// NOTE: do NOT change this after first startup.
	Defaults []string `json:"defaults,omitempty"`
	// {appName, namespace}
	index map[string]string
}

func newRouteConfig() *routeConfig {
	return &routeConfig{
		index: make(map[string]string),
	}
}

func (c *routeConfig) init() error {
	for ns, list := range c.Specifics {
		for _, name := range list {
			if old, ok := c.index[name]; ok {
				return fmt.Errorf("conflict namespace for app: %s (%s/%s)", name, ns, old)
			}
			c.index[name] = ns
		}
	}
	return nil
}

func (c *routeConfig) match(appName string) (string, error) {
	if ns, ok := c.matchSpecifies(appName); ok {
		return ns, nil
	}
	if len(c.Defaults) == 0 {
		return "", NoDefaultError
	}
	h := c.hash(appName)
	return c.Defaults[h%uint32(len(c.Defaults))], nil
}

// 多个匹配项情况下完全匹配优先，正则匹配按长度排序优先
func (c *routeConfig) matchSpecifies(appName string) (string, bool) {
	ns, ok := c.index[appName]
	if ok {
		return ns, true
	}
	var sortedIndex = make([]string, 0, len(c.index))
	for k := range c.index {
		sortedIndex = append(sortedIndex, k)
	}
	sort.Slice(sortedIndex, func(i, j int) bool {
		if len(sortedIndex[i]) == len(sortedIndex[j]) {
			return sortedIndex[i] > sortedIndex[j]
		}
		return len(sortedIndex[i]) > len(sortedIndex[j])
	})
	for i := range sortedIndex {
		p := sortedIndex[i]
		reg, err := regexp.Compile(p)
		if err != nil {
			glog.Errorf("Compile route config error: %s, %v", p, err)
		} else {
			if reg.MatchString(appName) {
				ns := c.index[p]
				return ns, true
			}
		}
	}
	return "", false
}

func (c *routeConfig) hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

const (
	// RouteConfigKey the config key name
	RouteConfigKey           = "namespace-router"
	FiberRouteConfigKey      = "fiber-router"
	FiberStageRouteConfigKey = "fiber-stage-router"
)

type configRouter struct {
	loader     config.Configer
	globalMode bool
}

// NewConfigRouter creates a config router which loads routing from configs.
func NewConfigRouter(loader config.Configer) Router {
	return &configRouter{
		loader:     loader,
		globalMode: false,
	}
}

func NewGlobalConfigRouter() Router {
	return &configRouter{globalMode: true}
}

func (r *configRouter) GetFiberId(app, az, region, stage, unit string) (string, error) {
	if app == "" || stage == "" {
		return "", fmt.Errorf("missing appName:%s or stage:%s", app, stage)
	}
	fiberId, err := r.route(FiberStageRouteConfigKey, stage)
	if fiberId != "" || (err != nil && err != NoDefaultError) {
		return fiberId, err
	}
	return r.route(FiberRouteConfigKey, app)
}

func (r *configRouter) GetNamespace(appName string) (string, error) {
	return r.route(RouteConfigKey, appName)
}

func (r *configRouter) route(configKey, name string) (string, error) {
	routeConfig := newRouteConfig()
	var err error
	if r.globalMode {
		err = common.GetGlobalConfigJSON(configKey, "", routeConfig)
	} else {
		err = r.loader.Get(configKey, routeConfig)
	}
	if err != nil {
		return "", fmt.Errorf("get route config failed: %v", err)
	}
	if err = routeConfig.init(); err != nil {
		return "", err
	}
	return routeConfig.match(name)
}
