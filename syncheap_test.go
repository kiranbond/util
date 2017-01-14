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

package util

import (
  "container/heap"
  "testing"

  "github.com/stretchr/testify/assert"
)

// This example inserts several ints into an SyncInt64Heap, checks the minimum,
// and removes them in order of priority.
func TestHeap(t *testing.T) {
  h := NewSyncInt64Heap()
  heap.Push(h, int64(3))
  heap.Push(h, int64(1))
  heap.Push(h, int64(6))
  heap.Push(h, int64(4))
  heap.Push(h, int64(2))

  var d int64
  d = heap.Pop(h).(int64)
  assert.Equal(t, int64(1), d)
  d = heap.Pop(h).(int64)
  assert.Equal(t, int64(2), d)
  d = heap.Pop(h).(int64)
  assert.Equal(t, int64(3), d)
  d = heap.Pop(h).(int64)
  assert.Equal(t, int64(4), d)
  d = heap.Pop(h).(int64)
  assert.Equal(t, int64(6), d)
}
