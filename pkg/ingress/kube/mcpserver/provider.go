// Copyright (c) 2025 Alibaba Group Holding Ltd.
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

package mcpserver

import (
	"reflect"
	"slices"
	"strings"
	"sync"
)

type McpServerProvider interface {
	GetMcpServers() []*McpServer
}

type McpRouteProviderAware interface {
	RegisterMcpServerProvider(provider McpServerProvider)
}

type McpServerCache struct {
	mcpServers []*McpServer
	mutex      sync.RWMutex
}

func (c *McpServerCache) GetMcpServers() []*McpServer {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.mcpServers
}

// SetMcpServers sets the mcp servers and returns true if the cached list is changed
func (c *McpServerCache) SetMcpServers(mcpServers []*McpServer) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	sortedMcpServers := make([]*McpServer, 0, len(mcpServers))
	sortedMcpServers = append(sortedMcpServers, mcpServers...)
	// Sort the mcp servers by PathMatchValue in descending order
	slices.SortFunc(sortedMcpServers, func(a, b *McpServer) int {
		return -strings.Compare(a.PathMatchValue, b.PathMatchValue)
	})

	if len(c.mcpServers) == len(mcpServers) {
		changed := false
		for i := range c.mcpServers {
			if !reflect.DeepEqual(c.mcpServers[i], mcpServers[i]) {
				changed = true
				break
			}
		}
		if !changed {
			return false
		}
	}

	c.mcpServers = sortedMcpServers
	return true
}
