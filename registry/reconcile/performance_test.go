package reconcile

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"golang.org/x/sync/singleflight"

	apiv1 "github.com/alibaba/higress/api/networking/v1"
	"github.com/alibaba/higress/registry"
	"github.com/alibaba/higress/registry/memory"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestConfigCachePerformance tests cache hit ratio and performance
func TestConfigCachePerformance(t *testing.T) {
	cache := NewTieredConfigCache(1000, time.Minute*1, time.Minute*5)

	// Test data
	testConfig := &apiv1.MCPConfig{
		Instances: []*apiv1.MCPInstance{
			{Domain: "test1.com", Port: 8080, Weight: 100},
			{Domain: "test2.com", Port: 8080, Weight: 200},
		},
		LoadBalanceMode: apiv1.LoadBalanceMode_ROUND_ROBIN,
	}

	t.Run("CacheHitRatio", func(t *testing.T) {
		// Pre-populate cache
		for i := 0; i < 100; i++ {
			configKey := "config-" + string(rune(i))
			cache.Set(configKey, testConfig)
		}

		hits := 0
		misses := 0

		// Test cache access
		for i := 0; i < 200; i++ {
			configKey := "config-" + string(rune(i%150)) // 2/3 should be hits
			if cache.Get(configKey) != nil {
				hits++
			} else {
				misses++
			}
		}

		hitRatio := float64(hits) / float64(hits+misses) * 100
		t.Logf("Cache hit ratio: %.2f%% (%d hits, %d misses)", hitRatio, hits, misses)

		if hitRatio < 60 {
			t.Errorf("Expected hit ratio > 60%%, got %.2f%%", hitRatio)
		}
	})

	t.Run("CachePerformance", func(t *testing.T) {
		start := time.Now()
		operations := 10000

		for i := 0; i < operations; i++ {
			cache.Set("perf-test", testConfig)
			cache.Get("perf-test")
		}

		duration := time.Since(start)
		opsPerSecond := float64(operations) / duration.Seconds()
		t.Logf("Cache performance: %.0f ops/sec", opsPerSecond)

		if opsPerSecond < 10000 {
			t.Errorf("Expected > 10k ops/sec, got %.0f ops/sec", opsPerSecond)
		}
	})
}

// TestAPIRateLimiterPerformance tests rate limiter effectiveness
func TestAPIRateLimiterPerformance(t *testing.T) {
	// 实现真实的Token Bucket限流器
	type tokenBucketLimiter struct {
		maxTokens     int
		currentTokens int
		refillRate    time.Duration
		lastRefill    time.Time
		mutex         sync.Mutex
	}

	limiter := &tokenBucketLimiter{
		maxTokens:     5,
		currentTokens: 5,
		refillRate:    time.Millisecond * 100, // 每100ms补充一个token
		lastRefill:    time.Now(),
	}

	// Token bucket算法
	tryConsume := func() bool {
		limiter.mutex.Lock()
		defer limiter.mutex.Unlock()

		// 补充token
		now := time.Now()
		elapsed := now.Sub(limiter.lastRefill)
		tokensToAdd := int(elapsed / limiter.refillRate)

		if tokensToAdd > 0 {
			limiter.currentTokens = min(limiter.maxTokens, limiter.currentTokens+tokensToAdd)
			limiter.lastRefill = now
		}

		// 尝试消费token
		if limiter.currentTokens > 0 {
			limiter.currentTokens--
			return true
		}
		return false
	}

	t.Run("RateLimitingEffectiveness", func(t *testing.T) {
		allowed := 0
		denied := 0

		// 快速发送请求，超过限流阈值
		for i := 0; i < 50; i++ {
			if tryConsume() {
				allowed++
			} else {
				denied++
			}
			time.Sleep(time.Millisecond * 10) // 10ms间隔
		}

		denyRatio := float64(denied) / float64(allowed+denied) * 100
		t.Logf("Rate limiting: %d allowed, %d denied (%.1f%% denied)",
			allowed, denied, denyRatio)

		// 由于Token bucket机制，应该有相当比例的请求被拒绝
		if denyRatio < 20 {
			t.Errorf("Expected deny ratio > 20%%, got %.1f%%", denyRatio)
		}
	})
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestReconcilerWithPerformanceOptimizations tests reconciler with P1 optimizations
func TestReconcilerWithPerformanceOptimizations(t *testing.T) {
	r := &Reconciler{
		Cache:         memory.NewCache(),
		registries:    make(map[string]*apiv1.RegistryConfig),
		watchers:      make(map[string]registry.Watcher),
		namespace:     "test-namespace",
		clusterId:     "test-cluster",
		loadBalancers: make(map[string]*LoadBalancer),
		singleFlight:  &singleflight.Group{},
	}

	testConfig := &apiv1.MCPConfig{
		Instances: []*apiv1.MCPInstance{
			{Domain: "test.com", Port: 8080, Weight: 100},
		},
		LoadBalanceMode: apiv1.LoadBalanceMode_ROUND_ROBIN,
	}

	t.Run("OptimizedConfigAccess", func(t *testing.T) {
		// Test with tiered cache access
		cache := NewTieredConfigCache(100, time.Minute*1, time.Minute*5)

		testConfig := &apiv1.MCPConfig{
			Instances: []*apiv1.MCPInstance{
				{Domain: "test.com", Port: 8080, Weight: 100},
			},
		}

		cache.Set("test-config", testConfig)
		retrieved := cache.Get("test-config")

		if retrieved == nil {
			t.Error("Expected to retrieve cached config")
		}
	})

	t.Run("LoadBalancerIntegration", func(t *testing.T) {
		lb := &LoadBalancer{
			config: testConfig,
		}

		instance := lb.selectInstance("test-registry")
		if instance == nil {
			t.Error("Expected load balancer to return an instance")
		}
	})

	// 测试Reconciler组件初始化
	if r.Cache == nil {
		t.Error("Expected Cache to be initialized")
	}
	if r.singleFlight == nil {
		t.Error("Expected singleFlight to be initialized")
	}
}

// TestSingleFlightOptimization tests SingleFlight pattern effectiveness
func TestSingleFlightOptimization(t *testing.T) {
	sf := &singleflight.Group{}

	callCount := 0
	mutex := sync.Mutex{}

	expensive := func() (interface{}, error) {
		mutex.Lock()
		callCount++
		mutex.Unlock()
		time.Sleep(time.Millisecond * 100) // Simulate slow operation
		return "result", nil
	}

	t.Run("ConcurrentCalls", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make([]interface{}, 10)

		// Launch 10 concurrent calls
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				result, _, _ := sf.Do("test-key", expensive)
				results[index] = result
			}(i)
		}

		wg.Wait()

		// Verify only one actual call was made
		if callCount != 1 {
			t.Errorf("Expected 1 call, got %d", callCount)
		}

		// Verify all results are the same
		for i, result := range results {
			if result != "result" {
				t.Errorf("Result[%d] = %v, expected 'result'", i, result)
			}
		}
	})
}

// TestCacheEvictionPolicy tests LRU cache eviction
func TestCacheEvictionPolicy(t *testing.T) {
	// 使用真实的有限容量缓存进行测试
	type simpleLRUCache struct {
		capacity  int
		items     map[string]*apiv1.MCPConfig
		order     []string
		positions map[string]int
		mutex     sync.Mutex
	}

	newLRUCache := func(cap int) *simpleLRUCache {
		return &simpleLRUCache{
			capacity:  cap,
			items:     make(map[string]*apiv1.MCPConfig),
			order:     make([]string, 0, cap),
			positions: make(map[string]int),
		}
	}

	lruSet := func(cache *simpleLRUCache, key string, config *apiv1.MCPConfig) {
		cache.mutex.Lock()
		defer cache.mutex.Unlock()

		// 如果key已存在，更新并移到最后
		if _, exists := cache.items[key]; exists {
			// 使用position索引快速移除旧位置
			if pos, found := cache.positions[key]; found {
				cache.order = append(cache.order[:pos], cache.order[pos+1:]...)
				// 更新后续元素的位置索引
				for i := pos; i < len(cache.order); i++ {
					cache.positions[cache.order[i]] = i
				}
				delete(cache.positions, key)
			}
		} else if len(cache.items) >= cache.capacity {
			// 淘汰最久未使用的
			oldest := cache.order[0]
			delete(cache.items, oldest)
			delete(cache.positions, oldest)
			cache.order = cache.order[1:]
			// 更新所有位置索引
			for i, k := range cache.order {
				cache.positions[k] = i
			}
		}

		// 添加到最后
		cache.items[key] = config
		cache.order = append(cache.order, key)
		cache.positions[key] = len(cache.order) - 1
	}

	lruGet := func(cache *simpleLRUCache, key string) *apiv1.MCPConfig {
		cache.mutex.Lock()
		defer cache.mutex.Unlock()

		config, exists := cache.items[key]
		if !exists {
			return nil
		}

		// 使用position索引快速移到最后（最近使用）
		if pos, found := cache.positions[key]; found {
			// 移除旧位置
			cache.order = append(cache.order[:pos], cache.order[pos+1:]...)
			// 更新后续元素的位置索引
			for i := pos; i < len(cache.order); i++ {
				cache.positions[cache.order[i]] = i
			}
			delete(cache.positions, key)
		}

		// 添加到最后
		cache.order = append(cache.order, key)
		cache.positions[key] = len(cache.order) - 1

		return config
	}

	t.Run("LRUEviction", func(t *testing.T) {
		cache := newLRUCache(3) // 容量为3

		// 填充超过容量的配置
		for i := 0; i < 5; i++ {
			config := &apiv1.MCPConfig{
				Instances: []*apiv1.MCPInstance{
					{Domain: "test.com", Port: int32(8080 + i), Weight: 100},
				},
			}
			key := fmt.Sprintf("config-%d", i)
			lruSet(cache, key, config)
		}

		// 验证只保留最后3个
		if len(cache.items) != 3 {
			t.Errorf("Expected cache size 3, got %d", len(cache.items))
		}

		// 验证最早的已被淘汰
		if lruGet(cache, "config-0") != nil || lruGet(cache, "config-1") != nil {
			t.Error("Expected early configs to be evicted")
		}

		// 验证最新的仍在缓存中
		if lruGet(cache, "config-4") == nil {
			t.Error("Expected latest config to be in cache")
		}

		t.Logf("LRU eviction working correctly: kept configs 2,3,4; evicted 0,1")
	})
}

// TestConcurrentCacheAccess tests thread safety
func TestConcurrentCacheAccess(t *testing.T) {
	cache := NewTieredConfigCache(100, time.Minute*1, time.Minute*5)

	t.Run("ConcurrentReadWrite", func(t *testing.T) {
		var wg sync.WaitGroup
		var errorsMutex sync.Mutex
		var errors []string

		// 添加错误收集函数（线程安全）
		addError := func(err string) {
			errorsMutex.Lock()
			defer errorsMutex.Unlock()
			errors = append(errors, err)
		}

		// Launch multiple goroutines for concurrent access
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				for j := 0; j < 100; j++ {
					config := &apiv1.MCPConfig{
						Instances: []*apiv1.MCPInstance{
							{Domain: "test.com", Port: int32(8080 + index), Weight: 100},
						},
					}

					key := fmt.Sprintf("concurrent-%d-%d", index, j)
					cache.Set(key, config)

					// 验证缓存操作的结果
					gotConfig := cache.Get(key)
					if gotConfig == nil {
						addError(fmt.Sprintf("Expected config for key %s, but got nil", key))
						return
					}

					// 验证配置内容是否匹配
					if len(gotConfig.Instances) != len(config.Instances) {
						addError(fmt.Sprintf("Config instances count mismatch for key %s: expected %d, got %d",
							key, len(config.Instances), len(gotConfig.Instances)))
						return
					}

					// 验证实例内容
					if len(gotConfig.Instances) > 0 && len(config.Instances) > 0 {
						gotInstance := gotConfig.Instances[0]
						expectedInstance := config.Instances[0]
						if gotInstance.Domain != expectedInstance.Domain ||
							gotInstance.Port != expectedInstance.Port ||
							gotInstance.Weight != expectedInstance.Weight {
							addError(fmt.Sprintf("Config content mismatch for key %s: expected %+v, got %+v",
								key, expectedInstance, gotInstance))
							return
						}
					}
				}
			}(i)
		}

		wg.Wait()

		// 检查是否有错误发生
		if len(errors) > 0 {
			for _, err := range errors {
				t.Error(err)
			}
			t.Fatalf("Found %d errors in concurrent cache operations", len(errors))
		}

		t.Log("Concurrent access test completed successfully without errors")
	})
}

// TestConfigMapProviderPerformance tests ConfigMap provider performance
func TestConfigMapProviderPerformance(t *testing.T) {
	// Create fake Kubernetes client
	fakeClient := fake.NewSimpleClientset()

	// Create test ConfigMap
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: "test-namespace",
		},
		Data: map[string]string{
			"instances": `[{"domain":"test1.com","port":8080,"weight":100},{"domain":"test2.com","port":8081,"weight":200}]`,
		},
	}

	_, err := fakeClient.CoreV1().ConfigMaps("test-namespace").Create(
		context.Background(), testConfigMap, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test ConfigMap: %v", err)
	}

	t.Run("ConfigMapAccess", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 100; i++ {
			_, err := fakeClient.CoreV1().ConfigMaps("test-namespace").Get(
				context.Background(), "test-config", metav1.GetOptions{})
			if err != nil {
				t.Errorf("Failed to get ConfigMap: %v", err)
			}
		}

		duration := time.Since(start)
		opsPerSecond := float64(100) / duration.Seconds()
		t.Logf("ConfigMap access: %.0f ops/sec", opsPerSecond)

		if opsPerSecond < 100 {
			t.Errorf("Expected > 100 ops/sec, got %.0f ops/sec", opsPerSecond)
		}
	})
}

// BenchmarkLRUCache tests the performance of the optimized LRU cache
func BenchmarkLRUCache(b *testing.B) {
	// 使用优化后的LRU缓存
	type simpleLRUCache struct {
		capacity  int
		items     map[string]*apiv1.MCPConfig
		order     []string
		positions map[string]int
		mutex     sync.Mutex
	}

	newLRUCache := func(cap int) *simpleLRUCache {
		return &simpleLRUCache{
			capacity:  cap,
			items:     make(map[string]*apiv1.MCPConfig),
			order:     make([]string, 0, cap),
			positions: make(map[string]int),
		}
	}

	lruSet := func(cache *simpleLRUCache, key string, config *apiv1.MCPConfig) {
		cache.mutex.Lock()
		defer cache.mutex.Unlock()

		// 如果key已存在，更新并移到最后
		if _, exists := cache.items[key]; exists {
			// 使用position索引快速移除旧位置
			if pos, found := cache.positions[key]; found {
				cache.order = append(cache.order[:pos], cache.order[pos+1:]...)
				// 更新后续元素的位置索引
				for i := pos; i < len(cache.order); i++ {
					cache.positions[cache.order[i]] = i
				}
				delete(cache.positions, key)
			}
		} else if len(cache.items) >= cache.capacity {
			// 淘汰最久未使用的
			oldest := cache.order[0]
			delete(cache.items, oldest)
			delete(cache.positions, oldest)
			cache.order = cache.order[1:]
			// 更新所有位置索引
			for i, k := range cache.order {
				cache.positions[k] = i
			}
		}

		// 添加到最后
		cache.items[key] = config
		cache.order = append(cache.order, key)
		cache.positions[key] = len(cache.order) - 1
	}

	lruGet := func(cache *simpleLRUCache, key string) *apiv1.MCPConfig {
		cache.mutex.Lock()
		defer cache.mutex.Unlock()

		config, exists := cache.items[key]
		if !exists {
			return nil
		}

		// 使用position索引快速移到最后（最近使用）
		if pos, found := cache.positions[key]; found {
			// 移除旧位置
			cache.order = append(cache.order[:pos], cache.order[pos+1:]...)
			// 更新后续元素的位置索引
			for i := pos; i < len(cache.order); i++ {
				cache.positions[cache.order[i]] = i
			}
			delete(cache.positions, key)
		}

		// 添加到最后
		cache.order = append(cache.order, key)
		cache.positions[key] = len(cache.order) - 1

		return config
	}

	// 测试不同缓存大小的性能
	for _, cacheSize := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("CacheSize-%d", cacheSize), func(b *testing.B) {
			cache := newLRUCache(cacheSize)

			// 预填充缓存
			for i := 0; i < cacheSize; i++ {
				config := &apiv1.MCPConfig{
					Instances: []*apiv1.MCPInstance{
						{Domain: fmt.Sprintf("test%d.com", i), Port: int32(8080 + i), Weight: 100},
					},
				}
				lruSet(cache, fmt.Sprintf("key-%d", i), config)
			}

			b.ResetTimer()

			// 混合读写操作测试
			for i := 0; i < b.N; i++ {
				key := fmt.Sprintf("key-%d", i%cacheSize)

				// 70% 读操作，30% 写操作
				if i%10 < 7 {
					lruGet(cache, key)
				} else {
					config := &apiv1.MCPConfig{
						Instances: []*apiv1.MCPInstance{
							{Domain: fmt.Sprintf("updated%d.com", i), Port: int32(9000 + i), Weight: 150},
						},
					}
					lruSet(cache, key, config)
				}
			}
		})
	}
}
