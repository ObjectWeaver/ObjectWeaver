package jsonSchema

import pb "github.com/ObjectWeaver/ObjectWeaver/grpc"

type RequestBody struct {
	Prompt     string                 `json:"prompt"`
	Definition *Definition `json:"definition"`
}

// Create a response struct
type Response struct {
	Data         map[string]any               `json:"data"` //this data can then be marshalled into the apprioate object type.
	UsdCost      float64                      `json:"usdCost"`
	DetailedData map[string]*pb.DetailedField `json:"detailedData"` //detailed metadata per field including tokens, cost, model, and choices
}