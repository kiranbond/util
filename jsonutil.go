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
  "strconv"
  "strings"
)

// This file implements JSON utility functions.

// ExtractFieldValue extracts value stored at a path in a decoded JSON object.
// For ex. from a JSON object below, extractFieldValue(jsn, "a[0].b") will
// extract value 1. It will be used to extract values when you are dealing with
// variable data structure from an external API.
// jsn = {"a":[
//         {
//          "b": 1
//         },
//         {}....
//       }
func ExtractFieldValue(obj interface{}, path string) (interface{}, error) {
  if path == "" {
    return nil, fmt.Errorf("path empty")
  }
  jsn, ok := obj.(map[string]interface{})
  if !ok {
    return nil, fmt.Errorf("error: found leaf node earlier than expected")
  }
  i := strings.IndexAny(path, "[.")
  if i < 0 {
    // at leaf node in the path, extract the val and return
    val, ok := jsn[path]
    if ok {
      return val, nil
    }
    return nil, fmt.Errorf("error finding path %s", path)
  }
  // we have encountered either a . or an opening bracket [
  switch path[i] {
  case '[':
    // encountered an array type object
    j := strings.Index(path[i+1:], "]")
    if j < 0 {
      return nil, fmt.Errorf("error closing bracket missing")
    }
    n, err := strconv.ParseInt(path[i+1:i+j+1], 10, 64)
    if err != nil {
      return nil, err
    }
    first := path[:i]
    items, ok := jsn[first].([]interface{})
    if !ok {
      return nil, fmt.Errorf("error did not find expected slice")
    }
    if int(n) > len(items)-1 {
      return nil, fmt.Errorf("error index out of bound")
    }
    if len(path) == i+j+2 {
      return items[n], nil
    }
    rest := path[i+j+3:]
    return ExtractFieldValue(items[n], rest)
  case '.':
    // TODO: Add support for escaping . character in path
    first, rest := path[:i], path[i+1:]
    return ExtractFieldValue(jsn[first], rest)
  }
  return nil, fmt.Errorf("value does not exist")
}
