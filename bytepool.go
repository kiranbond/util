// Copyright (c) 2017 ZeroStack, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// BytePool is a pool to hold some references to byte slices in memory so they
// are reused instead of being garbage collected. It is an unlimited pool since
// it will keep allocating as long as user keeps calling Get(). However, any
// slices returned to the pool are limited to a maximum that is set when the
// pool is created.

package util

import ()

// BytePool is a pool of []byte slices.
type BytePool struct {
  max  uint32 // maximum size of pool
  size uint32 // size of each allocation
  pool chan []byte
}

// NewBytePool returns a BytePool with "max" pool size and "size" initial
// allocation for each slice.
func NewBytePool(max, size uint32) *BytePool {
  return &BytePool{max: max, size: size, pool: make(chan []byte, max)}
}

// Get returns an slice from the pool or allocates a new slice and returns it.
func (p *BytePool) Get() []byte {
  var item []byte
  select {
  case item = <-p.pool:
  default:
    item = make([]byte, 0, p.size)
  }
  return item
}

// Put returns reference to the pool. Any reference larger than max is let
// go so runtime will garbage collect it.
func (p *BytePool) Put(item []byte) {
  // Make len 0 in case caller didn't take care of it. cap stays same.
  item = item[:0]
  select {
  case p.pool <- item:
  default:
  }
}
