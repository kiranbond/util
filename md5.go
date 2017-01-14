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
// This file implements utility functions for md5.

package util

import (
  "crypto/md5"
  "encoding/hex"
)

// BytesMD5 hashes a provided []byte using MD5 and returns the hash encoded as
// a string.
func BytesMD5(bytes []byte) string {
  h := md5.New()
  h.Write(bytes)
  return hex.EncodeToString(h.Sum(nil))
}

// StringMD5 hashes a provided string using MD5 and returns the hash encoded as
// a string.
func StringMD5(text string) string {
  h := md5.New()
  h.Write([]byte(text))
  return hex.EncodeToString(h.Sum(nil))
}
