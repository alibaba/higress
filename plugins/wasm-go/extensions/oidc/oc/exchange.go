//Copyright 2023 go-oidc
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//http://www.apache.org/licenses/LICENSE-2.0
//
//Unless required by applicable law or agreed to in writing, software
//distributed under the License is distributed on an "AS IS" BASIS,
//WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//See the License for the specific language governing permissions and
//limitations under the License.

package oc

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"io"
	"math"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Token struct {
	AccessToken  string
	TokenType    string
	RefreshToken string
	Expiry       time.Time
	Raw          interface{}
}

type tokenJSON struct {
	AccessToken  string         `json:"access_token"`
	TokenType    string         `json:"token_type"`
	RefreshToken string         `json:"refresh_token"`
	ExpiresIn    expirationTime `json:"expires_in"` // at least PayPal returns string, while most return number
}

type AuthCodeOption interface {
	setValue(url.Values)
}

type setParam struct{ k, v string }

func (p setParam) SetValue(m url.Values) { m.Set(p.k, p.v) }

func ReturnURL(RedirectURL, code string, opts ...AuthCodeOption) url.Values {
	v := url.Values{
		"grant_type": {"authorization_code"},
		"code":       {code},
	}
	if RedirectURL != "" {
		v.Set("redirect_uri", RedirectURL)
	}
	for _, opt := range opts {
		opt.setValue(v)
	}
	return v
}

func TokenFromInternal(t *Token) *Token {
	if t == nil {
		return nil
	}
	return &Token{
		AccessToken:  t.AccessToken,
		TokenType:    t.TokenType,
		RefreshToken: t.RefreshToken,
		Expiry:       t.Expiry,
		Raw:          t.Raw,
	}
}

func (e *tokenJSON) expiry() (t time.Time) {
	if v := e.ExpiresIn; v != 0 {
		return time.Now().Add(time.Duration(v) * time.Second)
	}
	return
}

type expirationTime int32

func (e *expirationTime) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	var n json.Number
	err := json.Unmarshal(b, &n)
	if err != nil {
		return err
	}
	i, err := n.Int64()
	if err != nil {
		return err
	}
	if i > math.MaxInt32 {
		i = math.MaxInt32
	}
	*e = expirationTime(i)
	return nil
}

type AuthStyle int

const (
	AuthStyleUnknown  AuthStyle = 0
	AuthStyleInParams AuthStyle = 1
	AuthStyleInHeader AuthStyle = 2
)

var authStyleCache struct {
	m map[string]AuthStyle // keyed by tokenURL
}

func LookupAuthStyle(tokenURL string) (style AuthStyle, ok bool) {
	style, ok = authStyleCache.m[tokenURL]
	return
}

// SetAuthStyle adds an entry to authStyleCache, documented above.
func SetAuthStyle(tokenURL string, v AuthStyle) {
	if authStyleCache.m == nil {
		authStyleCache.m = make(map[string]AuthStyle)
	}
	authStyleCache.m[tokenURL] = v
}

func (t *Token) Extra(key string) interface{} {
	if raw, ok := t.Raw.(map[string]interface{}); ok {
		return raw[key]
	}

	vals, ok := t.Raw.(url.Values)
	if !ok {
		return nil
	}

	v := vals.Get(key)
	switch s := strings.TrimSpace(v); strings.Count(s, ".") {
	case 0: // Contains no "."; try to parse as int
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}
	case 1: // Contains a single "."; try to parse as float
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
	}

	return v
}

func UnmarshalToken(token *Token, Headers http.Header, body []byte) (*Token, error) {
	if !gjson.ValidBytes(body) {
		return nil, fmt.Errorf("invalid JSON format in response body , get %v", string(body))
	}
	content, _, _ := mime.ParseMediaType(Headers.Get("Content-Type"))

	switch content {
	case "application/x-www-form-urlencoded", "text/plain":
		vals, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, err
		}
		token = &Token{
			AccessToken:  vals.Get("access_token"),
			TokenType:    vals.Get("token_type"),
			RefreshToken: vals.Get("refresh_token"),
			Raw:          vals,
		}
		e := vals.Get("expires_in")
		expires, _ := strconv.Atoi(e)
		if expires != 0 {
			token.Expiry = time.Now().Add(time.Duration(expires) * time.Second)
		}
	default:
		var tj tokenJSON
		if err := json.Unmarshal(body, &tj); err != nil {
			return nil, err
		}
		token = &Token{
			AccessToken:  tj.AccessToken,
			TokenType:    tj.TokenType,
			RefreshToken: tj.RefreshToken,
			Expiry:       tj.expiry(),
			Raw:          make(map[string]interface{}),
		}
		if err := json.Unmarshal(body, &token.Raw); err != nil {
			return nil, err
		}

		// no error checks for optional fields
	}
	if token.AccessToken == "" {
		return nil, errors.New("oauth2: server response missing access_token")
	}
	return token, nil
}

func NewTokenRequest(tokenURL, clientID, clientSecret string, v url.Values, authStyle AuthStyle) ([][2]string, []byte, error) {
	if authStyle == AuthStyleInParams {
		v = cloneURLValues(v)
		if clientID != "" {
			v.Set("client_id", clientID)
		}
		if clientSecret != "" {
			v.Set("client_secret", clientSecret)
		}
	}
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if authStyle == AuthStyleInHeader {
		req.SetBasicAuth(url.QueryEscape(clientID), url.QueryEscape(clientSecret))
	}
	var headerArray [][2]string
	for key, values := range req.Header {
		if len(values) > 0 {
			headerArray = append(headerArray, [2]string{key, values[0]})
		}
	}
	bodyBytes, err := io.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return nil, nil, err
	}

	return headerArray, bodyBytes, nil
}

func cloneURLValues(v url.Values) url.Values {
	v2 := make(url.Values, len(v))
	for k, vv := range v {
		v2[k] = append([]string(nil), vv...)
	}
	return v2
}

type RetrieveError struct {
	Response *http.Response
	Body     []byte
}

func (r *RetrieveError) Error() string {
	return fmt.Sprintf("oauth2: cannot fetch token: %v\nResponse: %s", r.Response.Status, r.Body)
}
