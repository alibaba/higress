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

package main

import (
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/proxy-redirect/proxy_redirect"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

const (
	name = "proxy-redirect"

	defaultRedirectRule = "default"

	httpStatusCodeTag = "http_status_code"
	proxyRedirectTag  = "proxy_redirect"

	locationKey          = "location"
	statusKey            = ":status"
	originalSchemeCtxKey = "proxy-redirect-original/scheme"
	originalHostCtxKey   = "proxy-redirect-original/authority"
	originalPathCtxKey   = "proxy-redirect-original/path"
)

func main() {
	wrapper.SetCtx(
		name,
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
	)
}

type Config struct {
	HTTPStatusCode []uint64 `json:"http_status_code"`
	ProxyRedirect  []string `json:"proxy_redirect"`
	proxyRedirect  [][2]string
}

func parseConfig(json gjson.Result, config *Config, log wrapper.Log) error {
	for _, item := range json.Get(httpStatusCodeTag).Array() {
		config.HTTPStatusCode = append(config.HTTPStatusCode, item.Uint())
	}
	for _, item := range json.Get(proxyRedirectTag).Array() {
		s := strings.TrimSpace(item.String())
		if s == defaultRedirectRule {
			config.ProxyRedirect = append(config.ProxyRedirect, defaultRedirectRule)
			config.proxyRedirect = append(config.proxyRedirect, [2]string{defaultRedirectRule})
			continue
		}
		list := strings.Split(s, " ")
		if len(list) < 2 {
			return errors.Errorf("invalid proxy_redirect: %s", item.String())
		}
		config.ProxyRedirect = append(config.ProxyRedirect, strings.Join([]string{list[0], list[len(list)-1]}, " "))
		config.proxyRedirect = append(config.proxyRedirect, [2]string{list[0], list[len(list)-1]})
	}
	if len(config.HTTPStatusCode) == 0 {
		config.HTTPStatusCode = []uint64{http.StatusCreated, http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect}
	}
	if len(config.ProxyRedirect) == 0 {
		config.ProxyRedirect = []string{defaultRedirectRule}
		config.proxyRedirect = [][2]string{{defaultRedirectRule}}
	}
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config Config, log wrapper.Log) types.Action {
	for k, v := range map[string]string{
		originalSchemeCtxKey: ctx.Scheme(),
		originalHostCtxKey:   ctx.Host(),
		originalPathCtxKey:   ctx.Path(),
	} {
		if v == "" {
			var err error
			proxywasm.LogWarnf("empty %s got onHttpRequestHeaders\n", path.Base(k))
			v, err = proxywasm.GetHttpRequestHeader(":" + path.Base(k))
			if err != nil {
				proxywasm.LogErrorf("failure to GetHttpRequestHeader %s: %v\n", ":"+path.Base(k), err)
			}
		}
		ctx.SetContext(k, v)
	}

	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config Config, log wrapper.Log) types.Action {
	// Only handle the http status code in the configuration list
	statusCode, err := proxywasm.GetHttpResponseHeader(statusKey)
	if err != nil {
		log.Errorf("failure to GetHttpResponseHeader %s", strconv.Quote(statusKey))
		return types.ActionContinue
	}
	var handleIt = false
	for _, code := range config.HTTPStatusCode {
		if handleIt = strconv.FormatUint(code, 10) == statusCode; handleIt {
			break
		}
	}
	if !handleIt {
		return types.ActionContinue
	}

	// Only handle the case if the http response header "location" can be fetched
	location, err := proxywasm.GetHttpResponseHeader(locationKey)
	if err != nil {
		log.Errorf("failure to GetHttpResponseHeader %s", strconv.Quote(locationKey))
		return types.ActionContinue
	}
	if location == "" {
		return types.ActionContinue
	}

	loc, err := url.Parse(location)
	if err != nil {
		log.Errorf("failure to Parse %s", location)
		return types.ActionContinue
	}

	originalScheme := ctx.GetContext(originalSchemeCtxKey).(string)
	originalHost := ctx.GetContext(originalHostCtxKey).(string)
	originalPath := ctx.GetContext(originalPathCtxKey).(string)
	origin, err := url.Parse(originalPath)
	if err != nil {
		log.Errorf("failure to Parse %s", originalPath)
		return types.ActionContinue
	}
	origin.Scheme = originalScheme
	origin.Host = originalHost

	var ctxPath = ctx.Path()
	var path_ = []string{"request"}
	for _, child := range []string{
		"path",
		"url_path",
		"host",
		"scheme",
		"method",
		"headers",
		"referer",
		"useragent",
		"time",
		"id",
		"protocol",
		"query",
	} {
		path_ := append(path_, child)
		property, err := proxywasm.GetProperty(path_)
		if err != nil {
			log.Errorf("failure to GetProperty %s\n", strings.Join(path_, "."))
		}
		log.Infof("%s: %s\n", strings.Join(path_, "."), string(property))
	}

	if ctxPath == "" {
		var err error
		if ctxPath, err = proxywasm.GetHttpResponseHeader(":path"); err != nil {
			proxywasm.LogErrorf("failure to GetHttpResponseHeader :path, err: %v\n", err)
			return types.ActionContinue
		}
	}
	upstream, err := url.Parse(ctxPath)
	if err != nil {
		log.Errorf("failure to Parse %s", ctxPath)
		return types.ActionContinue
	}

	for i := 0; i < len(config.ProxyRedirect); i++ {
		// if the redirect_proxy's redirect is "default"
		if config.ProxyRedirect[i] == defaultRedirectRule {
			action, ok, err := handleDefault(loc, origin, upstream)
			if err != nil {
				log.Error(err.Error())
				return action
			}
			if !ok {
				continue
			}
			return action
		}

		// if the redirect_proxy's redirect is regular expressions
		if strings.HasPrefix(config.proxyRedirect[i][0], "~") {
			action, ok, err := handleRegex(config.proxyRedirect[i][0], location, config.proxyRedirect[i][1])
			if err != nil {
				log.Errorf("failure to handleRegex: %v", err)
			}
			if !ok {
				continue
			}
			return action
		}

		// if the location matches the redirect_proxy's redirect text exactly
		if loc.String() == config.proxyRedirect[i][0] {
			if err := resetResponseHeaderLocation(config.proxyRedirect[i][1]); err != nil {
				log.Errorf("failure to resetResponseHeaderLocation: %v", err)
			}
			return types.ActionContinue
		}
	}

	log.Warnf("no proxy_redirect applied for Location: %s", location)
	return types.ActionContinue
}

func handleDefault(loc, origin, upstream *url.URL) (types.Action, bool, error) {
	// if the location host is some third host, do not reset the header location
	if loc.Host != "" && loc.Host != origin.Host && loc.Host != upstream.Host {
		return types.ActionContinue, true, nil
	}
	if loc.Host == origin.Host {
		return types.ActionContinue, true, nil
	}

	// If the original path is exactly the same as the upstream path, don't modify the path,
	// just modify the scheme and host of the location to be consistent with the origin
	if origin.Path == upstream.Path {
		if err := resetResponseHeaderLocation(loc.Path); err != nil {
			return 0, false, errors.Wrapf(err, "failure to resetResponseHeaderLocation %s", loc.String())
		}
		return types.ActionContinue, true, nil
	}

	// speculate on prefix matching rule and rewrite rule, reverse rewrite the header location
	prefix, rewrite, ok := proxy_redirect.SpeculatePrefixRewrite(origin.Path, upstream.Path)
	if !ok {
		return types.ActionContinue, false, nil
	}
	if !strings.HasPrefix(loc.Path, rewrite) {
		return types.ActionContinue, false, nil
	}
	loc.Path = path.Join(prefix, strings.TrimPrefix(loc.Path, rewrite))
	if err := resetResponseHeaderLocation(loc.String()); err != nil {
		return 0, false, errors.Wrapf(err, "failure to resetResponseHeaderLocation %s", loc.String())
	}
	return types.ActionContinue, true, nil
}

func handleRegex(redirect string, location string, replacement string) (types.Action, bool, error) {
	newLocation, ok, err := proxy_redirect.ReplaceSubstitution(redirect, location, replacement)
	if ok {
		if err := resetResponseHeaderLocation(newLocation); err != nil {
			return types.ActionContinue, false, err
		}
	}
	return types.ActionContinue, ok, err
}

func resetResponseHeaderLocation(value string) error {
	if err := proxywasm.RemoveHttpResponseHeader(locationKey); err != nil {
		return errors.Wrap(err, "failure to RemoveHttpResponseHeader")
	}
	if err := proxywasm.AddHttpResponseHeader(locationKey, value); err != nil {
		return errors.Wrap(err, "failure to AddHttpResponseHeader")
	}
	return nil
}
