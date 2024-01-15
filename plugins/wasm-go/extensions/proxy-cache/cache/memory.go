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

	"github.com/coocood/freecache"
)

type memoryCache struct {
	cache *freecache.Cache
	ttl   int
}

func NewMemoryCache(limit int, ttl int) (Cache, error) {
	cache := freecache.NewCache(limit)
	return &memoryCache{
		cache: cache,
		ttl:   ttl,
	}, nil
}

func (c *memoryCache) Get(key string) ([]byte, bool) {
	bytes, err := c.cache.Get([]byte(key))
	if err != nil {
		return nil, false
	}
	return bytes, true
}

func (c *memoryCache) Set(key string, value []byte) error {
	return c.cache.Set([]byte(key), value, c.ttl)
}

func (c *memoryCache) Delete(key string) error {
	ok := c.cache.Del([]byte(key))
	if !ok {
		return fmt.Errorf("delete cache failed")
	}
	return nil
}

func (c *memoryCache) Clean() error {
	c.cache.Clear()
	return nil
}
