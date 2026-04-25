// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/higress-group/wasm-go/pkg/log"
	"github.com/tidwall/gjson"

	"github.com/higress-group/wasm-go/pkg/wrapper"
)

func main() {}

func init() {
	wrapper.SetCtx(
		"rate-limit-advanced",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

// RateLimitConfig holds the plugin configuration.
type RateLimitConfig struct {
	maxRequests    int64
	windowSize     time.Duration
	limitByUser    bool
	limitByIP      bool
	userHeaderName string
	responseCode   uint32
	responseBody   string
	retryAfterSec  int64
	enableBurst    bool
	burstCapacity  int64
	burstRate      float64
	counters       *CounterStore
}

// CounterStore manages sliding window counters for rate limiting.
type CounterStore struct {
	mu       sync.Mutex
	counters map[string]*SlidingWindowCounter
}

// SlidingWindowCounter implements a sliding window rate limiter.
type SlidingWindowCounter struct {
	timestamps []int64
	maxSize    int64
	windowMs   int64
}

// TokenBucket implements a token bucket rate limiter as fallback.
type TokenBucket struct {
	tokens     float64
	capacity   float64
	rate       float64
	lastRefill int64
}

// NewCounterStore creates a new counter store.
func NewCounterStore() *CounterStore {
	return &CounterStore{
		counters: make(map[string]*SlidingWindowCounter),
	}
}

// NewSlidingWindowCounter creates a new sliding window counter.
func NewSlidingWindowCounter(maxRequests int64, windowMs int64) *SlidingWindowCounter {
	return &SlidingWindowCounter{
		timestamps: make([]int64, 0, maxRequests),
		maxSize:    maxRequests,
		windowMs:   windowMs,
	}
}

// NewTokenBucket creates a new token bucket.
func NewTokenBucket(capacity float64, rate float64) *TokenBucket {
	return &TokenBucket{
		tokens:     capacity,
		capacity:   capacity,
		rate:       rate,
		lastRefill: time.Now().UnixMilli(),
	}
}

// Allow checks if a request is allowed under the sliding window algorithm.
func (s *SlidingWindowCounter) Allow(nowMs int64) bool {
	cutoff := nowMs - s.windowMs

	// Remove expired timestamps
	validStart := 0
	for validStart < len(s.timestamps) && s.timestamps[validStart] <= cutoff {
		validStart++
	}
	if validStart > 0 {
		s.timestamps = s.timestamps[validStart:]
	}

	if int64(len(s.timestamps)) >= s.maxSize {
		return false
	}

	s.timestamps = append(s.timestamps, nowMs)
	return true
}

// Count returns the current number of requests in the window.
func (s *SlidingWindowCounter) Count(nowMs int64) int64 {
	cutoff := nowMs - s.windowMs
	count := int64(0)
	for _, ts := range s.timestamps {
		if ts > cutoff {
			count++
		}
	}
	return count
}

// Reset clears all timestamps.
func (s *SlidingWindowCounter) Reset() {
	s.timestamps = s.timestamps[:0]
}

// Allow checks if a request is allowed under the token bucket algorithm.
func (t *TokenBucket) Allow(nowMs int64) bool {
	elapsed := float64(nowMs-t.lastRefill) / 1000.0
	t.tokens += elapsed * t.rate
	if t.tokens > t.capacity {
		t.tokens = t.capacity
	}
	t.lastRefill = nowMs

	if t.tokens >= 1.0 {
		t.tokens -= 1.0
		return true
	}
	return false
}

// RemainingTokens returns the current token count.
func (t *TokenBucket) RemainingTokens() float64 {
	return t.tokens
}

// GetOrCreate returns an existing counter or creates a new one.
func (cs *CounterStore) GetOrCreate(key string, maxRequests int64, windowMs int64) *SlidingWindowCounter {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if counter, ok := cs.counters[key]; ok {
		return counter
	}
	counter := NewSlidingWindowCounter(maxRequests, windowMs)
	cs.counters[key] = counter
	return counter
}

// Cleanup removes expired counters.
func (cs *CounterStore) Cleanup(nowMs int64) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	for key, counter := range cs.counters {
		if counter.Count(nowMs) == 0 {
			delete(cs.counters, key)
		}
	}
}

// Size returns the number of tracked keys.
func (cs *CounterStore) Size() int {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return len(cs.counters)
}

func parseConfig(json gjson.Result, config *RateLimitConfig, log log.Log) error {
	maxReq := json.Get("max_requests").Int()
	if maxReq <= 0 {
		maxReq = 100
	}
	config.maxRequests = maxReq

	windowSec := json.Get("window_size_seconds").Int()
	if windowSec <= 0 {
		windowSec = 60
	}
	config.windowSize = time.Duration(windowSec) * time.Second

	config.limitByUser = json.Get("limit_by_user").Bool()
	config.limitByIP = json.Get("limit_by_ip").Bool()

	if !config.limitByUser && !config.limitByIP {
		config.limitByIP = true
	}

	userHeader := json.Get("user_header_name").String()
	if userHeader == "" {
		userHeader = "X-User-ID"
	}
	config.userHeaderName = userHeader

	respCode := json.Get("response_code").Int()
	if respCode <= 0 || respCode >= 600 {
		respCode = 429
	}
	config.responseCode = uint32(respCode)

	respBody := json.Get("response_body").String()
	if respBody == "" {
		respBody = "Rate limit exceeded. Please retry later."
	}
	config.responseBody = respBody

	retryAfter := json.Get("retry_after_seconds").Int()
	if retryAfter <= 0 {
		retryAfter = windowSec
	}
	config.retryAfterSec = retryAfter

	config.enableBurst = json.Get("enable_burst").Bool()

	burstCap := json.Get("burst_capacity").Int()
	if burstCap <= 0 {
		burstCap = maxReq * 2
	}
	config.burstCapacity = burstCap

	burstRate := json.Get("burst_rate").Float()
	if burstRate <= 0 {
		burstRate = float64(maxReq) / float64(windowSec)
	}
	config.burstRate = burstRate

	config.counters = NewCounterStore()

	log.Infof("[rate-limit-advanced] Config loaded: maxRequests=%d, window=%ds, limitByUser=%v, limitByIP=%v",
		config.maxRequests, windowSec, config.limitByUser, config.limitByIP)

	return nil
}

func extractClientKey(config RateLimitConfig, log log.Log) string {
	var key string

	if config.limitByUser {
		userID, err := proxywasm.GetHttpRequestHeader(config.userHeaderName)
		if err == nil && userID != "" {
			key = "user:" + userID
			return key
		}
	}

	if config.limitByIP {
		ip, err := proxywasm.GetHttpRequestHeader("x-forwarded-for")
		if err == nil && ip != "" {
			key = "ip:" + ip
			return key
		}
		ip, err = proxywasm.GetHttpRequestHeader("x-real-ip")
		if err == nil && ip != "" {
			key = "ip:" + ip
			return key
		}
	}

	key = "global"
	return key
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config RateLimitConfig, log log.Log) types.Action {
	clientKey := extractClientKey(config, log)
	nowMs := time.Now().UnixMilli()

	counter := config.counters.GetOrCreate(clientKey, config.maxRequests,
		config.windowSize.Milliseconds())

	config.counters.mu.Lock()
	allowed := counter.Allow(nowMs)
	remaining := config.maxRequests - counter.Count(nowMs)
	config.counters.mu.Unlock()

	if remaining < 0 {
		remaining = 0
	}

	if !allowed {
		if config.enableBurst {
			bucket := NewTokenBucket(float64(config.burstCapacity), config.burstRate)
			if bucket.Allow(nowMs) {
				log.Infof("[rate-limit-advanced] Burst allowed for key=%s", clientKey)
				setRateLimitHeaders(remaining, config)
				return types.ActionContinue
			}
		}

		log.Infof("[rate-limit-advanced] Rate limit exceeded for key=%s", clientKey)
		headers := [][2]string{
			{"X-RateLimit-Limit", fmt.Sprintf("%d", config.maxRequests)},
			{"X-RateLimit-Remaining", "0"},
			{"X-RateLimit-Reset", fmt.Sprintf("%d", config.retryAfterSec)},
			{"Retry-After", fmt.Sprintf("%d", config.retryAfterSec)},
		}
		proxywasm.SendHttpResponseWithDetail(config.responseCode,
			"rate-limit-advanced.rate_limited", headers,
			[]byte(config.responseBody), -1)
		return types.ActionContinue
	}

	setRateLimitHeaders(remaining, config)
	return types.ActionContinue
}

func setRateLimitHeaders(remaining int64, config RateLimitConfig) {
	_ = proxywasm.AddHttpRequestHeader("X-RateLimit-Limit",
		fmt.Sprintf("%d", config.maxRequests))
	_ = proxywasm.AddHttpRequestHeader("X-RateLimit-Remaining",
		fmt.Sprintf("%d", remaining))
}
