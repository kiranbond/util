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
// Utility functions to sort IP addresses.

package util

import (
  "bytes"
  "net"
)

// IPSlice is the sort package wrapper for net.IP type.
type IPSlice []net.IP

func (a IPSlice) Len() int {
  return len(a)
}

func (a IPSlice) Swap(i, j int) {
  a[i], a[j] = a[j], a[i]
}

func (a IPSlice) Less(i, j int) bool {
  return bytes.Compare([]byte(a[i]), []byte(a[j])) < 0
}
