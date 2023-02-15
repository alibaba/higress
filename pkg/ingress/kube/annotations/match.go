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
	"strings"

	. "github.com/alibaba/higress/pkg/ingress/log"
	networking "istio.io/api/networking/v1alpha3"
)

const (
	exact       = "exact"
	regex       = "regex"
	prefix      = "prefix"
	matchMethod = "match-method"
	matchQuery  = "match-query"
	matchHeader = "match-header"
	sep         = " "
)

var (
	methodList = []string{"GET", "HEAD", "POST", "PUT", "DELETE", "CONNECT", "OPTIONS", "TRACE", "PATCH"}
	methodMap  map[string]struct{}
)

type match struct{}

type MatchConfig struct {
	Methods     []string
	Headers     map[string]map[string]string
	QueryParams map[string]map[string]string
}

func (m match) Parse(annotations Annotations, config *Ingress, _ *GlobalContext) (err error) {
	config.Match = &MatchConfig{}

	if err = m.matchByMethod(annotations, config); err != nil {
		IngressLog.Errorf("parse methods error %v within ingress %s/%s", err, config.Namespace, config.Name)
	}

	if err = m.matchByHeader(annotations, config); err != nil {
		IngressLog.Errorf("parse headers error %v within ingress %s/%s", err, config.Namespace, config.Name)
	}

	if err = m.matchByUrlParam(annotations, config); err != nil {
		IngressLog.Errorf("parse query params error %v within ingress %s/%s", err, config.Namespace, config.Name)
	}

	return
}

func (m match) ApplyRoute(route *networking.HTTPRoute, ingressCfg *Ingress) {
	// apply route for method
	config := ingressCfg.Match
	if config.Methods != nil {
		for i := 0; i < len(route.Match); i++ {
			route.Match[i].Method = createMethodMatch(config.Methods...)
		}
	}

	// apply route for headers
	if config.Headers != nil {
		for i := 0; i < len(route.Match); i++ {
			addHeadersMatch(route.Match[i].Headers, config)
		}
	}

	if config.QueryParams != nil {
		for i := 0; i < len(route.Match); i++ {
			addQueryParamsMatch(route.Match[i].QueryParams, config)
		}
	}
}

func (m match) matchByMethod(annotations Annotations, ingress *Ingress) error {
	if !annotations.HasASAP(matchMethod) {
		return nil
	}

	config := ingress.Match
	str, err := annotations.ParseStringASAP(matchMethod)
	if err != nil {
		return err
	}

	methods := strings.Split(str, sep)
	set := make(map[string]struct{})
	for i := 0; i < len(methods); i++ {
		t := strings.ToUpper(methods[i])
		if _, ok := set[t]; !ok && isMethod(t) {
			set[t] = struct{}{}
			config.Methods = append(config.Methods, t)
		}
	}

	return nil
}

func (m match) matchByHeader(annotations Annotations, config *Ingress) error {
	for k, v := range annotations {
		if idx := strings.Index(k, matchHeader); idx != -1 {
			if config.Match.Headers == nil {
				config.Match.Headers = make(map[string]map[string]string)
			}
			if err := m.doMatchHeader(k, v, config, idx+len(matchHeader)+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m match) matchByUrlParam(annotations Annotations, config *Ingress) error {
	for k, v := range annotations {
		if idx := strings.Index(k, matchQuery); idx != -1 {
			if config.Match.QueryParams == nil {
				config.Match.QueryParams = make(map[string]map[string]string)
			}
			if err := m.doMatchQuery(k, v, config, idx+len(matchQuery)+1); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m match) doMatchHeader(k, v string, ingress *Ingress, start int) error {
	config := ingress.Match
	if start >= len(k) {
		return ErrInvalidAnnotationName
	}

	if idx := strings.Index(k, exact); idx == 0 { // if idx > 0, it means the "k" has the keywords
		if config.Headers[exact] == nil {
			config.Headers[exact] = make(map[string]string)
		}
		config.Headers[exact][k[start:]] = v
		return nil
	}
	if idx := strings.Index(k, regex); idx == 0 {
		if config.Headers[regex] == nil {
			config.Headers[regex] = make(map[string]string)
		}
		config.Headers[regex][k[start:]] = v
		return nil
	}
	if idx := strings.Index(k, prefix); idx == 0 {
		if config.Headers[prefix] == nil {
			config.Headers[prefix] = make(map[string]string)
		}
		config.Headers[prefix][k[start:]] = v
		return nil
	}

	return ErrInvalidAnnotationName
}

func (m match) doMatchQuery(k, v string, ingress *Ingress, start int) error {
	config := ingress.Match
	if start >= len(k) {
		return ErrInvalidAnnotationName
	}

	if idx := strings.Index(k, exact); idx == 0 {
		if config.QueryParams[exact] == nil {
			config.QueryParams[exact] = make(map[string]string)
		}
		config.QueryParams[exact][k[start:]] = v
	}
	if idx := strings.Index(k, regex); idx == 0 {
		if config.QueryParams[regex] == nil {
			config.QueryParams[regex] = make(map[string]string)
		}
		config.QueryParams[regex][k[start:]] = v
	}
	if idx := strings.Index(k, prefix); idx == 0 {
		if config.QueryParams[prefix] == nil {
			config.QueryParams[prefix] = make(map[string]string)
		}
		config.QueryParams[prefix][k[start:]] = v
	}
	return nil
}

func isMethod(s string) bool {
	if methodMap == nil || len(methodMap) == 0 {
		methodMap = make(map[string]struct{})
		for _, v := range methodList {
			methodMap[v] = struct{}{}
		}
	}

	_, ok := methodMap[s]
	return ok
}

func createMethodMatch(methods ...string) *networking.StringMatch {
	var sb strings.Builder
	for i := 0; i < len(methods); i++ {
		if i != 0 {
			sb.WriteString("|")
		}
		sb.WriteString(methods[i])
	}

	return &networking.StringMatch{
		MatchType: &networking.StringMatch_Regex{
			Regex: sb.String(),
		},
	}
}

func addHeadersMatch(headers map[string]*networking.StringMatch, config *MatchConfig) {
	merge(headers, config.Headers)
}

func addQueryParamsMatch(params map[string]*networking.StringMatch, config *MatchConfig) {
	merge(params, config.QueryParams)
}

// merge m2 to m1
func merge(m1 map[string]*networking.StringMatch, m2 map[string]map[string]string) {
	for typ, mmap := range m2 {
		for k, v := range mmap {
			switch typ {
			case exact:
				if _, ok := m1[k]; !ok {
					m1[k] = &networking.StringMatch{
						MatchType: &networking.StringMatch_Exact{
							Exact: v,
						},
					}
				}
			case prefix:
				if _, ok := m1[k]; !ok {
					m1[k] = &networking.StringMatch{
						MatchType: &networking.StringMatch_Prefix{
							Prefix: v,
						},
					}
				}
			case regex:
				if _, ok := m1[k]; !ok {
					m1[k] = &networking.StringMatch{
						MatchType: &networking.StringMatch_Regex{
							Regex: v,
						},
					}
				}
			default:
				IngressLog.Errorf("unknown type: %q is not supported HeaderMatch type", typ)
			}
		}

	}
}
