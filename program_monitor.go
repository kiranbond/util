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
  "flag"
  "net/http"
  "net/url"
  "os"
  "os/exec"
  "os/signal"
  "os/user"
  "path"
  "sync"
  "sync/atomic"
  "syscall"
  "time"

  "github.com/golang/glog"
)

// This file implements a program monitor using programMonitor struct that is
// created at the beginning of each main function. This will make the original
// process a parent process and spawn a child process with a special flag called
// "cChildIdentifierArg" to do the actual work. This child process again tries
// to create the parent but the programMonitor recognizes that it is a child and
// returns.

// The parent process then handles any signals sent to it,
// monitors any child deaths and respawns the child, if needed.
//
// The parent can also be instructed to send a HTTP GET to a specific URL
// given by child and kill/restart the child if the GET request times out.
//
// Example usage:
// func main () {
//   // Just in the beginning of main() in any program, do
//   monitoringURL := "http://127.0.0.1:port/healthz"
//   if err := SetProgramMonitor(os.Args, monitoringURL); err != nil {
//     glog.Fatalf("could not setup parent monitoring :: %v", err)
//   }
//   http.HandleFunc("/healthz", healthzHandler)
//
// }
//
// // Define a healthzHandler function:
// func healthzHandler(w http.ResponseWriter, r *http.Request) {
//   fmt.Fprintf(w, "I am alive")
// }

const cChildIdentifierArg string = "iamzchild"

// variable to guarantee that parent is allocated only once per program.
var hasParent int32

// Flags to control the behaviour of the parent program and monitoring
var (
  PingInterval = flag.Duration("ping_interval", 60*time.Second,
    "This specifies the interval after which HTTP GET requests are sent "+
      "from parent to child process.")
  ChildTimeout = flag.Duration("child_timeout", 2*time.Second,
    "Timeout a HTTP GET request sent to the child after this much time")
  NumRetries = flag.Int("num_http_retries", 1,
    "Number of times to retry the health monitoring GET requests before "+
      "restarting the child")
  StartGap = flag.Duration("start_gap", 300*time.Millisecond,
    "Wait for this much time before restarting the child once the death "+
      "is detected.")
  MaxRestarts = flag.Int("max_restarts", 65564,
    "Maximum number of restarts that the parent should do for a child")
)

// Main object that provides the functionality.
// programMonitor keeps the state for the monitoring, checking liveness and
// other information about the child process.
type programMonitor struct {
  programName string
  hostName    string
  userName    string

  psGroup                *processGroup
  monitoringURI          string         // uri used for monitoring child
  httpClient             *http.Client   // client used to send http get
  sigChan                chan os.Signal // channel to capture incoming signals
  numConsecutiveFailures int            // track consecutive failures
}

type processGroup struct {
  sync.Mutex   // mutex to guard the process set
  processSet   map[*os.Process]struct{}
  wg           sync.WaitGroup
  cmdPath      string
  cmdArgs      []string // command line args for the child
  restartCount int      // track total number of restarts
}

// SetProgramMonitor sets up the monitoring of a program using the command
// line args for the program and also does periodic monitoring using the
// specified monitoringURI. It starts a  child for the original program
// and starts monitoring.
func SetProgramMonitor(cmdLineArgs []string, monitoringURI string) error {
  // check if a parent already exists
  if atomic.CompareAndSwapInt32(&hasParent, 0, /*old val*/
    1 /*new val*/) == false {
    glog.Errorf("%s already has a parent, why ask for more?", cmdLineArgs[0])
    return nil
  }

  // check if this is a child process started by a parent. This is done by
  // checking for a special command line argument: cChildIdentifierArg
  for ii := range cmdLineArgs {
    if cmdLineArgs[ii] == cChildIdentifierArg {
      glog.Infof("%s is the child process started by parent", cmdLineArgs[0])
      // remove it from the command line args and return
      copy(cmdLineArgs[ii:], cmdLineArgs[ii+1:])
      cmdLineArgs[len(cmdLineArgs)-1] = ""
      cmdLineArgs = cmdLineArgs[:len(cmdLineArgs)-1]
      return nil
    }
  }

  parent := &programMonitor{
    monitoringURI: monitoringURI,
  }

  // initialize various fields in the parent
  if err := parent.init(cmdLineArgs); err != nil {
    glog.Errorf("error getting parent for: %s :: %v", cmdLineArgs[0], err)
    return err
  }

  glog.Infof("parent running : %v", os.Args)

  // Set command path
  cmdPath, errPath := exec.LookPath(parent.programName)
  if errPath != nil {
    glog.Warningf("could not find executable: %s :: %v", parent.programName,
      errPath)
    cmdPath = cmdLineArgs[0]
  }

  // start child now
  parent.psGroup = makeProcessGroup(cmdPath, cmdLineArgs)
  if _, err := parent.psGroup.startProcess(); err != nil {
    glog.Errorf("could not start process: %s :: %v", cmdPath, err)
    return err
  }
  // Set up a channel to capture signals
  parent.sigChan = make(chan os.Signal)
  signal.Notify(parent.sigChan, syscall.SIGINT, syscall.SIGTERM,
    syscall.SIGQUIT, syscall.SIGHUP, syscall.SIGPIPE)

  // handle the signals coming to the child
  go parent.handleSignals()
  // Start child monitoring!!
  go parent.startMonitor()

  parent.psGroup.WaitAll()

  // we cannot return from here, since that would start the client code
  // from where it called this function. So just call os.Exit()
  glog.Infof("parent exiting: %d ", os.Getpid())
  os.Exit(0)
  return nil
}

func (p *programMonitor) init(cmdArgs []string) error {
  // Set program name
  p.programName = path.Base(cmdArgs[0])

  // Set hostname
  hostName, errHostname := os.Hostname()
  if errHostname != nil {
    glog.Errorf("error determining host name :: %v", errHostname)
    return errHostname
  }
  p.hostName = hostName

  // Set username
  user, errUser := user.Current()
  if errUser != nil {
    glog.Errorf("error determining user name :: %v", errUser)
  }
  p.userName = user.Username

  // Create a http client
  p.httpClient = &http.Client{}

  return nil
}

func (p *programMonitor) startMonitor() error {

  // start monitoring with http get requests
  if len(p.monitoringURI) > 0 {
    //p.httpClient.Timeout = *ChildTimeout
    _, errURL := url.Parse(p.monitoringURI)
    if errURL != nil {
      glog.Errorf("could not parse monitoring url: %s :: %v", p.monitoringURI,
        errURL)
      return errURL
    }

    ticker := time.NewTicker(*PingInterval)
    defer ticker.Stop()
    var timeout bool

    for _ = range ticker.C {
      glog.V(1).Info("sending ping")
      getChan := make(chan *http.Response)
      go p.httpGetAndReport(p.monitoringURI, getChan)

      select {
      case <-time.After(*ChildTimeout):
        glog.Errorf("timedout waiting for GET response from child")
        timeout = true

      case rsp := <-getChan:
        rsp.Body.Close()
        timeout = false
      }

      if timeout {
        p.numConsecutiveFailures++
        if p.numConsecutiveFailures > *NumRetries {
          // send kill signal to child and restart
          glog.Infof("sending kill signal to child")
          p.psGroup.SignalAll(syscall.SIGHUP)
        }
      } else {
        // reset the failure count on success
        p.numConsecutiveFailures = 0
      }
    }
  }
  return nil
}

func (p *programMonitor) httpGetAndReport(uri string,
  respChan chan *http.Response) {

  req, errReq := http.NewRequest("GET", uri, nil)
  if errReq != nil {
    glog.Errorf("error creating http request for uri: %s :: %v", uri, errReq)
    return
  }
  req.Close = true

  resp, err := p.httpClient.Do(req)
  if err == nil {
    glog.Infof("http response: %v", resp.Status)
    respChan <- resp
  } else {
    glog.Errorf("got error on http GET for uri: %s :: %v", uri, err)
  }
  return
}

func (p *programMonitor) handleSignals() {
  for {
    signal := <-p.sigChan // get os.Signal
    syscallSignal := signal.(syscall.Signal)
    glog.Infof("parent: %d got signal: %v", os.Getpid(), signal)

    switch syscallSignal {
    case syscall.SIGHUP:
    case syscall.SIGINT:
    case syscall.SIGQUIT:
      p.psGroup.SignalAll(signal)
    case syscall.SIGPIPE:
      glog.Info("got signal SIGPIPE, ignoring")
      continue
    default:
      // Forward signal
      glog.Errorf("unhandled signal: %v", signal)
      p.psGroup.SignalAll(signal)
    }
  }
}

func makeProcessGroup(cmdPath string, args []string) *processGroup {
  return &processGroup{
    cmdPath:    cmdPath,
    cmdArgs:    args,
    processSet: make(map[*os.Process]struct{}),
  }
}

func (pg *processGroup) startProcess() (*os.Process, error) {
  pg.wg.Add(1)

  newArgs := append(pg.cmdArgs[1:], cChildIdentifierArg)
  glog.Infof("starting new child process %s, args: %v restartCount: %d",
    pg.cmdPath, newArgs, pg.restartCount)
  cmd := exec.Command(pg.cmdPath, newArgs...)
  cmd.Stdout = os.Stdout
  cmd.Stderr = os.Stderr
  if err := cmd.Start(); err != nil {
    glog.Errorf("could not start child process:%s :: %v", pg.cmdPath, err)
    return nil, err
  }

  // Add to set
  process := cmd.Process
  pg.Add(process)

  // Handle the process death
  go func() {
    state, err := process.Wait()

    glog.Error(process.Pid, state, err)

    // Remove from set
    pg.Remove(process)

    // Start the child process again if the exit status was non-zero
    if !state.Success() && pg.restartCount < *MaxRestarts {
      if *StartGap > 0 {
        glog.Infof("waiting start child process: %v", *StartGap)
        time.Sleep(*StartGap)
      }
      pg.restartCount++
      pg.startProcess()
    }

    // Process is gone now.
    pg.wg.Done()
  }()

  return process, nil
}

func (pg *processGroup) SignalAll(signal os.Signal) {
  pg.Each(func(process *os.Process) {
    process.Signal(signal)
  })
}

func (pg *processGroup) WaitAll() {
  pg.wg.Wait()
}

func (pg *processGroup) Add(process *os.Process) {
  pg.Lock()
  defer pg.Unlock()

  pg.processSet[process] = struct{}{}
}

func (pg *processGroup) Each(fn func(*os.Process)) {
  pg.Lock()
  defer pg.Unlock()

  for process := range pg.processSet {
    fn(process)
  }
}

func (pg *processGroup) Remove(process *os.Process) {
  pg.Lock()
  defer pg.Unlock()
  delete(pg.processSet, process)
}
