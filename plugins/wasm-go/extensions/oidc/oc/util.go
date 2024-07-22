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

package oc

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/gjson"
)

func ValidateHTTPResponse(statusCode int, headers http.Header, body []byte) error {
	contentType := headers.Get("Content-Type")
	if statusCode != http.StatusOK {
		return errors.New("call failed with status code")
	}
	if !strings.Contains(contentType, "application/json") {
		return fmt.Errorf("expected Content-Type = application/json , but got %s", contentType)
	}
	if !gjson.ValidBytes(body) {
		return errors.New("invalid JSON format in response body")
	}
	return nil
}

// GetParams 返回顺序 cookie code state
func GetParams(name, cookie, path, key string) (oidcCookieValue, code, state string, err error) {
	u, err := url.Parse(path)
	if err != nil {
		return "", "", "", err
	}
	query := u.Query()
	code, state = query.Get("code"), query.Get("state")

	cookiePairs := strings.Split(cookie, "; ")
	for _, pair := range cookiePairs {
		keyValue := strings.Split(pair, "=")
		if keyValue[0] == name {
			oidcCookieValue = keyValue[1]
			break
		}
	}

	oidcCookieValue, err = url.QueryUnescape(oidcCookieValue)
	if err != nil {
		return "", "", "", err
	}
	oidcCookieValue, err = Decrypt(oidcCookieValue, key)
	return oidcCookieValue, code, state, nil
}

func SendError(log *wrapper.Log, errMsg string, status int, statusDetail string) {
	log.Errorf(errMsg)
	proxywasm.SendHttpResponseWithDetail(uint32(status), statusDetail, nil, []byte(errMsg), -1)
}

type jsonTime time.Time

func (j *jsonTime) UnmarshalJSON(b []byte) error {
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	var unix int64

	if t, err := n.Int64(); err == nil {
		unix = t
	} else {
		f, err := n.Float64()
		if err != nil {
			return err
		}
		unix = int64(f)
	}
	*j = jsonTime(time.Unix(unix, 0))
	return nil
}

type audience []string

func (a *audience) UnmarshalJSON(b []byte) error {
	var s string
	if json.Unmarshal(b, &s) == nil {
		*a = audience{s}
		return nil
	}
	var auds []string
	if err := json.Unmarshal(b, &auds); err != nil {
		return err
	}
	*a = auds
	return nil
}
