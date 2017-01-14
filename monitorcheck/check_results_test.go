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

package monitorcheck

import (
  "flag"
  "fmt"
  "io/ioutil"
  "os"
  "testing"

  "github.com/golang/glog"
  "github.com/stretchr/testify/assert"
)

func TestMonitorCounts(t *testing.T) {
  flag.Set("v", "2")
  flag.Set("stderrthreshold", "0")

  // go through all the files in test directory and check that the count is same
  // as expected
  defer os.RemoveAll(monitorTestFolder)
  files, err := ioutil.ReadDir(monitorTestFolder)
  assert.Nil(t, err)
  for _, file := range files {
    filename := fmt.Sprintf("%s/%s", monitorTestFolder, file.Name())
    buf, errRead := ioutil.ReadFile(filename)
    assert.Nil(t, errRead)

    var count, expected int
    assert.NotEqual(t, len(buf), 0)
    numParsed, errParse := fmt.Sscanf(string(buf), "%d %d", &count, &expected)
    glog.V(2).Infof("buffer from file: %s buf: %s", file.Name(), string(buf))

    assert.Nil(t, errParse)
    assert.Equal(t, numParsed, 2)
    assert.Equal(t, count, expected)
  }
}
