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
	"errors"
	"log"

	"github.com/objectweaver/go-sdk/client"
	"github.com/objectweaver/go-sdk/converison"
	pb "github.com/objectweaver/go-sdk/grpc"
	"github.com/objectweaver/go-sdk/jsonSchema"
	"google.golang.org/protobuf/types/known/structpb"

	"objectweaver/checks"
	"objectweaver/orchestration/jos/domain"
	"objectweaver/orchestration/jos/factory"
)

// StreamGeneratedObjectsV2 - Uses V2 architecture for streaming object generation
func (s *Server) StreamGeneratedObjectsV2(req *pb.RequestBody, stream pb.JSONSchemaService_StreamGeneratedObjectsServer) error {
	// Convert protobuf request to internal format
	body := client.RequestBody{
		Prompt:     req.Prompt,
		Definition: converison.ConvertProtoToModel(req.Definition),
	}

	log.Println("Received streaming request for definition:", body.Definition)

	// Check for circular definitions
	if checks.CheckCircularDefinitions(body.Definition) {
		return errors.New("circular definitions found")
	}

	// Determine streaming mode based on schema configuration
	config := s.createStreamingConfig(body.Definition)

	// Create generator using factory
	generatorFactory := factory.NewGeneratorFactory(config)
	generator, err := generatorFactory.Create()
	if err != nil {
		log.Printf("Failed to create streaming generator: %v", err)
		return err
	}

	// Create generation request
	request := domain.NewGenerationRequest(body.Prompt, body.Definition).
		WithContext(stream.Context())

	// Determine which streaming method to use
	if config.Mode == factory.ModeStreamingProgressive {
		return s.streamProgressively(generator, request, stream)
	} else {
		return s.streamComplete(generator, request, stream)
	}
}

// streamComplete - Streams complete field values and accumulated state
func (s *Server) streamComplete(generator domain.Generator, request *domain.GenerationRequest, stream pb.JSONSchemaService_StreamGeneratedObjectsServer) error {
	// Get the streaming channel
	streamChan, err := generator.GenerateStream(request)
	if err != nil {
		log.Printf("Failed to start streaming generation: %v", err)
		return err
	}

	accumulatedData := make(map[string]interface{})
	totalCost := 0.0

	// Process each chunk
	for chunk := range streamChan {
		if chunk == nil {
			continue
		}

		// Update accumulated data with new field
		if chunk.Key != "" && chunk.Value != nil {
			accumulatedData[chunk.Key] = chunk.Value
		}

		// Use AccumulatedData if provided (for complete streaming mode)
		// Note: StreamChunk doesn't have AccumulatedData, we build it ourselves

		// Convert accumulated data to protobuf struct
		protoStruct, err := structpb.NewStruct(accumulatedData)
		if err != nil {
			log.Printf("Failed to convert map to protobuf struct: %v", err)
			return err
		}

		// Create streaming response
		response := &pb.StreamingResponse{
			Data:    protoStruct,
			UsdCost: totalCost,
			Status:  "Processing",
		}

		// Send the chunk
		if err := stream.Send(response); err != nil {
			log.Printf("Failed to send streaming response: %v", err)
			return err
		}

		// Track if this is the final chunk
		if chunk.IsFinal {
			break
		}
	}

	// Send final completion message
	finalStruct, err := structpb.NewStruct(accumulatedData)
	if err != nil {
		log.Printf("Failed to create final struct: %v", err)
		return err
	}

	finalResponse := &pb.StreamingResponse{
		Data:    finalStruct,
		UsdCost: totalCost,
		Status:  "Completed",
	}

	if err := stream.Send(finalResponse); err != nil {
		log.Printf("Failed to send final response: %v", err)
		return err
	}

	return nil
}

// streamProgressively - Streams token-by-token updates
func (s *Server) streamProgressively(generator domain.Generator, request *domain.GenerationRequest, stream pb.JSONSchemaService_StreamGeneratedObjectsServer) error {
	// Get the progressive streaming channel
	progressiveChan, err := generator.GenerateStreamProgressive(request)
	if err != nil {
		log.Printf("Failed to start progressive streaming: %v", err)
		return err
	}

	accumulatedData := make(map[string]interface{})
	totalCost := 0.0

	// Process each progressive chunk (token-level)
	for chunk := range progressiveChan {
		if chunk == nil {
			continue
		}

		// Update accumulated data with new token
		if chunk.NewToken != nil {
			if chunk.NewToken.Complete {
				// Token stream complete for this field, store final value
				accumulatedData[chunk.NewToken.Key] = chunk.NewToken.Partial
			} else {
				// Store partial value
				accumulatedData[chunk.NewToken.Key] = chunk.NewToken.Partial
			}
		}

		// Use CurrentMap if provided
		if chunk.CurrentMap != nil {
			accumulatedData = chunk.CurrentMap
		}

		// Convert to protobuf struct
		protoStruct, err := structpb.NewStruct(accumulatedData)
		if err != nil {
			// Log but don't fail - might be due to incomplete data
			log.Printf("Warning: Failed to convert partial data to struct: %v", err)
			continue
		}

		// Create streaming response
		response := &pb.StreamingResponse{
			Data:    protoStruct,
			UsdCost: totalCost,
			Status:  "Processing",
		}

		// Send the progressive update
		if err := stream.Send(response); err != nil {
			log.Printf("Failed to send progressive response: %v", err)
			return err
		}

		// Check if complete
		if chunk.IsFinal {
			break
		}
	}

	// Send final completion message
	finalStruct, err := structpb.NewStruct(accumulatedData)
	if err != nil {
		log.Printf("Failed to create final struct: %v", err)
		return err
	}

	finalResponse := &pb.StreamingResponse{
		Data:    finalStruct,
		UsdCost: totalCost,
		Status:  "Completed",
	}

	if err := stream.Send(finalResponse); err != nil {
		log.Printf("Failed to send final response: %v", err)
		return err
	}

	return nil
}

// createStreamingConfig creates streaming-specific configuration
func (s *Server) createStreamingConfig(schema *jsonSchema.Definition) *factory.GeneratorConfig {
	config := factory.DefaultGeneratorConfig()

	// Determine streaming granularity
	if s.shouldUseProgressiveStreaming(schema) {
		// Token-level streaming (most granular)
		config.Mode = factory.ModeStreamingProgressive
		config.Granularity = domain.GranularityToken
	} else {
		// Complete streaming (field-level with accumulated state)
		config.Mode = factory.ModeStreamingComplete
		config.Granularity = domain.GranularityField
	}

	// Configure based on schema properties
	config.MaxConcurrency = 10
	config.EnableCache = false // Disable cache for streaming

	return config
}

// shouldUseProgressiveStreaming determines if token-level streaming should be used
func (s *Server) shouldUseProgressiveStreaming(schema *jsonSchema.Definition) bool {
	// Check if any nested field has Stream enabled
	if schema.Properties != nil {
		for _, prop := range schema.Properties {
			if prop.Stream {
				return true // Use progressive streaming for token updates
			}
		}
	}

	// Default to field-level streaming
	return false
}
