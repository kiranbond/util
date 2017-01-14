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
  "reflect"
  "sync"
  "testing"

  "github.com/golang/glog"
  "github.com/stretchr/testify/assert"
)

func TestTryAcquire(t *testing.T) {
  idsl := NewIDSetLocker(reflect.Int)

  var id1, id2, id3, id4, id5, id6 int
  id1, id2, id3, id4, id5, id6 = 1, 2, 3, 4, 5, 6

  ans1 := idsl.tryAcquire(IntToIface([]int{id1, id2}),
    IntToIface([]int{}), nil)
  ans2 := idsl.tryAcquire(IntToIface([]int{id2}),
    IntToIface([]int{id3}), nil)
  ans3 := idsl.tryAcquire(IntToIface([]int{}),
    IntToIface([]int{id4}), nil)
  ans4 := idsl.tryAcquire(IntToIface([]int{id2, id5}),
    IntToIface([]int{id6}), nil)
  assert.True(t, ans1)
  assert.True(t, ans2)
  assert.True(t, ans3)
  assert.True(t, ans4)

  ans5 := idsl.tryAcquire(IntToIface([]int{id6}),
    IntToIface([]int{}), nil)
  assert.False(t, ans5)

  ans6 := idsl.tryAcquire(IntToIface([]int{id1, id2}),
    IntToIface([]int{id3}), nil)
  assert.False(t, ans6)

  // release and check again
  idsl.Release(IntToIface([]int{id1, id2, id6}))
  ans7 := idsl.tryAcquire(IntToIface([]int{id2}),
    IntToIface([]int{id1, id6}), nil)
  assert.True(t, ans7)

  idsl.Release(IntToIface([]int{id1, id2, id3, id4, id5, id6}))
  idsl.Release(IntToIface([]int{id2}))
  idsl.Release(IntToIface([]int{id2}))

  assert.True(t, idsl.IsEmpty())
}

func TestAcquireRelease(t *testing.T) {
  idsl := NewIDSetLocker(reflect.Int)

  var id1, id2, id3, id4, id5, id6 int
  id1, id2, id3, id4, id5, id6 = 1, 2, 3, 4, 5, 6

  idsl.Acquire(IntToIface([]int{id1, id2}),
    IntToIface([]int{id4, id5, id6}))
  idsl.Acquire(IntToIface([]int{id2}),
    IntToIface([]int{id3}))

  go idsl.Release(IntToIface([]int{id2, id6}))
  go idsl.Release(IntToIface([]int{id2, id3}))

  idsl.Acquire(IntToIface([]int{}),
    IntToIface([]int{id2}))
  go idsl.Release(IntToIface([]int{id1, id2, id4, id5}))
  idsl.Acquire(IntToIface([]int{id2}),
    IntToIface([]int{id1, id5}))
  idsl.Release(IntToIface([]int{id1, id2, id5}))

  assert.True(t, idsl.IsEmpty())
}

func TestAcquireReleaseInt64(t *testing.T) {
  idsl := NewIDSetLocker(reflect.Int64)

  var id1, id2, id3, id4, id5, id6 int64
  id1, id2, id3, id4, id5, id6 = 1, 2, 3, 4, 5, 6

  idsl.AcquireInt64([]int64{id1, id2}, []int64{id4, id5, id6})
  idsl.AcquireInt64([]int64{id2}, []int64{id3})

  go idsl.ReleaseInt64([]int64{id2, id6})
  go idsl.ReleaseInt64([]int64{id2, id3})

  idsl.AcquireInt64([]int64{}, []int64{id2})
  go idsl.ReleaseInt64([]int64{id1, id2, id4, id5})
  idsl.AcquireInt64([]int64{id2}, []int64{id1, id5})
  idsl.ReleaseInt64([]int64{id1, id2, id5})

  assert.True(t, idsl.IsEmpty())
}

func TestAcquireReleaseString(t *testing.T) {
  idsl := NewIDSetLocker(reflect.String)

  var id1, id2, id3, id4, id5, id6 string
  id1, id2, id3, id4, id5, id6 = "abc", "def", "ghi", "jkl", "mno", "pqr"

  idsl.AcquireStr([]string{id1, id2}, []string{id4, id5, id6})
  idsl.AcquireStr([]string{id2}, []string{id3})

  go idsl.ReleaseStr([]string{id2, id6})
  go idsl.ReleaseStr([]string{id2, id3})

  idsl.AcquireStr([]string{}, []string{id2})
  go idsl.ReleaseStr([]string{id1, id2, id4, id5})
  idsl.AcquireStr([]string{id2}, []string{id1, id5})
  idsl.ReleaseStr([]string{id1, id2, id5})

  assert.True(t, idsl.IsEmpty())
}

func TestAsynAcquire(t *testing.T) {
  idsl := NewIDSetLocker(reflect.Int)

  var id1, id2, id3, id4, id5, id6 int
  id1, id2, id3, id4, id5, id6 = 1, 2, 3, 4, 5, 6

  idsl.Acquire(IntToIface([]int{id1, id2, id3}),
    IntToIface([]int{id4, id5, id6}))
  done := make(chan bool, 1)
  idsl.AcquireAsync(IntToIface([]int{id1, id2}),
    IntToIface([]int{id4, id5}), done)
  idsl.Release(IntToIface([]int{id4, id5}))
  assert.True(t, <-done)

  idsl.AcquireAsync(IntToIface([]int{id1, id2, id3}),
    IntToIface([]int{id5, id6}), done)
  idsl.Release(IntToIface([]int{id5, id6}))
  assert.True(t, <-done)
}

func TestAcquireAll(t *testing.T) {
  idsl := NewIDSetLocker(reflect.Int64)

  var id1, id2, id3, id4, id5, id6 int64
  id1, id2, id3, id4, id5, id6 = 1, 2, 3, 4, 5, 6

  wg := &sync.WaitGroup{}

  // Make AcquireAll contend with other locking transactions
  loopCount := 2
  wg.Add(4 + loopCount)
  idsl.AcquireInt64([]int64{id1, id2}, []int64{id4, id5, id6})
  glog.Infof("lock acquired on ids %d %d %d %d %d", id1, id2, id4, id5, id6)
  for ii := 0; ii < loopCount; ii++ {
    go func() {
      idsl.AcquireAll()
      glog.Info("all locks acquired")
      idsl.ReleaseAll()
      glog.Info("all locks released")
      wg.Done()
    }()
  }

  go func() {
    idsl.AcquireInt64([]int64{}, []int64{id2})
    glog.Infof("lock acquired on id %d", id2)
    idsl.ReleaseInt64([]int64{id2})
    glog.Infof("lock released on id %d", id2)
    wg.Done()
  }()

  go func() {
    idsl.AcquireInt64([]int64{id2}, []int64{id3, id4})
    glog.Infof("lock acquired on ids %d %d %d", id2, id3, id4)
    idsl.ReleaseInt64([]int64{id2, id3, id4})
    glog.Infof("lock released on ids %d %d %d", id2, id3, id4)
    wg.Done()
  }()
  go func() {
    idsl.ReleaseInt64([]int64{id2, id6})
    glog.Infof("lock released on ids %d %d", id2, id6)
    wg.Done()
  }()
  go func() {
    idsl.ReleaseInt64([]int64{id1, id4, id5})
    glog.Infof("lock released on ids %d %d %d", id1, id4, id5)
    wg.Done()
  }()
  glog.Infof("trying to lock ids %d %d %d", id1, id4, id6)
  idsl.AcquireInt64([]int64{id1}, []int64{id4, id6})
  glog.Infof("lock acquired on ids %d %d %d", id1, id4, id6)
  idsl.ReleaseInt64([]int64{id1, id4, id6})
  glog.Infof("lock released on ids %d %d %d", id1, id4, id6)
  wg.Wait()

  // Make multiple AcquireAll contend with each other
  loopCount = 5
  wg.Add(loopCount)
  for ii := 0; ii < loopCount; ii++ {
    go func() {
      idsl.AcquireAll()
      glog.Info("all locks acquired")
      idsl.ReleaseAll()
      glog.Info("all locks released")
      wg.Done()
    }()
  }
  wg.Wait()
  assert.True(t, idsl.IsEmpty())
}
