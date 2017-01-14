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
  "fmt"
  "strings"
  "sync"
  "sync/atomic"
  "testing"
  "time"

  "github.com/stretchr/testify/assert"
)

func TestControllerBasic(t *testing.T) {
  ctl := NewController()
  errCanStop := ctl.CanStop()
  assert.Error(t, errCanStop)
  errSuspend := ctl.Suspend(1)
  assert.Error(t, errSuspend)
  errResume := ctl.Resume(1)
  assert.Error(t, errResume)
  token, errToken := ctl.NewToken(true /*checkStarted*/)
  assert.Error(t, errToken)
  token, errToken = ctl.NewToken(false /*checkStarted*/)
  assert.NoError(t, errToken)
  ctl.CloseToken(token)

  errCanStart := ctl.CanStart()
  assert.NoError(t, errCanStart)
  ctl.SetStarted(true)

  token, errToken = ctl.NewToken(true /*checkStarted*/)
  assert.NoError(t, errToken)
  errSuspend = ctl.Suspend(token)
  assert.NoError(t, errSuspend)
  errResume = ctl.Resume(token)
  assert.NoError(t, errResume)
  ctl.CloseToken(token)
  errCanStop = ctl.CanStop()
  assert.NoError(t, errCanStop)
  ctl.SetStopped(true)

  errClose := ctl.Close()
  assert.NoError(t, errClose)
  errCanStart = ctl.CanStart()
  assert.Error(t, errCanStart)
  errCanStop = ctl.CanStop()
  assert.Error(t, errCanStop)
  errSuspend = ctl.Suspend(token)
  assert.Error(t, errSuspend)
}

func doOp(t *testing.T, ctl *Controller, wg *sync.WaitGroup,
  opDuration, timeout time.Duration, success *uint32) {

  defer wg.Done()
  token, errToken := ctl.NewTokenWithTimeout(true, timeout)
  if errToken != nil {
    // the only expected error is a "context done".
    assert.True(t, strings.Contains(errToken.Error(), "context done"))
    return
  }
  _ = atomic.AddUint32(success, 1)
  time.Sleep(opDuration)
  ctl.CloseToken(token)
}

func TestControllerWait(t *testing.T) {
  ctl := NewController()
  errCanStart := ctl.CanStart()
  assert.NoError(t, errCanStart)
  ctl.SetStarted(true)

  var success uint32
  var wg sync.WaitGroup
  for ii := 0; ii < 5; ii++ {
    wg.Add(1)
    opDuration := 100 * time.Millisecond
    go doOp(t, ctl, &wg, opDuration, 0 /*timeout*/, &success)
  }
  wg.Wait()
  assert.Equal(t, success, uint32(5))
}

func TestControllerWaitTimeout(t *testing.T) {
  ctl := NewController()
  errCanStart := ctl.CanStart()
  assert.NoError(t, errCanStart)
  ctl.SetStarted(true)

  var success uint32
  var wg sync.WaitGroup
  for ii := 0; ii < 5; ii++ {
    wg.Add(1)
    opDuration := 100 * time.Millisecond
    timeout := 50 * time.Millisecond
    go doOp(t, ctl, &wg, opDuration, timeout, &success)
  }
  wg.Wait()
  // only the first operation should succeed and all others fail.
  assert.Equal(t, success, uint32(1))

  success = 0
  for ii := 0; ii < 5; ii++ {
    wg.Add(1)
    opDuration := 100 * time.Millisecond
    timeout := 5 * time.Second
    go doOp(t, ctl, &wg, opDuration, timeout, &success)
  }
  wg.Wait()
  // all operations must succeed
  assert.Equal(t, success, uint32(5))

  success = 0
  for ii := 0; ii < 5; ii++ {
    wg.Add(1)
    opDuration := 100 * time.Millisecond
    timeout := time.Duration(0) /* no timeout*/
    // have an operation in the middle timeout while waiting.
    if ii == 2 {
      timeout = 50 * time.Millisecond
    }
    go doOp(t, ctl, &wg, opDuration, timeout, &success)
  }
  wg.Wait()
  // 1 operation should timeout
  assert.Equal(t, success, uint32(4))
}

func TestControllerDurationTracking(t *testing.T) {
  ctl := NewController()
  errCanStart := ctl.CanStart()
  assert.NoError(t, errCanStart)
  ctl.SetStarted(true)

  var success uint32
  var wg sync.WaitGroup
  opDuration := 1000 * time.Millisecond
  wg.Add(1)
  go doOp(t, ctl, &wg, opDuration, 0 /*timeout*/, &success)
  wg.Add(1)
  go func(wg *sync.WaitGroup) {
    time.Sleep(500 * time.Millisecond)
    duration := ctl.BusyDuration()
    assert.True(t, duration <= opDuration,
      fmt.Sprintf("duration should be <= %v", opDuration))
    // Relies on the "doOp" goroutine having started before this one.
    assert.True(t, duration > 0, "duration should be > 0")
    wg.Done()
  }(&wg)
  wg.Wait()
  assert.Equal(t, int(ctl.BusyDuration()), 0)
}
