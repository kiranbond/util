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
  "runtime"
  "sync"
  "testing"
  "time"

  "github.com/stretchr/testify/assert"
)

func TestRestartTimer(t *testing.T) {
  dur := 100 * time.Millisecond
  timer := NewRestartTimer(dur)
  c := timer.C
  select {
  case <-time.After(2 * dur):
    assert.NotNil(t, nil, "did not get timer within expected duration")
  case <-timer.C:
  }

  // now reset timer to 300 ms and make sure it fires only after 200 ms
  // so we make sure it did not fire with previous value of 100 sec.
  newDur := 300 * time.Millisecond
  timer.Reset(newDur)
  // an expired timer can race, so a new timer must be created
  assert.NotEqual(t, c, timer.C)

  start := time.Now()
  select {
  case <-time.After(2 * newDur):
    assert.NotNil(t, nil, "did not get timer within expected duration")
  case <-timer.C:
    diff := time.Now().Sub(start)
    assert.True(t, diff > 200*time.Millisecond, "timer fired faster than 200ms")
  }

  c = timer.C
  timer.RestartExpired()
  // restarting the expired timer using expired api shoud not create a new timer
  assert.Equal(t, c, timer.C)
  timer.Restart()
  // a running timer cannot race, so the timer should remain same
  assert.Equal(t, c, timer.C)

  timer.Stop()
  timer.Restart()
  // cannot distinguish between stopped timer and timer expiry,
  // so a new timer must be created
  assert.NotEqual(t, c, timer.C)

  // now stop rightaway to make sure it does not fire when stopped
  // this is racy and can fail in a loaded system, for a test case this is ok
  timer.Stop()
  select {
  case <-time.After(newDur * 2):
  case <-timer.C:
    assert.NotNil(t, nil, "did not expect timer to fire")
  }
}

func TestLoadedTimer(t *testing.T) {

  count := 50

  var wg sync.WaitGroup
  wg.Add(count)
  for i := 0; i < count; i++ {
    go func() {
      load := "this is a test string"
      dur := 20 * time.Millisecond
      ch := make(chan interface{})
      tmr := NewLoadedTimer(dur, &load, ch)

      select {
      case msg := <-ch:
        str, ok := msg.(*string)
        assert.True(t, ok)
        assert.Equal(t, load, *str)

      case <-time.After(dur + 200*time.Millisecond):
        assert.NotNil(t, nil, "did not receive expected message on channel")
      }

      tmr.Stop()
      wg.Done()
    }()
  }

  wg.Wait()

  max := 17 + runtime.GOMAXPROCS(0)
  num := runtime.NumGoroutine()
  // kiran: 17 + MAXPROCS was found thru experiment, if we find this failing then we can
  // change the check to 20 or something which ensures it is not
  // anywhere near the number of timers we started above.
  assert.True(t, num <= max, "found more goroutines %d than max: %d", num, max)
}
