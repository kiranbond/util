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
// This file implements utility functions related to http call handling.

package util

import (
  "bytes"
  "context"
  "encoding/json"
  "errors"
  "fmt"
  "io"
  "io/ioutil"
  "mime/multipart"
  "net/http"
  "net/http/httputil"
  "os"
  "path/filepath"
  "strings"
  "time"

  "code.google.com/p/go-uuid/uuid"
  "github.com/gogo/protobuf/proto"
  "github.com/golang/glog"

  "zerostack/common/protofiles/starnetapi"
)

const (
  // ZSSeqIDHeader is used in libconn channel to keep sequence numbering
  // between messages since we allow out of order completion rather than
  // in-order responses.
  ZSSeqIDHeader string = "X-ZS-Client-Sequence-Id"
  // ZSKeepAliveHeader is used in libconn channel to indicate this is a
  // keepalive header and is not an end to end message.
  ZSKeepAliveHeader string = "X-ZS-KeepAlive"
)

// AddSeqIDHeader adds a ZSSeqIDHeader to response if request has one
func AddSeqIDHeader(rw http.ResponseWriter, req *http.Request) {
  seqhdr := req.Header.Get(ZSSeqIDHeader)
  if seqhdr != "" {
    rw.Header().Set(ZSSeqIDHeader, seqhdr)
  }
}

// AddKeepAliveHeader adds a ZSKeepAliveHeader to response if request has one
func AddKeepAliveHeader(rw http.ResponseWriter, req *http.Request) {
  kphdr := req.Header.Get(ZSKeepAliveHeader)
  if kphdr != "" {
    rw.Header().Set(ZSKeepAliveHeader, kphdr)
  }
}

// GenericDo is a wrapper around the Go http package.
func GenericDo(method, url string, input, output interface{},
  expStatus int, timeout time.Duration) (*http.Response, error) {

  var body io.Reader
  if input != nil {
    if s, ok := input.(string); ok {
      // do not JSON encode a string, just pass as is.
      body = bytes.NewBufferString(s)
    } else {
      inputBytes, err := json.Marshal(input)
      if err != nil {
        glog.Errorf("failed to marshal input :: %v", err)
        return nil, err
      }
      body = bytes.NewBuffer(inputBytes)
    }
  }
  req, err := http.NewRequest(method, url, body)
  if err != nil {
    glog.Errorf("failed to create new http request :: %v", err)
    return nil, err
  }
  glog.V(1).Infof("DoGeneric: method: %s url: %s", method, url)

  req.Header.Set("Content-Type", "application/json")

  cl := &http.Client{Timeout: timeout}

  if glog.V(2) {
    var out []byte
    out, err = httputil.DumpRequest(req, true)
    if err != nil {
      glog.Warningf("failed to dump http request :: %v", err)
    }
    glog.V(2).Infof("DoGeneric: full request: %s", out)
  }

  resp, err := cl.Do(req)
  if err != nil {
    glog.Errorf("failed to get the response :: %v", err)
    return nil, err
  }
  defer resp.Body.Close()

  if resp.StatusCode != expStatus {
    return nil, fmt.Errorf("unexpected Statuscode: %v", resp.StatusCode)
  }

  if output != nil { // expecting JSON response
    respBody, err := ioutil.ReadAll(resp.Body)
    if err != nil {
      glog.Errorf("failed to read the JSON body :: %v", err)
      return resp, err
    }

    glog.V(2).Infof("response body: %v", string(respBody))

    ct := resp.Header.Get("Content-Type")
    if !strings.Contains(ct, "json") {
      return nil, fmt.Errorf("error : unexpected content type in resp: %s", ct)
    }
    err = json.Unmarshal(respBody, output)
    if err != nil {
      glog.Errorf("failed to unmarshal the response body :: %v", err)
      return resp, err
    }
  }

  return resp, nil
}

// DoGetJSON does a http Get call and unmarshals JSON response from body.
func DoGetJSON(path string, timeout time.Duration, rspJSON interface{}) error {
  cl := http.Client{Timeout: timeout}
  var (
    rsp  *http.Response
    err  error
    data []byte
  )
  if rsp, err = cl.Get(path); err != nil {
    return err
  }
  if rsp == nil || rsp.Body == nil {
    return fmt.Errorf("nil rsp or rsp Body from GET")
  }

  defer rsp.Body.Close()
  if data, err = ioutil.ReadAll(rsp.Body); err != nil {
    return err
  }

  if rspJSON == nil {
    return fmt.Errorf("input param is nil for unmarshaling")
  }

  if err = json.Unmarshal(data, rspJSON); err != nil {
    return err
  }

  if rsp.StatusCode != http.StatusOK {
    return fmt.Errorf("http error :: %s", http.StatusText(rsp.StatusCode))
  }

  return nil
}

// DoPostJSON does a HTTP POST to the provided addr/path with the
// provided []byte as payload. If successful, it will try to parse response
// as JSON into the interface provided. The method also honors the "ctx"
// parameter.
func DoPostJSON(ctx context.Context, path string, payload []byte,
  rspJSON interface{}) (*http.Response, error) {

  var (
    rsp  *http.Response
    err  error
    data []byte
  )

  req, err := http.NewRequest("POST", path, bytes.NewBuffer(payload))
  if err != nil {
    return nil, fmt.Errorf("error creating new request: %v", err)
  }

  req.Header.Add("content-type", "application/json")
  req = req.WithContext(ctx)
  rsp, err = http.DefaultClient.Do(req)
  if err != nil {
    return nil, fmt.Errorf("error in Do: %v", err)
  }
  if rsp == nil || rsp.Body == nil {
    return nil, fmt.Errorf("nil rsp or rsp Body from POST call")
  }

  defer rsp.Body.Close()

  if rsp.StatusCode != http.StatusOK && rsp.StatusCode != http.StatusCreated &&
    rsp.StatusCode != http.StatusAccepted &&
    rsp.StatusCode != http.StatusNoContent {

    return nil, fmt.Errorf("error in http response: %s", rsp.Status)
  }

  if data, err = ioutil.ReadAll(rsp.Body); err != nil {
    return nil, fmt.Errorf("error reading response body: %v", err)
  }

  if rspJSON == nil {
    return rsp, fmt.Errorf("input param is nil for unmarshaling")
  }

  if err = json.Unmarshal(data, rspJSON); err != nil {
    return rsp, fmt.Errorf("error unmarshaling response data %s: %v",
      string(data), err)
  }
  return rsp, nil
}

// DoGet does a HTTP GET to the provided addr/path and headers.
func DoGet(path string, headers map[string]string, timeout time.Duration) (
  *http.Response, error) {

  // Compose HTTP request.
  req, err := http.NewRequest("GET", path, nil)
  if err != nil {
    return nil, err
  }

  for k, v := range headers {
    req.Header.Add(k, v)
  }

  // Make HTTP call.
  glog.V(1).Infof("making HTTP GET call, path: %v, timeout: %v", path, timeout)
  cl := http.Client{Timeout: timeout}
  return cl.Do(req)
}

// DoPost does a HTTP POST to the provided addr/path with the provided []byte as
// payload.
func DoPost(path string, payload []byte, headers map[string]string,
  timeout time.Duration) (*http.Response, error) {

  // Compose HTTP request.
  req, err := http.NewRequest("POST", path, bytes.NewBuffer(payload))
  if err != nil {
    return nil, err
  }

  for k, v := range headers {
    req.Header.Add(k, v)
  }

  // Make HTTP call.
  glog.V(1).Infof("making HTTP POST call, path: %v, payload: %v, timeout: %v",
    path, payload, timeout)
  cl := http.Client{Timeout: timeout}
  return cl.Do(req)
}

// WriteErrorHTTPResponse creates a error message of type ErrorMsg using a
// summary and detail for the message.
func WriteErrorHTTPResponse(rw http.ResponseWriter, summary, detail string,
  statusCode int) {

  errorMsg := starnetapi.ErrorMsg{
    Summary: proto.String(summary),
    Detail:  proto.String(detail),
  }
  WriteJSONHTTPResponse(rw, errorMsg, statusCode)
}

// WriteHTTPResponseBytes writes http reponse bytes to response writer.
func WriteHTTPResponseBytes(rw http.ResponseWriter, body []byte,
  statusCode int) {

  rw.Header().Set("Content-Type", "application/json")
  rw.WriteHeader(statusCode)
  rw.Write(body)
}

// WriteJSONHTTPResponse prepares a http response after marshaling the body
// parameter. The response is ready to be sent. After
// this function no further changes should be made to the response.
func WriteJSONHTTPResponse(rw http.ResponseWriter, body interface{},
  statusCode int) {

  bytes, errMarshal := json.Marshal(body)
  if errMarshal != nil {
    bytes, _ = json.Marshal("internal error while marshaling response")
    statusCode = http.StatusInternalServerError
  }
  WriteHTTPResponseBytes(rw, bytes, statusCode)
}

// WriteJSONHttpResponseIndented prepares a http response after indent marshaling
// the body parameter. The response is ready to be sent. After
// this function no further changes should be made to the response.
func WriteJSONHttpResponseIndented(rw http.ResponseWriter, body interface{},
  statusCode int) {

  bytes, errMarshal := json.MarshalIndent(body, "", "  ")
  if errMarshal != nil {
    bytes, _ = json.Marshal("internal error while marshaling response")
    statusCode = http.StatusInternalServerError
  }
  WriteHTTPResponseBytes(rw, bytes, statusCode)
}

// DownloadFromURL downloads a file from a URL and saves it to fileName (duh!)
// Returns number of bytes written to file and error.
func DownloadFromURL(url string, fileName string) (int64, error) {
  output, err := os.Create(fileName)
  if err != nil {
    return 0, fmt.Errorf("error creating file %s :: %v", fileName, err)
  }
  defer output.Close()

  response, err := http.Get(url)
  if err != nil {
    return 0, fmt.Errorf("error downloading %s :: %v", url, err)
  }
  defer response.Body.Close()

  n, err := io.Copy(output, response.Body)
  if err != nil {
    return 0, fmt.Errorf("error downloading %s :: %v", url, err)
  }
  return n, nil
}

// UploadFileToURL uploads a file to url with given fields as params.
func UploadFileToURL(url string, params map[string]string, paramName,
  path string) error {

  file, errOpen := os.Open(path)
  if errOpen != nil {
    return fmt.Errorf("error opening file %s :: %v", path, errOpen)
  }
  defer file.Close()

  body := &bytes.Buffer{}
  writer := multipart.NewWriter(body)
  for key, val := range params {
    _ = writer.WriteField(key, val)
  }
  part, errW := writer.CreateFormFile(paramName, filepath.Base(path))
  if errW != nil {
    return fmt.Errorf("error in createformfile for file %s :: %v",
      filepath.Base(path), errW)
  }
  if _, errC := io.Copy(part, file); errC != nil {
    return fmt.Errorf("error in io copy for file %s :: %v", file.Name(), errC)
  }
  if errC := writer.Close(); errC != nil {
    return fmt.Errorf("error closing writer :: %v", errC)
  }

  req, errH := http.NewRequest("POST", url, body)
  if errH != nil {
    return fmt.Errorf("error creating new http request for url %s :: %v",
      url, errH)
  }
  req.Header.Add("Content-Type", writer.FormDataContentType())

  client := &http.Client{}
  resp, errD := client.Do(req)
  if errD != nil {
    return fmt.Errorf("error in http client for url %s :: %v", req.URL, errD)
  }
  if resp.StatusCode != http.StatusOK {
    respBody := &bytes.Buffer{}
    _, errR := respBody.ReadFrom(resp.Body)
    defer resp.Body.Close()
    if errR != nil {
      return fmt.Errorf("http client failed with http status code %s, failed"+
        " to read http response :: %v", resp.Status, errR)
    }
    return fmt.Errorf("http client failed with http status code %s :: %s",
      resp.Status, respBody)
  }
  return nil
}

// ErrorInfo defines error messages in response to HTTP requests
type ErrorInfo struct {
  Summary string `json:"summary"`
  Detail  string `json:"detail"`

  // DebugMsg will be used for internal debugging and should not be used by to
  // display error messages in UI.
  DebugMsg string `json:"debug_msg"`
}

// SuccessInfo defines success messages in response to HTTP requests
type SuccessInfo struct {
  Message string `json:"message"`
}

// NewErrorInfo generates and returns an struct represnting the
// error message that is sent in response to failed HTTP requests
func NewErrorInfo(summary, detail, debugMsg string) *ErrorInfo {
  return &ErrorInfo{
    Summary:  summary,
    Detail:   detail,
    DebugMsg: debugMsg,
  }
}

// NewSuccessInfo generates and returns an struct represnting the
// success message that is sent in response to successful HTTP requests
func NewSuccessInfo(msg string) *SuccessInfo {
  var successInfo SuccessInfo
  if len(msg) > 0 {
    successInfo.Message = msg
  }
  return &successInfo
}

func validateUUID(uuidStr string) error {

  if len(uuidStr) == 0 {
    return errors.New("no uuid supplied")
  }

  uuid := uuid.Parse(uuidStr)
  if uuid == nil {
    return errors.New("bad UUID format")
  }
  return nil
}
