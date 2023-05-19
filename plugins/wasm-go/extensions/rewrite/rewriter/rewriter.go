package rewriter

import (
	"regexp"
	"strings"
)

type Rewriter struct {
	hostMatchers []*HostMatcher
	pathMatchers []*PathMatcher
	rewriteHost  string
	rewritePath  string
}

type HostMatcher struct {
	matchType MatchHostType
	host      string
}

type PathMatcher struct {
	matchType MatchPathType
	path      string
	reg       *regexp.Regexp
}

type MatchHostType int

const (
	HostPrefix MatchHostType = iota
	HostSuffix
	HostExact
	HostUnknown
)

type MatchPathType int

const (
	PathPrefix MatchPathType = iota
	PathExact
	PathRegex
	PathUnknown
)

func NewRewriter(hostNum, pathNum int, rewriteHost, rewritePath string) *Rewriter {
	return &Rewriter{
		hostMatchers: make([]*HostMatcher, 0, hostNum),
		pathMatchers: make([]*PathMatcher, 0, pathNum),
		rewriteHost:  rewriteHost,
		rewritePath:  rewritePath,
	}
}

func (r *Rewriter) AppendHostMatcher(matchType MatchHostType, host string) {
	r.hostMatchers = append(r.hostMatchers, &HostMatcher{
		matchType: matchType,
		host:      host,
	})
}

func (r *Rewriter) AppendPathMatcher(matchType MatchPathType, path, pattern string) {
	r.pathMatchers = append(r.pathMatchers, &PathMatcher{
		matchType: matchType,
		path:      path,
		reg:       regexp.MustCompile(pattern),
	})
}

func (r Rewriter) MatchAndRewrite(reqHost, reqPath string) (matched bool, rewriteHost, rewritePath string) {
	var hostMatched, pathMatched bool
	for _, hm := range r.hostMatchers {
		if hm.match(reqHost) {
			hostMatched = true
			rewriteHost = r.rewriteHost
			break
		}
	}
	if !hostMatched {
		return
	}

	for _, pm := range r.pathMatchers {
		if pm.match(reqPath) {
			pathMatched = true
			if pm.matchType == PathRegex {
				// e.g.
				// if:
				//   regexPattern = "/v1/(app)"
				//   reqPath = "/v1/app"
				//   r.rewritePath = "/$1"
				// then:
				//   rewritePath = "/app"
				rewritePath = pm.reg.ReplaceAllString(reqPath, r.rewritePath)
			} else {
				rewritePath = r.rewritePath
			}
			break
		}
	}
	if pathMatched {
		matched = true
		return
	}

	return
}

func (hm HostMatcher) match(reqHost string) bool {
	switch hm.matchType {
	case HostPrefix:
		return strings.HasPrefix(reqHost, hm.host)
	case HostSuffix:
		return strings.HasSuffix(reqHost, hm.host)
	case HostExact:
		return reqHost == hm.host
	}
	return false
}

func (pm PathMatcher) match(reqPath string) bool {
	switch pm.matchType {
	case PathPrefix:
		return strings.HasPrefix(reqPath, pm.path)
	case PathExact:
		return reqPath == pm.path
	case PathRegex:
		if ok := pm.reg.MatchString(reqPath); ok {
			return true
		}
	}
	return false
}
