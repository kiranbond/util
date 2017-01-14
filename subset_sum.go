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
  "math"

  "github.com/golang/glog"
)

// This file defines the SubsetSum struct that can be used to get the
// closest or exact sum to a certain value V given a sequence S of elements.
// This object is not thread safe and caller should make sure that it is used
// by a single thread only or is protected by a lock in caller.
// SubsetSum implements a SubsetSumFinder interface to provide this
// functionality.
//
// Example usage:
//
// (1) Create a SubsetSumFinder struct with a list of structs and a desired sum
//     ss = NewSubsetSumFinder()
//
//     ss.AddElements({e1, I1}, {e2, I2}, ...)
//     here e1, e2 are the actual elements and I1, I2 are some blobs associated
//     with each value. Ideally this should be a pointer to a struct, since
//     these may be copied during the computation.
//
//     Finally, one can find if the sum V exists using
//     v, out := ss.Solve(V)
//     the caller should check if returned sum is equal to V to check if exact
//     sum is found. In case of no exact match the closest sum to V is returned.
//     in any case, the out list contains the elements the make up the final sum

// Element is the single element that can be added to the list
type Element struct {
  val  int64       // the actual value of the element used for subsetsum
  blob interface{} // this is an opaque field is returned in the final list
}

// SubsetSumFinder defines the interface provided by SubsetSum.
type SubsetSumFinder interface {
  AddElements(list []Element)
  Solve(value int64) (int64, []Element)
}

// SubsetSum is the main object that provides the functionality.
type SubsetSum struct {
  elements []Element // list of elements to use for finding a target sum
}

// NewSubsetSumFinder creates and returns a pointer to SubsetSum.
func NewSubsetSumFinder() *SubsetSum {
  ss := &SubsetSum{}
  return ss
}

// AddElements adds the list of elements to the current list
func (ss *SubsetSum) AddElements(list []Element) {
  ss.elements = append(ss.elements, list...)
}

// Solve tries to solve for the exact sum that is specified by the input
// parameter. It returns the sum value found with the desired element list.
// The called should check to make sure that the returned value is the same as
// specified value or not.
func (ss *SubsetSum) Solve(target int64) (int64, []Element) {
  // first sort the elements in the list, using a simple n^2 sorting algorithm
  // TODO: replace with a better algorithm from golang library or
  // otherwise.
  count := len(ss.elements)
  for ii := 0; ii < count; ii++ {
    for jj := ii + 1; jj < count; jj++ {
      if ss.elements[ii].val > ss.elements[jj].val {
        ss.elements[ii], ss.elements[jj] = ss.elements[jj], ss.elements[ii]
      }
    }
  }

  // now lets use dynamic programming to get the list of all possible subset
  // sums. This is done by creating a row per element that captures the possible
  // sums using that element and possible combinations of previous elements.
  rows := make([]map[int64]bool, count, count)
  rows[0] = make(map[int64]bool)
  rows[0][0] = true
  rows[0][ss.elements[0].val] = true
  max := int64(0)
  lastRow := 0
  for ii := 1; ii < count; ii++ {
    rows[ii] = make(map[int64]bool)
    rows[ii][0] = true
    // copy the previous row into the new/current row and also add new elements
    for k := range rows[ii-1] {
      rows[ii][k] = true
      rows[ii][k+ss.elements[ii].val] = true
      if max < k+ss.elements[ii].val {
        max = k + ss.elements[ii].val
      }
    }
    lastRow = ii
    // this check should significantly reduce the runtime of the algorithm
    // from being exponential to a function of the target value itself.
    // we only need to reach a number higher than target to know if the value
    // is found. We are going a bit higher than that to handle the case when
    // the closest number is somewhat higher than target.
    if max > (target + target/5) {
      break
    }
  }

  // now check if the target is true for the last row. If true, we can get the
  // exact sum, otherwise we need to find the closest sum and return that to the
  // caller.
  closestSum := int64(0)
  if rows[lastRow][target] {
    glog.Infof("target found %d", target)
    closestSum = target
  } else {
    // find the closest sum
    diff := int64(math.MaxInt64)
    for possibleSum := range rows[lastRow] {
      absDiff := int64(math.Abs(float64(possibleSum - target)))
      if diff > absDiff {
        diff = absDiff
        closestSum = possibleSum
      }
    }
    glog.Infof("exact match not found, closest = %d target = %d", closestSum,
      target)
  }

  // now lets get the exact list of elements making the closestSum
  sumLeft := closestSum
  outputList := []Element{}
  for ii := lastRow; ii >= 1; ii-- {
    if rows[ii][sumLeft] == true && rows[ii-1][sumLeft] == false {
      outputList = append(outputList, ss.elements[ii])
      sumLeft = sumLeft - ss.elements[ii].val
    }
    if sumLeft == 0 {
      break
    }
  }
  if sumLeft > 0 {
    outputList = append(outputList, ss.elements[0])
  }
  return closestSum, outputList
}
