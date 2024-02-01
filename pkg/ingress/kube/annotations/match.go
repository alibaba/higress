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
	"fmt"
	"strings"

	. "github.com/alibaba/higress/pkg/ingress/log"
	networking "istio.io/api/networking/v1alpha3"
)

const (
	exact             = "exact"
	regex             = "regex"
	prefix            = "prefix"
	MatchMethod       = "match-method"
	MatchQuery        = "match-query"
	MatchHeader       = "match-header"
	MatchPseudoHeader = "match-pseudo-header"
	sep               = " "
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

	if config.Match.Headers, err = m.matchByHeaderOrQueryParma(annotations, MatchHeader, config.Match.Headers); err != nil {
		IngressLog.Errorf("parse headers error %v within ingress %s/%s", err, config.Namespace, config.Name)
	}

	var pseudoHeaderMatches map[string]map[string]string
	if pseudoHeaderMatches, err = m.matchByHeaderOrQueryParma(annotations, MatchPseudoHeader, pseudoHeaderMatches); err != nil {
		IngressLog.Errorf("parse headers error %v within ingress %s/%s", err, config.Namespace, config.Name)
	}
	if pseudoHeaderMatches != nil && len(pseudoHeaderMatches) > 0 {
		if config.Match.Headers == nil {
			config.Match.Headers = make(map[string]map[string]string)
		}
		for typ, mmap := range pseudoHeaderMatches {
			if config.Match.Headers[typ] == nil {
				config.Match.Headers[typ] = make(map[string]string)
			}
			for k, v := range mmap {
				config.Match.Headers[typ][":"+k] = v
			}
		}
	}

	if config.Match.QueryParams, err = m.matchByHeaderOrQueryParma(annotations, MatchQuery, config.Match.QueryParams); err != nil {
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
			IngressLog.Debug(fmt.Sprintf("match :%v, methods %v", route.Match[i].Name, route.Match[i].Method))
		}
	}

	// apply route for headers
	if config.Headers != nil {
		for i := 0; i < len(route.Match); i++ {
			if route.Match[i].Headers == nil {
				route.Match[i].Headers = map[string]*networking.StringMatch{}
			}
			addHeadersMatch(route.Match[i].Headers, config)
			IngressLog.Debug(fmt.Sprintf("match headers: %v, headers: %v", route.Match[i].Name, route.Match[i].Headers))
		}
	}

	if config.QueryParams != nil {
		for i := 0; i < len(route.Match); i++ {
			if route.Match[i].QueryParams == nil {
				route.Match[i].QueryParams = map[string]*networking.StringMatch{}
			}
			addQueryParamsMatch(route.Match[i].QueryParams, config)
			IngressLog.Debug(fmt.Sprintf("match : %v, queryParams: %v", route.Match[i].Name, route.Match[i].QueryParams))
		}
	}
}

func (m match) matchByMethod(annotations Annotations, ingress *Ingress) error {
	if !annotations.HasHigress(MatchMethod) {
		return nil
	}

	config := ingress.Match
	str, err := annotations.ParseStringForHigress(MatchMethod)
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

// matchByHeader to parse annotations to find MatchHeader config
func (m match) matchByHeaderOrQueryParma(annotations Annotations, key string, mmap map[string]map[string]string) (map[string]map[string]string, error) {
	for k, v := range annotations {
		if idx := strings.Index(k, key); idx != -1 {
			if mmap == nil {
				mmap = make(map[string]map[string]string)
			}
			if err := m.doMatch(k, v, mmap, idx+len(key)+1); err != nil {
				IngressLog.Errorf("matchByHeader() failed, the key: %v, value : %v, start: %d", k, v, idx+len(key)+1)
				return mmap, err
			}
		}
	}
	return mmap, nil
}

func (m match) doMatch(k, v string, mmap map[string]map[string]string, start int) error {
	if start >= len(k) {
		return ErrInvalidAnnotationName
	}

	var (
		idx      int
		legalIdx = len(HigressAnnotationsPrefix + "/") // the key has a higress prefix
	)

	// if idx == -1, it means don't have  exact|regex|prefix
	// if idx > legalIdx, it means the user key also has exact|regex|prefix. we just match the first one
	if idx = strings.Index(k, exact); idx == legalIdx {
		if mmap[exact] == nil {
			mmap[exact] = make(map[string]string)
		}
		mmap[exact][k[start:]] = v
		return nil
	}
	if idx = strings.Index(k, regex); idx == legalIdx {
		if mmap[regex] == nil {
			mmap[regex] = make(map[string]string)
		}
		mmap[regex][k[start:]] = v
		return nil
	}
	if idx = strings.Index(k, prefix); idx == legalIdx {
		if mmap[prefix] == nil {
			mmap[prefix] = make(map[string]string)
		}
		mmap[prefix][k[start:]] = v
		return nil
	}

	return ErrInvalidAnnotationName
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
	if m1 == nil {
		return
	}
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
				IngressLog.Errorf("unknown type: %q is not supported Match type", typ)
			}
		}

	}
}
