package main

import (
	"net"
	"net/http"
	"github.com/ObjectWeaver/ObjectWeaver/service"
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
			ReadHeaderTimeout: 10 * time.Second,
			ReadTimeout:       5 * time.Minute, // Allow up to 5 minutes for reading request
			WriteTimeout:      5 * time.Minute, // Allow up to 5 minutes for writing response
			IdleTimeout:       2 * time.Minute, // Allow 2 minutes for keep-alive
			MaxHeaderBytes:    1 << 20,         // 1 MB max header size
		},
		ready: r,
	}
}

func (hm *HTTPManager) Start() error {
	hm.ready <- true
	return hm.server.Serve(hm.listener)
}
