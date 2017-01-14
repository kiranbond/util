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

package monitortest2

import (
  "flag"
  "fmt"
  "net/http"
  "os"
  "testing"
  "time"

  "github.com/golang/glog"
  "github.com/stretchr/testify/assert"

  "zerostack/common/util"
  "zerostack/common/util/monitorcheck"
)

var numGetRequests = 0

func flakyHealthzHandler(w http.ResponseWriter, r *http.Request) {
  glog.V(2).Infof("child pid: %d alive at %s!", os.Getpid(), r.URL.Path[1:])
  numGetRequests++
  if numGetRequests > 1 {
    // this should cause a timeout and child restart
    time.Sleep(5 * time.Second)
  }
  fmt.Fprintf(w, "child pid: %d alive at %s!", os.Getpid(), r.URL.Path[1:])
}

func TestRestartTwice(t *testing.T) {
  flag.Set("v", "2")
  flag.Set("stderrthreshold", "0")

  *util.PingInterval = 2 * time.Second
  *util.ChildTimeout = time.Second
  *util.MaxRestarts = 2
  *util.NumRetries = 1
  monitoringURL := "http://127.0.0.1:53881/healthz"

  if err := util.SetProgramMonitor(os.Args, monitoringURL); err != nil {
    glog.Errorf("could not start parent for:%v :: %v", os.Args, err)
    assert.True(t, false)
  }

  glog.V(2).Infof("child running : %v", os.Args)
  http.HandleFunc("/healthz", flakyHealthzHandler)
  go util.HTTPListenAndServe4(":53881", nil)
  err := monitorcheck.IncrementLifeCount("TestRestartTwice", 3)
  assert.Nil(t, err)

  select {}
}
