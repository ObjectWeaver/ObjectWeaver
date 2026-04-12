package grpcService

import (
	"context"
	"crypto/tls"
	"log"
	"github.com/ObjectWeaver/ObjectWeaver/logger"
	"os"

	pb "github.com/ObjectWeaver/ObjectWeaver/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

// Server is used to implement the gRPC service with injected dependencies
type Server struct {
	pb.UnimplementedJSONSchemaServiceServer
	requestConverter RequestConverter
	circularChecker  CircularDefinitionChecker
	configFactory    ConfigFactory
	generatorService GeneratorService
	responseBuilder  ResponseBuilder
}

// NewServer creates a new Server with all dependencies injected
func NewServer(
	requestConverter RequestConverter,
	circularChecker CircularDefinitionChecker,
	configFactory ConfigFactory,
	generatorService GeneratorService,
	responseBuilder ResponseBuilder,
) *Server {
	return &Server{
		requestConverter: requestConverter,
		circularChecker:  circularChecker,
		configFactory:    configFactory,
		generatorService: generatorService,
		responseBuilder:  responseBuilder,
	}
}

// NewDefaultServer creates a new Server with default implementations
func NewDefaultServer() *Server {
	return NewServer(
		NewDefaultRequestConverter(),
		NewDefaultCircularDefinitionChecker(),
		NewDefaultConfigFactory(),
		NewDefaultGeneratorService(),
		NewDefaultResponseBuilder(),
	)
}

// GenerateObject implements the gRPC interface by calling GenerateObjectV2
func (s *Server) GenerateObject(ctx context.Context, req *pb.RequestBody) (*pb.Response, error) {
	return s.GenerateObjectV2(ctx, req)
}

// StreamGeneratedObjects implements the gRPC interface by calling StreamGeneratedObjectsV2
func (s *Server) StreamGeneratedObjects(req *pb.RequestBody, stream pb.JSONSchemaService_StreamGeneratedObjectsServer) error {
	return s.StreamGeneratedObjectsV2(req, stream)
}

// NewGRPCServer creates a new gRPC server with authentication interceptor.
func NewGRPCServer() *grpc.Server {

	// Load the TLS certificate and key
	certFile := os.Getenv("CERT_FILE")
	keyFile := os.Getenv("KEY_FILE")

	grpcServer := &grpc.Server{}
	if os.Getenv("GRPC_UNSECURE") != "true" {
		if certFile == "" || keyFile == "" {
			logger.Println("CERT_FILE and KEY_FILE environment variables must be set")
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
	pb.RegisterJSONSchemaServiceServer(grpcServer, NewDefaultServer())
	// Enable gRPC reflection
	reflection.Register(grpcServer)

	return grpcServer
}
