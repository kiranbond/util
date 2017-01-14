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
// This file implements helper functions w.r.t error handling.

package util

import (
  "bytes"
  "fmt"
  "os/exec"
  "sync"
  "syscall"

  "github.com/pkg/errors"
)

// FirstError function returns the first non-nil error from the
// 'errList'. Returns nil if all elements in the 'errList' are nil.
//
// It is useful when a function needs to perform several operations ignoring
// the failures, but needs to return the first error. An example use is as
// follows:
//
// func (service *SomeService) Stop() error {
//   e1 := service.sudo.StopService("first-daemon")
//   e2 := service.sudo.StopService("second-daemon")
//   e3 := service.sudo.StopService("third-daemon")
//   return FirstError(e1, e2, e3)
// }
//
func FirstError(errList ...error) error {
  for _, err := range errList {
    if err != nil {
      return err
    }
  }
  return nil
}

// ErrorExitCode attempts to extract an exit code from an error, if there is no
// exit code associated with the error it returns -1.
func ErrorExitCode(err error) int {
  if exitErr, ok := errors.Cause(err).(*exec.ExitError); ok {
    if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
      return status.ExitStatus()
    }
  }
  return -1
}

// FmtWrappedErrors returns a string that contains details of all
// wrapped errors in a stack.
func FmtWrappedErrors(err error) string {
  var b bytes.Buffer
  errors.Fprint(&b, err)
  return b.String()
}

// Errors provides a thread safe structure to handle multiple errors.
// This may be used in cases where multiple go routines are
// spawed and it is required to track errors from all the routines.
type Errors struct {
  mu   sync.Mutex
  errs []error
}

// NewErrors creates a threadsafe Errors object
func NewErrors() *Errors {
  return &Errors{}
}

// Error implements error interface
func (e *Errors) Error() string {
  e.mu.Lock()
  defer e.mu.Unlock()
  var str string
  for i, err := range e.errs {
    str = fmt.Sprintf("%s error_%d:%s", str, i, err.Error())
  }
  return str
}

// Add appends error to reciever. If argument is of Errors type then
// all errors in argument is appended to reciever.
func (e *Errors) Add(err error) {
  if err == nil {
    return
  }
  e.mu.Lock()
  defer e.mu.Unlock()
  if errs, ok := err.(*Errors); ok {
    count := errs.Count()
    for ii := 0; ii < count; ii++ {
      e.errs = append(e.errs, errs.Get(ii))
    }
  } else {
    e.errs = append(e.errs, err)
  }
}

// Count of accumulated errors
func (e *Errors) Count() int {
  e.mu.Lock()
  defer e.mu.Unlock()
  return len(e.errs)
}

// Get the nth accumulated error
func (e *Errors) Get(n int) error {
  if e == nil {
    return nil
  }
  e.mu.Lock()
  defer e.mu.Unlock()
  if n >= 0 && n < len(e.errs) {
    return e.errs[n]
  }
  return nil
}

// Errors returns slice of errors.
func (e *Errors) Errors() []error {
  e.mu.Lock()
  defer e.mu.Unlock()
  return e.errs
}

// Return is a helper function for returning Errors type.
// Use this to return Errors, so that calling function can
// use "err == nil" check.
func (e *Errors) Return() error {
  e.mu.Lock()
  defer e.mu.Unlock()
  if e.errs == nil {
    return nil
  }
  return e
}
