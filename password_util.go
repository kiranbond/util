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

  "code.google.com/p/go.crypto/bcrypt"
  "github.com/golang/glog"
)

// CreateHash creates a password hash using a random salt that is stored
// with the hash
func CreateHash(password *string) (*string, error) {

  pswdInBytes := []byte(*password)

  // Hashing the password with the cost of 10
  hashInBytes, err := bcrypt.GenerateFromPassword(pswdInBytes, 10)
  if err != nil {
    glog.Errorf("error creating password hash :: %v", err)
    return nil, err
  }
  hashedPassword := string(hashInBytes)
  return &hashedPassword, nil
}

// CheckPassword checks the actual password against the hash and returns
// true or false
func CheckPassword(hashedPassword *string, password *string) error {
  if hashedPassword == nil || len(*hashedPassword) == 0 {
    return errors.New("no password hash entered")
  }

  if password == nil || len(*password) == 0 {
    return errors.New("no password entered")
  }
  // Comparing the password with the hash
  err := bcrypt.CompareHashAndPassword([]byte(*hashedPassword),
    []byte(*password))
  if err != nil {
    glog.Infof("error validating password :: %v", err)
    return err
  }
  // nil means it is a match
  return nil

}
