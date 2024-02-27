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

package util

import (
	"fmt"
)

type BitMap struct {
	bits []byte
	max  int
}

func NewBitMap(max int) *BitMap {
	bits := make([]byte, (max>>3)+1)
	return &BitMap{bits: bits, max: max}
}

func (b *BitMap) Add(num uint) {
	index := num >> 3
	pos := num & 0x07
	b.bits[index] |= 1 << pos
}

func (b *BitMap) IsExist(num uint) bool {
	index := num >> 3
	pos := num & 0x07
	return b.bits[index]&(1<<pos) != 0
}

func (b *BitMap) Remove(num uint) {
	index := num >> 3
	pos := num & 0x07
	b.bits[index] = b.bits[index] & ^(1 << pos)
}

func (b *BitMap) Max() int {
	return b.max
}

func (b *BitMap) String() string {
	return fmt.Sprint(b.bits)
}
