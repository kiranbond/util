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
  "fmt"
  "io/ioutil"
  "os"

  "github.com/golang/glog"
)

const monitorTestFolder string = "/tmp/monitor_test"

// IncrementLifeCount increments the life count of the program in the file
// by 1. The file is created under monitorTestFolder if not present.
func IncrementLifeCount(name string, expectedCount int) error {
  if err := os.MkdirAll(monitorTestFolder, os.FileMode(0744)); err != nil {
    return err
  }
  filename := fmt.Sprintf("%s/%s", monitorTestFolder, name)
  // create file if not present
  if _, err := os.Stat(filename); err != nil {
    if os.IsNotExist(err) {
      glog.Infof("creating file %s", filename)
      file, errCreate := os.Create(filename)
      if errCreate != nil {
        return errCreate
      }
      file.Close()
    } else {
      glog.Errorf("error doing a stat on file: %s :: %v", filename, err)
      return err
    }
  }
  // open the file and update
  buf, errRead := ioutil.ReadFile(filename)
  if errRead != nil {
    glog.Errorf("error reading file %s :: %v", filename, errRead)
    return errRead
  }
  var count, expected int
  if len(buf) > 0 {
    numParsed, err := fmt.Sscanf(string(buf), "%d %d", &count, &expected)
    glog.Info("buffer from file: ", string(buf))
    if numParsed != 2 || err != nil {
      glog.Errorf("error parsing string, numParsed = %d :: %v", numParsed, err)
      return err
    }
  }
  expected = expectedCount
  count++
  glog.Infof("count = %d expected = %d", count, expected)
  writeStr := fmt.Sprintf("%d %d", count, expected)

  wBuf := []byte(writeStr)
  errWrite := ioutil.WriteFile(filename, wBuf, 0644)
  if errWrite != nil {
    glog.Errorf("error writing to file %s :: %v", filename, errWrite)
    return errWrite
  }
  return nil
}
