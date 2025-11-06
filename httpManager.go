// Copyright (C) 2025-present ObjectWeaver.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the Server Side Public License, version 1,
// as published by ObjectWeaver.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// Server Side Public License for more details.
//
// You should have received a copy of the Server Side Public License
// along with this program. If not, see
// <https://objectweaver.dev/licensing/server-side-public-license>.
package main

import (
	"objectweaver/service"
	"net"
	"net/http"
	"time"
)

type HTTPManager struct {
	listener net.Listener
	server   *http.Server
	ready    chan bool
}

func NewHTTPManager(l net.Listener, r chan bool) *HTTPManager {
	return &HTTPManager{
		listener: l,
		server: &http.Server{
			Handler:           service.NewHttpServer().Router,
			ReadHeaderTimeout: 5 * time.Second,
		},
		ready: r,
	}
}

func (hm *HTTPManager) Start() error {
	hm.ready <- true
	return hm.server.Serve(hm.listener)
}
