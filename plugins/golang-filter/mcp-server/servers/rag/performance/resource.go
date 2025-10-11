package performance

import (
	"context"
	"runtime"
	"sync"
	"time"
)

// ResourceManager handles memory and resource management for RAG operations
type ResourceManager struct {
	maxMemoryMB      int64
	maxConcurrent    int
	currentRequests  int
	requestSemaphore chan struct{}
	mutex            sync.RWMutex
	gcThreshold      int64 // Memory threshold for triggering GC
	lastGC           time.Time
}

// NewResourceManager creates a new resource manager
func NewResourceManager(maxMemoryMB int64, maxConcurrent int) *ResourceManager {
	if maxMemoryMB <= 0 {
		maxMemoryMB = 512 // Default 512MB
	}
	if maxConcurrent <= 0 {
		maxConcurrent = 100 // Default 100 concurrent requests
	}
	
	rm := &ResourceManager{
		maxMemoryMB:      maxMemoryMB,
		maxConcurrent:    maxConcurrent,
		requestSemaphore: make(chan struct{}, maxConcurrent),
		gcThreshold:      maxMemoryMB * 1024 * 1024 * 8 / 10, // 80% of max memory
		lastGC:           time.Now(),
	}
	
	// Start memory monitoring
	go rm.monitorMemory()
	
	return rm
}

// AcquireRequest acquires a request slot
func (rm *ResourceManager) AcquireRequest(ctx context.Context) error {
	select {
	case rm.requestSemaphore <- struct{}{}:
		rm.mutex.Lock()
		rm.currentRequests++
		rm.mutex.Unlock()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ReleaseRequest releases a request slot
func (rm *ResourceManager) ReleaseRequest() {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	
	if rm.currentRequests > 0 {
		rm.currentRequests--
		<-rm.requestSemaphore
	}
}

// WithResourceLimit executes a function with resource limits
func (rm *ResourceManager) WithResourceLimit(ctx context.Context, fn func() error) error {
	if err := rm.AcquireRequest(ctx); err != nil {
		return err
	}
	defer rm.ReleaseRequest()
	
	return fn()
}

// GetMemoryStats returns current memory statistics
func (rm *ResourceManager) GetMemoryStats() *MemoryStats {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	
	return &MemoryStats{
		AllocatedMB:    bytesToMB(m.Alloc),
		TotalAllocMB:   bytesToMB(m.TotalAlloc),
		SystemMB:       bytesToMB(m.Sys),
		NumGC:          int64(m.NumGC),
		MaxMemoryMB:    rm.maxMemoryMB,
		GCThresholdMB:  bytesToMB(rm.gcThreshold),
		MemoryUsagePercent: float64(m.Alloc) / float64(rm.maxMemoryMB*1024*1024) * 100,
	}
}

// GetResourceStats returns current resource statistics
func (rm *ResourceManager) GetResourceStats() *ResourceStats {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	return &ResourceStats{
		MaxConcurrent:     rm.maxConcurrent,
		CurrentRequests:   rm.currentRequests,
		AvailableSlots:    rm.maxConcurrent - rm.currentRequests,
		UsagePercent:      float64(rm.currentRequests) / float64(rm.maxConcurrent) * 100,
		LastGC:            rm.lastGC,
	}
}

// ForceGC forces garbage collection
func (rm *ResourceManager) ForceGC() {
	runtime.GC()
	rm.lastGC = time.Now()
}

// monitorMemory monitors memory usage and triggers GC when needed
func (rm *ResourceManager) monitorMemory() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		
		// Trigger GC if memory usage is high and it's been a while since last GC
		if int64(m.Alloc) > rm.gcThreshold && time.Since(rm.lastGC) > time.Minute {
			rm.ForceGC()
		}
	}
}

// MemoryStats contains memory usage statistics
type MemoryStats struct {
	AllocatedMB        int64   `json:"allocated_mb"`
	TotalAllocMB       int64   `json:"total_alloc_mb"`
	SystemMB           int64   `json:"system_mb"`
	NumGC              int64   `json:"num_gc"`
	MaxMemoryMB        int64   `json:"max_memory_mb"`
	GCThresholdMB      int64   `json:"gc_threshold_mb"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
}

// ResourceStats contains resource usage statistics
type ResourceStats struct {
	MaxConcurrent   int       `json:"max_concurrent"`
	CurrentRequests int       `json:"current_requests"`
	AvailableSlots  int       `json:"available_slots"`
	UsagePercent    float64   `json:"usage_percent"`
	LastGC          time.Time `json:"last_gc"`
}

// bytesToMB converts bytes to megabytes
func bytesToMB(bytes uint64) int64 {
	return int64(bytes / 1024 / 1024)
}

// ConnectionPool manages database/service connections
type ConnectionPool struct {
	connections chan *Connection
	factory     ConnectionFactory
	maxSize     int
	timeout     time.Duration
	mutex       sync.RWMutex
	closed      bool
}

// Connection represents a pooled connection
type Connection struct {
	ID        string
	Client    interface{}
	CreatedAt time.Time
	LastUsed  time.Time
	InUse     bool
}

// ConnectionFactory creates new connections
type ConnectionFactory interface {
	Create() (*Connection, error)
	Validate(*Connection) bool
	Close(*Connection) error
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(factory ConnectionFactory, maxSize int, timeout time.Duration) *ConnectionPool {
	if maxSize <= 0 {
		maxSize = 10
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	
	return &ConnectionPool{
		connections: make(chan *Connection, maxSize),
		factory:     factory,
		maxSize:     maxSize,
		timeout:     timeout,
	}
}

// Get acquires a connection from the pool
func (cp *ConnectionPool) Get(ctx context.Context) (*Connection, error) {
	cp.mutex.RLock()
	if cp.closed {
		cp.mutex.RUnlock()
		return nil, ErrPoolClosed
	}
	cp.mutex.RUnlock()
	
	select {
	case conn := <-cp.connections:
		// Validate connection
		if cp.factory.Validate(conn) {
			conn.LastUsed = time.Now()
			conn.InUse = true
			return conn, nil
		}
		// Connection invalid, create new one
		cp.factory.Close(conn)
		fallthrough
	default:
		// No connection available, create new one
		conn, err := cp.factory.Create()
		if err != nil {
			return nil, err
		}
		conn.LastUsed = time.Now()
		conn.InUse = true
		return conn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Put returns a connection to the pool
func (cp *ConnectionPool) Put(conn *Connection) {
	if conn == nil {
		return
	}
	
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	
	if cp.closed {
		cp.factory.Close(conn)
		return
	}
	
	conn.InUse = false
	
	select {
	case cp.connections <- conn:
		// Connection returned to pool
	default:
		// Pool is full, close connection
		cp.factory.Close(conn)
	}
}

// Close closes the connection pool
func (cp *ConnectionPool) Close() error {
	cp.mutex.Lock()
	defer cp.mutex.Unlock()
	
	if cp.closed {
		return nil
	}
	
	cp.closed = true
	close(cp.connections)
	
	// Close all remaining connections
	for conn := range cp.connections {
		cp.factory.Close(conn)
	}
	
	return nil
}

// Stats returns pool statistics
func (cp *ConnectionPool) Stats() *PoolStats {
	cp.mutex.RLock()
	defer cp.mutex.RUnlock()
	
	return &PoolStats{
		MaxSize:     cp.maxSize,
		Available:   len(cp.connections),
		InUse:       cp.maxSize - len(cp.connections),
		Closed:      cp.closed,
	}
}

// PoolStats contains connection pool statistics
type PoolStats struct {
	MaxSize   int  `json:"max_size"`
	Available int  `json:"available"`
	InUse     int  `json:"in_use"`
	Closed    bool `json:"closed"`
}

// Custom errors
type PoolError struct {
	Message string
}

func (e *PoolError) Error() string {
	return e.Message
}

var (
	ErrPoolClosed = &PoolError{"connection pool is closed"}
)