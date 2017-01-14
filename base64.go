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
// This file implements utility functions for base64.

package util

import (
  "encoding/base64"
)

// Base64Encode encodes the provided string into base64.
func Base64Encode(str string) string {
  return base64.StdEncoding.EncodeToString([]byte(str))
}

// Base64Decode decoded the given string from base64 to string.
func Base64Decode(str string) (string, error) {
  data, err := base64.StdEncoding.DecodeString(str)
  if err != nil {
    return "", err
  }
  return string(data), nil
}
