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

package cache

import (
	"fmt"
	"runtime"
	"sync"
)

type memoryCache struct {
	cache    map[string][]byte
	limit    int
	lock     sync.RWMutex
	memStats runtime.MemStats
}

func NewMemoryCache(limit int) (Cache, error) {
	return &memoryCache{
		cache:    make(map[string][]byte),
		lock:     sync.RWMutex{},
		limit:    limit,
		memStats: runtime.MemStats{},
	}, nil
}

func (c *memoryCache) Get(key string) ([]byte, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	val, ok := c.cache[key]
	return val, ok
}

func (c *memoryCache) Set(key string, value []byte) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	runtime.ReadMemStats(&c.memStats)
	for c.memStats.Alloc+uint64(len(value)) > uint64(c.limit) {
		return fmt.Errorf("memory limit exceeded")
	}
	c.cache[key] = value
	return nil
}

func (c *memoryCache) Delete(key string) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.cache, key)
	return nil
}
