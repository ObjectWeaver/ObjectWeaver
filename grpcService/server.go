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
// <https://github.com/ObjectWeaver/ObjectWeaver/blob/main/LICENSE.txt>.
package grpcService

import (
	"context"
	"crypto/tls"
	"log"
	"os"

	pb "github.com/objectweaver/go-sdk/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

// server is used to implement the gRPC service.
type Server struct {
	pb.UnimplementedJSONSchemaServiceServer
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
