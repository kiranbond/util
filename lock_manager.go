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
// lock manager manages the pool for mutexs.

package util

import (
  "fmt"
  "sync"
)

// LockManager represents the struct to manage mutexs.
type LockManager struct {
  lockMap map[int]*sync.Mutex
}

var errLockIndex = fmt.Errorf("lock index out of range")

// NewLockManager returns an instance of lockManager.
func NewLockManager(size int) *LockManager {
  lockMap := make(map[int]*sync.Mutex)
  for i := 0; i < size; i++ {
    lockMap[i] = new(sync.Mutex)
  }
  return &LockManager{lockMap: lockMap}
}

// GetLock grabs lock for a hash value.
func (lm *LockManager) GetLock(hash int) {
  lm.lockMap[hash%lm.getSize()].Lock()
}

// ReleaseLock releases lock for a hash value.
func (lm *LockManager) ReleaseLock(hash int) {
  lm.lockMap[hash%lm.getSize()].Unlock()
}

// getSize returns the size of lock manager.
func (lm *LockManager) getSize() int {
  return len(lm.lockMap)
}
