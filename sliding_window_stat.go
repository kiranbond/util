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
  "time"
)

// The desciption is copied from c++/open_util/base/sliding_window_stat.h
// and adapted to go language.
//
// The SlidingWindowStat class maintains a stat in a window that slides with
// time.
//
// Example usage:
//   // Create a sliding window stat with a window of 60s.
//   stat = NewSlidingWindowStat(60 * 1000000LL)
//
//   // Add some values at the current time to the stat.
//   stat.AddValue(10, -1)
//   stat.AddValue(20, -1)
//   ...
//   // Add a value at a specified time to the stat.
//   var timeUsecs int64
//   stat.AddValue(30, timeUsecs)
//
//   // Get the sum of values stored in the stat. This should return 10+20+30
//   // unless any of the above AddValue() calls were made more than 60s
//   // ago. If timeUsecs was more than 60s ago and the other two AddValue()
//   // calls were not, then this will return 10+20.
//   statValue := stat.Get()  // returns int64
//
//   // Remove the value 30 that was added at timeUsecs.
//   stat.AddValue(-30, timeUsecs)

const defaultIntervalCount int32 = 60

type position struct {
  index int32
  max   int32
}

// SlidingWindowStat object that provides the functionality for a single stat.
type SlidingWindowStat struct {
  WindowSizeUsecs int64    // Window size
  NumIntervals    int32    // Number of discrete intervals in a window
  IntervalUsecs   int64    // Duration of each interval
  latest          position // index of the latest position in intervalList
  // List of all intervals in increasing time order. It contains a total of
  // numIntervals in the slice.
  intervalList []intervalInfo
}

type intervalInfo struct {
  startTime int64 // Start time of this interval
  sum       int64 // Sum of stat for this interval
}

func (p *position) inc() {
  p.index = (p.index + 1) % p.max
}

func (p *position) dec() {
  p.index = (p.index + p.max - 1) % p.max
}

// NewSlidingWindowStat creates and returns a pointer to the newly created
// SlidingWindowStat. windowSizeUsecs provides the size of the sliding window.
// intervalCount provides the number of discrete intervals that this window
// is divided into. Values added into the stat are added to one of these
// discrete intervals. If intervalCount is -1, a default value of 60 is used.
func NewSlidingWindowStat(windowSizeUsecs int64,
  intervalCount int32) *SlidingWindowStat {
  if intervalCount == -1 {
    intervalCount = defaultIntervalCount
  }
  return &SlidingWindowStat{
    WindowSizeUsecs: windowSizeUsecs,
    NumIntervals:    intervalCount,
    IntervalUsecs:   windowSizeUsecs / int64(intervalCount),
    latest:          position{intervalCount - 1, intervalCount},
    intervalList:    make([]intervalInfo, intervalCount),
  }
}

// AddValue adds a value into this stat object at the specified time. If
// the specified time is larger than the time at which earlier values were
// added, then the sliding window moves forward. In this case, the right
// edge of the sliding window will correspond to the provided time.
//
// AddValue() is O(1) when timeUsecs is larger than any of the previous
// calls. If older times are used, complexity is O(NumIntervals).
//
// Note that 'value' may be +ve or -ve.
// If timeUsecs == -1, current time is used.
func (s *SlidingWindowStat) AddValue(value, timeUsecs int64) {
  if timeUsecs == -1 {
    timeUsecs = time.Now().Unix()
  }
  intervalStartUsec := timeUsecs / s.IntervalUsecs * s.IntervalUsecs
  latestInterval := s.intervalList[s.latest.index]

  if intervalStartUsec > latestInterval.startTime {
    // timeUsec belongs to a new interval, which is not present. Add it.
    // this also overwrites the oldest interval.
    s.latest.inc()
    s.intervalList[s.latest.index] = intervalInfo{intervalStartUsec, value}
    return
  }

  current := s.latest
  // now that the time is older than the latest interval, go through
  // the intervals and see if the corresponding interval already exists.
  // In that case, just add the value to the sum of that interval.
  // Otherwise, create a new interval, if the time is at least higher than
  // some interval. In that case, shift all the intervals by 1 and drop the
  // last one.
  for count := int32(0); count < s.NumIntervals; count++ {
    // interval found, just add the value to sum.
    if intervalStartUsec == s.intervalList[current.index].startTime {
      s.intervalList[current.index].sum += value
      break
    }
    // an older interval is found, so create a new one, move others back and
    // drop the last interval
    if intervalStartUsec > s.intervalList[current.index].startTime {
      prev := s.intervalList[current.index]
      s.intervalList[current.index] = intervalInfo{intervalStartUsec, value}
      count++
      for ; count < s.NumIntervals; count++ {
        current.dec()
        temp := s.intervalList[current.index]
        s.intervalList[current.index] = prev
        prev = temp
      }
      break
    }
    current.dec()
  }
  return
}

// Get returns the cumulate sum of all values added into the stat that lie
// within a time window W. If the latest time corresponding to any value
// added to the stat is L, then W is defined as follows:
//   * The left edge of W is given by
//       max(L, timeUsecs) - WindowSizeUsecs.
//   * The right edge of W is given by min(L, timeUsecs).
//   * W = (leftEdge, rightEdge]
// Note that this method does not otherwise affect L.
//
// Complexity of this method is O(NumIntervals).
// If timeUsecs == -1, current time is used.
func (s *SlidingWindowStat) Get(timeUsecs int64) int64 {
  if timeUsecs == -1 {
    timeUsecs = time.Now().Unix()
  }

  latestTime := s.intervalList[s.latest.index].startTime + s.IntervalUsecs - 1
  var leftEdge, rightEdge int64
  if timeUsecs > latestTime {
    leftEdge = timeUsecs - s.WindowSizeUsecs + 1
    rightEdge = latestTime
  } else {
    leftEdge = latestTime - s.WindowSizeUsecs + 1
    rightEdge = timeUsecs
  }

  // timeusecs is too far in future.
  if leftEdge > rightEdge {
    return 0
  }

  // sum of interval values within the window
  var sum int64
  // since left and right edges may not fall on interval boundary, only a
  // fraction of interval's value is counted toward sum. extrapolation keeps
  // the total fractional value on both the left and right side of window.
  var extrapolation float64

  current := s.latest
  // Now go through all the intervals and check for various cases:
  // Case 1: window W falls within the interval I
  // Solution: use  the fractional value of sum based on ratio of W/I
  // In this case we can also break, there is no further processing needed.
  //
  // Case 2: interval I falls within the window W
  // Solution: add the sum from the interval to total sum. Move to next
  // interval.
  //
  // Case 3: right side portion of interval I is in window W
  // Solution: add the fractional interval sum to total sum. We again do
  // not need any further processing, since next lower interval will be
  // outside the window.
  //
  // Case 4: left side portion of interval I is in window W
  // Solution: add the fractional sum based on the amount of interval in
  // window to total sum. Continue to the next lower interval, which may
  // still be in the window.
  for count := int32(0); count < s.NumIntervals; count++ {
    interval := s.intervalList[current.index]
    // check if window falls within this interval
    if interval.startTime <= leftEdge &&
      interval.startTime+s.IntervalUsecs > rightEdge {
      fraction := float64(rightEdge-leftEdge+1) / float64(s.IntervalUsecs)
      extrapolation += float64(interval.sum) * fraction
      break
    }

    // check if interval is within the window
    if interval.startTime >= leftEdge &&
      interval.startTime+s.IntervalUsecs-1 <= rightEdge {
      sum += interval.sum
      current.dec()
      continue
    }
    // check if suffix of interval is part of window
    if interval.startTime < leftEdge &&
      interval.startTime+s.IntervalUsecs > leftEdge {
      fraction := float64(leftEdge%s.IntervalUsecs) / float64(s.IntervalUsecs)
      extrapolation += float64(interval.sum) * (1.0 - fraction)
      break
    }
    // check if prefix of interval is part of window
    if interval.startTime <= rightEdge &&
      interval.startTime+s.IntervalUsecs-1 > rightEdge {
      fraction := float64((rightEdge%s.IntervalUsecs)+1) /
        float64(s.IntervalUsecs)
      extrapolation += float64(interval.sum) * (fraction)
      current.dec()
      continue
    }
    current.dec()
  }
  return sum + int64(extrapolation+0.5)
}
