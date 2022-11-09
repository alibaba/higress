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

package annotations

import (
	"regexp"
	"strings"

	networking "istio.io/api/networking/v1alpha3"
)

const (
	rewriteTarget = "rewrite-target"
	useRegex      = "use-regex"
	upstreamVhost = "upstream-vhost"

	re2Regex = "\\$[0-9]"
)

var (
	_ Parser       = &rewrite{}
	_ RouteHandler = &rewrite{}
)

type RewriteConfig struct {
	RewriteTarget string
	UseRegex      bool
	RewriteHost   string
}

type rewrite struct{}

func (r rewrite) Parse(annotations Annotations, config *Ingress, _ *GlobalContext) error {
	if !needRewriteConfig(annotations) {
		return nil
	}

	rewriteConfig := &RewriteConfig{}
	rewriteConfig.RewriteTarget, _ = annotations.ParseStringASAP(rewriteTarget)
	rewriteConfig.UseRegex, _ = annotations.ParseBoolASAP(useRegex)
	rewriteConfig.RewriteHost, _ = annotations.ParseStringASAP(upstreamVhost)

	if rewriteConfig.RewriteTarget != "" {
		// When rewrite target is present and not empty,
		// we will enforce regex match on all rules in this ingress.
		rewriteConfig.UseRegex = true

		// We should convert nginx regex rule to envoy regex rule.
		rewriteConfig.RewriteTarget = convertToRE2(rewriteConfig.RewriteTarget)
	}

	config.Rewrite = rewriteConfig
	return nil
}

func (r rewrite) ApplyRoute(route *networking.HTTPRoute, config *Ingress) {
	rewriteConfig := config.Rewrite
	if rewriteConfig == nil || (rewriteConfig.RewriteTarget == "" &&
		rewriteConfig.RewriteHost == "") {
		return
	}

	route.Rewrite = &networking.HTTPRewrite{}
	if rewriteConfig.RewriteTarget != "" {
		route.Rewrite.UriRegex = &networking.RegexMatchAndSubstitute{
			Pattern:      route.Match[0].Uri.GetRegex(),
			Substitution: rewriteConfig.RewriteTarget,
		}
	}

	if rewriteConfig.RewriteHost != "" {
		route.Rewrite.Authority = rewriteConfig.RewriteHost
	}
}

func convertToRE2(target string) string {
	if match, err := regexp.MatchString(re2Regex, target); err != nil || !match {
		return target
	}

	return strings.ReplaceAll(target, "$", "\\")
}

func NeedRegexMatch(annotations map[string]string) bool {
	target, _ := Annotations(annotations).ParseStringASAP(rewriteTarget)
	regex, _ := Annotations(annotations).ParseBoolASAP(useRegex)

	return regex || target != ""
}

func needRewriteConfig(annotations Annotations) bool {
	return annotations.HasASAP(rewriteTarget) || annotations.HasASAP(useRegex) ||
		annotations.HasASAP(upstreamVhost)
}
