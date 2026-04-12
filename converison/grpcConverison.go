package converison

import (
	"fmt"

	"github.com/ObjectWeaver/ObjectWeaver/jsonSchema"

	pb "github.com/ObjectWeaver/ObjectWeaver/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

// ConvertProtoToModel converts a protobuf Definition to your Go model Definition
func ConvertProtoToModel(protoDef *pb.Definition) *jsonSchema.Definition {
	if protoDef == nil {
		return nil
	}

	modelDef := &jsonSchema.Definition{
		Type:            jsonSchema.DataType(protoDef.Type),
		Instruction:     protoDef.Instruction,
		Properties:      make(map[string]jsonSchema.Definition),
		Items:           ConvertProtoToModel(protoDef.GetItems()), // Use Getters to handle nil cases
		Model:           protoDef.Model,
		ProcessingOrder: protoDef.ProcessingOrder,
		SystemPrompt:    getStringPointer(protoDef.GetSystemPrompt()), // Safe getter for pointers
		SelectFields:    protoDef.SelectFields,
		HashMap:         ConvertProtoToHashMap(protoDef.GetHashMap()),   // Check with Getters
		NarrowFocus:     ConvertProtoToFocus(protoDef.GetNarrowFocus()), // Handle nil safely
		Req:             ConvertProtoToRequestFormat(protoDef.GetReq()),
		SpeechToText:    convertProtoSpeechToText(protoDef.GetSpeechToText()), // Safely handle nested structs
		TextToSpeech:    convertProtoTextToSpeech(protoDef.GetTextToSpeech()),
		Image:           convertProtoImage(protoDef.GetImage()),         // Handle Image field
		SendImage:       convertProtoSendImage(protoDef.GetSendImage()), // Handle nil structs
		Stream:          protoDef.Stream,
		Priority:        protoDef.Priority,
		OverridePrompt:  getStringPointer(protoDef.GetOverridePrompt()),
		DecisionPoint:   convertProtoDecisionPoint(protoDef.GetDecisionPoint()),
		ScoringCriteria: convertProtoScoringCriteria(protoDef.GetScoringCriteria()),
		RecursiveLoop:   convertProtoRecursiveLoop(protoDef.GetRecursiveLoop()),
		Epistemic:       convertProtoEpistemicValidation(protoDef.Epistemic),
		ModelConfig:     convertProtoModelConfig(protoDef.GetModelConfig()),
	}

	// Handle Properties map
	if protoDef.Properties != nil {
		for key, protoProperty := range protoDef.Properties {
			modelDef.Properties[key] = *ConvertProtoToModel(protoProperty)
		}
	}

	return modelDef
}

// Helper function to safely get string pointers
func getStringPointer(val string) *string {
	if val == "" {
		return nil
	}
	return &val
}

// Helper function to safely get int pointers
func getIntPointer(val int32) *int {
	if val == 0 {
		return nil
	}
	intVal := int(val)
	return &intVal
}

// ConvertModelToProto converts your Go model Definition to a protobuf Definition
func ConvertModelToProto(modelDef *jsonSchema.Definition) *pb.Definition {
	if modelDef == nil {
		return nil
	}

	systemPrompt := ""
	if modelDef.SystemPrompt != nil {
		systemPrompt = *modelDef.SystemPrompt
	}

	overridePrompt := ""
	if modelDef.OverridePrompt != nil {
		overridePrompt = *modelDef.OverridePrompt
	}

	protoDef := &pb.Definition{
		Type:            string(modelDef.Type),
		Instruction:     modelDef.Instruction,
		Properties:      make(map[string]*pb.Definition),
		Items:           ConvertModelToProto(modelDef.Items),
		Model:           modelDef.Model,
		ProcessingOrder: modelDef.ProcessingOrder,
		SystemPrompt:    systemPrompt,
		SelectFields:    modelDef.SelectFields,
		HashMap:         ConvertModelToProtoHashMap(modelDef.HashMap),
		NarrowFocus:     ConvertModelToProtoFocus(modelDef.NarrowFocus),
		Req:             ConvertModelToProtoRequestFormat(modelDef.Req),
		Image:           convertModelImage(modelDef.Image),
		SpeechToText:    convertModelSpeechToText(modelDef.SpeechToText),
		TextToSpeech:    convertModelTextToSpeech(modelDef.TextToSpeech),
		SendImage:       convertModelSendImage(modelDef.SendImage),
		Stream:          modelDef.Stream,
		Priority:        modelDef.Priority,
		OverridePrompt:  overridePrompt,
		DecisionPoint:   convertModelDecisionPoint(modelDef.DecisionPoint),
		ScoringCriteria: convertModelScoringCriteria(modelDef.ScoringCriteria),
		RecursiveLoop:   convertModelRecursiveLoop(modelDef.RecursiveLoop),
		Epistemic:       convertModelEpistemicValidation(&modelDef.Epistemic),
		ModelConfig:     convertModelModelConfig(modelDef.ModelConfig),
	}

	// Handle Properties map
	if modelDef.Properties != nil {
		for key, modelProperty := range modelDef.Properties {
			protoDef.Properties[key] = ConvertModelToProto(&modelProperty)
		}
	}

	return protoDef
}

// Helper functions for SpeechToText, TextToSpeech, and other nested structs

func convertProtoSpeechToText(speechToText *pb.SpeechToText) *jsonSchema.SpeechToText {
	if speechToText == nil {
		return nil
	}
	return &jsonSchema.SpeechToText{
		Model:             speechToText.Model,
		AudioToTranscribe: speechToText.AudioToTranscribe,
		Language:          speechToText.Language,
		ToString:          speechToText.ToString,
		ToCaptions:        speechToText.ToCaptions,
		Format:            speechToText.Format,
	}
}

func convertProtoTextToSpeech(textToSpeech *pb.TextToSpeech) *jsonSchema.TextToSpeech {
	if textToSpeech == nil {
		return nil
	}
	return &jsonSchema.TextToSpeech{
		Model:         textToSpeech.Model,
		Voice:         textToSpeech.Voice,
		StringToAudio: textToSpeech.StringToAudio,
		Format:        textToSpeech.Format,
	}
}

func convertProtoImage(image *pb.Image) *jsonSchema.Image {
	if image == nil {
		return nil
	}
	return &jsonSchema.Image{
		Model: image.Model,
		Size:  image.Size,
	}
}

func convertProtoSendImage(sendImage *pb.SendImage) *jsonSchema.SendImage {
	if sendImage == nil {
		return nil
	}
	return &jsonSchema.SendImage{
		ImagesData: sendImage.ImagesData,
	}
}

func convertModelSpeechToText(speechToText *jsonSchema.SpeechToText) *pb.SpeechToText {
	if speechToText == nil {
		return nil
	}
	return &pb.SpeechToText{
		Model:             string(speechToText.Model),
		AudioToTranscribe: speechToText.AudioToTranscribe,
		Language:          speechToText.Language,
		ToString:          speechToText.ToString,
		ToCaptions:        speechToText.ToCaptions,
		Format:            string(speechToText.Format),
	}
}

func convertModelTextToSpeech(textToSpeech *jsonSchema.TextToSpeech) *pb.TextToSpeech {
	if textToSpeech == nil {
		return nil
	}
	return &pb.TextToSpeech{
		Model:         string(textToSpeech.Model),
		Voice:         string(textToSpeech.Voice),
		StringToAudio: textToSpeech.StringToAudio,
		Format:        string(textToSpeech.Format),
	}
}

func convertModelSendImage(sendImage *jsonSchema.SendImage) *pb.SendImage {
	if sendImage == nil {
		return nil
	}
	return &pb.SendImage{
		ImagesData: sendImage.ImagesData,
	}
}

func convertModelImage(image *jsonSchema.Image) *pb.Image {
	if image == nil {
		return nil
	}
	return &pb.Image{
		Model: string(image.Model),
		Size:  string(image.Size),
	}
}

// Helper functions for DecisionPoint, ScoringCriteria, and RecursiveLoop

func convertProtoDecisionPoint(dp *pb.DecisionPoint) *jsonSchema.DecisionPoint {
	if dp == nil {
		return nil
	}

	branches := make([]jsonSchema.ConditionalBranch, len(dp.Branches))
	for i, branch := range dp.Branches {
		branches[i] = *convertProtoConditionalBranch(branch)
	}

	return &jsonSchema.DecisionPoint{
		Name:             dp.Name,
		EvaluationPrompt: dp.EvaluationPrompt,
		Branches:         branches,
		Strategy:         jsonSchema.RoutingStrategy(dp.Strategy),
	}
}

func convertProtoConditionalBranch(cb *pb.ConditionalBranch) *jsonSchema.ConditionalBranch {
	if cb == nil {
		return nil
	}

	conditions := make([]jsonSchema.Condition, len(cb.Conditions))
	for i, cond := range cb.Conditions {
		conditions[i] = *convertProtoCondition(cond)
	}

	return &jsonSchema.ConditionalBranch{
		Name:       cb.Name,
		Conditions: conditions,
		Logic:      ConvertProtoToModel(cb.Logic),
		Then:       *ConvertProtoToModel(cb.Then),
		Priority:   int(cb.Priority),
	}
}

func convertProtoCondition(c *pb.Condition) *jsonSchema.Condition {
	if c == nil {
		return nil
	}

	var value interface{}
	// Handle oneof value field
	switch v := c.Value.(type) {
	case *pb.Condition_NumberValue:
		value = v.NumberValue
	case *pb.Condition_StringValue:
		value = v.StringValue
	case *pb.Condition_BoolValue:
		value = v.BoolValue
	}

	return &jsonSchema.Condition{
		Field:     c.Field,
		Operator:  jsonSchema.ComparisonOperator(c.Operator),
		Value:     value,
		FieldPath: c.FieldPath,
	}
}

func convertProtoScoringCriteria(sc *pb.ScoringCriteria) *jsonSchema.ScoringCriteria {
	if sc == nil {
		return nil
	}

	dimensions := make(map[string]jsonSchema.ScoringDimension)
	for key, dim := range sc.Dimensions {
		dimensions[key] = *convertProtoScoringDimension(dim)
	}

	return &jsonSchema.ScoringCriteria{
		Dimensions:        dimensions,
		EvaluationModel:   sc.EvaluationModel,
		AggregationMethod: jsonSchema.AggregationMethod(sc.AggregationMethod),
	}
}

func convertProtoScoringDimension(sd *pb.ScoringDimension) *jsonSchema.ScoringDimension {
	if sd == nil {
		return nil
	}

	return &jsonSchema.ScoringDimension{
		Description: sd.Description,
		Scale:       convertProtoScoreScale(sd.Scale),
		Type:        jsonSchema.ScoreType(sd.Type),
		Weight:      sd.Weight,
	}
}

func convertProtoScoreScale(ss *pb.ScoreScale) *jsonSchema.ScoreScale {
	if ss == nil {
		return nil
	}

	return &jsonSchema.ScoreScale{
		Min: int(ss.Min),
		Max: int(ss.Max),
	}
}

func convertProtoRecursiveLoop(rl *pb.RecursiveLoop) *jsonSchema.RecursiveLoop {
	if rl == nil {
		return nil
	}

	return &jsonSchema.RecursiveLoop{
		MaxIterations:           int(rl.MaxIterations),
		Selection:               jsonSchema.SelectionStrategy(rl.Selection),
		TerminationPoint:        convertProtoDecisionPoint(rl.TerminationPoint),
		FeedbackPrompt:          rl.FeedbackPrompt,
		IncludePreviousAttempts: rl.IncludePreviousAttempts,
	}
}

// Model to Proto conversions

func convertModelDecisionPoint(dp *jsonSchema.DecisionPoint) *pb.DecisionPoint {
	if dp == nil {
		return nil
	}

	branches := make([]*pb.ConditionalBranch, len(dp.Branches))
	for i, branch := range dp.Branches {
		branches[i] = convertModelConditionalBranch(&branch)
	}

	return &pb.DecisionPoint{
		Name:             dp.Name,
		EvaluationPrompt: dp.EvaluationPrompt,
		Branches:         branches,
		Strategy:         string(dp.Strategy),
	}
}

func convertModelConditionalBranch(cb *jsonSchema.ConditionalBranch) *pb.ConditionalBranch {
	if cb == nil {
		return nil
	}

	conditions := make([]*pb.Condition, len(cb.Conditions))
	for i, cond := range cb.Conditions {
		conditions[i] = convertModelCondition(&cond)
	}

	return &pb.ConditionalBranch{
		Name:       cb.Name,
		Conditions: conditions,
		Logic:      ConvertModelToProto(cb.Logic),
		Then:       ConvertModelToProto(&cb.Then),
		Priority:   int32(cb.Priority),
	}
}

func convertModelCondition(c *jsonSchema.Condition) *pb.Condition {
	if c == nil {
		return nil
	}

	condition := &pb.Condition{
		Field:     c.Field,
		Operator:  string(c.Operator),
		FieldPath: c.FieldPath,
	}

	// Set the appropriate oneof field based on the value type
	if c.Value != nil {
		switch v := c.Value.(type) {
		case float64:
			condition.Value = &pb.Condition_NumberValue{NumberValue: v}
		case float32:
			condition.Value = &pb.Condition_NumberValue{NumberValue: float64(v)}
		case int:
			condition.Value = &pb.Condition_NumberValue{NumberValue: float64(v)}
		case int32:
			condition.Value = &pb.Condition_NumberValue{NumberValue: float64(v)}
		case int64:
			condition.Value = &pb.Condition_NumberValue{NumberValue: float64(v)}
		case string:
			condition.Value = &pb.Condition_StringValue{StringValue: v}
		case bool:
			condition.Value = &pb.Condition_BoolValue{BoolValue: v}
		default:
			// For other types, try to convert to string as fallback
			condition.Value = &pb.Condition_StringValue{StringValue: fmt.Sprintf("%v", v)}
		}
	}

	return condition
}

func convertModelScoringCriteria(sc *jsonSchema.ScoringCriteria) *pb.ScoringCriteria {
	if sc == nil {
		return nil
	}

	dimensions := make(map[string]*pb.ScoringDimension)
	for key, dim := range sc.Dimensions {
		dimensions[key] = convertModelScoringDimension(&dim)
	}

	return &pb.ScoringCriteria{
		Dimensions:        dimensions,
		EvaluationModel:   sc.EvaluationModel,
		AggregationMethod: string(sc.AggregationMethod),
	}
}

func convertModelScoringDimension(sd *jsonSchema.ScoringDimension) *pb.ScoringDimension {
	if sd == nil {
		return nil
	}

	return &pb.ScoringDimension{
		Description: sd.Description,
		Scale:       convertModelScoreScale(sd.Scale),
		Type:        string(sd.Type),
		Weight:      sd.Weight,
	}
}

func convertModelScoreScale(ss *jsonSchema.ScoreScale) *pb.ScoreScale {
	if ss == nil {
		return nil
	}

	return &pb.ScoreScale{
		Min: int32(ss.Min),
		Max: int32(ss.Max),
	}
}

func convertModelRecursiveLoop(rl *jsonSchema.RecursiveLoop) *pb.RecursiveLoop {
	if rl == nil {
		return nil
	}

	return &pb.RecursiveLoop{
		MaxIterations:           int32(rl.MaxIterations),
		Selection:               string(rl.Selection),
		TerminationPoint:        convertModelDecisionPoint(rl.TerminationPoint),
		FeedbackPrompt:          rl.FeedbackPrompt,
		IncludePreviousAttempts: rl.IncludePreviousAttempts,
	}
}

// Helper functions for EpistemicValidation

func convertProtoEpistemicValidation(ev *pb.EpistemicValidation) jsonSchema.EpistemicValidation {
	if ev == nil {
		return jsonSchema.EpistemicValidation{}
	}

	return jsonSchema.EpistemicValidation{
		Active: ev.Active,
		Judges: int(ev.Judges),
	}
}

func convertModelEpistemicValidation(ev *jsonSchema.EpistemicValidation) *pb.EpistemicValidation {
	if ev == nil {
		return nil
	}

	return &pb.EpistemicValidation{
		Active: ev.Active,
		Judges: int32(ev.Judges),
	}
}

// Helper functions for ModelConfig

func convertProtoModelConfig(mc *pb.ModelConfig) *jsonSchema.ModelConfig {
	if mc == nil {
		return nil
	}

	// Convert logit bias map from int32 to int
	logitBias := make(map[string]int)
	for k, v := range mc.LogitBias {
		logitBias[k] = int(v)
	}

	// Convert seed from int32 to *int
	var seed *int
	if mc.Seed != 0 {
		seedVal := int(mc.Seed)
		seed = &seedVal
	}

	// Convert chatTemplateKwargs from protobuf Struct to map[string]any
	var chatTemplateKwargs map[string]any
	if mc.ChatTemplateKwargs != nil {
		chatTemplateKwargs = mc.ChatTemplateKwargs.AsMap()
	}

	return &jsonSchema.ModelConfig{
		MaxCompletionTokens: int(mc.MaxCompletionTokens),
		Temperature:         mc.Temperature,
		TopP:                mc.TopP,
		N:                   int(mc.N),
		Stream:              mc.Stream,
		Stop:                mc.Stop,
		PresencePenalty:     mc.PresencePenalty,
		Seed:                seed,
		FrequencyPenalty:    mc.FrequencyPenalty,
		LogitBias:           logitBias,
		LogProbs:            mc.LogProbs,
		TopLogProbs:         int(mc.TopLogProbs),
		User:                mc.User,
		Store:               mc.Store,
		ReasoningEffort:     mc.ReasoningEffort,
		Metadata:            mc.Metadata,
		ChatTemplateKwargs:  chatTemplateKwargs,
	}
}

func convertModelModelConfig(mc *jsonSchema.ModelConfig) *pb.ModelConfig {
	if mc == nil {
		return nil
	}

	// Convert logit bias map from int to int32
	logitBias := make(map[string]int32)
	for k, v := range mc.LogitBias {
		logitBias[k] = int32(v)
	}

	// Convert seed from *int to int32
	seed := int32(0)
	if mc.Seed != nil {
		seed = int32(*mc.Seed)
	}

	// Convert chatTemplateKwargs from map[string]any to protobuf Struct
	var chatTemplateKwargs *structpb.Struct
	if mc.ChatTemplateKwargs != nil {
		var err error
		chatTemplateKwargs, err = structpb.NewStruct(mc.ChatTemplateKwargs)
		if err != nil {
			// If conversion fails, leave it as nil
			chatTemplateKwargs = nil
		}
	}

	return &pb.ModelConfig{
		MaxCompletionTokens: int32(mc.MaxCompletionTokens),
		Temperature:         mc.Temperature,
		TopP:                mc.TopP,
		N:                   int32(mc.N),
		Stream:              mc.Stream,
		Stop:                mc.Stop,
		PresencePenalty:     mc.PresencePenalty,
		Seed:                seed,
		FrequencyPenalty:    mc.FrequencyPenalty,
		LogitBias:           logitBias,
		LogProbs:            mc.LogProbs,
		TopLogProbs:         int32(mc.TopLogProbs),
		User:                mc.User,
		Store:               mc.Store,
		ReasoningEffort:     mc.ReasoningEffort,
		Metadata:            mc.Metadata,
		ChatTemplateKwargs:  chatTemplateKwargs,
	}
}
