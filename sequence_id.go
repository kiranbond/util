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
  "sync/atomic"
)

// SequenceID implements an atomic monotonically increasing uint64 id.
type SequenceID struct {
  id uint64
}

// Incr increments the sequenceID by one and returns the new value.
func (s *SequenceID) Incr() uint64 {
  return atomic.AddUint64(&s.id, 1)
}

// Get returns the value of the sequenceID
func (s *SequenceID) Get() uint64 {
  return atomic.LoadUint64(&s.id)
}

// SetIfGT sets the sequenceID to the input param only if the input is
// greater than the current value of the sequenceID.
func (s *SequenceID) SetIfGT(val uint64) error {
  for snappedID := atomic.LoadUint64(&s.id); val > snappedID; snappedID = atomic.LoadUint64(&s.id) {

    if atomic.CompareAndSwapUint64(&s.id, snappedID, val) {
      return nil
    }
  }
  return fmt.Errorf("new id %v is less than last %v", val, s.id)
}

// Marshal returns a byte slice whose contents are the serialised sequence id.
func (s *SequenceID) Marshal() ([]byte, error) {
  return strconv.AppendUint(nil, s.id, 10), nil
}

// Unmarshal extracts the sequence ID from the given byte slice. It returns a
// parse error on failure.
func (s *SequenceID) Unmarshal(b []byte) error {
  id, err := strconv.ParseUint(string(b), 10, 64)
  if err != nil {
    return fmt.Errorf("error parsing sequence id :: %v", err)
  }
  s.id = id
  return nil
}
