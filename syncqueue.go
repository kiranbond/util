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
// SyncQueue is a simple thread-safe FIFO queue. It can be given a starting
// size for the circular buffer but can grow dynamically on demand.

package util

import "sync"

// SyncQueue provides a thread-safe dynamically growing queue.
type SyncQueue struct {
  items []interface{}
  size  int
  head  int
  tail  int
  count int
  mu    sync.Mutex
}

// NewSyncQueue initialized a SyncQueue with a starting size and returns it.
func NewSyncQueue(size int) *SyncQueue {
  return &SyncQueue{items: make([]interface{}, size), size: size}
}

// Push adds an item into the SyncQueue.
func (sq *SyncQueue) Push(element interface{}) {
  sq.mu.Lock()
  defer sq.mu.Unlock()

  if sq.head == sq.tail && sq.count > 0 {
    // If we are full then resize the queue by size.
    items := make([]interface{}, len(sq.items)+sq.size)
    copy(items, sq.items[sq.head:])
    copy(items[len(sq.items)-sq.head:], sq.items[:sq.head])
    sq.head = 0
    sq.tail = len(sq.items)
    sq.items = items
  }
  sq.items[sq.tail] = element
  sq.tail = (sq.tail + 1) % len(sq.items)
  sq.count++
}

// Pop removes and returns a node from the queue in first to last order.
func (sq *SyncQueue) Pop() interface{} {
  sq.mu.Lock()
  defer sq.mu.Unlock()

  if sq.count == 0 {
    return nil
  }
  element := sq.items[sq.head]
  sq.head = (sq.head + 1) % len(sq.items)
  sq.count--
  return element
}
