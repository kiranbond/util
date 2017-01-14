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
// This file implements helper functions w.r.t time.

package util

import (
  "time"
)

const (
  cOSNotificationTimeFormat    string = "2006-01-02 15:04:05.999999"
  cOSNotificationTimeFormatT   string = "2006-01-02T15:04:05.999999"
  cOSNotificationTimeFormatNZ  string = "2006-01-02 15:04:05+00:00"
  cOSNotificationTimeFormatNZT string = "2006-01-02T15:04:05+00:00"
)

// GetNowUTC returns current time in UTC timezone.
func GetNowUTC() time.Time {
  return time.Now().UTC()
}

// GetNowUTCUnix gets the current time as UTC Unix timestamp.
func GetNowUTCUnix() int64 {
  return time.Now().UTC().Unix()
}

// GetNowUTCUnixMS gets the current time in milliseconds as UTC Unix
// timestamp.
func GetNowUTCUnixMS() int64 {
  return NSToMS(GetNowUTCUnixNS())
}

// GetNowUTCUnixNS gets the current time in nanoseconds as UTC Unix
// timestamp.
func GetNowUTCUnixNS() int64 {
  return time.Now().UTC().UnixNano()
}

// GetUTCUnix returns t as a Unix time, with location set to UTC - the
// number of seconds since January 1, 1970 UTC.
func GetUTCUnix(t time.Time) int64 {
  return t.UTC().Unix()
}

// GetUTCUnixMS returns t as a Unix time, with location set to UTC - the
// number of milliseconds since January 1, 1970 UTC.
func GetUTCUnixMS(t time.Time) int64 {
  return t.UTC().UnixNano() / int64(time.Millisecond)
}

// GetUTCUnixNS returns t as a Unix time, with location set to UTC - the
// number of nanoseconds since January 1, 1970 UTC.
func GetUTCUnixNS(t time.Time) int64 {
  return t.UTC().UnixNano()
}

// UnixMSToUTC converts a timestamp into time
func UnixMSToUTC(ts int64) time.Time {
  return time.Unix(MSToSec(ts), 0).UTC()
}

// SecToMS converts seconds to milliseconds.
func SecToMS(t int64) int64 {
  return t * 1000
}

// SecToNS converts seconds to nanoseconds.
func SecToNS(t int64) int64 {
  return t * 1000 * 1000 * 1000
}

// MSToNS converts milliseconds to nanoseconds.
func MSToNS(t int64) int64 {
  return t * 1000 * 1000
}

// MSToSec converts milliseconds to seconds.
func MSToSec(t int64) int64 {
  return t / 1000
}

// NSToMS converts nanoseconds to milliseconds (precision lost).
func NSToMS(t int64) int64 {
  return t / (1000 * 1000)
}

// NSToSec converts nanoseconds to seconds.
func NSToSec(t int64) int64 {
  return t / (1000 * 1000 * 1000)
}

// TimeToEpoch converts Time instance to epoch.
func TimeToEpoch(t time.Time) int64 {
  return t.Unix()
}

// TimeToEpochNS converts Time instance to epoch in nanos.
func TimeToEpochNS(t time.Time) int64 {
  return t.UnixNano()
}

// TimeToEpochMS converts Time instance to epoch in millis.
func TimeToEpochMS(t time.Time) int64 {
  return NSToMS(TimeToEpochNS(t))
}

// DurationToMS converts Duration instance to millis.
func DurationToMS(d time.Duration) int64 {
  return d.Nanoseconds() / (1000 * 1000)
}

// ParseTimeStr parses a formatted string using a set of know formats and
// returns the time value it represents.
func ParseTimeStr(timestamp string) (time.Time, error) {
  if len(timestamp) == 0 {
    return time.Unix(0, 0), nil
  }
  t, err := time.Parse(cOSNotificationTimeFormat, timestamp)
  if err == nil {
    return t, nil
  }

  t, err = time.Parse(cOSNotificationTimeFormatT, timestamp)
  if err == nil {
    return t, nil
  }

  t, err = time.Parse(cOSNotificationTimeFormatNZ, timestamp)
  if err == nil {
    return t, nil
  }

  t, err = time.Parse(cOSNotificationTimeFormatNZT, timestamp)
  if err == nil {
    return t, nil
  }

  t, err = time.Parse(time.RFC3339, timestamp)
  if err == nil {
    return t, nil
  }

  t, err = time.Parse(time.RFC3339Nano, timestamp)
  if err == nil {
    return t, nil
  }

  return t, err
}
