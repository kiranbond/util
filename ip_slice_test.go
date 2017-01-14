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
  "net"
  "sort"
  "testing"
)

func TestIPSlice(t *testing.T) {
  ips := []net.IP{
    net.IPv4(127, 0, 0, 1),
    net.IPv4(127, 0, 0, 2),
    net.IPv4(127, 0, 0, 0),
  }

  sort.Sort(IPSlice(ips))

  if !net.IPv4(127, 0, 0, 0).Equal(ips[0]) {
    t.Fatalf("127.0.0.0 was not lesser than 127.0.0.1")
  }
  if !net.IPv4(127, 0, 0, 1).Equal(ips[1]) {
    t.Fatalf("127.0.0.1 was not greater than 127.0.0.0")
  }
  if !net.IPv4(127, 0, 0, 2).Equal(ips[2]) {
    t.Fatalf("127.0.0.2 was not greater than 127.0.0.1")
  }
}
