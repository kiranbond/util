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
// This file implements miscellaneous stringutility functions.

package util

import (
  "strconv"
  "strings"
)

// IntSliceToStringSlice accepts a list of ints, converts each element to a
// string and returns the resultant list of strings.
func IntSliceToStringSlice(list []int) []string {
  stringList := make([]string, len(list))
  for ii, elem := range list {
    stringList[ii] = strconv.Itoa(elem)
  }
  return stringList
}

// JoinIntSlice accepts a list of ints, converts each element to a string,
// performs a strings.Join() on them and returns the result.
func JoinIntSlice(list []int, sep string) string {
  return strings.Join(IntSliceToStringSlice(list), sep)
}

// TrimSpaceStringSlice accepts a list of strings, trims spaces from each entry
// and returns the result.
func TrimSpaceStringSlice(list []string) []string {
  for ii, elem := range list {
    list[ii] = strings.TrimSpace(elem)
  }
  return list
}

// IsAnyStringEmpty checks if empty string is present and return true if there is.
func IsAnyStringEmpty(list ...string) bool {
  for _, elem := range list {
    if len(elem) == 0 {
      return true
    }
  }
  return false
}

func getTotalElems(items [][]string) int {
  total := 0
  for _, item := range items {
    total = total + len(item)
  }
  return total
}

// InterleaveStrings interleaves strings from a list of strings.
// slices is a 2-d array of strings with variable length.
func InterleaveStrings(slices [][]string) []string {
  totalElems := getTotalElems(slices)
  numSlices := len(slices)
  ii := 0
  elemCount := 0

  out := make([]string, totalElems)
  for elemCount < totalElems {
    slice := ii % numSlices
    index := ii / numSlices

    if index < len(slices[slice]) {
      out[elemCount] = slices[slice][index]
      elemCount++
    }
    ii++
  }
  return out
}
