package performance

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ConcurrencyManager handles concurrent processing and load balancing
type ConcurrencyManager struct {
	workers      []*Worker
	workQueue    chan *Task
	resultQueue  chan *TaskResult
	workerPool   chan *Worker
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	maxWorkers   int
	taskTimeout  time.Duration
	metrics      *ConcurrencyMetrics
	mutex        sync.RWMutex
}

// Worker represents a worker in the pool
type Worker struct {
	ID        int
	TaskQueue chan *Task
	Quit      chan bool
	Active    bool
	LastTask  time.Time
}

// Task represents a unit of work
type Task struct {
	ID       string
	Type     string
	Data     interface{}
	Context  context.Context
	Handler  TaskHandler
	Priority int
	Created  time.Time
}

// TaskResult represents the result of a task
type TaskResult struct {
	TaskID   string
	Result   interface{}
	Error    error
	Duration time.Duration
}

// TaskHandler processes a task
type TaskHandler interface {
	Handle(ctx context.Context, data interface{}) (interface{}, error)
}

// ConcurrencyMetrics tracks concurrency performance
type ConcurrencyMetrics struct {
	TotalTasks      int64
	CompletedTasks  int64
	FailedTasks     int64
	ActiveTasks     int64
	AverageWaitTime time.Duration
	AverageExecTime time.Duration
	WorkerUtilization float64
	QueueSize       int
	mutex           sync.RWMutex
}

// NewConcurrencyManager creates a new concurrency manager
func NewConcurrencyManager(maxWorkers int, queueSize int, taskTimeout time.Duration) *ConcurrencyManager {
	if maxWorkers <= 0 {
		maxWorkers = 10
	}
	if queueSize <= 0 {
		queueSize = 1000
	}
	if taskTimeout <= 0 {
		taskTimeout = 30 * time.Second
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	cm := &ConcurrencyManager{
		workers:     make([]*Worker, 0, maxWorkers),
		workQueue:   make(chan *Task, queueSize),
		resultQueue: make(chan *TaskResult, queueSize),
		workerPool:  make(chan *Worker, maxWorkers),
		ctx:         ctx,
		cancel:      cancel,
		maxWorkers:  maxWorkers,
		taskTimeout: taskTimeout,
		metrics:     &ConcurrencyMetrics{},
	}
	
	// Start workers
	for i := 0; i < maxWorkers; i++ {
		worker := cm.createWorker(i)
		cm.workers = append(cm.workers, worker)
		cm.workerPool <- worker
	}
	
	// Start dispatcher
	go cm.dispatch()
	
	// Start metrics collector
	go cm.collectMetrics()
	
	return cm
}

// SubmitTask submits a task for processing
func (cm *ConcurrencyManager) SubmitTask(ctx context.Context, task *Task) error {
	task.Created = time.Now()
	
	select {
	case cm.workQueue <- task:
		cm.incrementTotalTasks()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrQueueFull
	}
}

// SubmitTaskWithResult submits a task and returns the result
func (cm *ConcurrencyManager) SubmitTaskWithResult(ctx context.Context, task *Task) (*TaskResult, error) {
	if err := cm.SubmitTask(ctx, task); err != nil {
		return nil, err
	}
	
	// Wait for result
	select {
	case result := <-cm.resultQueue:
		if result.TaskID == task.ID {
			return result, nil
		}
		// Not our result, put it back (this is a simplification)
		select {
		case cm.resultQueue <- result:
		default:
		}
		return nil, ErrResultNotFound
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(cm.taskTimeout):
		return nil, ErrTaskTimeout
	}
}

// GetMetrics returns current concurrency metrics
func (cm *ConcurrencyManager) GetMetrics() *ConcurrencyMetrics {
	cm.metrics.mutex.RLock()
	defer cm.metrics.mutex.RUnlock()
	
	// Calculate worker utilization
	activeWorkers := 0
	for _, worker := range cm.workers {
		if worker.Active {
			activeWorkers++
		}
	}
	
	utilization := float64(activeWorkers) / float64(cm.maxWorkers) * 100
	
	return &ConcurrencyMetrics{
		TotalTasks:        cm.metrics.TotalTasks,
		CompletedTasks:    cm.metrics.CompletedTasks,
		FailedTasks:       cm.metrics.FailedTasks,
		ActiveTasks:       cm.metrics.ActiveTasks,
		AverageWaitTime:   cm.metrics.AverageWaitTime,
		AverageExecTime:   cm.metrics.AverageExecTime,
		WorkerUtilization: utilization,
		QueueSize:         len(cm.workQueue),
	}
}

// Shutdown gracefully shuts down the concurrency manager
func (cm *ConcurrencyManager) Shutdown(timeout time.Duration) error {
	cm.cancel()
	
	// Stop all workers
	for _, worker := range cm.workers {
		worker.Stop()
	}
	
	// Wait for workers to finish or timeout
	done := make(chan struct{})
	go func() {
		cm.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return ErrShutdownTimeout
	}
}

// createWorker creates a new worker
func (cm *ConcurrencyManager) createWorker(id int) *Worker {
	worker := &Worker{
		ID:        id,
		TaskQueue: make(chan *Task),
		Quit:      make(chan bool),
	}
	
	cm.wg.Add(1)
	go worker.Start(cm)
	
	return worker
}

// dispatch dispatches tasks to available workers
func (cm *ConcurrencyManager) dispatch() {
	for {
		select {
		case task := <-cm.workQueue:
			select {
			case worker := <-cm.workerPool:
				worker.TaskQueue <- task
			case <-cm.ctx.Done():
				return
			}
		case <-cm.ctx.Done():
			return
		}
	}
}

// collectMetrics collects performance metrics
func (cm *ConcurrencyManager) collectMetrics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			cm.updateMetrics()
		case <-cm.ctx.Done():
			return
		}
	}
}

// updateMetrics updates performance metrics
func (cm *ConcurrencyManager) updateMetrics() {
	cm.metrics.mutex.Lock()
	defer cm.metrics.mutex.Unlock()
	
	cm.metrics.QueueSize = len(cm.workQueue)
	
	// Calculate worker utilization
	activeWorkers := 0
	for _, worker := range cm.workers {
		if worker.Active {
			activeWorkers++
		}
	}
	cm.metrics.WorkerUtilization = float64(activeWorkers) / float64(cm.maxWorkers) * 100
}

// incrementTotalTasks atomically increments total tasks
func (cm *ConcurrencyManager) incrementTotalTasks() {
	cm.metrics.mutex.Lock()
	defer cm.metrics.mutex.Unlock()
	cm.metrics.TotalTasks++
}

// incrementCompletedTasks atomically increments completed tasks
func (cm *ConcurrencyManager) incrementCompletedTasks() {
	cm.metrics.mutex.Lock()
	defer cm.metrics.mutex.Unlock()
	cm.metrics.CompletedTasks++
}

// incrementFailedTasks atomically increments failed tasks
func (cm *ConcurrencyManager) incrementFailedTasks() {
	cm.metrics.mutex.Lock()
	defer cm.metrics.mutex.Unlock()
	cm.metrics.FailedTasks++
}

// Worker methods

// Start starts the worker
func (w *Worker) Start(cm *ConcurrencyManager) {
	defer cm.wg.Done()
	
	for {
		select {
		case task := <-w.TaskQueue:
			w.processTask(cm, task)
			// Return worker to pool
			cm.workerPool <- w
		case <-w.Quit:
			return
		case <-cm.ctx.Done():
			return
		}
	}
}

// Stop stops the worker
func (w *Worker) Stop() {
	close(w.Quit)
}

// processTask processes a single task
func (w *Worker) processTask(cm *ConcurrencyManager, task *Task) {
	w.Active = true
	w.LastTask = time.Now()
	startTime := time.Now()
	
	defer func() {
		w.Active = false
		duration := time.Since(startTime)
		
		// Create result
		result := &TaskResult{
			TaskID:   task.ID,
			Duration: duration,
		}
		
		// Handle panic
		if r := recover(); r != nil {
			result.Error = &TaskPanicError{fmt.Sprintf("task panicked: %v", r)}
			cm.incrementFailedTasks()
		} else {
			cm.incrementCompletedTasks()
		}
		
		// Send result
		select {
		case cm.resultQueue <- result:
		default:
			// Result queue full, drop result
		}
	}()
	
	// Create task context with timeout
	ctx, cancel := context.WithTimeout(task.Context, cm.taskTimeout)
	defer cancel()
	
	// Process task
	result, err := task.Handler.Handle(ctx, task.Data)
	if err != nil {
		cm.incrementFailedTasks()
		// Send error result
		select {
		case cm.resultQueue <- &TaskResult{
			TaskID: task.ID,
			Error:  err,
			Duration: time.Since(startTime),
		}:
		default:
		}
		return
	}
	
	// Send success result
	select {
	case cm.resultQueue <- &TaskResult{
		TaskID: task.ID,
		Result: result,
		Duration: time.Since(startTime),
	}:
	default:
	}
}

// LoadBalancer distributes work across multiple concurrency managers
type LoadBalancer struct {
	managers []ConcurrencyManagerInterface
	strategy LoadBalancingStrategy
	current  int
	mutex    sync.RWMutex
}

// ConcurrencyManagerInterface defines the interface for concurrency managers
type ConcurrencyManagerInterface interface {
	SubmitTask(ctx context.Context, task *Task) error
	GetMetrics() *ConcurrencyMetrics
}

// LoadBalancingStrategy defines load balancing strategies
type LoadBalancingStrategy int

const (
	RoundRobin LoadBalancingStrategy = iota
	LeastLoaded
	Random
)

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(managers []ConcurrencyManagerInterface, strategy LoadBalancingStrategy) *LoadBalancer {
	return &LoadBalancer{
		managers: managers,
		strategy: strategy,
	}
}

// SubmitTask submits a task using the configured load balancing strategy
func (lb *LoadBalancer) SubmitTask(ctx context.Context, task *Task) error {
	manager := lb.selectManager()
	if manager == nil {
		return ErrNoAvailableManagers
	}
	
	return manager.SubmitTask(ctx, task)
}

// selectManager selects a manager based on the load balancing strategy
func (lb *LoadBalancer) selectManager() ConcurrencyManagerInterface {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()
	
	if len(lb.managers) == 0 {
		return nil
	}
	
	switch lb.strategy {
	case RoundRobin:
		manager := lb.managers[lb.current]
		lb.current = (lb.current + 1) % len(lb.managers)
		return manager
		
	case LeastLoaded:
		var bestManager ConcurrencyManagerInterface
		var minLoad int64 = -1
		
		for _, manager := range lb.managers {
			metrics := manager.GetMetrics()
			load := metrics.TotalTasks - metrics.CompletedTasks
			if minLoad == -1 || load < minLoad {
				minLoad = load
				bestManager = manager
			}
		}
		return bestManager
		
	case Random:
		// Simple random selection (using time-based)
		index := int(time.Now().UnixNano()) % len(lb.managers)
		return lb.managers[index]
		
	default:
		return lb.managers[0]
	}
}

// Custom errors
type ConcurrencyError struct {
	Message string
}

func (e *ConcurrencyError) Error() string {
	return e.Message
}

type TaskPanicError struct {
	Message string
}

func (e *TaskPanicError) Error() string {
	return e.Message
}

var (
	ErrQueueFull             = &ConcurrencyError{"task queue is full"}
	ErrTaskTimeout           = &ConcurrencyError{"task execution timeout"}
	ErrResultNotFound        = &ConcurrencyError{"task result not found"}
	ErrShutdownTimeout       = &ConcurrencyError{"shutdown timeout exceeded"}
	ErrNoAvailableManagers   = &ConcurrencyError{"no available concurrency managers"}
)