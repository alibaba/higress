// Copyright (c) 2023 Alibaba Group Holding Ltd.
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

package annotations

import (
	"strings"

	"github.com/alibaba/higress/pkg/ingress/kube/mcpserver"
	"github.com/alibaba/higress/pkg/ingress/log"
)

const (
	enableMcpServer            = "mcp-server"
	mcpServerMatchRuleDomains  = "mcp-server-match-rule-domains"
	mcpServerMatchRuleType     = "mcp-server-match-rule-type"
	mcpServerMatchRuleValue    = "mcp-server-match-rule-value"
	mcpServerUpstreamType      = "mcp-server-upstream-type"
	mcpServerEnablePathRewrite = "mcp-server-enable-path-rewrite"
	mcpServerPathRewritePrefix = "mcp-server-path-rewrite-prefix"
)

// help to conform mcpServer implements method of Parse
var (
	_ Parser = &mcpServer{}
)

type mcpServer struct{}

func (a mcpServer) Parse(annotations Annotations, config *Ingress, globalContext *GlobalContext) error {
	if globalContext == nil {
		return nil
	}

	ingressKey := config.Namespace + "/" + config.Name

	enabled, _ := annotations.ParseBoolASAP(enableMcpServer)
	if !enabled {
		return nil
	}

	var matchRuleDomains []string
	rawMatchRuleDomains, _ := annotations.ParseStringASAP(mcpServerMatchRuleDomains)
	if rawMatchRuleDomains == "" || rawMatchRuleDomains == "*" {
		// Match all domains. Leave an empty slice.
	} else if strings.Contains(rawMatchRuleDomains, ",") {
		matchRuleDomains = strings.Split(rawMatchRuleDomains, ",")
	} else {
		matchRuleDomains = []string{rawMatchRuleDomains}
	}

	matchRuleType, _ := annotations.ParseStringASAP(mcpServerMatchRuleType)
	if matchRuleType == "" {
		log.IngressLog.Errorf("ingress %s: mcp-server-match-rule-path-type is empty", ingressKey)
	} else if !mcpserver.ValidPathMatchTypes[matchRuleType] {
		log.IngressLog.Errorf("ingress %s: mcp-server-match-rule-path-type %s is not supported", ingressKey, matchRuleType)
	}

	matchRuleValue, _ := annotations.ParseStringASAP(mcpServerMatchRuleValue)

	upstreamType, _ := annotations.ParseStringASAP(mcpServerUpstreamType)
	if upstreamType != "" && !mcpserver.ValidUpstreamTypes[upstreamType] {
		log.IngressLog.Errorf("mcp-server-upstream-type %s is not supported", upstreamType)
		return nil
	}

	enablePathRewrite, _ := annotations.ParseBoolASAP(mcpServerEnablePathRewrite)
	pathRewritePrefix, _ := annotations.ParseStringASAP(mcpServerPathRewritePrefix)

	globalContext.McpServers = append(globalContext.McpServers, &mcpserver.McpServer{
		Name:              ingressKey,
		Domains:           matchRuleDomains,
		PathMatchType:     matchRuleType,
		PathMatchValue:    matchRuleValue,
		UpstreamType:      upstreamType,
		EnablePathRewrite: enablePathRewrite,
		PathRewritePrefix: pathRewritePrefix,
	})

	return nil
}
