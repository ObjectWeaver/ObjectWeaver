package grpcService

import (
	"context"
	"errors"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// AuthInterceptor intercepts incoming requests to check for a valid authorization token.
func AuthInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	// Extract metadata from the context
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("missing metadata")
	}

	// Log the metadata for debugging purposes
	fmt.Printf("Metadata received: %v\n", md)

	// Get the API key from the metadata, using a custom header (e.g., "x-api-key")
	apiKey, ok := md["x-api-key"]
	if !ok || len(apiKey) == 0 {
		return nil, errors.New("API key is not provided")
	}

	// Validate the API key (implement your validation logic here)
	if err := validateToken(apiKey[0]); err != nil {
		return nil, err
	}

	// Continue to the handler if the token is valid
	return handler(ctx, req)
}

// validateToken checks if the provided token is valid (you should implement your own logic here).
func validateToken(token string) error {
	// Example validation logic: check if the token is "Bearer some-valid-token"
	if token != os.Getenv("PASSWORD") {
		return errors.New("invalid authorization token")
	}
	return nil
}
