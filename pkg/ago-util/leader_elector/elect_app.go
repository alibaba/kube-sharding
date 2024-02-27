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

package elector

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/alibaba/kube-sharding/pkg/ago-util/utils"
	glog "k8s.io/klog"
)

// SpecSep is the seperator for service spec.
const SpecSep = "\n"

func newAppElector(url string, port int) (Elector, string, error) {
	if url == "" {
		return &NopElector{}, "", nil
	}
	opts, err := ParseElectOptions(url)
	if err != nil {
		return nil, "", err
	}
	opts.Key = filepath.Join(opts.Key, "LeaderElection", "leader_election0000000000")
	opts.ConnTimeout = 10
	opts.LostTimeout = 10
	ip, err := utils.GetLocalIP()
	if err != nil {
		return nil, "", err
	}
	spec := fmt.Sprintf("%s:%d", ip, port)
	opts.Val = strings.Join([]string{"_placeholder_", spec}, SpecSep)
	glog.Infof("new elector: %+v", *opts)
	elector, err := NewElector(opts)
	return elector, spec, err
}

// AppBase is the implemented application interface.
type AppBase interface {
	Start() error
	Recover() error
	Stop()
}

// ElectApp is the base election app to use.
type ElectApp struct {
	elector  Elector
	stopChan chan struct{}
	LostCh   <-chan struct{}
	impl     AppBase
	started  int32
	spec     string
}

// NewElectApp creates a new elect app.
func NewElectApp(urlRoot string, port int, impl AppBase) (*ElectApp, error) {
	elector, spec, err := newAppElector(urlRoot, port)
	if err != nil {
		return nil, err
	}
	return &ElectApp{elector: elector, stopChan: make(chan struct{}), impl: impl, spec: spec}, nil
}

// IsLeader check if it's the leader now.
func (app *ElectApp) IsLeader() bool {
	return app.elector.IsLeader() && atomic.LoadInt32(&app.started) > 0
}

// GetLeader return the leader spec. Call this function only if it's not the leader.
func (app *ElectApp) GetLeader() (string, bool, error) {
	spec, err := app.elector.GetLeader()
	if spec != "" {
		spec = strings.Split(spec, SpecSep)[1]
	}
	return spec, spec == app.spec, err
}

// Elect try to elect, wait until it became the leader or error occurs.
func (app *ElectApp) Elect(sigCh chan os.Signal) error {
	glog.Info("try to elect to leader...")
	quitCh := app.monitorSignal(sigCh)
	lostCh, err := app.elector.Elect(app.stopChan)
	if quitCh != nil {
		quitCh <- struct{}{}
	}
	if err != nil {
		glog.Errorf("elect error: %v", err)
		return err
	}
	glog.Info("elect leader success, start app...")
	app.LostCh = lostCh
	if err = app.impl.Recover(); err != nil {
		return err
	}
	if err = app.impl.Start(); err != nil {
		return err
	}
	atomic.StoreInt32(&app.started, 1)
	return nil
}

// Shutdown shuts the app.
func (app *ElectApp) Shutdown() error {
	glog.Info("shutdown elect app")
	close(app.stopChan)
	if err := app.elector.Close(); err != nil {
		return err
	}
	app.impl.Stop()
	return nil
}

func (app *ElectApp) monitorSignal(sigCh chan os.Signal) chan struct{} {
	if sigCh == nil {
		return nil
	}
	quitCh := make(chan struct{})
	go func() {
		select {
		case <-sigCh:
			glog.Info("os exit by signal")
			app.elector.Close()
			// see Locker impl, i have no place to stop the locking accquire
			os.Exit(-1)
		case <-quitCh:
			glog.Info("elect done, signal elect monitor stopped")
			close(quitCh)
		}
	}()
	return quitCh
}
