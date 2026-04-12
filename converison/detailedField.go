package converison

import (
	pb "objectweaver/grpc"
)

// ConvertDetailedFieldValue converts the DetailedField's value from structpb.Struct to map[string]any
func ConvertDetailedFieldValue(df *pb.DetailedField) (map[string]any, error) {
	if df == nil || df.Value == nil {
		return nil, nil
	}
	return ConvertStructpbToMap(df.Value)
}

// GetFieldMetadata extracts metadata from DetailedField
func GetFieldMetadata(df *pb.DetailedField) *pb.FieldMetadata {
	if df == nil {
		return nil
	}
	return df.Metadata
}

// ConvertDetailedDataToMap converts all DetailedField values in a map to Go maps
func ConvertDetailedDataToMap(detailedData map[string]*pb.DetailedField) (map[string]map[string]any, error) {
	if detailedData == nil {
		return nil, nil
	}

	result := make(map[string]map[string]any)
	for key, df := range detailedData {
		value, err := ConvertDetailedFieldValue(df)
		if err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, nil
}

// ConvertChoiceValue converts a Choice's value from structpb.Struct to map[string]any
func ConvertChoiceValue(choice *pb.Choice) (map[string]any, error) {
	if choice == nil || choice.Value == nil {
		return nil, nil
	}
	return ConvertStructpbToMap(choice.Value)
}
