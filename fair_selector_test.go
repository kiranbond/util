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
  "fmt"
  "testing"
)

func TestFairAlloc(t *testing.T) {
  buckets := []Bucket{
    {"h1", []BucketValue{{"v1", 800}, {"v2", 500}, {"v3", 1000}}},
    {"h2", []BucketValue{{"v3", 800}, {"v4", 1000}, {"v5", 500}}},
    {"h3", []BucketValue{{"v6", 800}, {"v7", 500}}},
    {"h4", []BucketValue{{"v6", 800}, {"v7", 500}}}}

  target := int64(3500)
  fmt.Printf("\nbuckets %v target %d\n", buckets, target)
  fs := NewFairSelector(buckets)
  size, result := fs.Select(target)
  fmt.Printf("result %v\nsize %d\n", result, size)

  target = int64(6000)
  buckets = []Bucket{
    {"h1", []BucketValue{{"v1", 500}, {"v2", 800}}},
    {"h2", []BucketValue{{"v3", 500}, {"v4", 800}}},
    {"h3", []BucketValue{{"v5", 500}, {"v6", 800},
      {"v7", 800}, {"v8", 800}, {"v9", 800}, {"v10", 800}}},
    {"h4", []BucketValue{{"v11", 500}, {"v12", 800},
      {"v13", 800}, {"v14", 800}, {"v15", 800}, {"v16", 800}}}}

  fmt.Printf("\nbuckets %v target %d\n", buckets, target)
  fs = NewFairSelector(buckets)
  size, result = fs.Select(target)
  fmt.Printf("result %v\nsize %d\n", result, size)

  target = int64(10000)
  buckets = []Bucket{
    {"h1", []BucketValue{{"v1", 500}, {"v2", 800}}},
    {"h2", []BucketValue{{"v3", 500}, {"v4", 800}}},
    {"h3", []BucketValue{{"v5", 500}, {"v6", 800},
      {"v7", 800}, {"v8", 800}, {"v9", 800}, {"v10", 800}}},
    {"h4", []BucketValue{{"v11", 500}, {"v12", 800},
      {"v13", 800}, {"v14", 800}, {"v15", 800}, {"v16", 800}}}}

  fmt.Printf("\nbuckets %v target %d\n", buckets, target)
  fs = NewFairSelector(buckets)
  size, result = fs.Select(target)
  fmt.Printf("result %v\nsize %d\n", result, size)

  fmt.Printf("size calculations for ssd combination 1")
  // This target gives max possible allocation for local pool on 4 nodes.
  target = int64(2371731353600) // size target for relhighiops
  buckets = []Bucket{
    {"h1", []BucketValue{{"v1", 592932838400}, {"v2", 799091268608}}},
    {"h2", []BucketValue{{"v3", 592932838400}, {"v4", 799091268608}}},
    {"h3", []BucketValue{{"v5", 592932838400}, {"v6", 799091268608}}},
    {"h4", []BucketValue{{"v7", 592932838400}, {"v8", 799091268608}}},
  }

  fmt.Printf("\nbuckets %v target %d\n", buckets, target)
  fs = NewFairSelector(buckets)
  size, result = fs.Select(target)
  fmt.Printf("result %v\nsize %d target %d difference %d\n", result, size, target, (target - size))

  fmt.Printf("size calculations for hdd combination 1")
  // This target gives max possible allocation for local pool on 4 nodes.
  target = int64(3996520312832) // size target for relhighicap
  buckets = []Bucket{
    {"h1", []BucketValue{{"v11", 999130078208}, {"v12", 999130078208}, {"v13", 999130078208}, {"v14", 999130078208}}},
    {"h1", []BucketValue{{"v21", 999130078208}, {"v22", 999130078208}, {"v23", 999130078208}, {"v24", 999130078208}}},
    {"h1", []BucketValue{{"v31", 999130078208}, {"v32", 999130078208}, {"v33", 999130078208}, {"v34", 999130078208}}},
    {"h1", []BucketValue{{"v41", 999130078208}, {"v42", 999130078208}, {"v43", 999130078208}, {"v44", 999130078208}}},
  }

  fmt.Printf("\nbuckets %v target %d\n", buckets, target)
  fs = NewFairSelector(buckets)
  size, result = fs.Select(target)
  fmt.Printf("result %v\nsize %d target %d difference %d\n", result, size, target, (target - size))

  fmt.Printf("size calculations for ssd combination 2")
  target = int64(3376981052416) // size target for relhighiops
  buckets = []Bucket{
    {"h1", []BucketValue{{"v1", 592932838400}, {"v2", 799091268608}}},
    {"h2", []BucketValue{{"v3", 592932838400}, {"v4", 799091268608}}},
    {"h3", []BucketValue{{"v5", 592932838400}, {"v6", 799091268608}}},
    {"h4", []BucketValue{{"v7", 592932838400}, {"v8", 799091268608}}},
  }

  fmt.Printf("\nbuckets %v target %d\n", buckets, target)
  fs = NewFairSelector(buckets)
  size, result = fs.Select(target)
  fmt.Printf("result %v\nsize %d target %d difference %d\n", result, size, target, (target - size))

  fmt.Printf("size calculations for hdd combination 2")
  target = int64(9991300782080) // size target for relhighicap
  buckets = []Bucket{
    {"h1", []BucketValue{{"v11", 999130078208}, {"v12", 999130078208}, {"v13", 999130078208}, {"v14", 999130078208}}},
    {"h1", []BucketValue{{"v21", 999130078208}, {"v22", 999130078208}, {"v23", 999130078208}, {"v24", 999130078208}}},
    {"h1", []BucketValue{{"v31", 999130078208}, {"v32", 999130078208}, {"v33", 999130078208}, {"v34", 999130078208}}},
    {"h1", []BucketValue{{"v41", 999130078208}, {"v42", 999130078208}, {"v43", 999130078208}, {"v44", 999130078208}}},
  }

  fmt.Printf("\nbuckets %v target %d\n", buckets, target)
  fs = NewFairSelector(buckets)
  size, result = fs.Select(target)
  fmt.Printf("result %v\nsize %d target %d difference %d\n", result, size, target, (target - size))

  fmt.Printf("size calculations for ssd combination 3")
  // This target gives max possible allocation for replicated pool on 4 nodes.
  target = int64(5568096428032) // size target for relhighiops
  buckets = []Bucket{
    {"h1", []BucketValue{{"v1", 592932838400}, {"v2", 799091268608}}},
    {"h2", []BucketValue{{"v3", 592932838400}, {"v4", 799091268608}}},
    {"h3", []BucketValue{{"v5", 592932838400}, {"v6", 799091268608}}},
    {"h4", []BucketValue{{"v7", 592932838400}, {"v8", 799091268608}}},
  }

  fmt.Printf("\nbuckets %v target %d\n", buckets, target)
  fs = NewFairSelector(buckets)
  size, result = fs.Select(target)
  fmt.Printf("result %v\nsize %d target %d difference %d\n", result, size, target, (target - size))

  fmt.Printf("size calculations for hdd combination 3")
  // This target gives max possible allocation for replicated pool on 4 nodes.
  target = int64(15986081251328) // size target for relhighicap
  buckets = []Bucket{
    {"h1", []BucketValue{{"v11", 999130078208}, {"v12", 999130078208}, {"v13", 999130078208}, {"v14", 999130078208}}},
    {"h1", []BucketValue{{"v21", 999130078208}, {"v22", 999130078208}, {"v23", 999130078208}, {"v24", 999130078208}}},
    {"h1", []BucketValue{{"v31", 999130078208}, {"v32", 999130078208}, {"v33", 999130078208}, {"v34", 999130078208}}},
    {"h1", []BucketValue{{"v41", 999130078208}, {"v42", 999130078208}, {"v43", 999130078208}, {"v44", 999130078208}}},
  }

  fmt.Printf("\nbuckets %v target %d\n", buckets, target)
  fs = NewFairSelector(buckets)
  size, result = fs.Select(target)
  fmt.Printf("result %v\nsize %d target %d difference %d\n", result, size, target, (target - size))

}
