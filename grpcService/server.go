package grpcService

import (
	"crypto/tls"
	pb "github.com/henrylamb/object-generation-golang/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"log"
	"os"
)

// server is used to implement the gRPC service.
type Server struct {
	pb.UnimplementedJSONSchemaServiceServer
}

// NewGRPCServer creates a new gRPC server with authentication interceptor.
func NewGRPCServer() *grpc.Server {

	// Load the TLS certificate and key
	certFile := os.Getenv("CERT_FILE")
	keyFile := os.Getenv("KEY_FILE")

	grpcServer := &grpc.Server{}
	if os.Getenv("GRPC_UNSECURE") != "true" {
		if certFile == "" || keyFile == "" {
			log.Println("CERT_FILE and KEY_FILE environment variables must be set")
			return nil
		}

		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			log.Fatalf("Failed to load key pair: %v", err)
		}

		// Create a certificate pool and add the server certificate
		creds := credentials.NewTLS(&tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		})

		grpcServer = grpc.NewServer(
			grpc.UnaryInterceptor(AuthInterceptor),
			grpc.Creds(creds),
		)
	} else {
		grpcServer = grpc.NewServer(
			grpc.UnaryInterceptor(AuthInterceptor),
		)
	}

	// Register your service implementation with the gRPC server
	pb.RegisterJSONSchemaServiceServer(grpcServer, &Server{})
	// Enable gRPC reflection
	reflection.Register(grpcServer)

	return grpcServer
}
