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
// This file implements workerpool manager.

package util

import (
  "fmt"
  "runtime"
  "sync"
  "sync/atomic"
  "time"

  "github.com/golang/glog"
)

// WorkerPoolManager manages the workers that performs Work.
type WorkerPoolManager struct {
  // WorkerPoolManager name.
  name string
  // Channel to post work.
  queueChan chan WorkManager
  // Channel to stop receiving work.
  stopQueueChan chan string
  // Channel to pickup work from.
  workChan chan WorkManager
  // Channel to stop workers.
  stopWorkChan chan struct{}
  // Synchronize workers shutdown.
  stopWG sync.WaitGroup
  // Maximum pending work allowed.
  maxQueuedWork int32
  // Atomically updated count of workers.
  numWorkers int32
  // Monotonically increasing id for workers.
  workerID int32
  // Atomically updated count of queued work.
  queuedWork int32
  // Atomically updated count of work being done.
  activeWork int32
  // Atomically updated count of work finished.
  completedWork int32
  // Start time since the first work was posted.
  startTime int64
  // Time after shutdown was complete.
  endTime int64
}

// WorkManager manages a work.
type WorkManager struct {
  work               Work
  resultChannel      chan error
  startTime          int64
  executionStartTime int64
  endTime            int64
  workPoolManager    *WorkerPoolManager
}

// Work is the job that needs execution.
type Work interface {
  DoWork()
  Info() string
}

// NewWorkerPoolManager creates a new WorkerPoolManager.
func NewWorkerPoolManager(name string, numWorkers int, maxQueuedWork int32) (
  *WorkerPoolManager, error) {

  wpm := &WorkerPoolManager{
    name:          name,
    queueChan:     make(chan WorkManager),
    stopQueueChan: make(chan string),
    workChan:      make(chan WorkManager),
    stopWorkChan:  make(chan struct{}),
    maxQueuedWork: maxQueuedWork,
    numWorkers:    0, // Updated when worker is created.
    workerID:      0,
    queuedWork:    0,
    activeWork:    0,
    startTime:     GetNowUTCUnixNS(),
    endTime:       0, // Updated on shutdown.
  }

  if errA := wpm.Add(numWorkers); errA != nil {
    return nil, errA
  }
  go wpm.run()
  glog.Infof("workpool manager %s started", wpm.name)
  return wpm, nil
}

// Add adds a workers to workpool manager.
func (wpm *WorkerPoolManager) Add(numWorkers int) error {
  if numWorkers <= 0 {
    return fmt.Errorf("invalid number of workers %d", numWorkers)
  }

  for worker := 0; worker < numWorkers; worker++ {
    go wpm.startWorker(atomic.AddInt32(&wpm.workerID, 1))
  }
  return nil
}

// Dispatch accepts a work for execution. Returns error if work is not queued
// because of queue being full.
func (wpm *WorkerPoolManager) Dispatch(work Work) (err error) {
  // Submit work on shutting down manager will panic, handle.
  defer catchPanic(&err, wpm.name, "Dispatch")

  wm := WorkManager{
    work:               work,
    resultChannel:      make(chan error),
    startTime:          GetNowUTCUnixNS(),
    executionStartTime: GetNowUTCUnixNS(),
    endTime:            0,
    workPoolManager:    wpm,
  }

  wpm.queueChan <- wm
  return <-wm.resultChannel
}

// stopQueueChannel stops the queue channel.
func (wpm *WorkerPoolManager) stopQueueChannel() {
  // Stop receiving work.
  wpm.stopQueueChan <- "Shutdown"
  // Make sure we stopped receiving for work.
  <-wpm.stopQueueChan
  // Cleaup all queue related channels.
  close(wpm.queueChan)
  close(wpm.stopQueueChan)
}

// stopWorkChannel stops the work channel.
func (wpm *WorkerPoolManager) stopWorkChannel() {
  // Inform works to stop the work.
  close(wpm.stopWorkChan)
  // Wait for all outstanding work to finish.
  wpm.stopWG.Wait()
  // Cleanup active work queue.
  close(wpm.workChan)
}

// WaitForCompletion is a waiting call until all the active and queued tasks are
// completed or timeout is reached.
func (wpm *WorkerPoolManager) WaitForCompletion(timeout time.Duration) (
  err error) {
  glog.Infof("waiting for completion of workerpool manager %s", wpm.name)
  defer catchPanic(&err, wpm.name, "WaitForCompletion")
  wpm.stopQueueChannel()

  ticker := time.NewTicker(time.Second)
  defer ticker.Stop()
  stop := time.After(timeout)
  for {
    select {
    case <-stop:
      go wpm.stopWorkChannel()
      return fmt.Errorf("timed out waiting for completion of workerpool "+
        "manager %s, going down", wpm.name)
    case <-ticker.C:
      // Verify no pending work and no ongoing work.
      if len(wpm.workChan) == 0 && atomic.LoadInt32(&wpm.activeWork) == 0 {
        wpm.stopWorkChannel()
        glog.Infof("workpool manager %s stopped", wpm.name)
        return nil
      }
    }
  }
}

// Shutdown stops a queue and work channel, and does _not_ wait for queued tasks
// to execute.
func (wpm *WorkerPoolManager) Shutdown(err error) {
  defer catchPanic(&err, wpm.name, "Shutdown")

  glog.Infof("shutting down workerpool manager %s", wpm.name)
  wpm.stopQueueChannel()
  wpm.stopWorkChannel()
  glog.Infof("shut down workerpool manager %s completed", wpm.name)
}

// startWorker starts a new worker and starts polling for Work.
func (wpm *WorkerPoolManager) startWorker(id int32) {
  wpm.stopWG.Add(1)
  defer wpm.stopWG.Done()
  for {
    select {
    case <-wpm.stopWorkChan:
      glog.V(2).Infof("shutting down worker %d", id)
      return

    case wm := <-wpm.workChan:
      wm.execWork()
      break
    }
  }
}

// execWork does the work in the context of the worker.
func (wm *WorkManager) execWork() {
  defer catchPanic(nil, wm.workPoolManager.name, "execWork")
  defer func() {
    atomic.AddInt32(&wm.workPoolManager.activeWork, -1)
    atomic.AddInt32(&wm.workPoolManager.completedWork, 1)
  }()

  atomic.AddInt32(&wm.workPoolManager.queuedWork, -1)
  atomic.AddInt32(&wm.workPoolManager.activeWork, 1)
  wm.executionStartTime = GetNowUTCUnixNS()
  wm.work.DoWork()
  wm.endTime = GetNowUTCUnixNS()
  glog.Infof("for %s, queuetime: %dms, executionTime: %dms", wm.work.Info(),
    NSToMS(wm.executionStartTime-wm.startTime),
    NSToMS(wm.endTime-wm.executionStartTime))
}

// run queues the posted work in workchannel.
func (wpm *WorkerPoolManager) run() {
  wpm.stopWG.Add(1)
  defer wpm.stopWG.Done()

  for {
    select {
    case msg := <-wpm.stopQueueChan:
      wpm.stopQueueChan <- msg
      return

    case queuedWork := <-wpm.queueChan:
      // If queue is at capacity, do not queue and return error.
      if atomic.LoadInt32(&wpm.queuedWork) == wpm.maxQueuedWork {
        queuedWork.resultChannel <- fmt.Errorf("queue full")
        continue
      }

      atomic.AddInt32(&wpm.queuedWork, 1)
      wpm.workChan <- queuedWork
      queuedWork.resultChannel <- nil
      break
    }
  }
}

// catchPanic catches panic if any and sets the appropriate error.
func catchPanic(err *error, entity, funcName string) {
  if r := recover(); r != nil {
    // Capture the stack trace.
    buf := make([]byte, 10000)
    runtime.Stack(buf, false)

    glog.Errorf("PANIC %s@%s : Stack Trace : %v",
      funcName, entity, string(buf))

    if err != nil {
      *err = fmt.Errorf("%v", r)
    }
  }
}
