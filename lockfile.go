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
// This file implements lockfile based process startup and shutdown
// functionality.

package util

import (
  "flag"
  "fmt"
  "io"
  "io/ioutil"
  "os"
  "os/exec"
  "path"
  "strconv"
  "strings"
  "time"

  "github.com/golang/glog"
)

// LockFile is simply the file path.
type LockFile string

// Start runs a command in the background with exclusive lock on the lockfile.
// stdout, stderr specify the io.Writer objects to which the command's stdout
// and stderr respectively, should be set to.
func (lf *LockFile) Start(
  command string, stdout, stderr io.Writer, args ...string) (*os.Process,
  error) {

  cmd := exec.Command("flock")
  cmd.Args = append(cmd.Args, "--exclusive", "--nonblock", string(*lf),
    command)
  cmd.Args = append(cmd.Args, args...)
  cmd.Stdout = stdout
  cmd.Stderr = stderr
  glog.V(1).Infof("running command %s with args %v", cmd.Path, cmd.Args)

  errRun := lf.startWithWait(cmd)
  if errRun != nil {
    return nil, errRun
  }
  return cmd.Process, nil
}

// StartAsUser runs a command as another user in the background with exclusive
// lock on the lockfile.
func (lf *LockFile) StartAsUser(
  user, pass, command string, args ...string) (*os.Process, error) {

  cmd := exec.Command("flock")
  cmd.Args = append(cmd.Args, "--exclusive", "--nonblock", string(*lf),
    "sudo", "-S", "-u", user, command)
  cmd.Args = append(cmd.Args, args...)
  cmd.Stdin = strings.NewReader(pass + "\n")
  glog.V(1).Infof("running command %s with args %v", cmd.Path, cmd.Args)

  errRun := lf.startWithWait(cmd)
  if errRun != nil {
    return nil, errRun
  }
  return cmd.Process, nil
}

// Command line flag to control KillAll behavior.
var maxKillAttempts = flag.Int("lockfile_max_kill_attempts", 5,
  "Maximum number times to attempt to kill processes.")

// KillAll sends a SIGKILL to all processes holding the lockfile. This function
// fails with an error if processes couldn't be killed in a fixed number of
// attempts.
func (lf *LockFile) KillAll() error {
  // Processes may get re-created continuously due to self monitoring, so retry
  // killing processes till none of them are left.
  max := *maxKillAttempts
  for ii := 0; ii < max && lf.IsLocked(); ii = ii + 1 {
    pids := lf.FindPids()
    if len(pids) == 0 {
      return nil
    }
    glog.Infof("attempt %d: found processes %v holding the lock file %v", ii,
      pids, *lf)

    pidStrs := make([]string, len(pids))
    for ii, pid := range pids {
      pidStrs[ii] = fmt.Sprintf("%d", pid)
    }

    // Use kill -9 <pids> to kill user's processes.
    cmd := exec.Command("kill", "-9")
    cmd.Args = append(cmd.Args, pidStrs...)
    _ = cmd.Run()
  }

  if lf.IsLocked() {
    glog.Errorf("could not kill processes holding the lock file %v", *lf)
    return fmt.Errorf("could not kill lock file processes")
  }
  return nil
}

// Command line flag to control KillAllAsUser behavior.
var maxKillAsUserAttempts = flag.Int("lockfile_max_kill_as_user_attempts", 5,
  "Maximum number kill operations to perform to kill sudo processes.")

// KillAllAsUser sends a SIGKILL to all processes holding a lockfile. This
// function will fail when processes do not die in a fixed number of attempts.
func (lf *LockFile) KillAllAsUser(user, pass string) error {

  // Since the sudo program used when starting processes previously has set-uid
  // bit on, it cannot be killed; but it should die when all its child
  // processes are killed. So kill status is considered failed if a process
  // doesn't die in a fixed number of attempts.

  numAttemptsMap := make(map[int]int)
  for lf.IsLocked() {
    pids := lf.FindPids()
    pidStrs := make([]string, len(pids))
    for ii, pid := range pids {
      pidStrs[ii] = fmt.Sprintf("%d", pid)

      numAttempts := numAttemptsMap[pid]
      if numAttempts >= *maxKillAsUserAttempts {
        return fmt.Errorf("couldn't kill process %d in %d attempts", pid,
          numAttempts)
      }
      numAttemptsMap[pid] = numAttempts + 1
    }

    // Use sudo -S -u user kill -9 <pids> to kill other user's processes.
    cmd := exec.Command("sudo")
    cmd.Args = append(cmd.Args, "-S", "-u", user, "kill", "-9")
    cmd.Args = append(cmd.Args, pidStrs...)
    cmd.Stdin = strings.NewReader(pass + "\n")
    glog.V(1).Infof("running command %s with args %v", cmd.Path, cmd.Args)
    _ = cmd.Run()
  }

  if lf.IsLocked() {
    glog.Errorf("could not kill processes holding the lock file %v", *lf)
    return fmt.Errorf("could not kill lock file processes")
  }
  return nil
}

// Command line flag to control FindPids behavior.
var flockFdIdxBegin = flag.Int("flock_fd_idx_begin", 3,
  "File descriptor index to start searching for lock file.")
var flockFdIdxEnd = flag.Int("flock_fd_idx_end", 10,
  "File descriptor index to stop searching for lock file.")

// hasLockFile searches the /proc node for the provided PID to see if a lockfile
// is present in the range of FDs to be checked.
func (lf *LockFile) hasLockFile(pid int, begin int, end int) bool {
  for ii := begin; ii <= end; ii++ {
    // Read the symlink target at the expected lock file index.
    fdPath := fmt.Sprintf("/proc/%d/fd/%d", pid, ii)
    target, errRead := os.Readlink(fdPath)
    if errRead == nil && path.Clean(target) == path.Clean(string(*lf)) {
      return true
    }
  }
  return false
}

// FindPids finds the process ids holding a lockfile.
func (lf *LockFile) FindPids() []int {

  // NOTE: There is no reliable and consistent way to figure out processes that
  // have opened a file. So, search for all processes opening the lockfile at
  // indices in the provided range.i This reduces the search space tremendously.

  dir, _ := os.Open("/proc")
  fileNames, _ := dir.Readdirnames(0) // this must always succeed on /proc
  dir.Close()

  pidSet := make(map[int]struct{})
  parentMap := make(map[int]int)
  for _, name := range fileNames {
    // Skip non-pid files.
    pid, err := strconv.Atoi(name)
    if err != nil {
      continue
    }

    // Read the /proc/*/status files to prepare parentMap.
    statusPath := fmt.Sprintf("/proc/%d/status", pid)
    status, errRead := ioutil.ReadFile(statusPath)
    if errRead != nil {
      if os.IsNotExist(errRead) {
        continue
      }
      glog.Warningf("couldn't read process status from %s: %s",
        statusPath, errRead)
      continue
    }
    fields := strings.Fields(string(status))
    for ii := 0; ii < len(fields); ii++ {
      if fields[ii] == "PPid:" {
        parent, _ := strconv.Atoi(fields[ii+1])
        parentMap[pid] = parent
        break
      }
    }

    if found := lf.hasLockFile(pid, *flockFdIdxBegin, *flockFdIdxEnd); !found {
      continue
    }
    pidSet[pid] = struct{}{}
  }

  // Figure out child processes and add them till no more processes to add.
  for true {
    count := 0
    for child, parent := range parentMap {
      if _, parentFound := pidSet[parent]; parentFound {
        if _, childFound := pidSet[child]; childFound {
          continue
        }
        pidSet[child] = struct{}{}
        count++
      }
    }
    if count == 0 {
      break
    }
  }

  // Convert pidSet into an array.
  pids := make([]int, 0, len(pidSet))
  for pid := range pidSet {
    pids = append(pids, pid)
  }
  return pids
}

// Command line flag to control Start* functions behavior.
var flockWait = flag.Duration("lockfile_flock_wait", 100*time.Millisecond,
  "Number of milliseconds to wait before treating a process as started.")

// startWithWait executes a command and waits for a timeout so that command got
// a chance to run properly. The timeout is not made as part of the signature
// because in future, when Go has flock support, we could wait till the
// lockfile gets locked or process dies without relying on timeout.
func (lf *LockFile) startWithWait(cmd *exec.Cmd) error {
  // Start the command so that cmd.Process is initialized.
  errStart := cmd.Start()
  if errStart != nil {
    return errStart
  }

  errChan := make(chan error)
  go func() {
    errChan <- cmd.Wait()
  }()

  // Wait for a timeout so that process gets the lock or quits.
  select {
  case errRun := <-errChan:
    if errRun != nil {
      glog.Errorf("command %s failed with lockfile %s: %s", cmd.Args,
        *lf, errRun)
      return errRun
    }
    glog.Warningf("command %s with lockfile %s has exited successfully",
      cmd.Args, *lf)
    return nil

  case <-time.After(*flockWait):
    return nil
  }
}

// IsLocked returns true if the lock file is in use.
func (lf *LockFile) IsLocked() bool {
  if _, err := os.Stat(string(*lf)); err == nil {
    cmd := exec.Command("flock", "--exclusive", "--nonblock", string(*lf),
      "true")
    if err := cmd.Run(); err != nil {
      return true
    }
  }
  return false
}
