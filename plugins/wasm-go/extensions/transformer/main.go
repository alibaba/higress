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
	"github.com/tidwall/sjson"
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
// type: request
// rules:
// - operate: remove
//   headers:
//   - key: X-remove
//   querys:
//   - key: k1
//   body:
//   - key: a1
// - operate: rename
//   headers:
//   - key: X-not-renamed
//     value: X-renamed
// - operate: replace
//   headers:
//   - key: X-replace
//     value: replaced
// - operate: add
//   headers:
//   - key: X-add-append
//     value: host-$1
//     host_pattern: ^(.*)\.com$
// - operate: append
//   headers:
//   - key: X-add-append
//     value: path-$1
//     path_pattern: ^.*?\/(\w+)[\?]{0,1}.*$
//   body:
//   - key: a1-new
//     value: t1-$1-append
//     value_type: string
//     host_pattern: ^(.*)\.com$
// - operate: map
//   headers:
//   - key: X-add-append
//     value: X-map
// - operate: dedupe
//   headers:
//   - key: X-dedupe-first
//     value: RETAIN_FIRST
// @End
type TransformerConfig struct {
	// @Title 转换器类型
	// @Description 指定转换器类型，可选值为 request, response。
	typ string `yaml:"type"`

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
	// @Title 参数的键
	// @Description 指定键值对的键
	key string `yaml:"key"`

	// @Title 参数的值
	// @Description 指定键值对的值，可能的含义有：空 (remove)，key (rename, map), value (replace, add, append), strategy (dedupe)
	value string `yaml:"value"`

	// @Title 值类型
	// @Description 当 content-type=application/json 时，为请求/响应体参数指定值类型，可选值为 object, boolean, number, string(default)
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

	config.rules = make([]TransformRule, 0)
	rules := json.Get("rules").Array()
	for _, r := range rules {
		var tRule TransformRule
		tRule.operate = strings.ToLower(r.Get("operate").String())
		if !isValidOperation(tRule.operate) {
			return errors.Errorf("invalid operate type %q", tRule.operate)
		}
		for _, h := range r.Get("headers").Array() {
			tRule.headers = append(tRule.headers, constructParam(&h, tRule.operate, ""))
		}
		for _, q := range r.Get("querys").Array() {
			tRule.querys = append(tRule.querys, constructParam(&q, tRule.operate, ""))
		}
		for _, b := range r.Get("body").Array() {
			valueType := strings.ToLower(b.Get("value_type").String())
			if valueType == "" { // default
				valueType = "string"
			}
			if !isValidJsonType(valueType) {
				return errors.Errorf("invalid body params type %q", valueType)
			}
			tRule.body = append(tRule.body, constructParam(&b, tRule.operate, valueType))
		}
		config.rules = append(config.rules, tRule)
	}

	switch config.typ {
	case "request":
		config.trans, err = newRequestTransformer(config)
	case "response":
		config.trans, err = newResponseTransformer(config)
	}
	if err != nil {
		return errors.Wrapf(err, "failed to new %s transformer", config.typ)
	}

	return nil
}

func constructParam(item *gjson.Result, op, valueType string) Param {
	p := Param{
		key:       item.Get("key").String(),
		value:     item.Get("value").String(),
		valueType: valueType,
	}
	if op == "replace" || op == "add" || op == "append" {
		p.hostPattern = item.Get("host_pattern").String()
		p.pathPattern = item.Get("path_pattern").String()
	}
	return p
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config TransformerConfig, log wrapper.Log) types.Action {
	// because it may be a response transformer, so the setting of host and path have to advance
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
		log.Warn(errGetRequestHost.Error())
		return types.ActionContinue
	}
	if hs[":path"] == nil {
		log.Warn(errGetRequestPath.Error())
		return types.ActionContinue
	}
	contentType := ""
	if hs["content-type"] != nil {
		contentType = hs["content-type"][0]
	}
	if config.trans.IsBodyChange() && isValidRequestContentType(contentType) {
		delete(hs, "content-length")
		ctx.SetContext("content-type", contentType)
	} else {
		ctx.DontReadRequestBody()
	}

	if config.trans.IsHeaderChange() {
		if err = config.trans.TransformHeaders(host, path, hs); err != nil {
			log.Warnf("failed to transform request headers: %v", err)
			return types.ActionContinue
		}
	}

	if config.trans.IsQueryChange() {
		qs, err := parseQueryByPath(path)
		if err != nil {
			log.Warnf("failed to parse query params by path: %v", err)
			return types.ActionContinue
		}
		if err = config.trans.TransformQuerys(host, path, qs); err != nil {
			log.Warnf("failed to transform request query params: %v", err)
			return types.ActionContinue
		}
		path, err = constructPath(path, qs)
		if err != nil {
			log.Warnf("failed to construct path: %v", err)
			return types.ActionContinue
		}
		hs[":path"] = []string{path}
	}

	headers = reconvertHeaders(hs)
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
	if !ok {
		log.Warn(errGetContentType.Error())
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
	if config.trans.IsBodyChange() && isValidResponseContentType(contentType) {
		delete(hs, "content-length")
		ctx.SetContext("content-type", contentType)
	} else {
		ctx.DontReadResponseBody()
	}

	if config.trans.IsHeaderChange() {
		if err = config.trans.TransformHeaders(host, path, hs); err != nil {
			log.Warnf("failed to transform response headers: %v", err)
			return types.ActionContinue
		}
	}

	headers = reconvertHeaders(hs)
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
	if !ok {
		log.Warn(errGetContentType.Error())
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
		return "", "", errGetRequestHost
	}
	path, ok = ctx.GetContext("path").(string)
	if !ok {
		return "", "", errGetRequestPath
	}
	return host, path, nil
}

type Transformer interface {
	TransformHeaders(host, path string, hs map[string][]string) error
	TransformQuerys(host, path string, qs map[string][]string) error
	TransformBody(host, path string, body interface{}) error
	IsHeaderChange() bool
	IsQueryChange() bool
	IsBodyChange() bool
}

var _ Transformer = (*requestTransformer)(nil)
var _ Transformer = (*responseTransformer)(nil)

type requestTransformer struct {
	headerHandler  *kvHandler
	queryHandler   *kvHandler
	bodyHandler    *requestBodyHandler
	isHeaderChange bool
	isQueryChange  bool
	isBodyChange   bool
}

func newRequestTransformer(config *TransformerConfig) (Transformer, error) {
	headerKvtGroup, isHeaderChange, err := newKvtGroup(config.rules, "headers")
	if err != nil {
		return nil, errors.Wrap(err, "failed to new kvt group for headers")
	}
	queryKvtGroup, isQueryChange, err := newKvtGroup(config.rules, "querys")
	if err != nil {
		return nil, errors.Wrap(err, "failed to new kvt group for querys")
	}
	bodyKvtGroup, isBodyChange, err := newKvtGroup(config.rules, "body")
	if err != nil {
		return nil, errors.Wrap(err, "failed to new kvt group for body")
	}
	return &requestTransformer{
		headerHandler: &kvHandler{headerKvtGroup},
		queryHandler:  &kvHandler{queryKvtGroup},
		bodyHandler: &requestBodyHandler{
			formDataHandler: &kvHandler{bodyKvtGroup},
			jsonHandler:     &jsonHandler{bodyKvtGroup},
		},
		isHeaderChange: isHeaderChange,
		isQueryChange:  isQueryChange,
		isBodyChange:   isBodyChange,
	}, nil
}

func (t requestTransformer) TransformHeaders(host, path string, hs map[string][]string) error {
	return t.headerHandler.handle(host, path, hs)
}

func (t requestTransformer) TransformQuerys(host, path string, qs map[string][]string) error {
	return t.queryHandler.handle(host, path, qs)
}

func (t requestTransformer) TransformBody(host, path string, body interface{}) error {
	switch body.(type) {
	case map[string][]string:
		return t.bodyHandler.formDataHandler.handle(host, path, body.(map[string][]string))

	case map[string]interface{}:
		m := body.(map[string]interface{})
		newBody, err := t.bodyHandler.handle(host, path, m["body"].([]byte))
		if err != nil {
			return err
		}
		m["body"] = newBody

	default:
		return errBodyType
	}

	return nil
}

func (t requestTransformer) IsHeaderChange() bool { return t.isHeaderChange }
func (t requestTransformer) IsQueryChange() bool  { return t.isQueryChange }
func (t requestTransformer) IsBodyChange() bool   { return t.isBodyChange }

type responseTransformer struct {
	headerHandler  *kvHandler
	bodyHandler    *responseBodyHandler
	isHeaderChange bool
	isBodyChange   bool
}

func newResponseTransformer(config *TransformerConfig) (Transformer, error) {
	headerKvtGroup, isHeaderChange, err := newKvtGroup(config.rules, "headers")
	if err != nil {
		return nil, errors.Wrap(err, "failed to new kvt group for headers")
	}
	bodyKvtGroup, isBodyChange, err := newKvtGroup(config.rules, "body")
	if err != nil {
		return nil, errors.Wrap(err, "failed to new kvt group for body")
	}
	return &responseTransformer{
		headerHandler:  &kvHandler{headerKvtGroup},
		bodyHandler:    &responseBodyHandler{&jsonHandler{bodyKvtGroup}},
		isHeaderChange: isHeaderChange,
		isBodyChange:   isBodyChange,
	}, nil
}

func (t responseTransformer) TransformHeaders(host, path string, hs map[string][]string) error {
	return t.headerHandler.handle(host, path, hs)
}

func (t responseTransformer) TransformQuerys(host, path string, qs map[string][]string) error {
	// the response does not need to transform the query params, always returns nil
	return nil
}

func (t responseTransformer) TransformBody(host, path string, body interface{}) error {
	switch body.(type) {
	case map[string]interface{}:
		m := body.(map[string]interface{})
		newBody, err := t.bodyHandler.handle(host, path, m["body"].([]byte))
		if err != nil {
			return err
		}
		m["body"] = newBody

	default:
		return errBodyType
	}

	return nil
}

func (t responseTransformer) IsHeaderChange() bool { return t.isHeaderChange }
func (t responseTransformer) IsQueryChange() bool  { return false } // the response does not need to transform the query params, always returns false
func (t responseTransformer) IsBodyChange() bool   { return t.isBodyChange }

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
			uniSet, uniques := make(map[string]struct{}), make([]string, 0)
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

func (h jsonHandler) handle(host, path string, oriData []byte) (data []byte, err error) {
	// order: remove → rename → replace → add → append → map → dedupe
	if !gjson.ValidBytes(oriData) {
		return nil, errors.New("invalid json body")
	}
	data = oriData

	// remove
	for _, key := range h.remove {
		if data, err = sjson.DeleteBytes(data, key); err != nil {
			return nil, errors.Wrap(err, errRemove.Error())
		}
	}

	// rename: 若指定 oldKey 不存在则无操作；否则将 oldKey 的值追加给 newKey，并删除 oldKey:value
	for _, item := range h.rename {
		oldKey, newKey := item.key, item.value
		value := gjson.GetBytes(data, oldKey)
		if !value.Exists() {
			continue
		}
		if data, err = sjson.SetBytes(data, newKey, value.Value()); err != nil {
			return nil, errors.Wrap(err, errRename.Error())
		}
		if data, err = sjson.DeleteBytes(data, oldKey); err != nil {
			return nil, errors.Wrap(err, errRename.Error())
		}
	}

	// replace: 若指定 key 不存在，则无操作；否则替换 value 为 newValue
	for _, item := range h.replace {
		key, value, valueType := item.key, item.value, item.typ
		if !gjson.GetBytes(data, key).Exists() {
			continue
		}
		if valueType == "string" && item.reg != nil {
			value = item.reg.matchAndReplace(value, host, path)
		}
		newValue, err := convertByJsonType(valueType, value)
		if err != nil {
			return nil, errors.Wrap(err, errReplace.Error())
		}
		if data, err = sjson.SetBytes(data, key, newValue); err != nil {
			return nil, errors.Wrap(err, errReplace.Error())
		}
	}

	// add: 若指定 key 存在则无操作；否则添加 key:value
	for _, item := range h.add {
		key, value, valueType := item.key, item.value, item.typ
		if gjson.GetBytes(data, key).Exists() {
			continue
		}
		if valueType == "string" && item.reg != nil {
			value = item.reg.matchAndReplace(value, host, path)
		}
		newValue, err := convertByJsonType(valueType, value)
		if err != nil {
			return nil, errors.Wrap(err, errAdd.Error())
		}
		if data, err = sjson.SetBytes(data, key, newValue); err != nil {
			return nil, errors.Wrap(err, errAdd.Error())
		}
	}

	// append: 若指定 key 存在，则追加同名 kv；否则相当于添加操作
	// 当原本的 value 为数组时，追加；当原本的 value 不为数组时，将原本的 value 和 appendValue 组成数组
	for _, item := range h.append {
		key, value, valueType := item.key, item.value, item.typ
		if valueType == "string" && item.reg != nil {
			value = item.reg.matchAndReplace(value, host, path)
		}
		appendValue, err := convertByJsonType(valueType, value)
		if err != nil {
			return nil, errors.Wrapf(err, errAppend.Error())
		}
		oldValue := gjson.GetBytes(data, key)
		if !oldValue.Exists() {
			if data, err = sjson.SetBytes(data, key, appendValue); err != nil { // key: appendValue
				return nil, errors.Wrap(err, errAppend.Error())
			}
			continue
		}

		// oldValue exists
		if oldValue.IsArray() {
			if len(oldValue.Array()) == 0 {
				if data, err = sjson.SetBytes(data, key, []interface{}{appendValue}); err != nil { // key: [appendValue]
					return nil, errors.Wrap(err, errAppend.Error())
				}
				continue
			}

			// len(oldValue.Array()) != 0
			oldValues := make([]interface{}, 0, len(oldValue.Array())+1)
			for _, val := range oldValue.Array() {
				oldValues = append(oldValues, val.Value())
			}
			if data, err = sjson.SetBytes(data, key, append(oldValues, appendValue)); err != nil { // key: [oldValue..., appendValue]
				return nil, errors.Wrap(err, errAppend.Error())
			}
			continue
		}

		// oldValue is not array
		if data, err = sjson.SetBytes(data, key, []interface{}{oldValue.Value(), appendValue}); err != nil { // key: [oldValue, appendValue]
			return nil, errors.Wrap(err, errAppend.Error())
		}
	}

	// map: 若指定 fromKey 不存在则无操作；否则将 fromKey 的值映射给 toKey 的值
	for _, item := range h.map_ {
		fromKey, toKey := item.key, item.value
		fromValue := gjson.GetBytes(data, fromKey)
		if !fromValue.Exists() {
			continue
		}
		if data, err = sjson.SetBytes(data, toKey, fromValue.Value()); err != nil {
			return nil, errors.Wrap(err, errMap.Error())
		}
	}

	// dedupe: 根据 strategy 去重：RETAIN_UNIQUE 保留所有唯一值，RETAIN_LAST 保留最后一个值，RETAIN_FIRST 保留第一个值 (default)
	for _, item := range h.dedupe {
		key, strategy := item.key, item.value
		value := gjson.GetBytes(data, key)
		if !value.Exists() || !value.IsArray() {
			continue
		}

		// value is array
		values := value.Array()
		if len(values) == 0 {
			continue
		}

		var dedupedVal interface{}
		switch strings.ToUpper(strategy) {
		case "RETAIN_UNIQUE":
			uniSet, uniques := make(map[string]struct{}), make([]interface{}, 0)
			for _, v := range values {
				vstr := v.String()
				if _, ok := uniSet[vstr]; !ok {
					uniSet[vstr] = struct{}{}
					uniques = append(uniques, v.Value())
				}
			}
			if len(uniques) == 1 {
				dedupedVal = uniques[0] // key: uniques[0]
			} else if len(uniques) > 1 {
				dedupedVal = uniques // key: [uniques...]
			}

		case "RETAIN_LAST":
			dedupedVal = values[len(values)-1].Value() // key: last

		case "RETAIN_FIRST":
			fallthrough
		default:
			dedupedVal = values[0].Value() // key: first
		}

		if dedupedVal == nil {
			continue
		}
		if data, err = sjson.SetBytes(data, key, dedupedVal); err != nil {
			return nil, errors.Wrap(err, errDedupe.Error())
		}
	}

	return data, nil
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

func newKvtGroup(rules []TransformRule, typ string) (g *kvtGroup, isChange bool, err error) {
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
				key := p.key
				if typ == "headers" {
					key = strings.ToLower(key)
				}
				g.remove = append(g.remove, key)

			case "rename", "map", "dedupe":
				var kt kvt
				kt.key, kt.value = p.key, p.value
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
				kr.key, kr.value = p.key, p.value
				if typ == "headers" {
					kr.key = strings.ToLower(kr.key)
				}
				if p.hostPattern != "" || p.pathPattern != "" {
					kr.reg, err = newReg(p.hostPattern, p.pathPattern)
					if err != nil {
						return nil, false, errors.Wrap(err, "failed to new reg")
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

	isChange = len(g.remove) != 0 ||
		len(g.rename) != 0 || len(g.replace) != 0 ||
		len(g.add) != 0 || len(g.append) != 0 ||
		len(g.map_) != 0 || len(g.dedupe) != 0

	return g, isChange, nil
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
		return
	}
	if pathPatten != "" {
		r.pathReg, err = regexp.Compile(pathPatten)
		return
	}
	return
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
