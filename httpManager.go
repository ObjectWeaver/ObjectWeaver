package main

import (
	"net"
	"net/http"
	"objectweaver/service"
	"time"
)

type HTTPManager struct {
	listener net.Listener
	server   *http.Server
	ready    chan bool
}

func NewHTTPManager(l net.Listener, r chan bool) *HTTPManager {
	// Create the singleton server with shared generator
	httpServer := service.NewHttpServer()

	return &HTTPManager{
		listener: l,
		server: &http.Server{
			Handler:           httpServer.Router,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      60 * time.Second,
			IdleTimeout:       30 * time.Second,
			MaxHeaderBytes:    1 << 20,          // 1 MB max header size
		},
		ready: r,
	}
}

func (hm *HTTPManager) Start() error {
	hm.ready <- true
	return hm.server.Serve(hm.listener)
}
