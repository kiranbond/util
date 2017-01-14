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
  "bytes"
  "encoding/binary"
  "encoding/json"
  "testing"

  "github.com/stretchr/testify/assert"
)

// We cannot use a protofile to test this since it causes an import cycle so
// let us use a custom struct with Marshal and Unmarshal methods.
type Info struct {
  Key   int32
  Value int32
}

func (i *Info) Marshal() ([]byte, error) {
  buf := new(bytes.Buffer)
  err := binary.Write(buf, binary.LittleEndian, i.Key)
  if err != nil {
    return nil, err
  }
  err = binary.Write(buf, binary.LittleEndian, i.Value)
  if err != nil {
    return nil, err
  }
  return buf.Bytes(), nil
}

func (i *Info) Unmarshal(data []byte) error {
  buf := bytes.NewBuffer(data)
  err := binary.Read(buf, binary.LittleEndian, &i.Key)
  if err != nil {
    return err
  }
  err = binary.Read(buf, binary.LittleEndian, &i.Value)
  if err != nil {
    return err
  }
  return nil
}

func (i *Info) MarshalTextString() ([]byte, error) {
  bytes, err := json.Marshal(i)
  if err != nil {
    return nil, err
  }
  return bytes, nil
}

func (i *Info) UnmarshalTextString(data []byte) error {
  err := json.Unmarshal(data, i)
  if err != nil {
    return err
  }
  return nil
}

// TestSnapshot tests the Snapshot functionality by saving and loading the
// Info object to a temp file.
func TestSnapshot(t *testing.T) {

  sn, err := NewSnapshot("/tmp/zerostack_snapshot_test", "test1")
  assert.Nil(t, err)
  assert.NotNil(t, sn)

  info := Info{Key: 1001, Value: 0220}

  err = sn.Save(&info)
  assert.Nil(t, err)

  expInfo := Info{}
  err = sn.Load(&expInfo)
  assert.Nil(t, err)

  assert.Equal(t, info, expInfo)

  err = sn.SaveText(&info)
  assert.Nil(t, err)

  expInfoText := Info{}
  err = sn.LoadText(&expInfoText)
  assert.Nil(t, err)

  assert.Equal(t, info, expInfoText)

  err = sn.Delete()
  assert.Nil(t, err)
}
