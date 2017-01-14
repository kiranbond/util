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
// RestartTimer - Stop/Restart timer with stored duration.
// LoadedTimer - Timer with embedded payload that is delivered on the channel
//               when the timer expires.

package util

import (
  "sync"
  "time"
)

// RestartTimer provides a timer that stores the duration internally so it can
// be restarted without needing to supply the duration every time. Essentially,
// it becomes a Stop/Restart ticker that we do not have natively in golang.
// Due to race in Stop() and firing of timer we use new timer in some cases
//
// IMPORTANT: r.C should not be cached since it can change
// when Reset or Restart are called.
type RestartTimer struct {
  d time.Duration
  C <-chan time.Time
  t *time.Timer
}

// NewRestartTimer creates a restartable timer.
func NewRestartTimer(dur time.Duration) *RestartTimer {
  timer := time.NewTimer(dur)
  return &RestartTimer{d: dur, C: timer.C, t: timer}
}

// RestartExpired will restart an expired timer. This version is racy if the
// timer has not already expired, if in doubt use Restart() or Reset() as
// required they are non-racy and handle races correctly.
// One correct use is as follows
// <-t.C
// t.RestartExpired()
func (r *RestartTimer) RestartExpired() {
  // the timer has expired so it is safe to just reset it
  r.t.Reset(r.d)
}

// Stop calls the underlying timer Stop and returns its return value.
func (r *RestartTimer) Stop() bool {
  return r.t.Stop()
}

// Close will close the timer.
func (r *RestartTimer) Close() {
  r.d = 0
  r.C = nil
  r.Stop()
  r.t = nil
}

// Reset will reset the timer with the new duration.
// IMPORTANT: r.C should not be cached since it can change
// when Reset or Restart are called
func (r *RestartTimer) Reset(dur time.Duration) {
  r.d = dur
  r.Restart()
}

// Restart will start the timer with the saved duration.
// IMPORTANT: r.C should not be cached since it can change
// when Reset or Restart are called
func (r *RestartTimer) Restart() {
  ok := r.Stop()
  if ok {
    // the timer has been successfully stopped just reset
    r.t.Reset(r.d)
    return
  }
  // possible race with old timer, create a new timer
  r.t = time.NewTimer(r.d)
  r.C = r.t.C
}

// LoadedTimer has a payload that it delivers on the channel when the specified
// duration expires. The payload is delivered in an async goroutine so the
// timer will not block but if the channel is full, the goroutine count will
// go up.
type LoadedTimer struct {
  *time.Timer
  mu      sync.Mutex
  Payload interface{}
  Ch      chan interface{}
  stopCh  chan struct{}
  doneCh  chan struct{}
}

// NewLoadedTimer creates a LoadedTimer instance that will deliver the load on
// the supplied chan when the duration expires.
func NewLoadedTimer(dur time.Duration, load interface{},
  ch chan interface{}) *LoadedTimer {

  timer := time.NewTimer(dur)
  l := LoadedTimer{
    Timer:   timer,
    Payload: load,
    Ch:      ch,
    stopCh:  make(chan struct{}),
    doneCh:  make(chan struct{}),
  }
  go l.run()
  return &l
}

// Stop overrides the Timer.Stop by stopping the run() goroutine.
// It makes sure that it waits for the run goroutine to stop by closing the
// stopCh and waiting for the doneCh. It handles any Stop after Stop calls
// (which user should not do) by using mutex and setting stopCh to nil.
func (l *LoadedTimer) Stop() {
  l.mu.Lock()
  defer l.mu.Unlock()

  if l.stopCh != nil {
    l.Timer.Stop()
    close(l.stopCh)
    <-l.doneCh
  }
  l.stopCh = nil
}

func (l *LoadedTimer) run() {
  select {
  case <-l.C:
    select {
    case l.Ch <- l.Payload:
    case <-l.stopCh:
    }
  case <-l.stopCh:
  }
  close(l.doneCh)
}
