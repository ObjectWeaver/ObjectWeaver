package main

import (
	"firechimp/grpcService"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
)

const defaultPort = "2008"

type ServerManager interface {
	Start() error
}

type GRPCManager struct {
	listener net.Listener
	server   *grpc.Server
	ready    chan bool
}

func NewGRPCManager(l net.Listener, r chan bool) *GRPCManager {
	return &GRPCManager{
		listener: l,
		server:   grpcService.NewGRPCServer(),
		ready:    r,
	}
}

func (gm *GRPCManager) Start() error {
	gm.ready <- true
	return gm.server.Serve(gm.listener)
}

func startServers(httpReady, grpcReady chan bool) {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}
	m := cmux.New(lis)
	grpcL := m.Match(cmux.HTTP2())
	httpL := m.Match(cmux.HTTP1Fast())

	grpcM := NewGRPCManager(grpcL, grpcReady)
	httpM := NewHTTPManager(httpL, httpReady)

	go func() {
		if err := grpcM.Start(); err != nil {
			log.Fatalf("gRPC server failed: %v", err)
		}
	}()
	go func() {
		if err := httpM.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()
	if err := m.Serve(); err != nil {
		log.Fatalf("cmux server failed: %v", err)
	}
}

func main() {
	defer handlePanic()

	// Channels to signal when servers are ready
	httpReady := make(chan bool)
	grpcReady := make(chan bool)

	// Start the servers
	go startServers(httpReady, grpcReady)

	// Wait for both servers to be ready
	<-httpReady
	<-grpcReady

	// Print ASCII art once both servers are up
	printAscii()

	// Block main thread to keep the process running
	select {}
}

func handlePanic() {
	if r := recover(); r != nil {
		// Print a simple error message instead of a full stack trace
		fmt.Println("Application encountered an error:", r)
		// Optionally, you can log this error into a file or elsewhere
		logToFile(fmt.Sprintf("%v", r))
		os.Exit(1)
	}
}

func logToFile(msg string) {
	// Open or create a log file
	f, err := os.OpenFile("error.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Could not open log file:", err)
		return
	}
	defer f.Close()

	logger := log.New(f, "", log.LstdFlags)
	logger.Println(msg)
}
