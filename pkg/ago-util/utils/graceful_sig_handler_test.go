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
	"syscall"
	"testing"
	"time"

	"context"

	"github.com/stretchr/testify/assert"
)

func TestGracefulHandleSig(t *testing.T) {
	h, _ := NewGracefulSigHandler(context.Background())
	h.NotifySigs(syscall.Signal(10))
	go func() {
		h.fakeSigs()
	}()

	err := h.Wait(time.Second*1, 0, func() {
		time.Sleep(time.Second * 4)
	})
	assert.NotNil(t, err)

	h1, _ := NewGracefulSigHandler(context.Background())
	h1.NotifySigs(syscall.Signal(10))
	go func() {
		h1.fakeSigs()
	}()

	err1 := h1.Wait(time.Second*3, 0, func() {
		time.Sleep(time.Second * 1)
	})
	assert.Nil(t, err1)

	h2, _ := NewGracefulSigHandler(context.Background())
	h2.NotifySigs(syscall.Signal(10))
	go func() {
		h2.fakeSigs()
	}()

	err2 := h2.Wait(0, 0, func() {
		time.Sleep(time.Second * 1)
	})
	assert.NotNil(t, err2)

	h3, _ := NewGracefulSigHandler(context.Background())
	h3.NotifySigs(syscall.Signal(10))
	go func() {
		h3.fakeSigs()
	}()

	err3 := h3.Wait(time.Second, 0, nil)
	assert.NotNil(t, err3)
}
