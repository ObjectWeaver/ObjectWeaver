package converison

import (
	"fmt"
	"google.golang.org/protobuf/types/known/structpb"
)

// ConvertStructpbToMap converts *structpb.Struct to map[string]interface{}
func ConvertStructpbToMap(s *structpb.Struct) (map[string]any, error) {
	if s == nil {
		return nil, fmt.Errorf("input structpb.Struct is nil")
	}

	result := make(map[string]any)

	for key, value := range s.GetFields() {
		convertedValue, err := convertStructpbValue(value)
		if err != nil {
			return nil, err
		}
		result[key] = convertedValue
	}

	return result, nil
}

// Helper function to convert individual *structpb.Value to Go types
func convertStructpbValue(value *structpb.Value) (any, error) {
	switch v := value.GetKind().(type) {
	case *structpb.Value_NullValue:
		return nil, nil
	case *structpb.Value_NumberValue:
		return v.NumberValue, nil
	case *structpb.Value_StringValue:
		return v.StringValue, nil
	case *structpb.Value_BoolValue:
		return v.BoolValue, nil
	case *structpb.Value_StructValue:
		return ConvertStructpbToMap(v.StructValue)
	case *structpb.Value_ListValue:
		return convertStructpbList(v.ListValue)
	default:
		return nil, fmt.Errorf("unsupported structpb.Value type: %T", v)
	}
}

// Helper function to convert *structpb.ListValue to a Go slice
func convertStructpbList(list *structpb.ListValue) ([]any, error) {
	result := make([]any, len(list.GetValues()))

	for i, value := range list.GetValues() {
		convertedValue, err := convertStructpbValue(value)
		if err != nil {
			return nil, err
		}
		result[i] = convertedValue
	}

	return result, nil
}
