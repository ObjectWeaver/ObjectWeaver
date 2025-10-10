package main

import (
	"firechimp/service"
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
