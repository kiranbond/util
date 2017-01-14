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

// This file implements tests for functionality defined in jsonutil.

import (
  "encoding/json"
  "strings"
  "testing"

  "github.com/stretchr/testify/assert"
)

func TestExtractFieldValue(t *testing.T) {
  tt := []struct {
    jsn    string
    path   string
    expVal interface{}
  }{
    {`{"a": 1}`, "a", float64(1)},
    {`{"a": {"b": {"c": 2}}}`, "a.b.c", float64(2)},
    {`{"a": [{}, {"b": "wow"}] }`, "a[1].b", "wow"},
    {`{"a": [1]}`, "a", []interface{}{float64(1)}},
  }

  for _, row := range tt {
    var jsn interface{}
    err := json.Unmarshal([]byte(row.jsn), &jsn)
    assert.Nil(t, err)
    got, err := ExtractFieldValue(jsn, row.path)
    assert.Nil(t, err)
    assert.Equal(t, got, row.expVal)
  }
}

func TestExtractFieldValueErrors(t *testing.T) {
  tt := []struct {
    jsn    string
    path   string
    expErr string
  }{
    {`{"a": 1}`, "a.b", "node earlier"},
    {`{"a": 1}`, "a.", "path empty"},
    {`{"a": 1}`, "a[10]", "did not find expected slice"},
    {`{"a": 1}`, "a[a]", "parsing"},
    {`{"a": {"b": {"c": 2}}}`, "a.c", "error finding path"},
    {`{"a": [{}, {"b": 2}] }`, "a[10].b", "error index out of bound"},
    {`{"a": [{}, {"b": 2}] }`, "a[1b", "error closing bracket missing"},
  }

  for _, row := range tt {
    var jsn interface{}
    err := json.Unmarshal([]byte(row.jsn), &jsn)
    assert.Nil(t, err)
    _, err = ExtractFieldValue(jsn, row.path)
    assert.NotNil(t, err)
    assert.True(t, strings.Contains(err.Error(), row.expErr))
  }
}
