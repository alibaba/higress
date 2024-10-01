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
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/gjson"
)

func setDefaultTag(k string, v string, log wrapper.Log) {
	if k == "" || v == "" {
		return
	}
	addTagHeader(k, v, log)
}

func getFullRequestURL() (string, error) {
	path, _ := proxywasm.GetHttpRequestHeader(":path")
	return path, nil
}

func parseCookie(cookieHeader string, key string) (string, bool) {
	cookies := strings.Split(cookieHeader, ";")
	for _, cookie := range cookies {
		cookie = strings.TrimSpace(cookie)
		if strings.HasPrefix(cookie, key+"=") {
			parts := strings.SplitN(cookie, "=", 2)
			if len(parts) == 2 {
				return parts[1], true
			}
		}
	}
	return "", false
}

func getQueryParameter(urlStr, paramKey string) (string, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}
	values, ok := u.Query()[paramKey]
	if !ok {
		return "", fmt.Errorf("parameter %s not found", paramKey)
	}
	return values[0], nil
}

func addTagHeader(key string, value string, log wrapper.Log) {
	existValue, _ := proxywasm.GetHttpRequestHeader(key)
	if existValue != "" {
		log.Infof("ADD HEADER failed: %s already exists, value: %s", key, existValue)
		return
	}
	if err := proxywasm.AddHttpRequestHeader(key, value); err != nil {
		log.Infof("failed to add tag header: %s", err)
		return
	}
	log.Infof("ADD HEADER: %s, value: %s", key, value)
}

func jsonValidate(json gjson.Result, log wrapper.Log) error {
	if !json.Exists() {
		log.Error("plugin config is missing in JSON")
		return errors.New("plugin config is missing in JSON")
	}

	jsonStr := strings.TrimSpace(json.Raw)
	if jsonStr == "{}" || jsonStr == "" {
		log.Error("plugin config is empty")
		return errors.New("plugin config is empty")
	}

	if !gjson.Valid(json.Raw) {
		log.Error("plugin config is invalid JSON")
		return errors.New("plugin config is invalid JSON")
	}

	return nil
}
