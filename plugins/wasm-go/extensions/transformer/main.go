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
	"regexp"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"

	"github.com/pkg/errors"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

func main() {
	wrapper.SetCtx(
		"transformer",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
		wrapper.ProcessRequestBodyBy(onHttpRequestBody),
		wrapper.ProcessResponseHeadersBy(onHttpResponseHeaders),
		wrapper.ProcessResponseBodyBy(onHttpResponseBody),
	)
}

// @Name transformer
// @Category custom
// @Phase UNSPECIFIED_PHASE
// @Priority 100
// @Title zh-CN 请求/响应转换器
// @Title en-US Request/Response Transformer
// @Description zh-CN transformer 插件可以对请求/响应头、请求查询参数、请求/响应体参数进行转换，支持的转换操作类型包括删除、重命名、更新、添加、追加、映射、去重。
// @Description en-US The transformer plugin can transform request/response headers, request query parameters, and request/response body parameters. Supported transform operations include remove, rename, replace, add, append, map, and dedupe.
// @IconUrl https://img.alicdn.com/imgextra/i1/O1CN018iKKih1iVx287RltL_!!6000000004419-2-tps-42-42.png
// @Version 1.0.0
//
// @Contact.name Higress Team
// @Contact.url http://higress.io/
// @Contact.email admin@higress.io
//
// @Example
//
// @End
type TransformerConfig struct {
	// @Title 转换器类型
	// @Description 指定转换器类型，可选值为 request, response。
	typ string `yaml:"type"`

	// @Title 键中是否包含点号
	// @Description 转换规则中 JSON 请求/响应体参数的键是否包含点号，例如 foo.bar:value。若为 true，则表示 foo.bar 作为键名，值为 value；否则表示嵌套关系，即 foo 的成员变量 bar 的值为 value
	dotsInKeys bool `yaml:"dots_in_keys"`

	// @Title 转换规则
	// @Description 指定转换操作类型以及请求/响应头、请求查询参数、请求/响应体参数的转换规则
	rules []TransformRule `yaml:"rules"`

	// this field is not exposed to the user and is used to store the request/response transformer instance
	trans Transformer `yaml:"-"`
}

type TransformRule struct {
	// @Title 转换操作类型
	// @Description 指定转换操作类型，可选值为 remove, rename, replace, add, append, map, dedupe
	operate string `yaml:"operate"`

	// @Title 请求/响应头转换规则
	// @Description 指定请求/响应头转换规则
	headers []Param `yaml:"headers"`

	// @Title 请求查询参数转换规则
	// @Description 指定请求查询参数转换规则
	querys []Param `yaml:"querys"`

	// @Title 请求/响应体参数转换规则
	// @Description 指定请求/响应体参数转换规则，请求体转换允许 content-type 为 application/json, application/x-www-form-urlencoded, multipart/form-data；响应体转换仅允许 content-type 为 application/json
	body []Param `yaml:"body"`
}

type Param struct {
	// @Title 键/键值对
	// @Description 指定键值对，当转换操作类型为 remove 时为 key，其他类型均为 key:value
	kv string `yaml:"kv"`

	// @Title 值类型
	// @Description 当 content-type=application/json 时，为请求/响应体参数指定值类型，可选值为 boolean, number, string(default)
	valueType string `yaml:"value_type"`

	// @Title 请求主机名匹配规则
	// @Description 指定主机名匹配规则，当转换操作类型为 replace, add, append 时有效
	hostPattern string `yaml:"host_pattern"`

	// @Title 请求路径匹配规则
	// @Description 指定路径匹配规则，当转换操作类型为 replace, add, append 时有效
	pathPattern string `yaml:"path_pattern"`
}

func parseConfig(json gjson.Result, config *TransformerConfig, log wrapper.Log) (err error) {
	config.typ = strings.ToLower(json.Get("type").String())
	if config.typ != "request" && config.typ != "response" {
		return errors.Errorf("invalid transformer type %q", config.typ)
	}
	config.dotsInKeys = json.Get("dots_in_keys").Bool()

	config.rules = make([]TransformRule, 0)
	rules := json.Get("rules").Array()
	for _, r := range rules {
		var tRule TransformRule
		tRule.operate = strings.ToLower(r.Get("operate").String())
		switch tRule.operate {
		case "remove", "rename", "replace", "add", "append", "map", "dedupe":
		default:
			return errors.Errorf("invalid operate type %q", tRule.operate)
		}
		for _, h := range r.Get("headers").Array() {
			var p Param
			p.kv = h.Get("kv").String()
			if tRule.operate == "replace" || tRule.operate == "add" || tRule.operate == "append" {
				p.hostPattern = h.Get("host_pattern").String()
				p.pathPattern = h.Get("path_pattern").String()
			}
			tRule.headers = append(tRule.headers, p)
		}
		for _, q := range r.Get("querys").Array() {
			var p Param
			p.kv = q.Get("kv").String()
			if tRule.operate == "replace" || tRule.operate == "add" || tRule.operate == "append" {
				p.hostPattern = q.Get("host_pattern").String()
				p.pathPattern = q.Get("path_pattern").String()
			}
			tRule.querys = append(tRule.querys, p)
		}
		for _, b := range r.Get("body").Array() {
			p := Param{
				kv:        b.Get("kv").String(),
				valueType: strings.ToLower(b.Get("value_type").String()),
			}
			if p.valueType == "" { // default
				p.valueType = "string"
			}
			if p.valueType != "boolean" && p.valueType != "number" && p.valueType != "string" {
				return errors.Errorf("invalid body params type %q", p.valueType)
			}
			if tRule.operate == "replace" || tRule.operate == "add" || tRule.operate == "append" {
				p.hostPattern = b.Get("host_pattern").String()
				p.pathPattern = b.Get("path_pattern").String()
			}
			tRule.body = append(tRule.body, p)
		}
		config.rules = append(config.rules, tRule)
	}

	// construct request/response transformer
	if config.typ == "request" {
		config.trans, err = newRequestTransformer(config)
		if err != nil {
			return errors.Wrap(err, "failed to new request transformer")
		}
	} else if config.typ == "response" {
		config.trans, err = newResponseTransformer(config)
		if err != nil {
			return errors.Wrap(err, "failed to new response transformer")
		}
	}

	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config TransformerConfig, log wrapper.Log) types.Action {
	// for response transformer
	host, path := ctx.Host(), ctx.Path()
	ctx.SetContext("host", host)
	ctx.SetContext("path", path)

	if config.typ == "response" {
		return types.ActionContinue
	}

	log.Debug("on http request headers ...")

	headers, err := proxywasm.GetHttpRequestHeaders()
	if err != nil {
		log.Warn("failed to get request headers")
		return types.ActionContinue
	}
	hs := convertHeaders(headers)
	if hs[":authority"] == nil {
		log.Warn("failed to get request host")
		return types.ActionContinue
	}
	if hs[":path"] == nil {
		log.Warn("failed to get request path")
		return types.ActionContinue
	}
	contentType := ""
	if hs["content-type"] != nil {
		contentType = hs["content-type"][0]
	}
	if contentType != "" {
		ctx.SetContext("content-type", contentType) // for request body
	}
	if config.trans.IsBodyChange() {
		delete(hs, "content-length")
	}
	qs, err := parseQueryByPath(path)
	if err != nil {
		log.Warnf("failed to parse query params by path: %v", err)
		return types.ActionContinue
	}

	if err = config.trans.TransformHeaderAndQuerys(host, path, hs, qs); err != nil {
		log.Warnf("failed to transform request header and query params: %v", err)
		return types.ActionContinue
	}

	path, err = constructPath(path, qs)
	if err != nil {
		log.Warnf("failed to construct path: %v", err)
		return types.ActionContinue
	}
	hs[":path"] = []string{path}
	headers = undoConvertHeaders(hs)
	if err = proxywasm.ReplaceHttpRequestHeaders(headers); err != nil {
		log.Warnf("failed to replace request headers: %v", err)
		return types.ActionContinue
	}

	return types.ActionContinue
}

func onHttpRequestBody(ctx wrapper.HttpContext, config TransformerConfig, body []byte, log wrapper.Log) types.Action {
	if config.typ == "response" || !config.trans.IsBodyChange() {
		return types.ActionContinue
	}

	log.Debug("on http request body ...")

	host, path, err := getHostAndPathFromHttpCtx(ctx)
	if err != nil {
		log.Warn(err.Error())
		return types.ActionContinue
	}
	contentType, ok := ctx.GetContext("content-type").(string)
	if !ok || !(contentType == "application/json" || contentType == "application/x-www-form-urlencoded" || contentType == "multipart/form-data") {
		log.Debugf("content-type=%q. there is no need to process the request body", contentType)
		return types.ActionContinue
	}
	structuredBody, err := parseBody(contentType, body)
	if err != nil {
		if !errors.Is(err, errEmptyBody) {
			log.Warnf("failed to parse request body: %v", err)
		}
		log.Debug("request body is empty")
		return types.ActionContinue
	}

	if err = config.trans.TransformBody(host, path, structuredBody); err != nil {
		log.Warnf("failed to transform request body: %v", err)
		return types.ActionContinue
	}

	body, err = constructBody(contentType, structuredBody)
	if err != nil {
		log.Warnf("failed to construct request body: %v", err)
		return types.ActionContinue
	}
	if err = proxywasm.ReplaceHttpRequestBody(body); err != nil {
		log.Warnf("failed to replace request body: %v", err)
		return types.ActionContinue
	}

	return types.ActionContinue
}

func onHttpResponseHeaders(ctx wrapper.HttpContext, config TransformerConfig, log wrapper.Log) types.Action {
	if config.typ == "request" {
		return types.ActionContinue
	}

	log.Debug("on http response headers ...")

	host, path, err := getHostAndPathFromHttpCtx(ctx)
	if err != nil {
		log.Warn(err.Error())
		return types.ActionContinue
	}
	headers, err := proxywasm.GetHttpResponseHeaders()
	if err != nil {
		log.Warnf("failed to get response headers: %v", err)
		return types.ActionContinue
	}
	hs := convertHeaders(headers)
	contentType := ""
	if hs["content-type"] != nil {
		contentType = hs["content-type"][0]
	}
	if contentType != "" {
		ctx.SetContext("content-type", contentType) // for response body
	}
	if config.trans.IsBodyChange() {
		delete(hs, "content-length")
	}

	if err = config.trans.TransformHeaderAndQuerys(host, path, hs, nil); err != nil {
		log.Warnf("failed to transform response header: %v", err)
		return types.ActionContinue
	}

	headers = undoConvertHeaders(hs)
	if err = proxywasm.ReplaceHttpResponseHeaders(headers); err != nil {
		log.Warnf("failed to replace response headers: %v", err)
		return types.ActionContinue
	}

	return types.ActionContinue
}

func onHttpResponseBody(ctx wrapper.HttpContext, config TransformerConfig, body []byte, log wrapper.Log) types.Action {
	if config.typ == "request" || !config.trans.IsBodyChange() {
		return types.ActionContinue
	}

	log.Debug("on http response body ...")

	host, path, err := getHostAndPathFromHttpCtx(ctx)
	if err != nil {
		log.Warn(err.Error())
		return types.ActionContinue
	}
	contentType, ok := ctx.GetContext("content-type").(string)
	if !ok || contentType != "application/json" { // only work with json response body
		log.Debugf("content-type=%q. there is no need to process the response body", contentType)
		return types.ActionContinue
	}
	structuredBody, err := parseBody(contentType, body)
	if err != nil {
		if !errors.Is(err, errEmptyBody) {
			log.Warnf("failed to parse response body: %v", err)
		}
		log.Debug("response body is empty")
		return types.ActionContinue
	}

	if err = config.trans.TransformBody(host, path, structuredBody); err != nil {
		log.Warnf("failed to transform response body: %v", err)
		return types.ActionContinue
	}

	body, err = constructBody(contentType, structuredBody)
	if err != nil {
		log.Warnf("failed to construct response body: %v", err)
		return types.ActionContinue
	}
	if err = proxywasm.ReplaceHttpResponseBody(body); err != nil {
		log.Warnf("failed to replace response body: %v", err)
		return types.ActionContinue
	}

	return types.ActionContinue
}

func getHostAndPathFromHttpCtx(ctx wrapper.HttpContext) (host, path string, err error) {
	host, ok := ctx.GetContext("host").(string)
	if !ok {
		err = errors.New("failed to get request host")
		return "", "", err
	}
	path, ok = ctx.GetContext("path").(string)
	if !ok {
		err = errors.New("failed to get request path")
		return "", "", err
	}
	return host, path, nil
}

type Transformer interface {
	TransformHeaderAndQuerys(host, path string, hs map[string][]string, qs map[string][]string) error
	TransformBody(host, path string, body interface{}) error
	IsBodyChange() bool
}

var _ Transformer = (*requestTransformer)(nil)
var _ Transformer = (*responseTransformer)(nil)

type requestTransformer struct {
	headerHandler *kvHandler
	queryHandler  *kvHandler
	bodyHandler   *requestBodyHandler
	isBodyChange  bool
}

func newRequestTransformer(config *TransformerConfig) (Transformer, error) {
	headerKvtGroup, err := newKvtGroup(config.rules, "headers")
	if err != nil {
		return nil, errors.Wrap(err, "failed to new kvt group for headers")
	}
	queryKvtGroup, err := newKvtGroup(config.rules, "querys")
	if err != nil {
		return nil, errors.Wrap(err, "failed to new kvt group for querys")
	}
	bodyKvtGroup, err := newKvtGroup(config.rules, "body")
	if err != nil {
		return nil, errors.Wrap(err, "failed to new kvt group for body")
	}
	return &requestTransformer{
		headerHandler: &kvHandler{headerKvtGroup},
		queryHandler:  &kvHandler{queryKvtGroup},
		bodyHandler: &requestBodyHandler{
			formDataHandler: &kvHandler{bodyKvtGroup},
			jsonHandler:     &jsonHandler{bodyKvtGroup, config.dotsInKeys},
		},
		isBodyChange: len(bodyKvtGroup.remove) != 0 ||
			len(bodyKvtGroup.rename) != 0 || len(bodyKvtGroup.replace) != 0 ||
			len(bodyKvtGroup.add) != 0 || len(bodyKvtGroup.append) != 0 ||
			len(bodyKvtGroup.map_) != 0 || len(bodyKvtGroup.dedupe) != 0,
	}, nil
}

func (t requestTransformer) TransformHeaderAndQuerys(host, path string, hs map[string][]string, qs map[string][]string) error {
	if err := t.headerHandler.handle(host, path, hs); err != nil {
		return errors.Wrap(err, "failed to handle headers")
	}
	if err := t.queryHandler.handle(host, path, qs); err != nil {
		return errors.Wrap(err, "failed to handle querys")
	}
	return nil
}

func (t requestTransformer) TransformBody(host, path string, body interface{}) error {
	var err error
	switch body.(type) {
	case map[string][]string:
		err = t.bodyHandler.formDataHandler.handle(host, path, body.(map[string][]string))
	case map[string]interface{}:
		err = t.bodyHandler.jsonHandler.handle(host, path, body.(map[string]interface{}))
	default:
		err = errors.New("unsupported body type")
	}
	if err != nil {
		return errors.Wrap(err, "failed to handle body")
	}
	return nil
}

func (t requestTransformer) IsBodyChange() bool {
	return t.isBodyChange
}

type responseTransformer struct {
	headerHandler *kvHandler
	bodyHandler   *responseBodyHandler
	isBodyChange  bool
}

func newResponseTransformer(config *TransformerConfig) (Transformer, error) {
	headerKvtGroup, err := newKvtGroup(config.rules, "headers")
	if err != nil {
		return nil, errors.Wrap(err, "failed to new kvt group for headers")
	}
	bodyKvtGroup, err := newKvtGroup(config.rules, "body")
	if err != nil {
		return nil, errors.Wrap(err, "failed to new kvt group for body")
	}
	return &responseTransformer{
		headerHandler: &kvHandler{headerKvtGroup},
		bodyHandler:   &responseBodyHandler{&jsonHandler{bodyKvtGroup, config.dotsInKeys}},
		isBodyChange: len(bodyKvtGroup.remove) != 0 ||
			len(bodyKvtGroup.rename) != 0 || len(bodyKvtGroup.replace) != 0 ||
			len(bodyKvtGroup.add) != 0 || len(bodyKvtGroup.append) != 0 ||
			len(bodyKvtGroup.map_) != 0 || len(bodyKvtGroup.dedupe) != 0,
	}, nil
}

func (t responseTransformer) TransformHeaderAndQuerys(host, path string, hs map[string][]string, qs map[string][]string) error {
	// the response does not need to transform the query params
	if err := t.headerHandler.handle(host, path, hs); err != nil {
		return errors.Wrap(err, "failed to handle headers")
	}
	return nil
}

func (t responseTransformer) TransformBody(host, path string, body interface{}) error {
	var err error
	switch body.(type) {
	case map[string]interface{}:
		err = t.bodyHandler.handle(host, path, body.(map[string]interface{}))
	default:
		err = errors.New("unsupported body type")
	}
	if err != nil {
		return errors.Wrap(err, "failed to handle body")
	}
	return nil
}

func (t responseTransformer) IsBodyChange() bool {
	return t.isBodyChange
}

type requestBodyHandler struct {
	formDataHandler *kvHandler
	*jsonHandler
}

type responseBodyHandler struct {
	*jsonHandler
}

type kvHandler struct {
	*kvtGroup
}

type jsonHandler struct {
	*kvtGroup
	dotInKeys bool
}

func (h kvHandler) handle(host, path string, kvs map[string][]string) error {
	// order: remove → rename → replace → add → append → map → dedupe

	// remove
	for _, key := range h.remove {
		delete(kvs, key)
	}

	// rename: 若指定 oldKey 不存在则无操作；否则将 oldKey 的值追加给 newKey，并删除 oldKey:value
	for _, item := range h.rename {
		oldKey, newKey := item.key, item.value
		if ovs, ok := kvs[oldKey]; ok {
			kvs[newKey] = append(kvs[newKey], ovs...)
			delete(kvs, oldKey)
		}
	}

	// replace: 若指定 key 不存在，则无操作；否则替换 value 为 newValue
	for _, item := range h.replace {
		key, newValue := item.key, item.value
		if _, ok := kvs[key]; !ok {
			continue
		}
		if item.reg != nil {
			newValue = item.reg.matchAndReplace(newValue, host, path)
		}
		kvs[item.key] = []string{newValue}
	}

	// add: 若指定 key 存在则无操作；否则添加 key:value
	for _, item := range h.add {
		key, value := item.key, item.value
		if _, ok := kvs[key]; ok {
			continue
		}
		if item.reg != nil {
			value = item.reg.matchAndReplace(value, host, path)
		}
		kvs[key] = []string{value}
	}

	// append: 若指定 key 存在，则追加同名 kv；否则相当于添加操作
	for _, item := range h.append {
		key, appendValue := item.key, item.value
		if item.reg != nil {
			appendValue = item.reg.matchAndReplace(appendValue, host, path)
		}
		kvs[key] = append(kvs[key], appendValue)
	}

	// map: 若指定 fromKey 不存在则无操作；否则将 fromKey 的值映射给 toKey 的值
	for _, item := range h.map_ {
		fromKey, toKey := item.key, item.value
		if vs, ok := kvs[fromKey]; ok {
			kvs[toKey] = vs
		}
	}

	// dedupe: 根据 strategy 去重：RETAIN_UNIQUE 保留所有唯一值，RETAIN_LAST 保留最后一个值，RETAIN_FIRST 保留第一个值 (default)
	for _, item := range h.dedupe {
		key, strategy := item.key, item.value
		switch strings.ToUpper(strategy) {
		case "RETAIN_UNIQUE":
			uniSet := make(map[string]struct{})
			uniques := make([]string, 0)
			for _, v := range kvs[key] {
				if _, ok := uniSet[v]; !ok {
					uniSet[v] = struct{}{}
					uniques = append(uniques, v)
				}
			}
			kvs[key] = uniques

		case "RETAIN_LAST":
			if vs, ok := kvs[key]; ok && len(vs) >= 1 {
				kvs[key] = vs[len(vs)-1:]
			}

		case "RETAIN_FIRST":
			fallthrough
		default:
			if vs, ok := kvs[key]; ok && len(vs) >= 1 {
				kvs[key] = vs[:1]
			}
		}
	}

	return nil
}

func (h jsonHandler) handle(host, path string, data map[string]interface{}) error {
	// order: remove → rename → replace → add → append → map → dedupe

	// remove
	for _, key := range h.remove {
		if err := remove(data, h.dotInKeys, key); err != nil {
			return err
		}
	}

	// rename: 若指定 oldKey 不存在则无操作；否则将 oldKey 的值追加给 newKey，并删除 oldKey:value
	for _, item := range h.rename {
		oldKey, newKey := item.key, item.value
		if err := rename(data, h.dotInKeys, oldKey, newKey); err != nil {
			return err
		}
	}

	// replace: 若指定 key 不存在，则无操作；否则替换 value 为 newValue
	for _, item := range h.replace {
		key, value, valueType := item.key, item.value, item.typ
		if valueType == "string" && item.reg != nil {
			value = item.reg.matchAndReplace(value, host, path)
		}
		newValue := convertByJsonType(valueType, value)
		err := replace(data, h.dotInKeys, key, newValue)
		if err != nil {
			return err
		}
	}

	// add: 若指定 key 存在则无操作；否则添加 key:value
	for _, item := range h.add {
		key, value, valueType := item.key, item.value, item.typ
		if valueType == "string" && item.reg != nil {
			value = item.reg.matchAndReplace(value, host, path)
		}
		newValue := convertByJsonType(valueType, value)
		err := add(data, h.dotInKeys, key, newValue)
		if err != nil {
			return err
		}
	}

	// append: 若指定 key 存在，则追加同名 kv；否则相当于添加操作
	for _, item := range h.append {
		key, value, valueType := item.key, item.value, item.typ
		if valueType == "string" && item.reg != nil {
			value = item.reg.matchAndReplace(value, host, path)
		}
		appendValue := convertByJsonType(valueType, value)
		err := append_(data, h.dotInKeys, key, appendValue)
		if err != nil {
			return err
		}
	}

	// map: 若指定 fromKey 不存在则无操作；否则将 fromKey 的值映射给 toKey 的值
	for _, item := range h.map_ {
		fromKey, toKey := item.key, item.value
		err := map_(data, h.dotInKeys, fromKey, toKey)
		if err != nil {
			return err
		}
	}

	// dedupe: 根据 strategy 去重：RETAIN_UNIQUE 保留所有唯一值，RETAIN_LAST 保留最后一个值，RETAIN_FIRST 保留第一个值 (default)
	for _, item := range h.dedupe {
		key, strategy := item.key, item.value
		err := dedupe(data, h.dotInKeys, key, strategy)
		if err != nil {
			return err
		}
	}

	return nil
}

type kvtGroup struct {
	remove  []string // key
	rename  []kvt    // oldKey:newKey
	replace []kvtReg // key:newValue
	add     []kvtReg // newKey:newValue
	append  []kvtReg // key:appendValue
	map_    []kvt    // fromKey:toKey
	dedupe  []kvt    // key:strategy
}

func newKvtGroup(rules []TransformRule, typ string) (g *kvtGroup, err error) {
	g = &kvtGroup{}
	for _, r := range rules {
		var prams []Param
		switch typ {
		case "headers":
			prams = r.headers
		case "querys":
			prams = r.querys
		case "body":
			prams = r.body
		}

		for _, p := range prams {
			switch r.operate {
			case "remove":
				key := p.kv
				if typ == "headers" {
					key = strings.ToLower(key)
				}
				g.remove = append(g.remove, key)

			case "rename", "map", "dedupe":
				var kt kvt
				kv := strings.Split(p.kv, ":")
				if len(kv) != 2 {
					return nil, errors.Errorf("invalid %s kv %q", typ, p.kv)
				}
				kt.key, kt.value = strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
				if typ == "headers" {
					kt.key = strings.ToLower(kt.key)
					if r.operate == "rename" || r.operate == "map" {
						kt.value = strings.ToLower(kt.value)
					}
				}
				if typ == "body" {
					kt.typ = p.valueType
				}
				switch r.operate {
				case "rename":
					g.rename = append(g.rename, kt)
				case "map":
					g.map_ = append(g.map_, kt)
				case "dedupe":
					g.dedupe = append(g.dedupe, kt)
				}

			case "replace", "add", "append":
				var kr kvtReg
				kv := strings.Split(p.kv, ":")
				if len(kv) != 2 {
					return nil, errors.Errorf("invalid %s kv %q", typ, p.kv)
				}
				kr.key, kr.value = strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
				if typ == "headers" {
					kr.key = strings.ToLower(kr.key)
				}
				if p.hostPattern != "" || p.pathPattern != "" {
					kr.reg, err = newReg(p.hostPattern, p.pathPattern)
					if err != nil {
						return nil, errors.Wrap(err, "failed to new reg")
					}
				}
				if typ == "body" {
					kr.typ = p.valueType
				}
				switch r.operate {
				case "replace":
					g.replace = append(g.replace, kr)
				case "add":
					g.add = append(g.add, kr)
				case "append":
					g.append = append(g.append, kr)
				}
			}
		}

	}

	return g, nil
}

type kvtReg struct {
	kvt
	*reg
}

type kvt struct {
	key   string
	value string
	typ   string
}

type reg struct {
	hostReg *regexp.Regexp
	pathReg *regexp.Regexp
}

// you can only choose one between host and path
func newReg(hostPatten, pathPatten string) (r *reg, err error) {
	r = &reg{}
	if hostPatten != "" {
		r.hostReg, err = regexp.Compile(hostPatten)
		if err != nil {
			return nil, err
		}
		return r, nil
	}
	if pathPatten != "" {
		r.pathReg, err = regexp.Compile(pathPatten)
		if err != nil {
			return nil, err
		}
		return r, nil
	}

	return r, nil
}

func (r reg) matchAndReplace(value, host, path string) string {
	if r.hostReg != nil && r.hostReg.MatchString(host) {
		return r.hostReg.ReplaceAllString(host, value)
	}
	if r.pathReg != nil && r.pathReg.MatchString(path) {
		return r.pathReg.ReplaceAllString(path, value)
	}
	return value
}
