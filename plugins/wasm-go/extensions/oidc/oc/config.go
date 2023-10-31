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
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"golang.org/x/oauth2"
)

type Accessmod int

const (
	Access  Accessmod = 0
	SenBack Accessmod = 1
)

var invalidRedirectRegex = regexp.MustCompile(`[/\\](?:[\s\v]*|\.{1,2})[/\\]`)

type IDConfig struct {
	ClientID             string
	SupportedSigningAlgs []string
	SkipExpiryCheck      bool
	//SkipIssuerCheck 用于特殊情况，其中调用者希望推迟对签发者的验证。
	//当启用时，调用者必须独立验证令牌的签发者是否为已知的有效值。
	//
	//
	//不匹配的签发者通常指示客户端配置错误。如果不希望发生不匹配，请检查所提供的签发者URL是否正确，而不是启用这个选项。
	SkipIssuerCheck bool
	SkipNonceCheck  bool
	Now             func() time.Time
}

type idToken struct {
	Issuer       string                 `json:"iss"`
	Subject      string                 `json:"sub"`
	Audience     audience               `json:"aud"`
	Expiry       jsonTime               `json:"exp"`
	IssuedAt     jsonTime               `json:"iat"`
	NotBefore    *jsonTime              `json:"nbf"`
	Nonce        string                 `json:"nonce"`
	AtHash       string                 `json:"at_hash"`
	ClaimNames   map[string]string      `json:"_claim_names"`
	ClaimSources map[string]claimSource `json:"_claim_sources"`
}

type claimSource struct {
	Endpoint    string `json:"endpoint"`
	AccessToken string `json:"access_token"`
}

type Oatuh2Config struct {
	oauth2.Config
	Issuer               string
	JwksURL              string
	ClientUrl            string
	Path                 string
	SupportedSigningAlgs []string
	SkipExpiryCheck      bool
	Timeout              int
	Client               wrapper.HttpClient
	SkipNonceCheck       bool

	Option       *OidcOption
	CookieOption *CookieOption
	CookieData   *CookieData
}

type OidcOption struct {
	StateStr   string
	Nonce      string
	Code       string
	Mod        Accessmod
	RawIdToken string
	AuthStyle  AuthStyle
}

func IsValidRedirect(redirect string) error {
	if !strings.HasSuffix(redirect, "oauth2/callback") {
		return errors.New("redirect URL must end with oauth2/callback")
	}
	switch {
	case redirect == "":
		return errors.New("redirect URL is empty")
	case strings.HasPrefix(redirect, "/"):
		if strings.HasPrefix(redirect, "//") || invalidRedirectRegex.MatchString(redirect) {
			return errors.New("invalid local redirect URL")
		}
		return nil
	case strings.HasPrefix(redirect, "http://"), strings.HasPrefix(redirect, "https://"):
		_, err := url.ParseRequestURI(redirect)
		if err != nil {
			return errors.New("invalid remote redirect URL")
		}
		return nil
	default:
		return errors.New("redirect URL must start with /, http://, or https://")
	}
}
