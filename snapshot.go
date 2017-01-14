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
// Snapshot implements a simple interface to save and load a struct to disk in
// a pre-configured file. The input is written to a tmp file before copying
// over to the real file. The Save and Load methods expect the struct to have
// Marshal and Unmarshal methods.

package util

import (
  "fmt"
  "io/ioutil"
  "os"

  "github.com/golang/glog"
)

// Marshaler is the interface implemented by objects that
// can marshal and unmarshal themselves.
type Marshaler interface {
  Marshal() ([]byte, error)
  Unmarshal([]byte) error
}

// TextMarshaler is the interface implemented by objects that
// can marshal and unmarshal themselves.
type TextMarshaler interface {
  MarshalTextString() ([]byte, error)
  UnmarshalTextString([]byte) error
}

// Snapshot saves the params for the file where the snapshot is stored.
type Snapshot struct {
  dir      string
  fileName string
  path     string
}

// NewSnapshot returns a Snapshot object using the dir and name of the file.
func NewSnapshot(dir string, fileName string) (*Snapshot, error) {
  snap := &Snapshot{dir: dir, fileName: fileName, path: dir + "/" + fileName}
  err := os.MkdirAll(dir, os.FileMode(0777))
  if err != nil {
    glog.Errorf("could not create directory : %s", dir)
    return nil, err
  }
  return snap, nil
}

// Exists checks if the snapshot file exists.
func (s *Snapshot) Exists() bool {
  stat, err := os.Stat(s.path)
  if err != nil || stat.IsDir() {
    return false
  }
  return true
}

// Save marshals the data and writes the snapshot top disk.
func (s *Snapshot) Save(item Marshaler) error {
  data, err := item.Marshal()
  if err != nil {
    glog.Errorf("could not marshal snapshot item :: %v", err)
    return err
  }
  return s.saveData(data)
}

// SaveText marshals the data as string and writes the snapshot to disk.
func (s *Snapshot) SaveText(item TextMarshaler) error {
  data, err := item.MarshalTextString()
  if err != nil {
    glog.Errorf("could not marshal snapshot item :: %v", err)
    return err
  }
  return s.saveData(data)
}

// saveData writes the marshaled data of the struct to disk using a tempfile.
func (s *Snapshot) saveData(data []byte) error {
  tempfile, err := ioutil.TempFile(s.dir, s.fileName)
  if err != nil {
    glog.Errorf("error creating temp file :: %v", err)
    return err
  }
  defer func() {
    _ = os.Remove(tempfile.Name())
  }()

  _, err = tempfile.Write(data)
  if err != nil {
    glog.Errorf("could not write to file :: %v", err)
    return err
  }

  tempfile.Close()

  if err = os.Rename(tempfile.Name(), s.path); err != nil {
    glog.Errorf("error renaming %s to %s :: %v", tempfile.Name(), s.path, err)
    return err
  }

  return nil
}

// Load loads the snapshot from file and unmarshals the bytes into the
// struct in the interface.
func (s *Snapshot) Load(item Marshaler) error {
  buf, err := s.loadData()
  if err != nil || buf == nil {
    glog.Errorf("error reading data from snapshot file :: %v", err)
    return err
  }
  err = item.Unmarshal(buf)
  if err != nil {
    glog.Errorf("error unmarshaling into param :: %v", err)
    return err
  }
  return nil
}

// LoadText loads the snapshot string from file and unmarshals the bytes into
// the struct in the interface.
func (s *Snapshot) LoadText(item TextMarshaler) error {
  buf, err := s.loadData()
  if err != nil || buf == nil {
    glog.Warningf("error reading data from snapshot file :: %v", err)
    return err
  }
  err = item.UnmarshalTextString(buf)
  if err != nil {
    glog.Errorf("error unmarshaling into param :: %v", err)
    return err
  }
  return nil
}

func (s *Snapshot) loadData() ([]byte, error) {
  file, err := os.OpenFile(s.path, os.O_RDONLY, os.FileMode(0777))
  if err != nil {
    glog.Warningf("could not read snapshot file : %s :: %v", s.path, err)
    return nil, err
  }

  fs, err := file.Stat()
  if err != nil {
    glog.Errorf("could not stat file : %s :: %v", s.path, err)
    return nil, err
  }

  size := fs.Size()

  glog.V(2).Infof("reading %d bytes from snapshot %s", size, s.path)

  buf := make([]byte, size)
  cnt, err := file.Read(buf)
  if err != nil {
    glog.Errorf("error reading from snapshot : %s :: %v", s.path, err)
    return nil, err
  }
  if int64(cnt) < size {
    glog.Errorf("error reading from snapshot : %s, expected %d bytes read %d",
      s.path, size, cnt)
    return nil, fmt.Errorf("error reading file")
  }

  return buf, nil
}

// Delete deletes the snapshot file.
func (s *Snapshot) Delete() error {
  return os.Remove(s.path)
}
