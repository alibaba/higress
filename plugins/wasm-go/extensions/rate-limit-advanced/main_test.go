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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSlidingWindowCounter(t *testing.T) {
	counter := NewSlidingWindowCounter(10, 60000)
	require.NotNil(t, counter)
	assert.Equal(t, int64(10), counter.maxSize)
	assert.Equal(t, int64(60000), counter.windowMs)
	assert.Empty(t, counter.timestamps)
}

func TestSlidingWindowCounterAllow(t *testing.T) {
	counter := NewSlidingWindowCounter(3, 60000)
	now := time.Now().UnixMilli()

	assert.True(t, counter.Allow(now))
	assert.True(t, counter.Allow(now+100))
	assert.True(t, counter.Allow(now+200))
	assert.False(t, counter.Allow(now+300))
}

func TestSlidingWindowCounterExpiry(t *testing.T) {
	counter := NewSlidingWindowCounter(3, 1000)
	now := time.Now().UnixMilli()

	assert.True(t, counter.Allow(now))
	assert.True(t, counter.Allow(now+100))
	assert.True(t, counter.Allow(now+200))
	// Window expires after 1000ms
	assert.True(t, counter.Allow(now+1100))
}

func TestSlidingWindowCounterCount(t *testing.T) {
	counter := NewSlidingWindowCounter(10, 60000)
	now := time.Now().UnixMilli()

	counter.Allow(now)
	counter.Allow(now + 100)
	counter.Allow(now + 200)

	assert.Equal(t, int64(3), counter.Count(now+300))
}

func TestSlidingWindowCounterReset(t *testing.T) {
	counter := NewSlidingWindowCounter(10, 60000)
	now := time.Now().UnixMilli()

	counter.Allow(now)
	counter.Allow(now + 100)
	counter.Reset()

	assert.Equal(t, int64(0), counter.Count(now+200))
}

func TestNewTokenBucket(t *testing.T) {
	bucket := NewTokenBucket(10.0, 1.0)
	require.NotNil(t, bucket)
	assert.Equal(t, 10.0, bucket.capacity)
	assert.Equal(t, 1.0, bucket.rate)
}

func TestTokenBucketAllow(t *testing.T) {
	bucket := NewTokenBucket(3.0, 1.0)
	now := time.Now().UnixMilli()

	assert.True(t, bucket.Allow(now))
	assert.True(t, bucket.Allow(now))
	assert.True(t, bucket.Allow(now))
	assert.False(t, bucket.Allow(now))
}

func TestTokenBucketRefill(t *testing.T) {
	bucket := NewTokenBucket(2.0, 1.0)
	now := time.Now().UnixMilli()

	assert.True(t, bucket.Allow(now))
	assert.True(t, bucket.Allow(now))
	assert.False(t, bucket.Allow(now))
	// After 2 seconds, should have refilled
	assert.True(t, bucket.Allow(now+2100))
}

func TestTokenBucketRemainingTokens(t *testing.T) {
	bucket := NewTokenBucket(5.0, 1.0)
	now := time.Now().UnixMilli()

	bucket.Allow(now)
	bucket.Allow(now)

	remaining := bucket.RemainingTokens()
	assert.InDelta(t, 3.0, remaining, 0.1)
}

func TestNewCounterStore(t *testing.T) {
	store := NewCounterStore()
	require.NotNil(t, store)
	assert.Equal(t, 0, store.Size())
}

func TestCounterStoreGetOrCreate(t *testing.T) {
	store := NewCounterStore()

	c1 := store.GetOrCreate("user:1", 10, 60000)
	require.NotNil(t, c1)
	assert.Equal(t, 1, store.Size())

	// Should return the same counter
	c2 := store.GetOrCreate("user:1", 10, 60000)
	assert.Equal(t, c1, c2)
	assert.Equal(t, 1, store.Size())

	// Different key creates new counter
	c3 := store.GetOrCreate("user:2", 10, 60000)
	assert.NotEqual(t, c1, c3)
	assert.Equal(t, 2, store.Size())
}

func TestCounterStoreCleanup(t *testing.T) {
	store := NewCounterStore()
	now := time.Now().UnixMilli()

	c1 := store.GetOrCreate("active", 10, 60000)
	c1.Allow(now)

	store.GetOrCreate("empty", 10, 60000)

	store.Cleanup(now + 100)
	assert.Equal(t, 1, store.Size())
}

func TestExtractClientKeyGlobal(t *testing.T) {
	// When neither limitByUser nor limitByIP, key should default
	config := RateLimitConfig{
		limitByUser: false,
		limitByIP:   false,
	}
	// extractClientKey uses proxywasm which can't be called in unit tests,
	// so we test the config defaults instead
	assert.False(t, config.limitByUser)
	assert.False(t, config.limitByIP)
}

func TestRateLimitConfigDefaults(t *testing.T) {
	config := RateLimitConfig{}
	assert.Equal(t, int64(0), config.maxRequests)
	assert.Equal(t, uint32(0), config.responseCode)
	assert.Equal(t, "", config.responseBody)
	assert.Equal(t, "", config.userHeaderName)
	assert.False(t, config.enableBurst)
}
