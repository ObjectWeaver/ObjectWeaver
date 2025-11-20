package execution

import (
	"container/heap"
	"context"
	"objectweaver/logger"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type job struct {
	fn       func()
	ctx      context.Context
	priority int       // Lower = higher priority (0 = highest)
	enqueued time.Time // For latency tracking
	index    int       // Heap index for efficient removal
}

type jobHeap []*job

func (h jobHeap) Len() int           { return len(h) }
func (h jobHeap) Less(i, j int) bool { return h[i].priority < h[j].priority }
func (h jobHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *jobHeap) Push(x interface{}) {
	n := len(*h)
	item := x.(*job)
	item.index = n
	*h = append(*h, item)
}

func (h *jobHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	*h = old[0 : n-1]
	return item
}

type GlobalWorkerPool struct {
	shards           []*workerPoolShard
	numShards        uint64
	nextShard        atomic.Uint64
	maxWorkers       int
	totalSubmissions atomic.Int64
	started          atomic.Bool
	startOnce        sync.Once
}

type workerPoolShard struct {
	heap          jobHeap
	mu            sync.Mutex
	notEmpty      *sync.Cond
	workers       []*worker
	activeWorkers atomic.Int64
	shutdown      chan struct{}
	wg            sync.WaitGroup
}

type worker struct {
	id    int
	shard *workerPoolShard
}

var (
	globalPool     *GlobalWorkerPool
	globalPoolOnce sync.Once
)

// returns the singleton global worker pool
func GetGlobalWorkerPool() *GlobalWorkerPool {
	globalPoolOnce.Do(func() {
		maxWorkers := 5000 // balanced for i/o-bound workload without excessive goroutines

		if envWorkers := os.Getenv("WORKER_POOL_SIZE"); envWorkers != "" {
			if parsed, err := strconv.Atoi(envWorkers); err == nil && parsed > 0 {
				maxWorkers = parsed
			}
		} else {
			cores := runtime.GOMAXPROCS(0)
			if cores <= 4 {
				// Likely test/dev environment
				maxWorkers = cores * 10
			} else {
				maxWorkers = 2000
			}
		}

		globalPool = newGlobalWorkerPool(maxWorkers)
		globalPool.start()
		logger.Printf("[WorkerPool] Initialized global worker pool with %d persistent workers (cores: %d) - Optimized for I/O-bound workload", maxWorkers, runtime.GOMAXPROCS(0))
	})
	return globalPool
}

func newGlobalWorkerPool(maxWorkers int) *GlobalWorkerPool {
	numShards := 32
	workersPerShard := maxWorkers / numShards
	if workersPerShard < 1 {
		workersPerShard = 1
		numShards = maxWorkers
	}

	gwp := &GlobalWorkerPool{
		shards:     make([]*workerPoolShard, numShards),
		numShards:  uint64(numShards),
		maxWorkers: maxWorkers,
	}

	for i := 0; i < numShards; i++ {
		shard := &workerPoolShard{
			heap:     make(jobHeap, 0, 1024),
			workers:  make([]*worker, workersPerShard),
			shutdown: make(chan struct{}),
		}
		shard.notEmpty = sync.NewCond(&shard.mu)
		heap.Init(&shard.heap)

		for j := 0; j < workersPerShard; j++ {
			shard.workers[j] = &worker{
				id:    (i * workersPerShard) + j,
				shard: shard,
			}
		}
		gwp.shards[i] = shard
	}

	return gwp
}

func (gwp *GlobalWorkerPool) start() {
	gwp.startOnce.Do(func() {
		for _, shard := range gwp.shards {
			for _, w := range shard.workers {
				shard.wg.Add(1)
				go w.run()
			}
		}
		gwp.started.Store(true)
	})
}

func (w *worker) run() {
	defer w.shard.wg.Done()

	for {
		select {
		case <-w.shard.shutdown:
			return
		default:
		}

		w.shard.mu.Lock()
		for len(w.shard.heap) == 0 {
			select {
			case <-w.shard.shutdown:
				w.shard.mu.Unlock()
				return
			default:
				w.shard.notEmpty.Wait() // Releases lock, waits, reacquires
			}
		}

		job := heap.Pop(&w.shard.heap).(*job)
		w.shard.mu.Unlock()

		if job.ctx != nil {
			select {
			case <-job.ctx.Done():
				continue // Skip cancelled jobs
			default:
			}
		}

		w.shard.activeWorkers.Add(1)
		job.fn()
		w.shard.activeWorkers.Add(-1)
	}
}

func (gwp *GlobalWorkerPool) Submit(fn func()) {
	gwp.SubmitWithPriority(fn, nil, 50) // Default mid-priority
}

func (gwp *GlobalWorkerPool) SubmitWithContext(ctx context.Context, fn func()) bool {
	select {
	case <-ctx.Done():
		return false
	default:
		gwp.SubmitWithPriority(fn, ctx, 50) // Default mid-priority
		return true
	}
}

func (gwp *GlobalWorkerPool) SubmitWithPriority(fn func(), ctx context.Context, priority int) {
	if ctx != nil {
		select {
		case <-ctx.Done():
			return
		default:
		}
	}

	shardIdx := gwp.nextShard.Add(1) % gwp.numShards
	shard := gwp.shards[shardIdx]

	shard.mu.Lock()

	job := &job{
		fn:       fn,
		ctx:      ctx,
		priority: priority,
		enqueued: time.Now(),
	}
	heap.Push(&shard.heap, job)
	gwp.totalSubmissions.Add(1)

	shard.mu.Unlock()

	shard.notEmpty.Signal()
}

func (gwp *GlobalWorkerPool) AvailableWorkers() int {
	active := int64(0)
	for _, shard := range gwp.shards {
		active += shard.activeWorkers.Load()
	}
	return gwp.maxWorkers - int(active)
}

func (gwp *GlobalWorkerPool) MaxWorkers() int {
	return gwp.maxWorkers
}

func (gwp *GlobalWorkerPool) ActiveWorkers() int64 {
	active := int64(0)
	for _, shard := range gwp.shards {
		active += shard.activeWorkers.Load()
	}
	return active
}

func (gwp *GlobalWorkerPool) TotalSubmissions() int64 {
	return gwp.totalSubmissions.Load()
}

func (gwp *GlobalWorkerPool) QueueDepth() int {
	depth := 0
	for _, shard := range gwp.shards {
		shard.mu.Lock()
		depth += len(shard.heap)
		shard.mu.Unlock()
	}
	return depth
}

func (gwp *GlobalWorkerPool) QueueCapacity() int {
	capTotal := 0
	for _, shard := range gwp.shards {
		shard.mu.Lock()
		capTotal += cap(shard.heap)
		shard.mu.Unlock()
	}
	return capTotal
}

func (gwp *GlobalWorkerPool) Shutdown() {
	for _, shard := range gwp.shards {
		close(shard.shutdown)
		shard.wg.Wait()
	}
}

// per-request facade borrowing workers from global pool
type WorkerPool struct {
	global       *GlobalWorkerPool
	quota        int           // Soft limit: max concurrent workers this request should use
	active       atomic.Int32  // Current workers borrowed by this request
	semaphore    chan struct{} // Per-request semaphore for quota enforcement
	enforceQuota bool          // Whether to block when quota exceeded
}

func NewWorkerPool(maxWorkers int) *WorkerPool {
	if maxWorkers == 0 {
		return &WorkerPool{
			global:       nil, // Don't use global pool for tests
			quota:        1,
			semaphore:    make(chan struct{}, 1),
			enforceQuota: false, // Disabled for test mode
		}
	}

	if maxWorkers < 0 {
		maxWorkers = runtime.GOMAXPROCS(0) * 5000
	}

	return &WorkerPool{
		global:       GetGlobalWorkerPool(),
		quota:        maxWorkers,
		semaphore:    make(chan struct{}, maxWorkers),
		enforceQuota: true,
	}
}

func (wp *WorkerPool) Submit(fn func()) {
	if wp.global == nil {
		go fn()
		return
	}

	if wp.enforceQuota {
		wp.semaphore <- struct{}{}
	}

	wp.active.Add(1)

	wrappedFn := func() {
		defer func() {
			wp.active.Add(-1)
			if wp.enforceQuota {
				<-wp.semaphore // Release quota slot
			}
		}()
		fn()
	}

	wp.global.Submit(wrappedFn)
}

func (wp *WorkerPool) SubmitWithContext(ctx context.Context, fn func()) bool {
	select {
	case <-ctx.Done():
		return false
	default:
	}

	if wp.global == nil {
		go fn()
		return true
	}

	if wp.enforceQuota {
		select {
		case <-ctx.Done():
			return false
		case wp.semaphore <- struct{}{}:
		}
	}

	wp.active.Add(1)

	wrappedFn := func() {
		defer func() {
			wp.active.Add(-1)
			if wp.enforceQuota {
				<-wp.semaphore
			}
		}()
		fn()
	}

	submitted := wp.global.SubmitWithContext(ctx, wrappedFn)

	if !submitted {
		wp.active.Add(-1)
		if wp.enforceQuota {
			<-wp.semaphore
		}
	}

	return submitted
}

func (wp *WorkerPool) AvailableWorkers() int {
	if wp.global == nil {
		return 100 // Test mode: return arbitrary value
	}
	return wp.global.AvailableWorkers()
}

func (wp *WorkerPool) MaxWorkers() int {
	if wp.global == nil {
		return 100 // Test mode: return arbitrary value
	}
	return wp.global.MaxWorkers()
}

func (wp *WorkerPool) ActiveForRequest() int {
	return int(wp.active.Load())
}

func (wp *WorkerPool) QuotaUtilization() float64 {
	return float64(wp.active.Load()) / float64(wp.quota) * 100.0
}

func (wp *WorkerPool) RemainingQuota() int {
	if !wp.enforceQuota {
		if wp.global == nil {
			return 100 // Test mode: return arbitrary value
		}
		return wp.global.AvailableWorkers()
	}
	return wp.quota - int(wp.active.Load())
}
