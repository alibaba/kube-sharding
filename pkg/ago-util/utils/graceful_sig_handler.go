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
	"context"
	"fmt"
	stdLog "log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// GracefulSigHandler GracefulSigHandler
type GracefulSigHandler struct {
	cancel context.CancelFunc
	ctx    context.Context
}

// NewGracefulSigHandler NewGracefulSigHandler
func NewGracefulSigHandler(pctx context.Context) (*GracefulSigHandler, context.Context) {
	var g = &GracefulSigHandler{}
	return g, g.init(pctx)
}

func (g *GracefulSigHandler) init(pctx context.Context) context.Context {
	g.ctx, g.cancel = context.WithCancel(pctx)
	return g.ctx
}

// NotifySigs NotifySigs
func (g *GracefulSigHandler) NotifySigs(signals ...os.Signal) {
	sigs := make(chan os.Signal, 1)
	// `signal.Notify` registers the given channel to
	// receive notifications of the specified signals.
	if nil == signals {
		signals = []os.Signal{}
	}
	signal.Notify(sigs, append(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)...)
	for i := range signals {
		signal.Notify(sigs, signals[i])
	}
	// This goroutine executes a blocking receive for
	// signals. When it gets one it'll print it out
	// and then notify the program that it can finish.
	go func() {
		sig := <-sigs
		stdLog.Printf("catch sig %s", sig)
		g.cancel()
	}()
}

func (g *GracefulSigHandler) fakeSigs() {
	g.cancel()
}

// Wait Wait
func (g *GracefulSigHandler) Wait(maxWait, blockingTime time.Duration, closeAllRoutine func()) error {
	select {
	case <-g.ctx.Done():
		stdLog.Printf("wait for %s to exit", maxWait)
		select {
		// 等待关闭所有协程完成
		case <-func() chan struct{} {
			var flagChan = make(chan struct{}, 1)
			go func() {
				time.Sleep(blockingTime)
				if nil != closeAllRoutine {
					closeAllRoutine()
					flagChan <- struct{}{}
				}
			}()
			return flagChan
		}():
			stdLog.Printf("wait All Routine Done")
			return nil
		// 或者超时
		case <-time.After(maxWait):
			stdLog.Printf("already wait for %s to exit", maxWait)
			return fmt.Errorf("force shutdown after %v", maxWait)
		}
	}
}
