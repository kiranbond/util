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
// This file implements miscellaneous utility functions.

package util

import (
  "bufio"
  "bytes"
  "crypto/rand"
  "crypto/sha1"
  "encoding/hex"
  "fmt"
  "hash/fnv"
  "io"
  "io/ioutil"
  "math/big"
  "net"
  "os"
  "os/exec"
  "os/user"
  "reflect"
  "regexp"
  "runtime"
  "strconv"
  "strings"
  "time"

  "github.com/golang/glog"
  "zerostack/common/constants"
)

const (
  cMACLenBytes int = 6
)

// MAC is a type that represents an ethernet MAC address
type MAC []byte

// IPtoMAC takes in the cidr as input and generates a MAC as the output
// The format is shown as follows:
// ipcidr = "192.168.100.1/24" type = "1" -> mac = "02:01:c0:a8:64:01"
func IPtoMAC(ipcidr string, prefix string) (string, error) {
  if prefix != "1" && prefix != "2" && prefix != "3" && prefix != "4" {
    return "", fmt.Errorf("invalid prefix for the MAC address")
  }
  ip, _, _ := net.ParseCIDR(ipcidr)
  mac := "02:0" + prefix
  digits := strings.Split(ip.String(), ".")
  for _, digit := range digits {
    d, _ := strconv.Atoi(digit)
    hd := fmt.Sprintf(":%x%x", d>>4, d&0xf)
    mac = mac + hd
  }
  return mac, nil
}

// MACtoHex16 converts a MAC address to a 16 digit Hex representation with
// zero padding. The format is as follows:
// "00:01:c0:a8:64:01" -> "000001c0a86401"
func MACtoHex16(mac string) string {
  macHex := "0000" + strings.Replace(mac, ":", "", -1)
  return macHex
}

// ParseIP is a helper function to extract the IP address contained in a string
// e.g., tcp://10.150.30.1:10000/?uuid=021d4743-2319-4535-a4ac-a59a1802bc70 will
// result in 10.150.30.1
func ParseIP(str string) string {
  octet := "(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9]?[0-9])"
  rxp := octet + "\\." + octet + "\\." + octet + "\\." + octet
  re := regexp.MustCompile(rxp)
  ipStr := re.FindString(str)
  return ipStr
}

// FileExists checks if the file at filePath exists.
func FileExists(filePath string) bool {
  stat, err := os.Stat(filePath)
  if err != nil || stat.IsDir() {
    return false
  }
  return true
}

// GenerateRandomMAC generates a random MAC address and returns that.
func GenerateRandomMAC() (MAC, error) {
  randBytes := make([]byte, 5)
  if _, err := rand.Read(randBytes); err != nil {
    glog.Errorf("error generating random bytes for mac :: %v", err)
    return nil, err
  }

  mac := MAC{
    2, // set the local bit, so that it doesn't clash with any global address
    randBytes[0], randBytes[1], randBytes[2], randBytes[3], randBytes[4]}

  return mac, nil
}

// ParseMAC parses a string representation of a MAC address and returns
// a MAC type.
func ParseMAC(s string) (MAC, error) {
  bytes := strings.Split(s, ":")

  if len(bytes) != cMACLenBytes {
    return nil, fmt.Errorf("incorrect mac format")
  }

  mac := make(MAC, cMACLenBytes)
  for ii, bs := range bytes {
    bsStr, err := strconv.ParseUint(bs, 16, 8)
    if err != nil {
      return nil, fmt.Errorf("error parsing mac %s :: %v", s, err)
    }
    mac[ii] = byte(bsStr)
  }

  return mac, nil
}

func (mac MAC) String() string {
  return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
    mac[0], mac[1], mac[2], mac[3], mac[4], mac[5])
}

// WriteToTempFile creates a temporary file with the input string "data".
// If "dir" is an empty string, the temporary file is created under that
// default directory for temporary files; otherwise it is created under the
// "dir" directory". Returns the path to temporary file on success.
func WriteToTempFile(
  dir string, data string, mode os.FileMode) (string, error) {
  return WriteToTempFileWithPrefix(dir, "tempfile", data, mode)
}

// WriteToTempFileWithPrefix creates temporary file in directory "dir" starting
// with prefix "prefix" with the content "data" and file access mode "mode".
// Returns the temporary file path on success.
func WriteToTempFileWithPrefix(
  dir, prefix, data string, mode os.FileMode) (string, error) {

  file, errTemp := ioutil.TempFile(dir, prefix)
  if errTemp != nil {
    glog.Errorf("could not create temporary file :: %v", errTemp)
    return "", errTemp
  }
  defer file.Close()

  if _, err := file.WriteString(data); err != nil {
    glog.Errorf("could not write to temporary file %s :: %v", file.Name(), err)
    _ = os.Remove(file.Name())
    return "", err
  }

  if err := os.Chmod(file.Name(), mode); err != nil {
    glog.Errorf("could not update file permissions %s :: %v", file.Name(), err)
    _ = os.Remove(file.Name())
    return "", err
  }
  return file.Name(), nil
}

// RawWriteToTempFile creates a temporary file with the input bytes. If "dir" is
// an empty string, the temporary file is created under that default directory
// for temporary files, otherwise, it is created under the "dir" directory.
// Returns path to temporary file on suceess.
func RawWriteToTempFile(
  dir string,
  data []byte,
  mode os.FileMode) (string, error) {

  return RawWriteToTempFileWithPrefix(dir, "tempfile", data, mode)
}

// RawWriteToTempFileWithPrefix creates temporary file in directory "dir"
// starting with prefix "prefix" with the content "data" and file access mode
// "mode". Returns the temporary file path on success.
func RawWriteToTempFileWithPrefix(
  dir, prefix string,
  data []byte,
  mode os.FileMode) (string, error) {

  file, errTemp := ioutil.TempFile(dir, prefix)
  if errTemp != nil {
    glog.Errorf("could not create temporary file :: %v", errTemp)
    return "", errTemp
  }
  defer func() {
    file.Sync()
    file.Close()
  }()

  if _, err := file.Write(data); err != nil {
    glog.Errorf("could not write to temporary file %s :: %v", file.Name(), err)
    _ = os.Remove(file.Name())
    return "", err
  }

  if err := os.Chmod(file.Name(), mode); err != nil {
    glog.Errorf("could not update file permissions %s :: %v", file.Name(), err)
    _ = os.Remove(file.Name())
    return "", err
  }
  return file.Name(), nil
}

// NewRandomStr returns a random alphanumeric string of length n
func NewRandomStr(n int) (string, error) {
  const alphanum string = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
  symbols := big.NewInt(int64(len(alphanum)))
  states := big.NewInt(0)
  states.Exp(symbols, big.NewInt(int64(n)), nil)
  r, err := rand.Int(rand.Reader, states)
  if err != nil {
    glog.Errorf("could not create random string :: %v", err)
    return "", err
  }
  var bytes = make([]byte, n)
  r2 := big.NewInt(0)
  symbol := big.NewInt(0)
  for i := range bytes {
    r2.DivMod(r, symbols, symbol)
    r, r2 = r2, r
    bytes[i] = alphanum[symbol.Int64()]
  }
  return string(bytes), nil
}

// Contains returns "true" if "val" is present in the "slice".
func Contains(slice interface{}, val interface{}) bool {
  sv := reflect.ValueOf(slice)
  if sv.Kind() != reflect.Slice {
    glog.Errorf("invoked with non-slice type:%s", sv.Kind())
    return false
  }
  for i := 0; i < sv.Len(); i++ {
    // Using "DeepEqual()" as it correctly handles recursive types, for
    // which "==" may cause a runtime panic.
    if reflect.DeepEqual(sv.Index(i).Interface(), val) {
      return true
    }
  }
  return false
}

// MinInt returns the minimum of 2 integers.
func MinInt(a, b int) int {
  if a < b {
    return a
  }
  return b
}

// MaxInt returns the maximum of 2 integers.
func MaxInt(a, b int) int {
  if a > b {
    return a
  }
  return b
}

// ReadFile is a helper function that trims and returns file content as string.
// In case of an error, default value is returned along with the error.
func ReadFile(path string, defValue string) (string, error) {
  data, errRead := ioutil.ReadFile(path)
  if errRead != nil {
    return defValue, errRead
  }
  return strings.TrimSpace(string(data)), nil
}

// WaitForFuncToSucceed is a helper function which retries a given function
// till it succeeds or times out. fnLabel is a short description of the
// function. An ex.
//
//  var dbConn *dbconn.DBConnCassandra
//  res, err := waitForFuncToSucceed("cassandra db connection",
//    func() (interface{}, error) {
//      return dbconn.NewDBConnCassandra(dbServers, zsKeyspace)
//    },
//    10*time.Second,
//    1*time.Second)
//  if err != nil {
//    glog.Fatalf("error setting up db connection :: %v", err)
//  } else {
//    dbConn, ok := res.(*dbconn.DBConnCassandra)
//    if !ok {
//      glog.Fatalf("error setting up db connection :: %v", err)
//    } else {
//      glog.Infof("initialized cassandra db connection: %v successfully",
//        dbConn)
//    }
//  }
func WaitForFuncToSucceed(fnLabel string, fn func() (interface{}, error),
  timeout, pollInternal time.Duration) (result interface{}, err error) {

  stop := time.After(timeout)
  // poll every pollInternal seconds.
  ticker := time.NewTicker(pollInternal)
  defer ticker.Stop()

  for {
    if result, err = fn(); err != nil {
      glog.V(2).Infof("waiting for %s to succeed, failed with :: %s", fnLabel,
        err.Error())
    } else {
      glog.V(2).Infof("%s succeeded", fnLabel)
      return result, nil
    }
    select {
    case <-stop:
      glog.Errorf("timed out waiting for %s to succeed", fnLabel)
      return result, fmt.Errorf("timed out waiting for %s to succeed :: %v",
        fnLabel, err)
    case <-ticker.C:
      // block at the top (inside for) will be executed
    }
  }
}

// ParseKeyValues parses command line parameters in key=value form into a key
// value map and others into arguments.
func ParseKeyValues(params []string) (map[string]string, []string, error) {
  args := []string{}
  kvMap := make(map[string]string)
  for _, param := range params {
    split := strings.SplitN(param, "=", 2)
    if len(split) == 1 {
      args = append(args, param)
    } else if len(split) == 2 {
      kvMap[split[0]] = split[1]
    } else {
      glog.Errorf("invalid parameter %s", param)
      return nil, nil, os.ErrInvalid
    }
  }
  return kvMap, args, nil
}

// GetSSHPublicKey returns the public key for local host. It returns an
// empty key in case of non production environment.
// Returns actual key on success.
func GetSSHPublicKey(sshPublicKeyPath string) ([]byte, error) {
  // check if this is a non-production system by looking at username.
  userInfo, errUser := user.Current()
  if errUser != nil {
    glog.Error("error getting main user info: ", errUser)
    return nil, errUser
  }
  if userInfo.Username != "zerostack" {
    return []byte{}, nil
  }
  // get the actual key for a production system
  data, errRead := ioutil.ReadFile(sshPublicKeyPath)
  if errRead != nil {
    glog.Errorf("could not read ssh public key from %s :: %v",
      sshPublicKeyPath, errRead)
    return nil, errRead
  }
  return bytes.TrimSpace(data), nil
}

// SetDiffBytesSlice compares slice of byte array and returns element present in
// "from" and not present in "to".
// Returns a set difference, "from" - "to".
func SetDiffBytesSlice(from [][]byte, to [][]byte) [][]byte {
  ret := [][]byte{}
  for _, src := range from {
    found := false
    for _, dest := range to {
      if bytes.Equal(src, dest) {
        found = true
        break
      }
    }
    if !found {
      ret = append(ret, src)
    }
  }
  return ret
}

// StartProcess starts a process locally, waits for a small timeout to see that
// it starts and returns the process pointer on success.
// It also waits for process to complete in a goroutine, so that the process
// does not become zombie on getting killed.
// Returns nil and error in case of failure.
func StartProcess(name, env, binPath string, timeout time.Duration,
  stdout, stderr io.Writer, options ...string) (*os.Process, error) {

  cmd := exec.Command(binPath, options...)
  if stdout != nil {
    cmd.Stdout = stdout
  }
  if stderr != nil {
    cmd.Stderr = stderr
  }
  if len(env) > 0 {
    envProc := os.Environ()
    envProc = append(envProc, env)
    cmd.Env = envProc
  }
  return StartCmdProcess(name, cmd, timeout)
}

// StartCmdProcess starts the process specified in "cmd".
// waits for a small timeout to see that it starts and returns the
// process pointer on success.
func StartCmdProcess(name string, cmd *exec.Cmd,
  timeout time.Duration) (*os.Process, error) {

  errStart := cmd.Start()
  if errStart != nil {
    glog.Errorf("could not start service %s, command %s failed :: %v",
      name, cmd.Args, errStart)
    return nil, errStart
  }

  // create a buffered channel of size 1 to avoid leaking a go-routine
  // on a channel write attempt.
  errChan := make(chan error, 1)
  go func() {
    errChan <- cmd.Wait()
  }()

  // Wait for a timeout so that process starts properly or quits.
  // This small timeout is to see if the process started without any problems.
  select {
  case errRun := <-errChan:
    if errRun != nil {
      glog.Errorf("command %s failed :: %v", cmd.Args, errRun)
      return nil, errRun
    }
    glog.Infof("command %s has exited successfully", cmd.Args)
    return nil, nil

  case <-time.After(timeout):
    break
  }

  return cmd.Process, nil
}

// BytesToUint64 converts byte slice to uint64.
func BytesToUint64(slice []byte) (uint64, error) {
  ret, err := strconv.ParseUint(
    strings.TrimSpace(string(slice)), 10, 64)
  if err != nil {
    return 0, fmt.Errorf("error parsing string %s to uint :: %v", slice, err)
  }
  return ret, nil
}

// EnvToSetMaxCPUs returns env variable to set max CPU's used at run time.
func EnvToSetMaxCPUs(numCPU int) string {
  var env string
  if numCPU == 0 {
    env = ""
  } else if numCPU > runtime.NumCPU() {
    env = fmt.Sprintf("GOMAXPROCS=%d", runtime.NumCPU())
  } else {
    env = fmt.Sprintf("GOMAXPROCS=%d", numCPU)
  }
  return env
}

// ReadFileByLine reads a file and returns a slice of lines
func ReadFileByLine(filePath string) ([]string, error) {
  if len(filePath) == 0 {
    return nil, fmt.Errorf("no file provided")
  }

  fd, errOpen := os.Open(filePath)
  if errOpen != nil {
    return nil, fmt.Errorf("failed to open file %s :: "+
      "%v", filePath, errOpen)
  }
  defer fd.Close()

  reader := bufio.NewReader(fd)
  scanner := bufio.NewScanner(reader)

  var lines []string
  for scanner.Scan() {
    lines = append(lines, scanner.Text())
  }
  return lines, nil
}

// SliceContainsStr returns if a string slice contains the given string.
func SliceContainsStr(list []string, str string) bool {
  for _, elem := range list {
    if strings.EqualFold(elem, str) {
      return true
    }
  }
  return false
}

// FindDisks finds all disks on the system.
func FindDisks() ([]string, error) {
  cmd := exec.Command("lsblk", "-o", "KNAME,TYPE")
  out, err := cmd.CombinedOutput()
  if err != nil {
    return nil, fmt.Errorf("error executing command %s :: %v", cmd.Args, err)
  }
  lines := strings.Split(string(out), "\n")
  disks := make([]string, 0, len(lines))
  for _, line := range lines {
    if strings.Contains(line, "disk") {
      tokens := strings.Fields(line)
      if len(tokens) != 2 {
        return nil, fmt.Errorf("error parsing command %s output", cmd.Args)
      }
      disks = append(disks, tokens[0])
    }
  }
  return disks, nil
}

// HashStr computes the hash of a given string
func HashStr(key string) int {
  h := fnv.New32a()
  h.Write([]byte(key))
  return int(h.Sum32())
}
