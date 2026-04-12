package LLM

import (
	"context"
	"errors"
	"github.com/ObjectWeaver/ObjectWeaver/llmManagement/clientManager"
	"github.com/ObjectWeaver/ObjectWeaver/logger"
	"sync/atomic"

	"github.com/ObjectWeaver/ObjectWeaver/jsonSchema"

	gogpt "github.com/sashabaranov/go-openai"
	"golang.org/x/sync/semaphore"
)

// DirectJobSubmitter bypasses the worker queue and calls the HTTP client directly.
// This eliminates 6 layers of indirection (WorkerChannel → ChannelBridge → JobQueue →
// StartManager → jobChan → Workers) and reduces latency from 18+ seconds to ~150ms.
//
// Architecture:
//
//	Request → processConcurrentFields (spawn N goroutines)
//	  ↓
//	Semaphore.Acquire() ← limits concurrent requests (default: 100)
//	  ↓
//	clientAdapter.Process() ← direct HTTP call (10-50ms)
//	  ↓
//	Semaphore.Release()
//	  ↓
//	Return result
//
// This approach:
// - Eliminates queue polling overhead (was 10μs per poll)
// - Eliminates channel synchronization overhead (6 channel hops)
// - Uses Go's native HTTP connection pooling (MaxIdleConns=1000)
// - Provides natural backpressure (goroutines block when semaphore full)
// - Simplifies debugging (fewer moving parts)
type adapterShard struct {
	adapter   clientManager.ClientAdapter
	semaphore *semaphore.Weighted
}

type DirectJobSubmitter struct {
	shards        []*adapterShard
	numShards     uint64
	nextShard     atomic.Uint64
	maxConcurrent int64
	verbose       bool
}

// NewDirectJobSubmitter creates a submitter that calls the HTTP client directly.
// maxConcurrent: Maximum number of concurrent HTTP requests (default: 3000)
func NewDirectJobSubmitter(adapters []clientManager.ClientAdapter, maxConcurrent int, verbose bool) *DirectJobSubmitter {
	if maxConcurrent <= 0 {
		maxConcurrent = 3000 // Balanced for I/O throughput without overwhelming backend
	}

	if len(adapters) == 0 {
		panic("at least one client adapter is required")
	}

	logger.Printf("[DirectJobSubmitter] Initialized with maxConcurrent=%d (bypassing worker queue) and %d adapters", maxConcurrent, len(adapters))

	numShards := len(adapters)
	shards := make([]*adapterShard, numShards)

	// Distribute concurrency limit across shards
	shardCapacity := int64(maxConcurrent) / int64(numShards)
	if shardCapacity < 1 {
		shardCapacity = 1
	}

	for i, adapter := range adapters {
		shards[i] = &adapterShard{
			adapter:   adapter,
			semaphore: semaphore.NewWeighted(shardCapacity),
		}
	}

	return &DirectJobSubmitter{
		shards:        shards,
		numShards:     uint64(numShards),
		maxConcurrent: int64(maxConcurrent),
		verbose:       verbose,
	}
}

// SubmitJob processes the job directly without going through the worker queue.
func (d *DirectJobSubmitter) SubmitJob(job *Job, _ chan *Job) (any, *gogpt.Usage, error) {
	if job == nil {
		return "", nil, errors.New("job is nil")
	}

	// Round-robin shard selection
	idx := d.nextShard.Add(1) % d.numShards
	shard := d.shards[idx]

	// Acquire semaphore for this specific shard
	// Use context from job inputs if available
	ctx := job.Inputs.Ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if err := shard.semaphore.Acquire(ctx, 1); err != nil {
		return "", nil, errors.New("failed to acquire semaphore: " + err.Error())
	}
	defer shard.semaphore.Release(1)

	// Call HTTP client directly (no queue, no polling, no channel hops)
	result, err := shard.adapter.Process(job.Inputs)

	if err != nil {
		return "", nil, err
	}

	// Handle different result types
	if job.Inputs.Def.Type == jsonSchema.Vector {
		if result.EmbeddingRes == nil || len(result.EmbeddingRes.Data) == 0 {
			return "", nil, errors.New("embedding response data is empty")
		}
		return result.EmbeddingRes.Data[0].Embedding, nil, nil
	}

	// Validate chat completion result
	if result == nil {
		return "", nil, errors.New("result is nil")
	}
	if result.ChatRes.Choices == nil || len(result.ChatRes.Choices) < 1 {
		return "", nil, errors.New("result has no choices")
	}

	return result.ChatRes.Choices[0].Message.Content, &result.ChatRes.Usage, nil
}

// GetMaxConcurrent returns the maximum concurrent requests allowed
func (d *DirectJobSubmitter) GetMaxConcurrent() int64 {
	return d.maxConcurrent
}
