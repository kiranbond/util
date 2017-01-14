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
  "sync"
  "sync/atomic"
  "testing"

  "github.com/stretchr/testify/assert"
)

func testInc(sq *SequenceID, wg *sync.WaitGroup, work int, result *[]uint64) {
  res := *result
  for i := 0; i < work; i++ {
    incr := sq.Incr()
    atomic.AddUint64(&res[incr-1], 1)
  }
  wg.Done()
}

func testIfGT(sq *SequenceID, wg *sync.WaitGroup, work int, errCnt *uint64) {
  for i := 0; i < work; i++ {
    incr := sq.Get()
    err := sq.SetIfGT(incr + 1)
    if err != nil {
      atomic.AddUint64(errCnt, 1)
    }
  }
  wg.Done()
}

func TestSequenceID(t *testing.T) {
  var wg sync.WaitGroup
  workers := 100
  work := 50000

  var sqInc SequenceID
  result := make([]uint64, workers*work)

  for i := 0; i < workers; i++ {
    wg.Add(1)
    go testInc(&sqInc, &wg, work, &result)
  }
  wg.Wait()
  assert.Equal(t, uint64(workers*work), sqInc.Get())
  for idx, val := range result {
    assert.Equal(t, val, uint64(1), "unexpected at index %d", idx)
  }

  var sqIfGT SequenceID
  errCnt := uint64(0)
  for i := 0; i < workers; i++ {
    wg.Add(1)
    go testIfGT(&sqIfGT, &wg, work, &errCnt)
  }
  wg.Wait()

  // Current sequence count + error should be equal total work done
  assert.Equal(t, uint64(workers*work), sqIfGT.Get()+errCnt,
    "current seq %d err cnt %d", sqIfGT.Get(), errCnt)
}
