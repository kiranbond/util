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
  "sort"

  "github.com/golang/glog"
)

// This file describes a fair selector that takes a group of buckets with
// elements and selects a list of elements to reach a certain sum, such that the
// selected elements are equally distributed across the buckets.
// An example use case for this is to select a list of disks for a pool such
// that the sum of disk capacities add up to the desired pool capacity and
// the disks are equally distributed across hosts.
//
// Example usage:
//
// (1) Create a FairSelector object with a group of buckets each with a list
//     of elements.
//     fs := NewFairSelector([]Bucket)
//
//     value, result := fs.Select(target)
//     Select function chooses a set of elements to sum up to a certain target.
//     It returns the value as the achieved sum and a group with selected
//     elements. Caller should compare the value with target to see if exact
//     target is reached.
//
//     note: this object is not thread-safe and also it operates on the input in
//     place and modifies that to return the result. Caller should not use the
//     input after creating FairSelector object with that.

// BucketValue represents each individual value of all values in a bucket.
type BucketValue struct {
  VID   string // ID of this value, useful for de-multiplexing output.
  Value int64  // value of this element.
}

// Bucket defines the input to the FairSelector.
type Bucket struct {
  BID    string        // id for this bucket
  Values []BucketValue // elements in this bucket
}

// FairSelector type implements the functionality of selecting entries from
// a list of buckets, such that they sum to a target and the distribution
// across buckets is fair.
type FairSelector struct {
  buckets []Bucket // list of input buckets
  used    []int64  // to keep track of allocation per bucket
}

// ByBucketValue implements sort.Interface for []BucketValue based on value
// field of BucketValue
type ByBucketValue []BucketValue

func (bktVal ByBucketValue) Len() int {
  return len(bktVal)
}

func (bktVal ByBucketValue) Swap(i, j int) {
  bktVal[i], bktVal[j] = bktVal[j], bktVal[i]
}

func (bktVal ByBucketValue) Less(i, j int) bool {
  return bktVal[i].Value < bktVal[j].Value
}

// NewFairSelector return a pointer to FairSelector object after initializing it
// with the given list.
func NewFairSelector(list []Bucket) *FairSelector {
  fs := &FairSelector{}
  // note: we are not copying the elements but using the input list as it is
  fs.buckets = list

  // sort all the elements in each bucket
  for _, bb := range fs.buckets {
    sort.Sort(ByBucketValue(bb.Values))
  }
  fs.used = make([]int64, len(list))
  return fs
}

// Select picks the elements in all buckets such that they add up to a target
// and the overall allocation across buckets is balanced.
func (fs *FairSelector) Select(target int64) (int64, []Bucket) {
  result := make([]Bucket, len(fs.buckets))
  for ii, bb := range fs.buckets {
    result[ii].BID = bb.BID
  }
  size := int64(0)

  for {
    found, valID, idx, min := fs.findMinUsed()
    if !found {
      glog.Errorf("did not find min buckets %v used %v", fs.buckets, fs.used)
      return size, result
    }

    glog.V(1).Infof("found min %d in bucket[%d] buckets %v used %v", min, idx,
      fs.buckets, fs.used)

    // check if we are going to exceed the target with this new element and
    // if the excess is higher than the current deficit, return now.
    diff := size + min - target
    if diff > 0 && diff > target-size {
      return size, result
    }

    fs.used[idx] += min
    size += min
    result[idx].Values = append(result[idx].Values, BucketValue{Value: min, VID: valID})
    fs.buckets[idx].Values =
      fs.buckets[idx].Values[1:len(fs.buckets[idx].Values)]
    if size > target {
      glog.V(1).Infof("answer size %d results %v", size, result)
      return size, result
    }
  }
}

// findMinUsed finds min out of used value for all buckets. If used value is
// same, the bucket with the smallest first element is selected.
func (fs *FairSelector) findMinUsed() (bool, string, int, int64) {
  min := int64(math.MaxInt64)
  minUsed := int64(math.MaxInt64)
  idx := 0
  found := false
  vID := ""

  for ii, bb := range fs.buckets {
    if len(bb.Values) == 0 {
      continue
    }
    if fs.used[ii] < minUsed && bb.Values[0].Value <= min ||
      fs.used[ii] == minUsed && bb.Values[0].Value < min {

      min = bb.Values[0].Value
      minUsed = fs.used[ii]
      idx = ii
      vID = bb.Values[0].VID
      found = true
    }
  }
  return found, vID, idx, min
}
