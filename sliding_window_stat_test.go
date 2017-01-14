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
  "flag"
  "testing"

  "github.com/stretchr/testify/assert"
)

// TestSlidingWindowStat does the basic testing for sliding window stat.
// It is pretty much copied from the cpp util test case.
func TestSlidingWindowStat(t *testing.T) {
  flag.Set("stderrthreshold", "0")
  window := int64(100)
  intervals := int32(10)
  stat := NewSlidingWindowStat(window, intervals)
  stat.AddValue(20, 50)                           // add 20 at time = 50
  assert.Equal(t, int64(20), stat.Get(60))        // (-40, 60]
  assert.Equal(t, int64(20), stat.Get(80))        // (-20, 80]
  assert.Equal(t, int64(0), stat.Get(40))         // (-60, 40]
  assert.Equal(t, int64(0), stat.Get(20))         // (-80, 20]
  assert.Equal(t, int64(20), stat.Get(49+window)) // (49, 149]
  assert.Equal(t, int64(18), stat.Get(50+window)) // (50, 150] => 0.9
  assert.Equal(t, int64(8), stat.Get(55+window))  // (55, 155] => 0.4
  assert.Equal(t, int64(0), stat.Get(60+window))  // (60, 160] => 0
  assert.Equal(t, int64(0), stat.Get(80+window))  // (80, 180] => 0

  // Add values such that previous interval [50-59] is removed from window.
  // Do it in unordered manner to test interval addition in between.
  stat.AddValue(1, 61)
  stat.AddValue(1, 81)
  stat.AddValue(1, 101)
  stat.AddValue(1, 71)
  stat.AddValue(1, 121)
  stat.AddValue(1, 91)
  stat.AddValue(1, 131)
  stat.AddValue(1, 141)
  stat.AddValue(1, 111)
  stat.AddValue(1, 151)

  assert.Equal(t, int64(9), stat.Get(165)) // Window partially in rightmost interval
  assert.Equal(t, int64(10), stat.Get(155))
  assert.Equal(t, int64(9), stat.Get(151))
  assert.Equal(t, int64(9), stat.Get(145))
  assert.Equal(t, int64(8), stat.Get(141))
  assert.Equal(t, int64(7), stat.Get(131))
  assert.Equal(t, int64(6), stat.Get(121))
  assert.Equal(t, int64(5), stat.Get(111))
  assert.Equal(t, int64(4), stat.Get(101))
  assert.Equal(t, int64(3), stat.Get(91))
  assert.Equal(t, int64(2), stat.Get(81))
  assert.Equal(t, int64(1), stat.Get(71)) // Window lies partially in leftmost interval
  assert.Equal(t, int64(0), stat.Get(61)) // Window fits entirely in [60, 69] of stat.
  assert.Equal(t, int64(0), stat.Get(51)) // No interval of stat lies in window.

  // Add values such that interval [60-69] is no more part of window.
  stat.AddValue(2, 161)
  assert.Equal(t, int64(11), stat.Get(169)) // [70 - 169]
  assert.Equal(t, int64(9), stat.Get(159))  // [60 - 159], but [60-69] is absent.

  // Add values such that window does not move.
  stat.AddValue(2, 161)
  stat.AddValue(2, 151)
  assert.Equal(t, int64(15), stat.Get(169)) // [70 - 169]
  assert.Equal(t, int64(11), stat.Get(159)) // [60 - 159]

  // Add a value much larger than current right edge of window.
  stat.AddValue(5, 301)
  assert.Equal(t, int64(0), stat.Get(169))
  assert.Equal(t, int64(0), stat.Get(199))
  assert.Equal(t, int64(0), stat.Get(209))
  assert.Equal(t, int64(0), stat.Get(259))
  assert.Equal(t, int64(5), stat.Get(350))
  assert.Equal(t, int64(5), stat.Get(399))
  assert.Equal(t, int64(5), stat.Get(400))

  // Add a value to an interval less than the latest interval,
  // such that the added interval is not yet present.
  stat.AddValue(5, 281)
  stat.AddValue(5, 271)
  assert.Equal(t, int64(10), stat.Get(380))
  assert.Equal(t, int64(15), stat.Get(370))
}
