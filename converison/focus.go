package converison

import (
	"github.com/ObjectWeaver/ObjectWeaver/jsonSchema"

	pb "github.com/ObjectWeaver/ObjectWeaver/grpc"
)

// ConvertProtoToFocus converts a protobuf Focus to the Go model Focus
func ConvertProtoToFocus(protoFocus *pb.Focus) *jsonSchema.Focus {
	if protoFocus == nil {
		return nil
	}

	return &jsonSchema.Focus{
		Prompt:       protoFocus.Prompt,
		Fields:       protoFocus.Fields,
		KeepOriginal: protoFocus.KeepOriginal,
	}
}

// ConvertModelToProtoFocus converts a Go model Focus to a protobuf Focus
func ConvertModelToProtoFocus(modelFocus *jsonSchema.Focus) *pb.Focus {
	if modelFocus == nil {
		return nil
	}

	return &pb.Focus{
		Prompt:       modelFocus.Prompt,
		Fields:       modelFocus.Fields,
		KeepOriginal: modelFocus.KeepOriginal,
	}
}
