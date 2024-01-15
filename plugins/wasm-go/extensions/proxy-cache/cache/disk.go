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
	"github.com/boltdb/bolt"
	"time"
)

const (
	DefaultBucket = "default"
)

type diskCache struct {
	db  *bolt.DB
	ttl int
}

type DiskCacheOptions struct {
	RootDir     string
	DiskLimit   int
	MemoryLimit int
	TTL         int
}

func NewDiskCache(opt DiskCacheOptions) (Cache, error) {
	options := bolt.DefaultOptions
	options.Timeout = time.Duration(opt.TTL) * time.Second
	db, err := bolt.Open(opt.RootDir, 0600, options)
	if err != nil {
		return nil, err
	}
	// create bucket
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(DefaultBucket))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &diskCache{
		db: db,
	}, nil
}

func (c *diskCache) Get(key string) ([]byte, bool) {
	var value []byte
	err := c.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(DefaultBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", DefaultBucket)
		}
		value = bucket.Get([]byte(key))
		return nil
	})
	if err != nil {
		return nil, false
	}
	return value, true
}

func (c *diskCache) Set(key string, value []byte) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(DefaultBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", DefaultBucket)
		}
		return bucket.Put([]byte(key), value)
	})
}

func (c *diskCache) Delete(key string) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(DefaultBucket))
		if bucket == nil {
			return fmt.Errorf("bucket %s not found", DefaultBucket)
		}
		return bucket.Delete([]byte(key))
	})
}

func (c *diskCache) Clean() error {
	return c.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(DefaultBucket))
	})
}
