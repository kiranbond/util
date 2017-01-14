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
  "testing"

  "github.com/golang/glog"
  "github.com/stretchr/testify/assert"
)

func TestSubsetSumExactMatch(t *testing.T) {
  ss := NewSubsetSumFinder()

  list := []Element{{2, nil}, {3, nil}, {7, nil}, {13, nil}, {4, nil}}
  ss.AddElements(list)
  val, output := ss.Solve(14)
  assert.Equal(t, int(val), 14, "output not equal to target")
  sum := int64(0)
  for _, el := range output {
    sum += el.val
    glog.V(1).Infof("element selected %d", el.val)
  }
  glog.Infof("closest sum found = %d", sum)
  assert.Equal(t, int(sum), 14, "sum of values not equal to target")
}

func TestSubsetSumExactMatchLarge(t *testing.T) {
  ss := NewSubsetSumFinder()

  list := []Element{{2000, nil}, {8500, nil}, {3200, nil}, {13500, nil},
    {1200, nil}, {1200, nil}, {1200, nil}, {800, nil}, {800, nil}, {800, nil},
    {800, nil}, {800, nil}, {500, nil}, {500, nil}, {500, nil}, {500, nil},
    {500, nil}, {500, nil}}
  ss.AddElements(list)
  val, output := ss.Solve(10800)
  assert.Equal(t, int(val), 10800, "output not equal to target")
  sum := int64(0)
  for _, el := range output {
    sum += el.val
    glog.V(1).Infof("element selected %d", el.val)
  }
  glog.Infof("closest sum found = %d", sum)
  assert.Equal(t, int(sum), 10800, "sum of values not equal to target")
}

func TestSubsetSumNoMatch(t *testing.T) {
  ss := NewSubsetSumFinder()

  list := []Element{{20, nil}, {3, nil}, {7, nil}, {13, nil}, {40, nil}}
  ss.AddElements(list)
  val, output := ss.Solve(51)
  assert.NotEqual(t, int(val), 51, "output equal to target, not unexpected")
  sum := int64(0)
  for _, el := range output {
    sum += el.val
    glog.V(1).Infof("element selected %d", el.val)
  }
  glog.Infof("closest sum found = %d", sum)
  assert.Equal(t, int(sum), 50, "sum of values not equal to 50")
}

func TestSubsetSumNoMatchLarge(t *testing.T) {
  ss := NewSubsetSumFinder()

  list := []Element{{2000, nil}, {8500, nil}, {3200, nil}, {13500, nil},
    {1200, nil}, {1200, nil}, {1200, nil}, {800, nil}, {800, nil}, {800, nil},
    {800, nil}, {800, nil}, {500, nil}, {500, nil}, {500, nil}, {500, nil},
    {500, nil}, {500, nil}}
  ss.AddElements(list)
  val, output := ss.Solve(10980)
  assert.NotEqual(t, int(val), 10980, "output not equal to target")
  sum := int64(0)
  for _, el := range output {
    sum += el.val
    glog.V(1).Infof("element selected %d", el.val)
  }
  glog.Infof("closest sum found = %d", sum)
  assert.Equal(t, int(sum), 11000, "sum of values not equal to target")
}
