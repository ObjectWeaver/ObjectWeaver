package converison

import "google.golang.org/protobuf/types/known/structpb"

// Convert map[string]interface{} to *structpb.Struct
func ConvertMapToStruct(m map[string]interface{}) (*structpb.Struct, error) {
	return structpb.NewStruct(m)
}

// Convert *structpb.Struct to map[string]interface{}
func ConvertStructToMap(s *structpb.Struct) (map[string]interface{}, error) {
	return s.AsMap(), nil
}
