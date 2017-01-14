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
  "strconv"
  "sync"
  "testing"

  "github.com/stretchr/testify/assert"
)

// TestErrors veryfies Errors structure is capable to record
// errors in a multi thread environment.
func TestErrors(t *testing.T) {

  N := 20
  errors := NewErrors()
  var wg sync.WaitGroup
  check := make(map[int]bool)

  // add N errors
  for i := 0; i < N; i++ {
    wg.Add(1)
    check[i] = false
    go func(i int) {
      defer wg.Done()
      errors.Add(fmt.Errorf("%d", i))
    }(i)
  }
  wg.Wait()

  // check we have the correct number of errors
  assert.Equal(t, errors.Count(), N)

  // check all the errors were received
  for i := 0; i < N; i++ {
    err := errors.Get(i)
    n, err := strconv.Atoi(err.Error())
    assert.Equal(t, err, nil)
    check[n] = true
  }

  for i := 0; i < N; i++ {
    assert.Equal(t, check[i], true)
  }
}

// TestErrorNone checks Errors handles absence of errors correctly
func TestErrorsNone(t *testing.T) {
  N := 20
  errors := NewErrors()

  // add N errors
  for i := 0; i < N; i++ {
    errors.Add(nil)
  }

  // check we have the correct number of errors
  assert.Equal(t, errors.Count(), 0)

  // check Return gives nil
  assert.Equal(t, errors.Return(), nil)
}
