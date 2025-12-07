package concurrency

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// WorkerPool manages a pool of workers
type WorkerPool struct {
	workers int
	jobs    chan Job
	results chan Result
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

// Job represents a job to be processed
type Job struct {
	ID   string
	Task func(ctx context.Context) (interface{}, error)
}

// Result represents a job result
type Result struct {
	JobID string
	Data  interface{}
	Error error
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workers int, bufferSize int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())

	return &WorkerPool{
		workers: workers,
		jobs:    make(chan Job, bufferSize),
		results: make(chan Result, bufferSize),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

// worker processes jobs
func (wp *WorkerPool) worker() {
	defer wp.wg.Done()

	for {
		select {
		case <-wp.ctx.Done():
			return
		case job, ok := <-wp.jobs:
			if !ok {
				return
			}

			data, err := job.Task(wp.ctx)
			wp.results <- Result{
				JobID: job.ID,
				Data:  data,
				Error: err,
			}
		}
	}
}

// Submit submits a job to the pool
func (wp *WorkerPool) Submit(job Job) {
	wp.jobs <- job
}

// Results returns the results channel
func (wp *WorkerPool) Results() <-chan Result {
	return wp.results
}

// Stop stops the worker pool immediately
func (wp *WorkerPool) Stop() {
	wp.cancel()
	close(wp.jobs)
	wp.wg.Wait()
	close(wp.results)
}

// Shutdown gracefully shuts down the worker pool, waiting for context
func (wp *WorkerPool) Shutdown(ctx context.Context) error {
	close(wp.jobs)
	
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		close(wp.results)
		return nil
	case <-ctx.Done():
		wp.cancel() // Force cancel remaining
		return ctx.Err()
	}
}

// Cancel cancels all jobs
func (wp *WorkerPool) Cancel() {
	wp.cancel()
	wp.wg.Wait()
}

// RateLimiter limits the rate of operations
type RateLimiter struct {
	rate     int
	interval time.Duration
	tokens   chan struct{}
	stop     chan struct{}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate int, interval time.Duration) *RateLimiter {
	rl := &RateLimiter{
		rate:     rate,
		interval: interval,
		tokens:   make(chan struct{}, rate),
		stop:     make(chan struct{}),
	}

	// Fill initial tokens
	for i := 0; i < rate; i++ {
		rl.tokens <- struct{}{}
	}

	// Start refilling
	go rl.refill()

	return rl
}

// Wait waits for a token
func (rl *RateLimiter) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-rl.stop:
		return fmt.Errorf("rate limiter stopped")
	case <-rl.tokens:
		return nil
	}
}

// refill refills tokens
func (rl *RateLimiter) refill() {
	ticker := time.NewTicker(rl.interval)
	defer ticker.Stop()

	for {
		select {
		case <-rl.stop:
			return
		case <-ticker.C:
			// Try to add tokens up to rate
			for i := 0; i < rl.rate; i++ {
				select {
				case rl.tokens <- struct{}{}:
				default:
					// Channel full, skip
				}
			}
		}
	}
}

// Stop stops the rate limiter
func (rl *RateLimiter) Stop() {
	close(rl.stop)
}

// Semaphore limits concurrent access
type Semaphore struct {
	permits chan struct{}
}

// NewSemaphore creates a new semaphore
func NewSemaphore(maxConcurrent int) *Semaphore {
	return &Semaphore{
		permits: make(chan struct{}, maxConcurrent),
	}
}

// Acquire acquires a permit
func (s *Semaphore) Acquire(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.permits <- struct{}{}:
		return nil
	}
}

// Release releases a permit
func (s *Semaphore) Release() {
	<-s.permits
}

// TryAcquire tries to acquire without blocking
func (s *Semaphore) TryAcquire() bool {
	select {
	case s.permits <- struct{}{}:
		return true
	default:
		return false
	}
}

// Barrier synchronizes multiple goroutines
type Barrier struct {
	count int
	wait  chan struct{}
	mu    sync.Mutex
}

// NewBarrier creates a new barrier
func NewBarrier(count int) *Barrier {
	return &Barrier{
		count: count,
		wait:  make(chan struct{}),
	}
}

// Wait waits at the barrier
func (b *Barrier) Wait() {
	b.mu.Lock()
	b.count--
	if b.count == 0 {
		close(b.wait)
		b.mu.Unlock()
		return
	}
	b.mu.Unlock()
	<-b.wait
}

// Reset resets the barrier
func (b *Barrier) Reset(count int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.count = count
	b.wait = make(chan struct{})
}

// TaskQueue manages a queue of tasks with priority
type TaskQueue struct {
	tasks    []Task
	mu       sync.Mutex
	notEmpty *sync.Cond
	closed   bool
}

// Task represents a task with priority
type Task struct {
	ID       string
	Priority int
	Func     func() error
}

// NewTaskQueue creates a new task queue
func NewTaskQueue() *TaskQueue {
	tq := &TaskQueue{
		tasks: make([]Task, 0),
	}
	tq.notEmpty = sync.NewCond(&tq.mu)
	return tq
}

// Push adds a task to the queue
func (tq *TaskQueue) Push(task Task) error {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	if tq.closed {
		return fmt.Errorf("queue is closed")
	}

	// Insert in priority order
	inserted := false
	for i, t := range tq.tasks {
		if task.Priority > t.Priority {
			tq.tasks = append(tq.tasks[:i], append([]Task{task}, tq.tasks[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		tq.tasks = append(tq.tasks, task)
	}

	tq.notEmpty.Signal()
	return nil
}

// Pop removes and returns the highest priority task
func (tq *TaskQueue) Pop() (Task, error) {
	tq.mu.Lock()
	defer tq.mu.Unlock()

	for len(tq.tasks) == 0 && !tq.closed {
		tq.notEmpty.Wait()
	}

	if tq.closed && len(tq.tasks) == 0 {
		return Task{}, fmt.Errorf("queue is closed")
	}

	task := tq.tasks[0]
	tq.tasks = tq.tasks[1:]

	return task, nil
}

// Len returns the number of tasks in the queue
func (tq *TaskQueue) Len() int {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	return len(tq.tasks)
}

// Close closes the queue
func (tq *TaskQueue) Close() {
	tq.mu.Lock()
	defer tq.mu.Unlock()
	tq.closed = true
	tq.notEmpty.Broadcast()
}

// BatchProcessor processes items in batches
type BatchProcessor struct {
	batchSize int
	maxWait   time.Duration
	processor func([]interface{}) error
	items     []interface{}
	mu        sync.Mutex
	timer     *time.Timer
	stop      chan struct{}
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(batchSize int, maxWait time.Duration, processor func([]interface{}) error) *BatchProcessor {
	return &BatchProcessor{
		batchSize: batchSize,
		maxWait:   maxWait,
		processor: processor,
		items:     make([]interface{}, 0, batchSize),
		stop:      make(chan struct{}),
	}
}

// Add adds an item to the batch
func (bp *BatchProcessor) Add(item interface{}) error {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	bp.items = append(bp.items, item)

	if bp.timer == nil {
		bp.timer = time.AfterFunc(bp.maxWait, func() {
			bp.flush()
		})
	}

	if len(bp.items) >= bp.batchSize {
		bp.timer.Stop()
		bp.timer = nil
		return bp.flushUnlocked()
	}

	return nil
}

// flush flushes the batch
func (bp *BatchProcessor) flush() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	return bp.flushUnlocked()
}

// flushUnlocked flushes without locking
func (bp *BatchProcessor) flushUnlocked() error {
	if len(bp.items) == 0 {
		return nil
	}

	items := bp.items
	bp.items = make([]interface{}, 0, bp.batchSize)

	return bp.processor(items)
}

// Close closes the batch processor
func (bp *BatchProcessor) Close() error {
	close(bp.stop)
	if bp.timer != nil {
		bp.timer.Stop()
	}
	return bp.flush()
}
