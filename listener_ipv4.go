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
// This file implements helper functions that listen only on ipv4 addresses

package util

import (
  "net"
  "net/http"
  "time"
)

const cTCPKeepAlivePeriod time.Duration = 3 * time.Minute

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
// Copied from net/http/server.go
type tcpKeepAliveListener struct {
  *net.TCPListener
}

// Accept is copied from net/http/server.go in addition to normal
// accept it sets TCP Keep Alive to cTCPKeepAlivePeriod (3 min)
func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
  tc, err := ln.AcceptTCP()
  if err != nil {
    return
  }
  tc.SetKeepAlive(true)
  tc.SetKeepAlivePeriod(cTCPKeepAlivePeriod)
  return tc, nil
}

// Listener4 is a helper function to create a listener on ipv4 only
func Listener4(addr string) (net.Listener, error) {
  if addr == "" {
    addr = ":http"
  }
  listener, err := net.Listen("tcp4", addr)
  if err != nil {
    return nil, err
  }
  return tcpKeepAliveListener{listener.(*net.TCPListener)}, nil
}

// HTTPListenAndServe4 is equivalent to http.ListenAndServe, it listens on
// TCP IPv4 network address only, not on IPv6 addresses
func HTTPListenAndServe4(addr string, handler http.Handler) error {
  ln, errLis := Listener4(addr)
  if errLis != nil {
    return errLis
  }
  return http.Serve(ln, handler)
}

// HTTPServerListenAndServe4 is equivalent to ListenAndServe method on http.Server
// it listens on TCP IPv4 network address only, not on IPv6 addresses
func HTTPServerListenAndServe4(srv *http.Server) error {
  ln, errLis := Listener4(srv.Addr)
  if errLis != nil {
    return errLis
  }
  return srv.Serve(ln)
}
