package execution

import (
	"context"
	"fmt"
	"objectweaver/logger"
	"objectweaver/orchestration/jos/domain"
	"sync"
	"time"

	"github.com/objectweaver/go-sdk/jsonSchema"
)

type FieldProcessor struct {
	llmProvider          domain.LLMProvider
	promptBuilder        domain.PromptBuilder
	generator            domain.Generator
	decisionProcessor    *DecisionProcessor
	epstimicOrchestrator EpstimicOrchestrator
}

type EpstimicOrchestrator interface {
	EpstimicValidation(task *domain.FieldTask, context *domain.ExecutionContext, generateFn func(*domain.FieldTask, *domain.ExecutionContext) (any, *domain.ProviderMetadata, error)) (*domain.TaskResult, *domain.ProviderMetadata, error)
}

type ResultCollector interface {
	Collect(ctx context.Context, results []*domain.TaskResult) error
}

type ChannelCollector struct {
	ch chan<- []*domain.TaskResult
}

func (c *ChannelCollector) Collect(ctx context.Context, results []*domain.TaskResult) error {
	select {
	case c.ch <- results:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type MapCollector struct {
	results  map[string]interface{}
	metadata *domain.ResultMetadata
	mu       sync.Mutex
}

func (c *MapCollector) Collect(ctx context.Context, results []*domain.TaskResult) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, result := range results {
		if result != nil {
			c.results[result.Key()] = result.Value()
			if result.Metadata() != nil && c.metadata != nil {
				c.metadata.AddCost(result.Metadata().Cost)
				c.metadata.AddTokens(result.Metadata().TokensUsed)
			}
		}
	}
	return nil
}

func NewFieldProcessor(llmProvider domain.LLMProvider, promptBuilder domain.PromptBuilder) *FieldProcessor {
	return &FieldProcessor{
		llmProvider:   llmProvider,
		promptBuilder: promptBuilder,
	}
}

func (fp *FieldProcessor) SetGenerator(generator domain.Generator) {
	fp.generator = generator
	fp.decisionProcessor = NewDecisionProcessor(generator)
}

func (fp *FieldProcessor) SetEpstimicOrchestrator(orchestrator EpstimicOrchestrator) {
	fp.epstimicOrchestrator = orchestrator
}

func (fp *FieldProcessor) ProcessFieldsStart(ctx context.Context, schema *jsonSchema.Definition, parentTask *domain.FieldTask, execContext *domain.ExecutionContext) <-chan []*domain.TaskResult {
	bufferSize := 50 // Reasonable default for most cases
	if schema != nil && schema.Properties != nil {
		propertyCount := len(schema.Properties)
		if propertyCount > 100 {
			bufferSize = 100
		} else if propertyCount > 10 {
			bufferSize = propertyCount
		}
	}
	ch := make(chan []*domain.TaskResult, bufferSize)

	go func() {
		defer close(ch)
		collector := &ChannelCollector{ch: ch}
		fp.ProcessFields(ctx, schema, parentTask, execContext, collector)
	}()

	return ch
}

func (fp *FieldProcessor) ProcessFields(ctx context.Context, schema *jsonSchema.Definition, parentTask *domain.FieldTask, execContext *domain.ExecutionContext, collector ResultCollector) {
	if schema == nil || schema.Properties == nil {
		logger.Printf("Schema or Properties is nil")
		return
	}

	orderedKeys, remainingKeys := getOrderedKeys(schema)

	currentGen := make(map[string]interface{})
	fp.processSequentialFields(ctx, collector, schema, parentTask, execContext, currentGen, orderedKeys)

	fp.processConcurrentFieldsAndWait(ctx, collector, schema, parentTask, execContext, currentGen, remainingKeys)
}

type fieldContinuation struct {
	ctx         context.Context
	collector   ResultCollector
	wg          sync.WaitGroup
	mu          sync.Mutex
	fieldCount  int
	completedAt time.Time
}

func (fc *fieldContinuation) waitForCompletion() {
	start := time.Now()
	fc.wg.Wait()
	fc.mu.Lock()
	fc.completedAt = time.Now()
	fc.mu.Unlock()
	waitDuration := time.Since(start)
	logger.Printf("[FieldProcessor] Continuation completed: %d fields processed in %v", fc.fieldCount, waitDuration)
}

func (fp *FieldProcessor) processSequentialFields(
	ctx context.Context,
	collector ResultCollector,
	schema *jsonSchema.Definition,
	parentTask *domain.FieldTask,
	execContext *domain.ExecutionContext,
	currentGen map[string]interface{},
	orderedKeys []string,
) {
	batches := fp.analyzeFieldDependencies(orderedKeys, schema)
	logger.Printf("[FieldProcessor] ProcessingOrder analysis: %d fields split into %d batches", len(orderedKeys), len(batches))

	for batchIdx, batch := range batches {
		select {
		case <-ctx.Done():
			logger.Printf("[FieldProcessor] Context cancelled during batch %d/%d", batchIdx+1, len(batches))
			return
		default:
		}

		logger.Printf("[FieldProcessor] Processing batch %d/%d with %d independent fields", batchIdx+1, len(batches), len(batch))

		if len(batch) == 1 {
			// Process single field synchronously
			key := batch[0]
			childDef := schema.Properties[key]
			childDefCopy := childDef
			task := domain.NewFieldTask(key, &childDefCopy, parentTask)

			results := fp.processField(ctx, task, execContext)

			if len(results) > 0 {
				for _, result := range results {
					if result != nil {
						currentGen[result.Key()] = result.Value()
						execContext.SetGeneratedValue(result.Key(), result.Value())
					}
				}
				if err := collector.Collect(ctx, results); err != nil {
					logger.Printf("[FieldProcessor] Context cancelled during collect in sequential processing")
					return
				}
			}
		} else {
			// Process multiple independent fields concurrently
			fp.processSpeculativeBatch(ctx, collector, schema, parentTask, execContext, currentGen, batch)
		}
	}
}

func (fp *FieldProcessor) processField(ctx context.Context, task *domain.FieldTask, execContext *domain.ExecutionContext) []*domain.TaskResult {
	select {
	case <-ctx.Done():
		logger.Printf("[FieldProcessor] Context cancelled during processField: %v", ctx.Err())
		return nil
	default:
	}

	if task.Definition().Type == jsonSchema.Object {
		return fp.processObjectField(ctx, task, execContext)
	}

	processor := fp.getProcessorForType(task.Definition())

	result, err := processor.Process(ctx, task, execContext)
	if err != nil {
		logger.Printf("Error processing field %s: %v", task.Key(), err)
		return nil
	}

	if result == nil {
		return nil
	}

	execContext.SetGeneratedValue(result.Key(), result.Value())
	logger.Printf("[FieldProcessor] Generated field '%s', value: %v", result.Key(), result.Value())

	logger.Printf("[FieldProcessor] Checking scoring criteria for field '%s': %v", task.Key(), task.Definition().ScoringCriteria != nil)
	if task.Definition().ScoringCriteria != nil {
		logger.Printf("[FieldProcessor] Evaluating scores for field '%s'", task.Key())
		scores, err := fp.evaluateScores(ctx, result, task.Definition().ScoringCriteria, execContext)
		if err != nil {
			logger.Printf("[FieldProcessor] Warning: scoring failed for field %s: %v", task.Key(), err)
		} else {
			fp.attachScoresToResult(result, scores)
			logger.Printf("[FieldProcessor] Field '%s' scores: %v", result.Key(), scores)
		}
	}

	logger.Printf("[FieldProcessor] Checking decision point for field '%s': %v", task.Key(), task.Definition().DecisionPoint != nil)
	if task.Definition().DecisionPoint != nil {
		logger.Printf("[FieldProcessor] Processing decision point for field %s", task.Key())

		results, err := fp.decisionProcessor.ProcessDecisionPoint(ctx, task, result, execContext)
		if err != nil {
			logger.Printf("[FieldProcessor] Decision point processing failed: %v", err)
			return []*domain.TaskResult{result}
		}

		if len(results) > 1 {
			logger.Printf("[FieldProcessor] Decision point created %d additional fields", len(results)-1)
			for _, branchResult := range results[1:] {
				execContext.SetGeneratedValue(branchResult.Key(), branchResult.Value())
				logger.Printf("[FieldProcessor] Added branch field to context: %s = %v", branchResult.Key(), branchResult.Value())
			}
		}

		return results
	}

	return []*domain.TaskResult{result}
}

func (fp *FieldProcessor) processObjectField(ctx context.Context, task *domain.FieldTask, execContext *domain.ExecutionContext) []*domain.TaskResult {
	logger.Printf("[FieldProcessor] Processing nested object field '%s'", task.Key())

	nestedResults := make(map[string]interface{})
	aggregatedMetadata := domain.NewResultMetadata()

	collector := &MapCollector{
		results:  nestedResults,
		metadata: aggregatedMetadata,
	}

	// Process nested fields synchronously to reuse stack
	fp.ProcessFields(ctx, task.Definition(), task, execContext, collector)

	result := domain.NewTaskResult(task.ID(), task.Key(), nestedResults, aggregatedMetadata)
	result = result.WithPath(task.Path())

	return []*domain.TaskResult{result}
}

func (fp *FieldProcessor) processConcurrentFieldsAndWait(
	ctx context.Context,
	collector ResultCollector,
	schema *jsonSchema.Definition,
	parentTask *domain.FieldTask,
	execContext *domain.ExecutionContext,
	currentGen map[string]interface{},
	remainingKeys []string,
) {
	if len(remainingKeys) == 0 {
		return
	}

	wp := execContext.WorkerPool()
	availableWorkers := wp.AvailableWorkers()

	// If less than 10% of workers are available, run sequentially to avoid deadlock
	lowWorkerThreshold := wp.MaxWorkers() / 10
	if lowWorkerThreshold < 10 {
		lowWorkerThreshold = 10
	}

	if availableWorkers < lowWorkerThreshold {
		logger.Printf("[FieldProcessor] Low worker availability (%d < %d), processing %d fields sequentially to avoid deadlock",
			availableWorkers, lowWorkerThreshold, len(remainingKeys))

		for _, key := range remainingKeys {
			select {
			case <-ctx.Done():
				return
			default:
			}

			childDef := schema.Properties[key]
			childDefCopy := childDef
			task := domain.NewFieldTask(key, &childDefCopy, parentTask)

			results := fp.processField(ctx, task, execContext)

			if len(results) > 0 {
				if err := collector.Collect(ctx, results); err != nil {
					return
				}
			}
		}
		return
	}

	continuation := &fieldContinuation{
		ctx:        ctx,
		collector:  collector,
		fieldCount: len(remainingKeys),
	}

	logger.Printf("[FieldProcessor] Starting work continuation for %d concurrent fields", len(remainingKeys))

	for _, key := range remainingKeys {
		childDef := schema.Properties[key]
		childDefCopy := childDef

		fieldKey := key
		fieldDef := childDefCopy

		continuation.wg.Add(1)

		submitted := execContext.WorkerPool().SubmitWithContext(ctx, func() {
			defer continuation.wg.Done()

			select {
			case <-ctx.Done():
				logger.Printf("[FieldProcessor] Context cancelled, skipping field: %s", fieldKey)
				return
			default:
			}

			task := domain.NewFieldTask(fieldKey, &fieldDef, parentTask)
			results := fp.processField(ctx, task, execContext)

			select {
			case <-ctx.Done():
				logger.Printf("[FieldProcessor] Context cancelled after processing field: %s", fieldKey)
				return
			default:
			}

			if len(results) > 0 {
				if err := collector.Collect(ctx, results); err != nil {
					logger.Printf("[FieldProcessor] Context cancelled during collect for field: %s", fieldKey)
					return
				}
			}
		})

		if !submitted {
			logger.Printf("[FieldProcessor] Failed to submit field %s, marking as complete", fieldKey)
			continuation.wg.Done()
		}
	}

	continuation.waitForCompletion()
}

func (fp *FieldProcessor) getProcessorForType(def *jsonSchema.Definition) domain.TypeProcessor {
	if def.TextToSpeech != nil || def.Image != nil || def.SpeechToText != nil {
		return NewByteProcessor(fp.llmProvider, fp.promptBuilder)
	}

	if def.RecursiveLoop != nil && fp.generator != nil {
		baseProcessor := fp.getBaseProcessorForType(def.Type)
		decisionProcessor := NewDecisionProcessor(fp.generator)
		return NewRecursiveLoopProcessor(baseProcessor, fp.generator, decisionProcessor)
	}

	return fp.getBaseProcessorForType(def.Type)
}

func (fp *FieldProcessor) getBaseProcessorForType(schemaType jsonSchema.DataType) domain.TypeProcessor {
	switch schemaType {
	case jsonSchema.Array:
		return NewArrayProcessorWithFieldProcessor(fp.llmProvider, fp.promptBuilder, fp)
	case jsonSchema.Map:
		return NewMapProcessorWithFieldProcessor(fp.llmProvider, fp.promptBuilder, fp)
	case jsonSchema.Boolean:
		processor := NewBooleanProcessor(fp.llmProvider, fp.promptBuilder)
		if fp.epstimicOrchestrator != nil {
			processor.SetEpstimicOrchestrator(fp.epstimicOrchestrator)
		}
		return processor
	case jsonSchema.Number, jsonSchema.Integer:
		processor := NewNumberProcessor(fp.llmProvider, fp.promptBuilder)
		if fp.epstimicOrchestrator != nil {
			processor.SetEpstimicOrchestrator(fp.epstimicOrchestrator)
		}
		return processor
	default:
		processor := NewPrimitiveProcessor(fp.llmProvider, fp.promptBuilder)
		if fp.epstimicOrchestrator != nil {
			processor.SetEpstimicOrchestrator(fp.epstimicOrchestrator)
		}
		return processor
	}
}

func getOrderedKeys(schema *jsonSchema.Definition) ([]string, []string) {
	var remainingKeys []string
	allKeys := make(map[string]struct{})
	for key := range schema.Properties {
		allKeys[key] = struct{}{}
	}

	if schema.ProcessingOrder != nil {
		for _, key := range schema.ProcessingOrder {
			delete(allKeys, key)
		}
	}

	for key := range allKeys {
		remainingKeys = append(remainingKeys, key)
	}

	return schema.ProcessingOrder, remainingKeys
}

func (fp *FieldProcessor) analyzeFieldDependencies(orderedKeys []string, schema *jsonSchema.Definition) [][]string {
	if len(orderedKeys) == 0 {
		return nil
	}

	// Map each field to its dependencies via SelectFields
	dependencies := make(map[string]map[string]bool)
	for _, key := range orderedKeys {
		fieldDef := schema.Properties[key]
		if len(fieldDef.SelectFields) > 0 {
			deps := make(map[string]bool)
			for _, fieldPath := range fieldDef.SelectFields {
				// Extract root field name from path (e.g., "car.color" -> "car")
				rootField := extractRootFieldName(fieldPath)
				if rootField != "" && rootField != key {
					deps[rootField] = true
				}
			}
			if len(deps) > 0 {
				dependencies[key] = deps
			}
		}
	}

	batches := make([][]string, 0)
	processed := make(map[string]bool)

	for len(processed) < len(orderedKeys) {
		currentBatch := make([]string, 0)

		for _, key := range orderedKeys {
			if processed[key] {
				continue
			}

			// Check if all dependencies are satisfied
			canProcess := true
			if deps, hasDeps := dependencies[key]; hasDeps {
				for dep := range deps {
					if !processed[dep] {
						canProcess = false
						break
					}
				}
			}

			if !canProcess {
				continue
			}

			fieldDef := schema.Properties[key]

			// Avoid batching objects as they're complex
			if fieldDef.Type == jsonSchema.Object {
				if len(currentBatch) > 0 {
					// Flush current batch, process object alone
					break
				}
				currentBatch = append(currentBatch, key)
				break
			}

			if len(currentBatch) >= 5 {
				break
			}

			currentBatch = append(currentBatch, key)
		}

		for _, key := range currentBatch {
			processed[key] = true
		}

		if len(currentBatch) > 0 {
			batches = append(batches, currentBatch)
		} else {
			for _, key := range orderedKeys {
				if !processed[key] {
					logger.Printf("[FieldProcessor] Warning: circular or unresolvable dependency detected for field '%s', processing anyway", key)
					batches = append(batches, []string{key})
					processed[key] = true
					break
				}
			}
		}
	}

	return batches
}

// extractRootFieldName extracts the root field name from a field path
// Examples: "car" -> "car", "car.color" -> "car", "cars.color" -> "cars"
func extractRootFieldName(fieldPath string) string {
	if fieldPath == "" {
		return ""
	}

	// Split by dot and return first part
	for i, char := range fieldPath {
		if char == '.' {
			return fieldPath[:i]
		}
	}

	return fieldPath
}

// processSpeculativeBatch processes a batch of independent fields concurrently
func (fp *FieldProcessor) processSpeculativeBatch(
	ctx context.Context,
	collector ResultCollector,
	schema *jsonSchema.Definition,
	parentTask *domain.FieldTask,
	execContext *domain.ExecutionContext,
	currentGen map[string]interface{},
	batch []string,
) {
	continuation := &fieldContinuation{
		ctx:        ctx,
		collector:  collector,
		fieldCount: len(batch),
	}

	logger.Printf("[FieldProcessor] Processing speculative batch: %d fields", len(batch))

	for _, key := range batch {
		childDef := schema.Properties[key]
		childDefCopy := childDef

		fieldKey := key
		fieldDef := childDefCopy

		continuation.wg.Add(1)

		submitted := execContext.WorkerPool().SubmitWithContext(ctx, func() {
			defer continuation.wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
			}

			task := domain.NewFieldTask(fieldKey, &fieldDef, parentTask)
			results := fp.processField(ctx, task, execContext)

			if len(results) > 0 {
				for _, result := range results {
					if result != nil {
						execContext.SetGeneratedValue(result.Key(), result.Value())
					}
				}
				if err := collector.Collect(ctx, results); err != nil {
					return
				}
			}
		})

		if !submitted {
			continuation.wg.Done()
		}
	}

	continuation.waitForCompletion()
}

func (fp *FieldProcessor) evaluateScores(
	ctx context.Context,
	result *domain.TaskResult,
	criteria *jsonSchema.ScoringCriteria,
	execContext *domain.ExecutionContext,
) (map[string]float64, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	if fp.generator == nil {
		return nil, fmt.Errorf("generator not set, cannot evaluate scores")
	}

	properties := make(map[string]jsonSchema.Definition)
	for dimensionName, dimension := range criteria.Dimensions {
		var fieldType jsonSchema.DataType
		switch dimension.Type {
		case jsonSchema.ScoreNumeric:
			fieldType = jsonSchema.Number
		case jsonSchema.ScoreBoolean:
			fieldType = jsonSchema.Boolean
		default:
			fieldType = jsonSchema.String
		}

		instruction := dimension.Description
		if dimension.Scale != nil {
			instruction += fmt.Sprintf(" (Range: %d-%d)", dimension.Scale.Min, dimension.Scale.Max)
		}

		properties[dimensionName] = jsonSchema.Definition{
			Type:        fieldType,
			Instruction: instruction,
		}
	}

	scoringSchema := &jsonSchema.Definition{
		Type:        jsonSchema.Object,
		Properties:  properties,
		Instruction: "Evaluate the content according to the specified dimensions",
	}

	prompt := "Evaluate the following content:\n\n"
	prompt += fmt.Sprintf("%v\n\n", result.Value())
	prompt += "Provide scores for each dimension as requested."

	if criteria.EvaluationModel != "" {
		scoringSchema.Model = criteria.EvaluationModel
	}

	request := domain.NewGenerationRequest(prompt, scoringSchema)
	scoreResult, err := fp.generator.Generate(request)
	if err != nil {
		return nil, fmt.Errorf("score generation failed: %w", err)
	}

	scoreData := scoreResult.Data()
	scores := make(map[string]float64)
	for key, value := range scoreData {
		if numValue, ok := toFloat64(value); ok {
			scores[key] = numValue
		}
	}

	if criteria.AggregationMethod != "" {
		aggregate := fp.calculateAggregate(scores, criteria)
		scores["_aggregate"] = aggregate
	}

	return scores, nil
}

func (fp *FieldProcessor) calculateAggregate(scores map[string]float64, criteria *jsonSchema.ScoringCriteria) float64 {
	switch criteria.AggregationMethod {
	case jsonSchema.AggregateWeightedAverage:
		var sum, totalWeight float64
		for dimension, score := range scores {
			if dimDef, exists := criteria.Dimensions[dimension]; exists {
				weight := dimDef.Weight
				if weight == 0 {
					weight = 1.0 / float64(len(criteria.Dimensions))
				}
				sum += score * weight
				totalWeight += weight
			}
		}
		if totalWeight > 0 {
			return sum / totalWeight
		}
		return 0

	case jsonSchema.AggregateMinimum:
		min := float64(100)
		for _, score := range scores {
			if score < min {
				min = score
			}
		}
		return min

	case jsonSchema.AggregateMaximum:
		max := float64(0)
		for _, score := range scores {
			if score > max {
				max = score
			}
		}
		return max

	default:
		var sum float64
		for _, score := range scores {
			sum += score
		}
		return sum / float64(len(scores))
	}
}

func (fp *FieldProcessor) attachScoresToResult(result *domain.TaskResult, scores map[string]float64) {
	if result == nil || result.Metadata() == nil {
		return
	}

	if aggregateScore, hasAggregate := scores["_aggregate"]; hasAggregate {
		choice := domain.Choice{
			Score: int(aggregateScore),
		}
		result.Metadata().Choices = append(result.Metadata().Choices, choice)
	}
}
