/*
Copyright 2022 FerryProxy Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package server

import (
	"net"
	"net/http"
	"sync"

	"github.com/go-logr/logr"
)

type Controller struct {
	mut    sync.Mutex
	logger logr.Logger
}

func (c *Controller) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	switch r.Method {
	case http.MethodHead:
	case http.MethodGet:
		c.Get(rw, r)
	default:
		http.Error(rw, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
}

// Get GET /ports/unused
func (c *Controller) Get(rw http.ResponseWriter, r *http.Request) {
	c.mut.Lock()
	defer c.mut.Unlock()

	listen, err := net.Listen("tcp", ":0")
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	defer listen.Close()
	addr := listen.Addr().String()
	_, port, _ := net.SplitHostPort(addr)
	rw.Write([]byte(port))
}
