package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8" // Redis客户端
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// 全局变量定义
var (
	// Redis客户端
	redisClient *redis.Client
	ctx         = context.Background()

	// 定义Metric（与Higress原有Metric命名规范对齐）
	// 1. 各模型的累计cached input token
	aiCachedInputTokenTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "higress_ai_cache_cached_input_token_total",
			Help: "Total number of cached input tokens per AI model",
		},
		[]string{"model"}, // 标签：模型名称
	)

	// 2. 各模型的累计cached output token
	aiCachedOutputTokenTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "higress_ai_cache_cached_output_token_total",
			Help: "Total number of cached output tokens per AI model",
		},
		[]string{"model"},
	)

	// 3. 各模型的缓存命中次数
	aiCacheHitCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "higress_ai_cache_hit_count",
			Help: "Number of cache hits per AI model",
		},
		[]string{"model"},
	)

	// 4. 各模型的缓存未命中次数
	aiCacheMissCount = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "higress_ai_cache_miss_count",
			Help: "Number of cache misses per AI model",
		},
		[]string{"model"},
	)

	// 5. 缓存命中率（计算后的值，可选）
	aiCacheHitRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "higress_ai_cache_hit_rate",
			Help: "Cache hit rate per AI model (0-1)",
		},
		[]string{"model"},
	)
)

// 初始化Redis客户端
func initRedis(addr, password string, db int) {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     addr,     // Redis地址，如"redis:6379"
		Password: password, // Redis密码（无则为空）
		DB:       db,       // Redis数据库编号
	})

	// 测试Redis连接
	_, err := redisClient.Ping(ctx).Result()
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to Redis: %v", err))
	}
	fmt.Println("Connected to Redis successfully")
}

// collectRedisMetrics 从Redis采集数据并更新Prometheus Metric
func collectRedisMetrics() {
	// 步骤1：获取所有模型名称（通过Redis的Key匹配）
	// 匹配模式：ai_cache:token:*（存储模型Token的Key）
	modelKeys, err := redisClient.Keys(ctx, "ai_cache:token:*").Result()
	if err != nil {
		fmt.Printf("Failed to get model keys from Redis: %v\n", err)
		return
	}

	// 提取模型名称（从Key中解析：ai_cache:token:gpt-3.5-turbo → gpt-3.5-turbo）
	var models []string
	for _, key := range modelKeys {
		parts := strings.Split(key, ":")
		if len(parts) >= 3 {
			model := parts[2]
			models = append(models, model)
		}
	}

	// 步骤2：遍历每个模型，采集数据
	for _, model := range models {
		// 2.1 读取该模型的cached input/output token（Hash类型）
		tokenHashKey := fmt.Sprintf("ai_cache:token:%s", model)
		inputTokenStr, err := redisClient.HGet(ctx, tokenHashKey, "input_token").Result()
		if err != nil && err != redis.Nil {
			fmt.Printf("Failed to get input token for model %s: %v\n", model, err)
			continue
		}
		outputTokenStr, err := redisClient.HGet(ctx, tokenHashKey, "output_token").Result()
		if err != nil && err != redis.Nil {
			fmt.Printf("Failed to get output token for model %s: %v\n", model, err)
			continue
		}

		// 转换为int64
		inputToken, _ := strconv.ParseInt(inputTokenStr, 10, 64)
		outputToken, _ := strconv.ParseInt(outputTokenStr, 10, 64)

		// 2.2 读取缓存命中/未命中次数（String类型，自增计数器）
		hitCountStr, err := redisClient.Get(ctx, fmt.Sprintf("ai_cache:hit:%s", model)).Result()
		if err != nil && err != redis.Nil {
			fmt.Printf("Failed to get hit count for model %s: %v\n", model, err)
			hitCountStr = "0"
		}
		missCountStr, err := redisClient.Get(ctx, fmt.Sprintf("ai_cache:miss:%s", model)).Result()
		if err != nil && err != redis.Nil {
			fmt.Printf("Failed to get miss count for model %s: %v\n", model, err)
			missCountStr = "0"
		}

		hitCount, _ := strconv.ParseFloat(hitCountStr, 64)
		missCount, _ := strconv.ParseFloat(missCountStr, 64)

		// 2.3 计算缓存命中率（避免除零错误）
		var hitRate float64
		total := hitCount + missCount
		if total > 0 {
			hitRate = hitCount / total
		} else {
			hitRate = 0
		}

		// 步骤3：更新Prometheus Metric
		aiCachedInputTokenTotal.WithLabelValues(model).Set(float64(inputToken))
		aiCachedOutputTokenTotal.WithLabelValues(model).Set(float64(outputToken))
		aiCacheHitCount.WithLabelValues(model).Set(hitCount)
		aiCacheMissCount.WithLabelValues(model).Set(missCount)
		aiCacheHitRate.WithLabelValues(model).Set(hitRate)
	}

	fmt.Println("Metrics updated successfully")
}
