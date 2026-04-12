package converison

import (
	"github.com/ObjectWeaver/ObjectWeaver/jsonSchema"

	pb "github.com/ObjectWeaver/ObjectWeaver/grpc"
)

// ConvertProtoToRequestFormat converts a protobuf RequestFormat to the Go model RequestFormat
func ConvertProtoToRequestFormat(protoReq *pb.RequestFormat) *jsonSchema.RequestFormat {
	if protoReq == nil {
		return nil
	}

	body, _ := ConvertStructToMap(protoReq.Body)

	return &jsonSchema.RequestFormat{
		URL:           protoReq.Url,
		Method:        protoReq.Method,
		Headers:       protoReq.Headers,
		Body:          body,
		Authorization: protoReq.Authorization,
		RequireFields: protoReq.RequireFields,
	}
}

// ConvertModelToProtoRequestFormat converts a Go model RequestFormat to a protobuf RequestFormat
func ConvertModelToProtoRequestFormat(modelReq *jsonSchema.RequestFormat) *pb.RequestFormat {
	if modelReq == nil {
		return nil
	}

	body, _ := ConvertMapToStruct(modelReq.Body)

	return &pb.RequestFormat{
		Url:           modelReq.URL,
		Method:        string(modelReq.Method),
		Headers:       modelReq.Headers,
		Body:          body,
		Authorization: modelReq.Authorization,
		RequireFields: modelReq.RequireFields,
	}
}
