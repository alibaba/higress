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
	"istio.io/istio/pkg/config"
)

var (
	GvkMcpServer = config.GroupVersionKind{Group: "networking.higress.io", Version: "v1alpha1", Kind: "McpServer"}
)

const (
	UpstreamTypeRest       string = "rest"
	UpstreamTypeSSE        string = "sse"
	UpstreamTypeStreamable string = "streamable"

	ExactMatchType    string = "exact"
	PrefixMatchType   string = "prefix"
	SuffixMatchType   string = "suffix"
	ContainsMatchType string = "contains"
	RegexMatchType    string = "regex"
)

var (
	ValidUpstreamTypes = map[string]bool{
		UpstreamTypeRest:       true,
		UpstreamTypeSSE:        true,
		UpstreamTypeStreamable: true,
	}
	ValidPathMatchTypes = map[string]bool{
		ExactMatchType:    true,
		PrefixMatchType:   true,
		SuffixMatchType:   true,
		ContainsMatchType: true,
		RegexMatchType:    true,
	}
)

type McpServer struct {
	Name              string   `json:"name,omitempty"`
	Domains           []string `json:"domains,omitempty"`
	PathMatchType     string   `json:"path_match_type,omitempty"`
	PathMatchValue    string   `json:"path_match_value,omitempty"`
	UpstreamType      string   `json:"upstream_type,omitempty"`
	EnablePathRewrite bool     `json:"enable_path_rewrite,omitempty"`
	PathRewritePrefix string   `json:"path_rewrite_prefix,omitempty"`
}
