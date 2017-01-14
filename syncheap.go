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
// Simple min-heap of integers made thread-safe from golang examples.
// http://golang.org/pkg/container/heap/
// Look at the test for example usage.
//
// NOTE: heap will panic if you Pop an empty heap. Call Len() first to check.

package util

import (
  "container/heap"
  "sync"
)

// SyncInt64Heap is a min-heap of int64s.
type SyncInt64Heap struct {
  data []int64
  mu   sync.RWMutex
}

// NewSyncInt64Heap creates and initializes an empty heap of int64s.
func NewSyncInt64Heap() *SyncInt64Heap {
  h := &SyncInt64Heap{}
  heap.Init(h)
  return h
}

// Len returns the size of the heap contents.
func (h *SyncInt64Heap) Len() int {
  h.mu.RLock()
  defer h.mu.RUnlock()
  return len(h.data)
}

// Functions below are for internal use by container/heap.
// User should use heap.Push(h, i) and heap.Pop(h, i)

// Less is needed by heap interface to manage the heap.
func (h *SyncInt64Heap) Less(i, j int) bool {
  h.mu.RLock()
  defer h.mu.RUnlock()
  return h.data[i] < h.data[j]
}

// Swap is needed by heap interface to manage the heap.
func (h *SyncInt64Heap) Swap(i, j int) {
  h.mu.Lock()
  defer h.mu.Unlock()
  h.data[i], h.data[j] = h.data[j], h.data[i]
}

// Push is for internal use by container/heap. User should call heap.Push(h, i).
func (h *SyncInt64Heap) Push(x interface{}) {
  h.mu.Lock()
  defer h.mu.Unlock()
  // Push and Pop use pointer receivers because they modify the slice's length,
  // not just its contents.
  h.data = append(h.data, x.(int64))
}

// Pop is for internal use by container/heap. User should call heap.Pop(h)
func (h *SyncInt64Heap) Pop() interface{} {
  h.mu.Lock()
  defer h.mu.Unlock()
  old := h.data
  n := len(old)
  if n == 0 {
    return nil
  }
  item := old[n-1]
  h.data = old[0 : n-1]
  return item
}

//////////////////////////////////////////////////////////////////////////////

// SyncUint64Heap is a min-heap of int64s.
type SyncUint64Heap struct {
  data []uint64
  mu   sync.RWMutex
}

// NewSyncUint64Heap creates and initializes an empty heap of int64s.
func NewSyncUint64Heap() *SyncUint64Heap {
  h := &SyncUint64Heap{}
  heap.Init(h)
  return h
}

// Len returns the size of the heap contents.
func (h *SyncUint64Heap) Len() int {
  h.mu.RLock()
  defer h.mu.RUnlock()
  return len(h.data)
}

// Functions below are for internal use by container/heap.
// User should use heap.Push(h, i) and heap.Pop(h, i)

// Less is needed by heap interface to manage the heap.
func (h *SyncUint64Heap) Less(i, j int) bool {
  h.mu.RLock()
  defer h.mu.RUnlock()
  return h.data[i] < h.data[j]
}

// Swap is needed by heap interface to manage the heap.
func (h *SyncUint64Heap) Swap(i, j int) {
  h.mu.Lock()
  defer h.mu.Unlock()
  h.data[i], h.data[j] = h.data[j], h.data[i]
}

// Push is for internal use by container/heap. User should call heap.Push(h, i).
func (h *SyncUint64Heap) Push(x interface{}) {
  h.mu.Lock()
  defer h.mu.Unlock()
  // Push and Pop use pointer receivers because they modify the slice's length,
  // not just its contents.
  h.data = append(h.data, x.(uint64))
}

// Pop is for internal use by container/heap. User should call heap.Pop(h)
func (h *SyncUint64Heap) Pop() interface{} {
  h.mu.Lock()
  defer h.mu.Unlock()
  old := h.data
  n := len(old)
  if n == 0 {
    return nil
  }
  item := old[n-1]
  h.data = old[0 : n-1]
  return item
}
