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
// This file implements utility functions for type conversions.

package util

import (
  "strconv"

  "github.com/golang/glog"
)

const (
  _ int64 = 1 << (10 * iota) // ignore first value by assigning to blank identifier
  // KB denotes kilobyte multiplier
  KB
  // MB denotes megabyte multiplier
  MB
  // GB denotes gigabyte multiplier
  GB
  // TB denotes terabyte multiplier
  TB
  // PB denotes petabyte multiplier
  PB
)

// StrToint32 parses an input string as a decimal number and returns it as a
// int32. Any errors in the conversion are dropped. ParseUint returns a int64
// which is cast into int32 so any loss of precision is callers responsbility.
func StrToint32(s string) int32 {
  n, _ := strconv.ParseInt(s, 10, 32)
  return int32(n)
}

// StrToint64 parses an input string as a decimal number and returns it as a
// int64. Any errors in the conversion are dropped.
func StrToint64(s string) int64 {
  n, _ := strconv.ParseInt(s, 10, 64)
  return int64(n)
}

// StrToUint32 parses an input string as a decimal number and returns it as a
// uint32. Any errors in the conversion are dropped. ParseUint returns a uint64
// which is cast into uint32 so any loss of precision is callers responsbility.
func StrToUint32(s string) uint32 {
  n, _ := strconv.ParseUint(s, 10, 32)
  return uint32(n)
}

// StrToUint64 parses an input string as a decimal number and returns it as a
// uint64. Any errors in the conversion are dropped.
func StrToUint64(s string) uint64 {
  n, _ := strconv.ParseUint(s, 10, 64)
  return n
}

// ParseIntDefault is a helper function to convert a string to integer. Instead
// of returning an error on parse failure, it returns the user chosen default
// value.
func ParseIntDefault(str string, base, bitSize int, defValue int64) int64 {
  value, errParse := strconv.ParseInt(str, base, bitSize)
  if errParse != nil {
    glog.Errorf("could not parse integer from %s :: %v", str, errParse)
    return defValue
  }
  return value
}

// GBToBytes converts input unit in GB to bytes
func GBToBytes(sizeInGB int) int64 {
  return int64(sizeInGB) * GB
}

// BytesToGB converts input unit in bytes to GB
func BytesToGB(sizeInBytes int64) int64 {
  return sizeInBytes / int64(GB)
}

// BytesToMB converts input unit in bytes to MB
func BytesToMB(sizeInBytes int64) int64 {
  return sizeInBytes / int64(MB)
}
