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

// Cache is the interface that wraps the basic Get, Set, Delete and Clean methods.
type Cache interface {
	// Get returns the value stored in the cache for a key, or nil if no value is present.
	Get(key string) ([]byte, bool)
	// Set stores a value for a key.
	Set(key string, value []byte) error
	// Delete deletes a value for a key.
	Delete(key string) error
	// Clean deletes all values in the cache.
	Clean() error
}
