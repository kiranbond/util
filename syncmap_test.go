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
  "strconv"
  "sync"
  "testing"

  "github.com/stretchr/testify/assert"
)

type data struct {
  i int
  s string
}

func TestSyncMapCmpFunc(t *testing.T) {
  sm := NewSyncMap()
  sm.Add(1, &data{5, "five"})
  sm.Add(5, &data{1, "one"})
  sm.Add(4, &data{2, "two"})
  sm.Add(2, &data{4, "four"})
  sm.Add(3, &data{3, "three"})

  minIntFunc := func(a interface{}, b interface{}) bool {
    return a.(*data).i < b.(*data).i
  }

  keyInf, valInf := sm.CmpFunc(minIntFunc)
  key := keyInf.(int)
  val := valInf.(*data)
  assert.Equal(t, key, 5)
  assert.Equal(t, val.i, 1)
}

// makeTestSyncMap is a helper function to create a sync map
// with test data. It populates SyncMap with n entries in parallel.
func makeTestSyncMap(n int) (sm *SyncMap) {
  var wg sync.WaitGroup
  sm = NewSyncMap()

  wg.Add(n)
  for i := 0; i < n; i++ {
    go func(ii int) {
      sm.Add(ii, &data{ii, strconv.Itoa(ii)})
      wg.Done()
    }(i)
  }

  wg.Wait()
  return sm
}

// TestSynMapGet tests parallel Get
func TestSyncMapGet(t *testing.T) {
  var wg sync.WaitGroup
  n := 1000

  sm := makeTestSyncMap(n)
  assert.Equal(t, sm.Len(), n)

  wg.Add(n)
  for i := 0; i < n; i++ {
    go func(ii int) {
      val, _ := sm.Get(ii)
      assert.Equal(t, ii, val.(*data).i)
      wg.Done()
    }(i)
  }

  wg.Wait()
}

// TestSyncMapDelete tests parallel Delete
func TestSyncMapDelete(t *testing.T) {
  var wg sync.WaitGroup
  n := 1000

  sm := makeTestSyncMap(n)
  wg.Add(n)
  for i := 0; i < n; i++ {
    go func(ii int) {
      val, _ := sm.Del(ii)
      assert.Equal(t, ii, val.(*data).i)
      wg.Done()
    }(i)
  }

  wg.Wait()
  assert.Equal(t, sm.Len(), 0)
}

// TestSyncMapClear tests Clear
func TestSyncMapClear(t *testing.T) {
  n := 1000

  sm := makeTestSyncMap(n)
  sm.Clear()
  assert.Equal(t, sm.Len(), 0)
}

// TestSyncMapFindFunc tests FindFunc
func TestSyncMapFindFunc(t *testing.T) {
  var wg sync.WaitGroup
  n := 1000
  sm := makeTestSyncMap(n)

  wg.Add(n)
  for i := 0; i < n; i++ {
    go func(ii int) {
      findFn := func(k interface{}, v interface{}) bool {
        return v.(*data).i == ii
      }
      key, val := sm.FindFunc(findFn)
      assert.Equal(t, ii, val.(*data).i)
      assert.Equal(t, key, ii)
      wg.Done()
    }(i)
  }

  wg.Wait()
}

// TestSyncMapUpdate tests concurrent add/update
func TestSyncMapUpdate(t *testing.T) {
  var wg sync.WaitGroup
  sm := NewSyncMap()
  n := 1000

  wg.Add(2 * n)

  for i := 0; i < n; i++ {
    go func(ii int) {
      sm.Add(ii, &data{ii, strconv.Itoa(ii)})
      wg.Done()
    }(i)
    go func(ii int) {
      // here we try to double the value contained at i
      sm.Add(ii, &data{2 * ii, strconv.Itoa(2 * ii)})
      wg.Done()
    }(i)
  }
  wg.Wait()
  assert.Equal(t, sm.Len(), n)
  for i := 0; i < n; i++ {
    val, _ := sm.Get(i)
    assert.Equal(t, val.(*data).i == i || val.(*data).i == 2*i, true)
  }
}
