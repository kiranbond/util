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
  "io/ioutil"
  "os"
  "os/user"
  "strings"
  "testing"
  "time"

  "github.com/stretchr/testify/assert"
)

func TestLockFileStart(t *testing.T) {
  tmpFile, errTemp := ioutil.TempFile("", "lockfile_start")
  assert.Nil(t, errTemp)
  lockFile := LockFile(tmpFile.Name())
  tmpFile.Close()
  defer os.Remove(string(lockFile))

  // Start a command with lockfile.
  proc, errStart := lockFile.Start("sleep", os.Stdout, os.Stderr, "2")
  assert.Nil(t, errStart)
  assert.NotNil(t, proc)

  // Sleep for some time so that command begins execution.
  time.Sleep(time.Second)

  // Make sure that file is locked.
  _, errAgain := lockFile.Start("sleep", os.Stdout, os.Stderr, "2")
  assert.NotNil(t, errAgain)

  // Log file descriptor targets.
  for ii := 0; ii < 4; ii++ {
    path := fmt.Sprintf("/proc/%d/fd/%d", proc.Pid, ii)
    target, errRead := os.Readlink(path)
    t.Logf("%s -> %s %v", path, target, errRead)
  }
}

func TestLockFileFindPids(t *testing.T) {
  tmpFile, errTemp := ioutil.TempFile("", "lockfile_findpids")
  assert.Nil(t, errTemp)
  lockFile := LockFile(tmpFile.Name())
  tmpFile.Close()
  defer os.Remove(string(lockFile))

  // Start a command with lockfile.
  proc, errStart := lockFile.Start("sleep", os.Stdout, os.Stderr, "2")
  assert.Nil(t, errStart)
  assert.NotNil(t, proc)

  // Sleep for some time so that command begins execution.
  time.Sleep(time.Second)

  // Make sure that lockfile is locked.
  _, errAgain := lockFile.Start("sleep", os.Stdout, os.Stderr, "2")
  assert.NotNil(t, errAgain)

  // Log the pids holding the lockfile.
  pids := lockFile.FindPids()
  for _, pid := range pids {
    cmdPath := fmt.Sprintf("/proc/%d/comm", pid)
    command, errRead := ioutil.ReadFile(cmdPath)
    assert.Nil(t, errRead)

    fdPath := fmt.Sprintf("/proc/%d/fd/3", pid)
    target, errRead := os.Readlink(fdPath)
    t.Logf("%d %s -> %s %v", pid, strings.TrimSpace(string(command)), target,
      errRead)
  }
}

func TestLockFileStartAsUserFindPids(t *testing.T) {
  tmpFile, errTemp := ioutil.TempFile("", "lockfile_startasuser_findpids")
  assert.Nil(t, errTemp)
  lockFile := LockFile(tmpFile.Name())
  tmpFile.Close()
  defer os.Remove(string(lockFile))

  // Start a sudo command with lockfile.
  currentUser, _ := user.Current()
  proc, errStart := lockFile.StartAsUser(
    currentUser.Username, "", "sleep", "2")
  assert.Nil(t, errStart)
  assert.NotNil(t, proc)

  // Sleep for some time so that command begins execution.
  time.Sleep(time.Second)

  // Make sure that lockfile is locked.
  _, errAgain := lockFile.Start("sleep", os.Stdout, os.Stderr, "2")
  assert.NotNil(t, errAgain)

  // Log the pids holdings the lockfile.
  pids := lockFile.FindPids()
  for _, pid := range pids {
    cmdPath := fmt.Sprintf("/proc/%d/comm", pid)
    command, errRead := ioutil.ReadFile(cmdPath)
    assert.Nil(t, errRead)

    fdPath := fmt.Sprintf("/proc/%d/fd/3", pid)
    target, errRead := os.Readlink(fdPath)
    t.Logf("%d %s -> %s %v", pid, strings.TrimSpace(string(command)), target,
      errRead)
  }
}

func TestLockFileKillAll(t *testing.T) {
  tmpFile, errTemp := ioutil.TempFile("", "lockfile_killall")
  assert.Nil(t, errTemp)
  lockFile := LockFile(tmpFile.Name())
  tmpFile.Close()
  defer os.Remove(string(lockFile))

  // Start a command with lockfile.
  proc, errStart := lockFile.Start("sleep", os.Stdout, os.Stderr, "10")
  assert.Nil(t, errStart)
  assert.NotNil(t, proc)

  // Sleep for some time so that command begins execution.
  time.Sleep(time.Second)

  // Make sure that lockfile is locked.
  _, errAgain := lockFile.Start("sleep", os.Stdout, os.Stderr, "2")
  assert.NotNil(t, errAgain)

  pidsBefore := lockFile.FindPids()
  t.Logf("pids before kill are %v", pidsBefore)

  // Kill all processes.
  errKill := lockFile.KillAll()
  if errKill != nil {
    t.Logf("couldn't kill pids: %v", errKill)
    assert.Nil(t, errKill)
  }

  pidsAfter := lockFile.FindPids()
  t.Logf("pids after kill are %v", pidsAfter)
  assert.Equal(t, len(pidsAfter), 0)
}

func TestLockFileKillAllAsUser(t *testing.T) {
  tmpFile, errTemp := ioutil.TempFile("", "lockfile_killallasuser")
  assert.Nil(t, errTemp)
  lockFile := LockFile(tmpFile.Name())
  tmpFile.Close()
  defer os.Remove(string(lockFile))

  // Start a sudo command with lockfile.
  currentUser, _ := user.Current()
  proc, errStart := lockFile.StartAsUser(
    currentUser.Username, "", "sleep", "10")
  assert.Nil(t, errStart)
  assert.NotNil(t, proc)

  // Sleep for some time so that command begins execution.
  time.Sleep(time.Second)

  // Make sure that lockfile is locked.
  _, errAgain := lockFile.StartAsUser(currentUser.Username, "", "sleep", "10")
  assert.NotNil(t, errAgain)

  pidsBefore := lockFile.FindPids()
  t.Logf("pids before kill are %v", pidsBefore)

  // Kill all sudo processes.
  errKill := lockFile.KillAllAsUser(currentUser.Username, "")
  if errKill != nil {
    t.Logf("couldn't kill sudo pids: %v", errKill)
    assert.Nil(t, errKill)
  }

  pidsAfter := lockFile.FindPids()
  t.Logf("pids after kill are %v", pidsAfter)
  assert.Equal(t, len(pidsAfter), 0)
}
