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
// This file defines Controller type whose objects help in implementing Start,
// Stop and Close functions with uniform semantics.
//
// USAGE
//
// type SomeType struct {
//   ctl *util.Controller
//   ...
// }
//
// func (st *SomeType) Close() error {
//   if err := st.ctl.Close(); err != nil {
//     return err
//   }
//
//   ...try releasing resources...
//
//   return result
// }
//
// func (st *SomeType) Start() error {
//   if err := st.ctl.CanStart(); err != nil {
//     return err
//   }
//
//   ...try startup work...
//
//   if result == nil {
//     st.ctl.SetStarted(true)
//   } else {
//     st.ctl.SetStarted(false)
//   }
//   return result
// }
//
// func (st *SomeType) Stop() error {
//   if err := st.ctl.CanStop(); err != nil {
//     return err
//   }
//
//   ...try shutdown work...
//
//   if result == nil {
//     st.ctl.SetStopped(true)
//   } else {
//     st.ctl.SetStopped(false)
//   }
//   return result
// }
//
// func (st *SomeType) SomeOperation() error {
//   token, errToken := st.ctl.NewToken(true /* checkStarted */)
//   if errToken != nil {
//     return errToken
//   }
//   defer st.ctl.CloseToken(token)
//
//   ...handle the operation...
//
//   return result
// }

package util

import (
  "context"
  "flag"
  "fmt"
  "os"
  "sync"
  "time"
)

var panicDuration = flag.Duration("controller_token_timeout", 0,
  "When non-zero, panics the process if token wait time is more than "+
    "the allowed duration.")

// Controller type implements synchronization, rpc admission control and other
// functionality that helps in implement thread-safe network objects and
// Start/Stop/Close functions. See ensemble.go for usage.
type Controller struct {
  sync.Mutex

  // channels used for waiting. Each wait happens on a separate channel, to
  // ensure correct timeout handling.
  waitChans []chan struct{}

  // Flag indicating an operation is active.
  busy bool

  // time at which an operation was started.
  busyStart time.Time

  // Flag indicating if Close() method is called. If an object is closed, it is
  // effectively destroyed, so all other functions must fail.
  isClosed bool

  // Flag indicating if object services are activated.
  isStarted bool

  // Unique token id assigned to every operation.
  token int64
}

// NewController creates a new Controller object.
func NewController() *Controller {
  ctl := &Controller{}
  ctl.waitChans = []chan struct{}{}
  return ctl
}

// wait waits for a signal. This method should be invoked with the ctl.Lock()
// held.
func (ctl *Controller) wait() {
  // create a channel with a size of "1", to ensure that even after a timeout,
  // the signalling go-routine does not hang.
  waitCh := make(chan struct{}, 1)
  ctl.waitChans = append(ctl.waitChans, waitCh)

  // release the lock before the wait.
  ctl.Unlock()
  <-waitCh
  // re-acquire the lock after the wait.
  ctl.Lock()
}

// waitWithContext waits to receive a signal. The method
// returns error in case the "ctx expires before the signal is received,
// "nil" otherwise.
// This method should be invoked with the ctl.Lock() held.
func (ctl *Controller) waitWithContext(ctx context.Context) error {
  // create a channel with a size of "1", to ensure that after a timeout,
  // the signalling go-routine does not hang.
  waitCh := make(chan struct{}, 1)
  ctl.waitChans = append(ctl.waitChans, waitCh)

  // release the lock before the wait.
  ctl.Unlock()
  // re-acquire the lock after the wait.
  defer ctl.Lock()

  select {
  case <-waitCh:
    return nil
  case <-ctx.Done():
    return ctx.Err()
  }
}

// broadcast signals all the  waiting go-routine's "wait()" to unblock and
// proceed. This method should be invoked with ctl.Lock() held.
func (ctl *Controller) broadcast() {
  for _, waitCh := range ctl.waitChans {
    // all wait channels have a size of 1, so a write to a channel
    // corresponding to a go-routine whose wait has timed-out also works fine.
    waitCh <- struct{}{}
  }
  // clear "waitChans", as all the current waiting go-routines have been
  // signalled.
  ctl.waitChans = []chan struct{}{}
}

// Close returns nil if Close operation can be performed. Once a Close is
// invoked, object is effectively destroyed, irrespective of whether Close
// function returns success or not.
func (ctl *Controller) Close() error {
  if *panicDuration > 0 {
    cancelCh := startWatchdog(*panicDuration)
    defer close(cancelCh)
  }

  ctl.Lock()
  defer ctl.Unlock()

  if ctl.isClosed {
    return os.ErrInvalid
  }
  ctl.isClosed = true

  for ctl.busy {
    ctl.wait()
  }

  ctl.busy = false
  ctl.broadcast()
  return nil
}

// Suspend closes a controller temporarily. Caller must have a valid token.
func (ctl *Controller) Suspend(token int64) error {
  ctl.Lock()
  defer ctl.Unlock()

  if ctl.token != token {
    return os.ErrInvalid
  }

  if ctl.isClosed {
    return os.ErrInvalid
  }

  ctl.busy = false
  ctl.isClosed = true
  return nil
}

// Resume makes a suspended controller object live again. Caller must have a
// valid token. Token must be closed separately.
func (ctl *Controller) Resume(token int64) error {
  ctl.Lock()
  defer ctl.Unlock()

  if ctl.token != token {
    return os.ErrInvalid
  }

  if !ctl.isClosed {
    return os.ErrInvalid
  }

  ctl.busy = true
  ctl.busyStart = time.Now()
  ctl.isClosed = false
  return nil
}

// CanStart returns nil if Start operation can be performed.
func (ctl *Controller) CanStart() error {
  ctl.Lock()
  defer ctl.Unlock()

  for {
    if ctl.isClosed {
      return os.ErrInvalid
    }
    if ctl.isStarted {
      return fmt.Errorf("already started")
    }
    if !ctl.busy {
      break
    }
    ctl.wait()
  }

  ctl.busy = true
  ctl.busyStart = time.Now()
  return nil
}

// SetStarted marks the object as started.
//
// ok: flag that indicates if Start was successful or not.
func (ctl *Controller) SetStarted(ok bool) {
  ctl.Lock()
  defer ctl.Unlock()

  ctl.isStarted = ok
  ctl.busy = false
  ctl.broadcast()
}

// CanStop returns nil if Stop operation can be performed.
func (ctl *Controller) CanStop() error {
  ctl.Lock()
  defer ctl.Unlock()

  for {
    if ctl.isClosed {
      return os.ErrInvalid
    }
    if !ctl.isStarted {
      return fmt.Errorf("not started")
    }
    if !ctl.busy {
      break
    }
    ctl.wait()
  }

  ctl.busy = true
  ctl.busyStart = time.Now()
  return nil
}

// SetStopped marks the object as stopped.
//
// ok: flag that indicates if Stop was successful or not.
func (ctl *Controller) SetStopped(ok bool) {
  ctl.Lock()
  defer ctl.Unlock()

  if ok {
    ctl.isStarted = false
  } else {
    ctl.isStarted = true
  }
  ctl.busy = false
  ctl.broadcast()
}

// NewTokenWithContext returns a unique operation id when object is not closed.
// Returns an error if the token cannot be acquired before "ctx" expiry.
func (ctl *Controller) NewTokenWithContext(ctx context.Context,
  checkStarted bool) (int64, error) {

  if *panicDuration > 0 {
    cancelCh := startWatchdog(*panicDuration)
    defer close(cancelCh)
  }

  ctl.Lock()
  defer ctl.Unlock()

  for {
    if ctl.isClosed {
      return -1, os.ErrInvalid
    }
    if checkStarted && !ctl.isStarted {
      return -1, fmt.Errorf("not started")
    }
    if !ctl.busy {
      break
    }
    if err := ctl.waitWithContext(ctx); err != nil {
      // context expired.
      return -1, fmt.Errorf("context done :: %v", err)
    }
  }

  ctl.token++
  ctl.busy = true
  ctl.busyStart = time.Now()
  return ctl.token, nil
}

// NewTokenWithTimeout returns a unique operation id when object is not closed.
// times out with an error if the token cannot be acquired within "timeout".
// A timeout value of "0" implies no timeout.
func (ctl *Controller) NewTokenWithTimeout(checkStarted bool,
  timeout time.Duration) (int64, error) {

  ctx := context.Background()
  if timeout != 0 {
    var cancel context.CancelFunc
    ctx, cancel = context.WithTimeout(context.Background(), timeout)
    defer cancel()
  }
  return ctl.NewTokenWithContext(ctx, checkStarted)
}

// NewToken returns a unique operation id when object is not closed.
func (ctl *Controller) NewToken(checkStarted bool) (int64, error) {
  return ctl.NewTokenWithTimeout(checkStarted, 0 /*timeout*/)
}

// CloseToken marks an operation with given token as complete.
func (ctl *Controller) CloseToken(token int64) {
  ctl.Lock()
  defer ctl.Unlock()

  if ctl.token != token {
    panic(fmt.Sprintf("unexpected token:%d expected:%d", token, ctl.token))
  }

  ctl.busy = false
  ctl.broadcast()
}

// startWatchdog begins a watchdog timer for the specified duration. Returns a
// channel that can cancel the watchdog. Returns nil if timeout is zero.
func startWatchdog(timeout time.Duration) chan struct{} {
  if timeout <= 0 {
    return nil
  }
  cancelCh := make(chan struct{})
  go func() {
    select {
    case <-time.After(timeout):
      panic("controller watchdog timed out")
    case <-cancelCh:
      return
    }
  }()
  return cancelCh
}

// BusyDuration returns the duration for which the current operation has
// been running. Returns "0" if no operation is running.
func (ctl *Controller) BusyDuration() time.Duration {
  ctl.Lock()
  defer ctl.Unlock()
  if !ctl.busy {
    return 0
  }
  return time.Since(ctl.busyStart)
}
