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
  "fmt"
  "reflect"
  "sync"

  "github.com/golang/glog"
)

// This file defines the IDSetLock struct that can be used to obtain locks on
// a set of idSt. A slice of idSt of some type like int, int64, string etc.
// needs to be converted to a slice of interfaces. To avoid deadlocks, the locks
// should either be taken in a certain order, or else all at once. The lock
// releases for the various idSt can be done in any order. Locks can either be
// acquired synchronously or asynchronously.  In async case a channel is used to
// signal that the locks are acquired.
//
// Lock acquisition is starvation free and is performed in a mostly FIFO order.
//
// Thread-safety: All methods of this class are thread-safe.
//
// Example usage:
//
// (1) Synchronous lock acquire:
//
//     idsl = NewIDSetLocker()
//     // Acquire a shared lock on id set (1, 2) and an exclusive lock on
//     // id set (3, 4).
//     // Look at IntToInterface() function to see how to call this using
//     // int slices. Similar function can be used for other types.
//     idsl.Acquire({1, 2}, {3, 4})
//     ...
//     // Release locks on (1, 2, 3) first and on 4 later.
//     // Look at IntToInterface() function to see how to call this using
//     // int slices. Similar function can be used for other types.
//     idsl.Release({1, 2, 3})
//     ...
//     idsl.Release(4)
//
// (2) Asynchronous lock acquire:
//
//     // create a channel that is signaled when locks are acquired.
//     done := make(chan bool, 1)
//
//     idsl.AcquireAsync({1, 2}, {3, 4}, done)
//
//     // The caller can then block on done channel at some point after
//     // calling AcquireAsyn, or in a separate go routine.
//     go continueAfterLock() {
//       <-done
//       .... // do the rest of the work here
//     }
//

// IDSetLocker defines the interface provided by an IDSetLock.
type IDSetLocker interface {
  Acquire(sharedIDList []interface{}, exclusiveIDList []interface{})
  AcquireStr(sharedIDList []string, exclusiveIDList []string)
  AcquireInt64(sharedIDList []int64, exclusiveIDList []int64)
  AcquireAsync(sharedIDList []interface{}, exclusiveIDList []interface{},
    done chan bool)
  AcquireAll()
  ReleaseAll()
  IsEmpty() bool
  Release(idList []interface{})
  ReleaseStr(idList []string)
  ReleaseInt64(idList []int64)
}

// IDSetLock is the main object that provides the functionality.
type IDSetLock struct {
  mu         sync.RWMutex             // lock that protects the shared variables
  idType     reflect.Kind             // data type of ids that will be used
  allLckCond *sync.Cond               // condition variable used to lock all ids
  allLocked  bool                     // true if all IDs are locked
  allIDList  []interface{}            // list of all locks AcquireAll waits on
  idMap      map[interface{}]*idState // mapping from ID to the state (idState)
}

// waiterState keeps the state of the threads waiting to acquire locks
type waiterState struct {
  locked       chan bool     // used to signal once all locks are acquired
  sharedIds    []interface{} // list of shared ids to be locked
  exclusiveIds []interface{} // list of exclusive ids to be locked
}

// waiterInfo keeps the state of a waiter along with the info if a waiter
// is requesting any exclusive lock
type waiterInfo struct {
  wsPtr       *waiterState
  isExclusive bool
}

// idState keeps the state of each id and waiters associated with it
type idState struct {
  // the current lock value, 0 is unlocked, -1 is exclusive locked
  // and a positive number indicates the number of shared locks
  lockValue int
  // list of waiters waiting on this id. We can't use map because we want
  // to preserve the FIFO order and avoid any starvation.
  wInfoList []*waiterInfo
}

func (ids *idState) String() string {
  str := fmt.Sprintf("lockValue: %d\n", ids.lockValue)
  for _, wi := range ids.wInfoList {
    if wi == nil {
      glog.Fatalf("waiter info is nil")
    }
    str += fmt.Sprintf("exclusive: %v locked: %v shIds: %v exIds: %v\n",
      wi.isExclusive, wi.wsPtr.locked, wi.wsPtr.sharedIds, wi.wsPtr.exclusiveIds)
  }
  return str
}

// NewIDSetLocker creates and returns a pointer to IDSetLock.
func NewIDSetLocker(idType reflect.Kind) *IDSetLock {
  if idType != reflect.Int &&
    idType != reflect.Int16 &&
    idType != reflect.Int32 &&
    idType != reflect.Int64 &&
    idType != reflect.String {
    glog.Errorf("unsupported type %v", idType)
    return nil
  }
  idsl := &IDSetLock{
    idType:    idType,
    allLocked: false,
    idMap:     make(map[interface{}]*idState),
  }
  // keeping it outside the initializer, so that we can use the same lock that
  // is part of idsl.
  idsl.allLckCond = sync.NewCond(&(idsl.mu))
  return idsl
}

// Acquire tries to acquire a set of shared and exclusive locks. These two sets
// need to be disjoint, which is caller's responsibility.
// Each set is represented by a slice of interface type.
// The functions blocks if the locks cannot be acquired.
func (sl *IDSetLock) Acquire(sharedIDList []interface{},
  exclusiveIDList []interface{}) {

  // check if all IDs are not locked, wait till that condition becomes false
  sl.mu.Lock()
  for sl.allLocked {
    sl.allLckCond.Wait()
  }

  // check first if all locks can be acquired directly.
  if sl.tryAcquire(sharedIDList, exclusiveIDList, nil) {
    sl.mu.Unlock()
    return
  }

  // prepare waiter state and add it
  ws := &waiterState{
    locked:       make(chan bool, 1),
    sharedIds:    sharedIDList,
    exclusiveIds: exclusiveIDList,
  }
  sl.addWaiterState(ws)
  sl.mu.Unlock()

  // block on a channel until the locks are acquired
  <-ws.locked
}

// addWaiterState adds the waiterInfo structure for all the ids that the
// waiter is trying to acquire.
func (sl *IDSetLock) addWaiterState(ws *waiterState) {
  for _, id := range ws.sharedIds {
    var idSt *idState
    val, ok := sl.idMap[id]
    if !ok {
      idSt = &idState{}
    } else {
      idSt = val
    }
    winfo := &waiterInfo{ws, false}
    idSt.wInfoList = append(idSt.wInfoList, winfo)
  }
  for _, id := range ws.exclusiveIds {
    idSt, ok := sl.idMap[id]
    if !ok {
      idSt = &idState{}
    }
    winfo := &waiterInfo{ws, true}
    idSt.wInfoList = append(idSt.wInfoList, winfo)
  }
}

// tryAcquire checks if all the locks can be acquired and does so if possible
func (sl *IDSetLock) tryAcquire(sharedIDList []interface{},
  exclusiveIDList []interface{}, ws *waiterState) bool {

  // First check if all the locks can be granted. If so, we will move to
  // the next step and record the appropriate changes.
  for _, id := range sharedIDList {
    if reflect.TypeOf(id).Kind() != sl.idType {
      glog.Errorf("type mismatch, input interface type: %v required: %v",
        reflect.TypeOf(id).Kind(), sl.idType)
      // we are simply ignoring the bad input. Can potentially return an error
      return false
    }
    val, ok := sl.idMap[id]
    if !ok {
      continue // no one is holding this lock
    }
    idSt := val
    if idSt.lockValue < 0 {
      return false // this id is locked in exclusive mode
    }
    // Check all waiters for this id. As long as we see requests for shared
    // locks, we are ok. We will stop if we see exclusive waiter or our waiter
    // state
    for _, winfo := range idSt.wInfoList {
      if winfo.wsPtr == ws {
        break
      }
      if winfo.isExclusive {
        return false // someone is waiting on this id for an exclusive lock
      }
    }
  }

  // now let's go through the exclusive id list
  for _, id := range exclusiveIDList {
    if reflect.TypeOf(id).Kind() != sl.idType {
      glog.Errorf("type mismatch, input interface type: %v required: %v",
        reflect.TypeOf(id).Kind(), sl.idType)
      // we are simply ignoring the bad input. Can potentially return an error
      return false
    }
    idSt, ok := sl.idMap[id]
    if !ok {
      continue // no one is holding this lock
    }
    if idSt.lockValue != 0 {
      return false // this id is already locked
    }
    // Check all waiters for this id. If there is any waiter ahead of us, we
    // can't grab the lock at this time.
    if len(idSt.wInfoList) > 0 && idSt.wInfoList[0].wsPtr != ws {
      return false
    }
  }

  // We reach here only if all locks can be granted. Let's modify the
  // structures to grant all the locks.
  for _, id := range sharedIDList {
    idSt, ok := sl.idMap[id]
    if !ok {
      idSt = &idState{}
      sl.idMap[id] = idSt
    }
    idSt.lockValue++ // acquire in shared mode
    glog.V(2).Infof("acquire shared lock: %v lockValue: %d", id, idSt.lockValue)
    if ws != nil && len(idSt.wInfoList) > 0 {
      // remove waiter state from the idSt.wInfoList
      for ii, witer := range idSt.wInfoList {
        if witer.wsPtr == ws {
          copy(idSt.wInfoList[ii:], idSt.wInfoList[ii+1:])
          idSt.wInfoList[len(idSt.wInfoList)-1] = nil
          idSt.wInfoList = idSt.wInfoList[:len(idSt.wInfoList)-1]
          break
        }
      }
    }
  }

  for _, id := range exclusiveIDList {
    idSt, ok := sl.idMap[id]
    if !ok {
      idSt = &idState{}
      sl.idMap[id] = idSt
    }
    idSt.lockValue = -1 // acquire in exclusive mode
    glog.V(2).Infof("acquire excl lock: %v lockValue: %d", id, idSt.lockValue)
    if ws != nil && len(idSt.wInfoList) > 0 {
      // it has to be the first entry in the list, remove it
      if idSt.wInfoList[0].wsPtr != ws {
        glog.Errorf("NEVER HAPPEN: waiter is not the first entry in list")
      } else {
        copy(idSt.wInfoList[0:], idSt.wInfoList[1:])
        idSt.wInfoList[len(idSt.wInfoList)-1] = nil
        idSt.wInfoList = idSt.wInfoList[:len(idSt.wInfoList)-1]
      }
    }
  }
  return true
}

// IsEmpty returns true if no locks are held.
func (sl *IDSetLock) IsEmpty() bool {
  return len(sl.idMap) == 0
}

// AcquireAsync is an async version of Acquire function. It returns immediately
// and signals a channel once the locks are acquired.
func (sl *IDSetLock) AcquireAsync(sharedIDList []interface{},
  exclusiveIDList []interface{}, done chan bool) {
  go sl.acquireAndSignal(sharedIDList, exclusiveIDList, done)
  return
}

// acquireAndSignal is a helper function that acquires a lock and signals the
// passed in channel.
func (sl *IDSetLock) acquireAndSignal(sharedIDList []interface{},
  exclusiveIDList []interface{}, done chan bool) {
  sl.Acquire(sharedIDList, exclusiveIDList)
  done <- true
}

// AcquireAll acquires lock on all possible IDs. It works by acquiring locks on
// all locked IDs and setting a condition variable so that no other lock
// operations will succeed. Once these locks are acquired, they are released
// as part of ReleaseAll() and the condition that all locks are acquired is
// also set to false.
func (sl *IDSetLock) AcquireAll() {
  sl.mu.Lock()

  for sl.allLocked {
    sl.allLckCond.Wait()
  }
  // setting it upfront, since no other call should be able to acquire any
  // locks till this is set to false.
  sl.allLocked = true
  if sl.IsEmpty() {
    sl.allIDList = []interface{}{}
    sl.mu.Unlock()
    return
  }

  // get list of all IDs in the locker
  allIDList := make([]interface{}, len(sl.idMap))
  index := 0
  for key := range sl.idMap {
    allIDList[index] = key
    index++
  }
  sl.allIDList = allIDList
  glog.Infof("trying to lock all ids %v", allIDList)
  // prepare waiter state and add it
  ws := &waiterState{
    locked:       make(chan bool, 1),
    sharedIds:    []interface{}{},
    exclusiveIds: allIDList,
  }
  sl.addWaiterState(ws)
  sl.mu.Unlock()

  // block on a channel until the locks are acquired
  <-ws.locked
  glog.V(0).Infof("acquired all locks %v", allIDList)
}

// Release releases the locks on the ids in the given list.
func (sl *IDSetLock) Release(idList []interface{}) {
  if len(idList) == 0 {
    return
  }
  if reflect.TypeOf(idList[0]).Kind() != sl.idType {
    // this is really a compile time bug, that should be caught asap.
    panic(fmt.Sprintf("type mismatch, input interface type: %v required: %v",
      reflect.TypeOf(idList[0]).Kind(), sl.idType))
  }

  sl.mu.Lock()
  defer sl.mu.Unlock()

  sl.releaseLocked(idList)
}

// releaseLocked releases the locks on the ids in the given list without
// acquiring any locks. The caller should have acquired all the necessary
// locks.
func (sl *IDSetLock) releaseLocked(idList []interface{}) {
  var unlockedList []*idState

  for _, id := range idList {
    idSt, ok := sl.idMap[id]
    if !ok {
      glog.Errorf("caller has gone nuts, the unlock id not in map: %v", id)
      continue // continue, since this can be a caller mistake
    }
    if idSt.lockValue < 0 {
      idSt.lockValue = 0
      glog.V(2).Infof("release excl lock: %v lockValue: %d", id, idSt.lockValue)
    } else if idSt.lockValue > 0 {
      idSt.lockValue--
      glog.V(2).Infof("release shared lock: %v lockValue: %d", id,
        idSt.lockValue)
    } else {
      glog.Fatalf("NEVER HAPPEN: lockValue is zero for a lock in map: %v", id)
    }
    if idSt.lockValue == 0 {
      if len(idSt.wInfoList) == 0 {
        // this id is unused, remove from map
        delete(sl.idMap, id)
      } else {
        unlockedList = append(unlockedList, idSt)
      }
    }
  }

  // Go through the unlocked list and try to grant locks
  for len(unlockedList) > 0 {
    idSt := unlockedList[0]
    isFirst := true

    origList := make([]*waiterInfo, len(idSt.wInfoList))
    numElements := copy(origList, idSt.wInfoList)
    if numElements != len(idSt.wInfoList) {
      glog.Fatal("could not copy all elements in slice")
    }
    for _, winfo := range origList {
      if winfo == nil {
        glog.Fatalf("winfo is nil, origList= %v id state= %v", origList, idSt)
      }
      wstate, isExclusive := winfo.wsPtr, winfo.isExclusive
      // we allow exclusive lock only if the waiter is the first in list
      if isFirst {
        isFirst = false
      } else if isExclusive {
        break
      }
      if sl.tryAcquire(wstate.sharedIds, wstate.exclusiveIds, wstate) {
        // signal the successful acquisition of locks
        wstate.locked <- true
      }
      // if this was an exclusive waiter, we stop examining more waiters
      // irrespective of whether we could grant the locks to this one or not.
      if isExclusive {
        break
      }
    }

    // remove the top element of unlockedList
    unlockedList = unlockedList[1:]
  }
}

// ReleaseAll releases all locks, this is a complimentary call to AcquireAll.
// Since AcquireAll doesn't really acquire physical locks on IDs, we simply
// set the allLocked to false here.
func (sl *IDSetLock) ReleaseAll() {
  sl.mu.Lock()
  defer sl.mu.Unlock()
  sl.releaseLocked(sl.allIDList)
  sl.allLocked = false
  sl.allLckCond.Broadcast()
}

// AcquireStr acquires locks on string ids in the given lists.
// This is a convenience wrapper on top of Acquire() function.
func (sl *IDSetLock) AcquireStr(sharedIDList, exclusiveIDList []string) {
  if sl.idType != reflect.String {
    // this is a serious problem and should never happen.
    // hence doing glog.Fatal
    glog.Fatalf("type mismatch, input interface type: string required: %v",
      sl.idType)
  }

  sl.Acquire(StrToIface(sharedIDList), StrToIface(exclusiveIDList))
}

// AcquireInt64 acquires locks on int64 ids in the given lists.
// This is a convenience wrapper on top of Acquire() function.
func (sl *IDSetLock) AcquireInt64(sharedIDList, exclusiveIDList []int64) {
  if sl.idType != reflect.Int64 {
    // this is a serious problem and should never happen.
    // hence doing glog.Fatal
    glog.Fatalf("type mismatch, input interface type: int64 required: %v",
      sl.idType)
  }

  sl.Acquire(Int64ToIface(sharedIDList), Int64ToIface(exclusiveIDList))
}

// ReleaseStr releases locks on string ids in the given list.
// This is a convenience wrapper on top of Release() function.
func (sl *IDSetLock) ReleaseStr(idList []string) {
  if sl.idType != reflect.String {
    // this is a serious problem and should never happen.
    // hence doing glog.Fatal
    glog.Fatalf("type mismatch, input interface type: string required: %v",
      sl.idType)
  }

  sl.Release(StrToIface(idList))
}

// ReleaseInt64 releases locks on int64 ids in the given list.
// This is a convenience wrapper on top of Release() function.
func (sl *IDSetLock) ReleaseInt64(idList []int64) {
  if sl.idType != reflect.Int64 {
    // this is a serious problem and should never happen.
    // hence doing glog.Fatal
    glog.Fatalf("type mismatch, input interface type: int64 required: %v",
      sl.idType)
  }

  sl.Release(Int64ToIface(idList))
}

// IntToIface converts a slice of ints to a slice of interface.
// This is a helper function used to call various IDSetLock APIs.
func IntToIface(in []int) []interface{} {
  out := make([]interface{}, len(in))
  for i, v := range in {
    out[i] = v
  }
  return out
}

// Int32ToIface converts a slice of int32 to a slice of interface.
// This is a helper function used to call various IDSetLock APIs.
func Int32ToIface(in []int32) []interface{} {
  out := make([]interface{}, len(in))
  for i, v := range in {
    out[i] = v
  }
  return out
}

// Int64ToIface converts a slice of int64 to a slice of interface.
// This is a helper function used to call various IDSetLock APIs.
func Int64ToIface(in []int64) []interface{} {
  out := make([]interface{}, len(in))
  for i, v := range in {
    out[i] = v
  }
  return out
}

// StrToIface converts a slice of string to a slice of interface.
// This is a helper function used to call various IDSetLock APIs.
func StrToIface(in []string) []interface{} {
  out := make([]interface{}, len(in))
  for i, v := range in {
    out[i] = v
  }
  return out
}
