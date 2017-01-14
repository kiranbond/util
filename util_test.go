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
  "errors"
  "strings"
  "testing"
  "time"

  "github.com/stretchr/testify/assert"
)

func TestMAC(t *testing.T) {
  mac1, _ := GenerateRandomMAC()
  mac2, _ := GenerateRandomMAC()

  assert.NotEqual(t, mac1, mac2, "two generated MACs should not be identical")
  assert.NotEqual(t, mac1.String(), "00:00:00:00:00:00",
    "generated MACs are not zero")

  const testMAC string = "11:AA:22:BB:DE:AD"
  mac3, macErr := ParseMAC(testMAC)
  assert.NoError(t, macErr, "test MAC should parse")
  assert.Equal(t, mac3.String(), strings.ToLower(testMAC))

  mac3, macErr = ParseMAC("notamac")
  assert.Error(t, macErr, "invalid string should not parse")
  assert.Nil(t, mac3)

  mac3, macErr = ParseMAC("11:AA:22:BB:DE:GG")
  assert.Error(t, macErr, "invalid mac should not parse")
  assert.Nil(t, mac3)

  mac3, macErr = ParseMAC("11:AA:22:BB:DE:AD:BE:EF")
  assert.Error(t, macErr, "too long mac should not parse")
  assert.Nil(t, mac3)
}

func TestParseIP(t *testing.T) {
  str1 := "tcp://10.150.30.1:10000/?uuid=021d4743-2319-4535-a4ac-a59a1802bc70"
  ipStr1 := ParseIP(str1)
  assert.Equal(t, "10.150.30.1", ipStr1)

  str2 := "tcp://10.20.24.31:10000/"
  ipStr2 := ParseIP(str2)
  assert.Equal(t, "10.20.24.31", ipStr2)

  str3 := "foobar"
  ipStr3 := ParseIP(str3)
  assert.Equal(t, "", ipStr3)

}

func TestWaitForFuncToSucceedReturnErrorMessage(t *testing.T) {
  fnLabel := "LABEL"
  errorMessage := "ERROR"
  _, err := WaitForFuncToSucceed(fnLabel,
    func() (interface{}, error) {
      return 0, errors.New(errorMessage)
    },
    time.Millisecond,
    time.Millisecond)

  assert.Equal(t, "timed out waiting for "+fnLabel+" to succeed :: "+errorMessage, err.Error())
}
