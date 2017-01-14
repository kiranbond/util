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
// SyncMap - SynchronizedMap aka ConcurrentMap).

package util

import (
  "sync"
)

// SyncMap implements a thread-safe map[interface{}]interface{}. You can put a
// key and value of any type into it.
type SyncMap struct {
  mu sync.RWMutex
  m  map[interface{}]interface{}
}

// NewSyncMap initializes and returns a new SyncMap.
func NewSyncMap() *SyncMap {
  m := make(map[interface{}]interface{})
  return &SyncMap{m: m}
}

// Add adds the key and val to the SyncMap
func (s *SyncMap) Add(key interface{}, val interface{}) {
  s.mu.Lock()
  defer s.mu.Unlock()
  s.m[key] = val
}

// Get looks up the key in the map and returns the val+found pair.
func (s *SyncMap) Get(key interface{}) (val interface{}, ok bool) {
  s.mu.RLock()
  defer s.mu.RUnlock()
  val, ok = s.m[key]
  return val, ok
}

// CloneToMap returns a map of all keys and values in the sync map.
func (s *SyncMap) CloneToMap() map[interface{}]interface{} {
  out := make(map[interface{}]interface{})
  s.mu.RLock()
  defer s.mu.RUnlock()
  for k, v := range s.m {
    out[k] = v
  }
  return out
}

// Len returns the length of the map.
func (s *SyncMap) Len() int {
  s.mu.RLock()
  defer s.mu.RUnlock()
  return len(s.m)
}

// Del deletes the key from the SyncMap and returns the val+found pair.
func (s *SyncMap) Del(key interface{}) (val interface{}, ok bool) {
  s.mu.Lock()
  defer s.mu.Unlock()
  val, ok = s.m[key]
  delete(s.m, key)
  return val, ok
}

// Clear empties the map.
func (s *SyncMap) Clear() {
  s.mu.Lock()
  defer s.mu.Unlock()
  for key := range s.m {
    delete(s.m, key)
  }
}

// Replace calls the function with the value for the given key and replaces it
// with the returned value if the function returns true for second param.
//
// This is different from Add since the value can be RMW under a single write
// lock rather than Get followed by Add under two locks.
//
// The value and a bool are returned. If the key was found the bool param is
// true.
func (s *SyncMap) Replace(key interface{},
  replace func(interface{}) (interface{}, bool)) (interface{}, bool) {

  s.mu.Lock()
  defer s.mu.Unlock()

  if val, ok := s.m[key]; ok {
    newval, do := replace(val)
    if do {
      s.m[key] = newval
      return newval, true
    }
    return val, true
  }
  return nil, false
}

// ReplaceParam calls the function with the value and the provided param for the
// given key and replaces it with the returned value if the function returns
// true via its second OUT param.
//
// This is different from Add since the value can be RMW under a single write
// lock rather than Get followed by Add under two locks.
//
// The value and a bool are returned. If the key was found the bool param is
// true.
func (s *SyncMap) ReplaceParam(key interface{},
  replaceParam func(interface{}, interface{}) (interface{}, bool),
  param interface{}) (interface{}, bool) {

  s.mu.Lock()
  defer s.mu.Unlock()

  if val, ok := s.m[key]; ok {
    newval, do := replaceParam(val, param)
    if do {
      s.m[key] = newval
      return newval, true
    }
    return val, true
  }
  return nil, false
}

// FindFunc iterates over the map with the user supplied find function.
// It stops when the find function returns true.
// FindFunc can be used to do search for custom fields inside the stored value.
func (s *SyncMap) FindFunc(find func(interface{}, interface{}) bool) (
  interface{}, interface{}) {

  s.mu.RLock()
  defer s.mu.RUnlock()

  for key, val := range s.m {
    if find(key, val) {
      return key, val
    }
  }
  return nil, nil
}

// FindReplaceFunc iterates over the map and replaces the value for every key
// with the supplied new value if the replace function returns true for second
// return value.
func (s *SyncMap) FindReplaceFunc(replace func(interface{}) (interface{},
  bool)) error {

  s.mu.Lock()
  defer s.mu.Unlock()

  for key, val := range s.m {
    if newval, ok := replace(val); ok {
      s.m[key] = newval
    }
  }
  return nil
}

// CmpFunc iterates over the map with the user supplied compare function.
// Can be used to do Min, Max etc. for custom fields inside the stored value.
// CmpFunc will pass the elements of the map pairwise to the func and the
// function should return true or false for each pair.
func (s *SyncMap) CmpFunc(
  cmp func(interface{}, interface{}) bool) (interface{}, interface{}) {

  s.mu.RLock()
  defer s.mu.RUnlock()

  var goldkey interface{}
  var gold interface{}
  first := true
  for key, val := range s.m {
    if first {
      goldkey = key
      gold = val
      first = false
    } else if cmp(val, gold) {
      goldkey = key
      gold = val
    }
  }
  return goldkey, gold
}
