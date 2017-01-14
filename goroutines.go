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

// Helper methods to determine goroutine profile information.

package util

import (
  "fmt"
  "io/ioutil"
  "net/http"
  "strings"
)

// GoroutineProfile returns a string dump containing the goroutine profile of
// the provided service at the provided addresss.
func GoroutineProfile(svcAddr string, profPort int) (string, error) {
  profURL := fmt.Sprintf("http://%s:%d/debug/pprof/goroutine?debug=3",
    strings.TrimPrefix(svcAddr, "http://"), profPort)

  rsp, err := http.Get(profURL)
  if err != nil {
    return "", fmt.Errorf("error making GET call to profile port :: %v", err)
  }
  if rsp == nil {
    return "", fmt.Errorf("nil response")
  }
  if rsp.Body == nil {
    return "", fmt.Errorf("nil response body")
  }

  defer rsp.Body.Close()

  if rsp.StatusCode != http.StatusOK {
    return "", fmt.Errorf("unexpected status code in response :: %v",
      rsp.StatusCode)
  }
  body, err := ioutil.ReadAll(rsp.Body)
  if err != nil {
    return "", fmt.Errorf("error reading response body :: %v", err)
  }
  return string(body), nil
}
