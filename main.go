package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"objectweaver/grpcService"
	"objectweaver/logger"
	"os"
	"runtime"

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

	// Check if TLS is enabled via environment variables
	certPEM := os.Getenv("SERVER_CERT_PEM")
	keyPEM := os.Getenv("SERVER_KEY_PEM")
	caCertPEM := os.Getenv("CA_CERT_PEM")

	var lis net.Listener
	var err error

	// If TLS environment variables are set, use TLS
	if certPEM != "" && keyPEM != "" && caCertPEM != "" {
		logger.Println("Starting server with TLS enabled")

		// Load server certificate and key with support for negative serial numbers
		cert, err := loadX509KeyPairWithNegativeSerial([]byte(certPEM), []byte(keyPEM))
		if err != nil {
			log.Fatalf("Failed to create key pair: %v", err)
		}

		// Load CA certificate pool
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM([]byte(caCertPEM)) {
			log.Fatal("Failed to add CA cert to pool.")
		}

		// Configure TLS with mTLS support
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    caCertPool,
			VerifyConnection: func(cs tls.ConnectionState) error {
				// Custom verification that allows certificates with negative serial numbers
				// This is needed for compatibility with Go 1.23+ on Windows
				opts := x509.VerifyOptions{
					Roots:         caCertPool,
					Intermediates: x509.NewCertPool(),
					KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
				}

				for _, cert := range cs.PeerCertificates[1:] {
					opts.Intermediates.AddCert(cert)
				}

				_, err := cs.PeerCertificates[0].Verify(opts)
				return err
			},
		}

		// Create TLS listener
		lis, err = tls.Listen("tcp", ":"+port, tlsConfig)
		if err != nil {
			log.Fatalf("Failed to listen on port %s with TLS: %v", port, err)
		}
	} else {
		log.Println("Starting server without TLS (plain TCP)")
		// Create plain TCP listener
		lis, err = net.Listen("tcp", ":"+port)
		if err != nil {
			log.Fatalf("Failed to listen on port %s: %v", port, err)
		}
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

	intialThreads := runtime.GOMAXPROCS(0)
	runtime.GOMAXPROCS(intialThreads * 2)

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
