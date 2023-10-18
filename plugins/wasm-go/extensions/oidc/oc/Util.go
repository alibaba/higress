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
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/gjson"
)

func ValidateHTTPResponse(statusCode int, headers http.Header, body []byte) error {
	contentType := headers.Get("Content-Type")
	if statusCode != http.StatusOK {
		return errors.New("call failed with status code")
	}
	if !strings.Contains(contentType, "application/json") {
		return fmt.Errorf("expected Content-Type = application/json or application/json;charset=UTF-8, but got %s", contentType)
	}
	if !gjson.ValidBytes(body) {
		return errors.New("invalid JSON format in response body")
	}
	return nil
}

// GetParams 返回顺序 cookie code state
func GetParams(cookie string, path string) (string, string, string) {
	u, err := url.Parse(path)
	if err != nil {
		return "", "", ""
	}
	query := u.Query()
	code, state := query.Get("code"), query.Get("state")

	cookiePairs := strings.Split(cookie, "; ")
	var oidcCookieValue string
	for _, pair := range cookiePairs {
		keyValue := strings.Split(pair, "=")
		if keyValue[0] == "oidc_oauth2_wasm_plugin" {
			oidcCookieValue = keyValue[1]
			break
		}
	}
	oidcCookieValue, _ = url.QueryUnescape(oidcCookieValue)
	oidcCookieValue, err = Decrypt(oidcCookieValue)

	return oidcCookieValue, code, state
}

func SendError(log *wrapper.Log, errMsg string, status int) {
	log.Errorf(errMsg)
	proxywasm.SendHttpResponse(uint32(status), nil, []byte(errMsg), -1)
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
