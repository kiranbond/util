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
  "testing"

  "github.com/stretchr/testify/assert"
)

type request struct {
  i int
  s string
}

func TestSyncQueue(t *testing.T) {
  sq := NewSyncQueue(1)
  sq.Push(&request{1, "one"})
  sq.Push(&request{2, "two"})
  sq.Push(&request{3, "three"})
  sq.Push(&request{4, "four"})
  ri := sq.Pop()
  r := ri.(*request)
  assert.Equal(t, r.i, 1)
  assert.Equal(t, r.s, "one")
  ri = sq.Pop()
  r = ri.(*request)
  assert.Equal(t, r.i, 2)
  assert.Equal(t, r.s, "two")

  sq.Push(&request{5, "five"})

  ri = sq.Pop()
  r = ri.(*request)
  assert.Equal(t, r.i, 3)
  assert.Equal(t, r.s, "three")
  ri = sq.Pop()
  r = ri.(*request)
  assert.Equal(t, r.i, 4)
  assert.Equal(t, r.s, "four")
  ri = sq.Pop()
  r = ri.(*request)
  assert.Equal(t, r.i, 5)
  assert.Equal(t, r.s, "five")
}
