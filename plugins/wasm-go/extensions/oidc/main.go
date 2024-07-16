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
	"net/http"
	"net/url"
	"oidc/oc"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"golang.org/x/oauth2"
)

const OAUTH2CALLBACK = "oauth2/callback"

type OidcConfig struct {
	Issuer          string
	Path            string
	ClientID        string
	ClientSecret    string
	RedirectURL     string
	ClientURL       string
	Timeout         int
	CookieName      string
	CookieSecret    string
	CookieDomain    string
	CookiePath      string
	CookieSameSite  string
	CookieSecure    bool
	CookieHTTPOnly  bool
	Scopes          []string
	SkipExpiryCheck bool
	SkipNonceCheck  bool
	Client          wrapper.HttpClient
}

func main() {
	wrapper.SetCtx(
		"oidc",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

func parseConfig(json gjson.Result, config *OidcConfig, log wrapper.Log) error {
	config.Issuer = json.Get("issuer").String()
	if config.Issuer == "" {
		return errors.New("missing issuer in config")
	}

	config.ClientID = json.Get("client_id").String()
	if config.ClientID == "" {
		return errors.New("missing client_id in config")
	}

	config.ClientSecret = json.Get("client_secret").String()
	if config.ClientSecret == "" {
		return errors.New("missing client_secret in config")
	}
	config.ClientURL = json.Get("client_url").String()
	_, err := url.ParseRequestURI(config.ClientURL)
	if err != nil {
		return errors.New("missing client_url in config or err format")
	}

	err = oc.IsValidRedirect(json.Get("redirect_url").String())
	if err != nil {
		return err
	}
	config.RedirectURL = json.Get("redirect_url").String()

	config.SkipExpiryCheck = json.Get("skip_expiry_check").Bool()
	config.SkipNonceCheck = json.Get("skip_nonce_check").Bool()
	for _, item := range json.Get("scopes").Array() {
		scopes := item.String()
		config.Scopes = append(config.Scopes, scopes)
	}
	parsedURL, err := url.Parse(config.Issuer)
	if err != nil {
		return errors.New("failed to parse issuer URL")
	}
	config.Path = parsedURL.Path

	timeout := json.Get("timeout_millis").Int()
	if timeout <= 0 {
		config.Timeout = 500
	} else {
		config.Timeout = int(timeout)
	}

	//cookie

	config.CookieSecret = oc.Set32Bytes(config.ClientSecret)
	config.CookieName = json.Get("cookie_name").String()
	if config.CookieName == "" {
		config.CookieName = "_oidc_wasm"
	}
	config.CookieDomain = json.Get("cookie_domain").String()
	if config.CookieDomain == "" {
		return errors.New("missing cookie_domain in config or err format")
	}
	config.CookiePath = json.Get("cookie_path").String()
	if config.CookiePath == "" {
		config.CookiePath = "/"
	}
	config.CookieSecure = json.Get("cookie_secure").Bool()
	config.CookieSecure = json.Get("cookie_httponly").Bool()

	config.CookieSameSite = json.Get("cookie_samesite").String()
	if config.CookieSameSite == "" {
		config.CookieSameSite = "Lax"
	}

	serviceSource := json.Get("service_source").String()
	serviceName := json.Get("service_name").String()
	servicePort := json.Get("service_port").Int()
	serviceHost := json.Get("service_host").String()
	if serviceName == "" || servicePort == 0 {
		return errors.New("invalid service config")
	}
	switch serviceSource {
	case "ip":
		config.Client = wrapper.NewClusterClient(&wrapper.StaticIpCluster{
			ServiceName: serviceName,
			Host:        serviceHost,
			Port:        servicePort,
		})
		log.Debugf("%v %v %v", serviceName, serviceHost, servicePort)
		return nil
	case "dns":
		domain := json.Get("service_domain").String()
		if domain == "" {
			return errors.New("missing service_domain in config")
		}
		config.Client = wrapper.NewClusterClient(&wrapper.DnsCluster{
			ServiceName: serviceName,
			Port:        servicePort,
			Domain:      domain,
		})
		return nil
	default:
		return errors.New("unknown service source: " + serviceSource)
	}
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config OidcConfig, log wrapper.Log) types.Action {

	defaultHandler := oc.NewDefaultOAuthHandler()
	cookieString, _ := proxywasm.GetHttpRequestHeader("cookie")
	oidcCookieValue, code, state, err := oc.GetParams(config.CookieName, cookieString, ctx.Path(), config.CookieSecret)
	if err != nil {
		oc.SendError(&log, fmt.Sprintf("GetParams err : %v", err), http.StatusBadRequest, "oidc.get_params_failed")
		return types.ActionContinue
	}
	nonce, _ := oc.Nonce(32)
	nonceStr := oc.GenState(nonce, config.ClientSecret, config.RedirectURL)
	createdAtTime := time.Now()
	cfg := &oc.Oatuh2Config{
		Config: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.RedirectURL,
			Scopes:       config.Scopes,
		},
		Issuer:          config.Issuer,
		ClientUrl:       config.ClientURL,
		Path:            config.Path,
		SkipExpiryCheck: config.SkipExpiryCheck,
		Timeout:         config.Timeout,
		Client:          config.Client,
		SkipNonceCheck:  config.SkipNonceCheck,
		Option:          &oc.OidcOption{},
		CookieOption: &oc.CookieOption{
			Name:     config.CookieName,
			Domain:   config.CookieDomain,
			Secret:   config.CookieSecret,
			Path:     config.CookiePath,
			SameSite: config.CookieSameSite,
			Secure:   config.CookieSecure,
			HTTPOnly: config.CookieHTTPOnly,
		},
		CookieData: &oc.CookieData{
			Nonce:     []byte(nonceStr),
			CreatedAt: createdAtTime,
		},
	}
	log.Debugf("path :%v host :%v state :%v code :%v cookie :%v", ctx.Path(), ctx.Host(), state, code, oidcCookieValue)

	if oidcCookieValue == "" {
		if code == "" {
			if err := defaultHandler.ProcessRedirect(&log, cfg); err != nil {
				oc.SendError(&log, fmt.Sprintf("ProcessRedirect error : %v", err), http.StatusInternalServerError, "oidc.process_redirect_failed")
				return types.ActionContinue
			}
			return types.ActionPause
		}
		if strings.Contains(ctx.Path(), OAUTH2CALLBACK) {
			parts := strings.Split(state, ".")
			if len(parts) != 2 {
				oc.SendError(&log, "State signature verification failed", http.StatusUnauthorized, "oidc.bad_state")
				return types.ActionContinue
			}
			stateVal, signature := parts[0], parts[1]
			if err := oc.VerifyState(stateVal, signature, cfg.ClientSecret, cfg.RedirectURL); err != nil {
				oc.SendError(&log, fmt.Sprintf("State signature verification failed : %v", err), http.StatusUnauthorized, "oidc.invalid_state")
				return types.ActionContinue
			}

			cfg.Option.Code = code
			cfg.Option.Mod = oc.SenBack
			if err := defaultHandler.ProcessExchangeToken(&log, cfg); err != nil {
				oc.SendError(&log, fmt.Sprintf("ProcessExchangeToken error : %v", err), http.StatusInternalServerError, "oidc.process_exchange_token_failed")
				return types.ActionContinue
			}
			return types.ActionPause
		}
		oc.SendError(&log, fmt.Sprintf("redirect URL must end with oauth2/callback"), http.StatusBadRequest, "oidc.bad_redirect_url")
		return types.ActionContinue
	}

	cookieData, err := oc.DeserializeCookieData(oidcCookieValue)
	if err != nil {
		oc.SendError(&log, fmt.Sprintf("DeserializeCookieData err : %v", err), http.StatusInternalServerError, "oidc.bad_cookie_value")
		return types.ActionContinue
	}

	cfg.CookieData = &oc.CookieData{
		IDToken:   cookieData.IDToken,
		Secret:    cfg.CookieOption.Secret,
		Nonce:     cookieData.Nonce,
		CreatedAt: cookieData.CreatedAt,
		ExpiresOn: cookieData.ExpiresOn,
	}
	cfg.Option.RawIdToken = cfg.CookieData.IDToken
	cfg.Option.Mod = oc.Access
	if err := defaultHandler.ProcessVerify(&log, cfg); err != nil {
		oc.SendError(&log, fmt.Sprintf("ProcessVerify error : %v", err), http.StatusUnauthorized, "oidc.unauthorized")
		return types.ActionContinue
	}

	return types.ActionPause
}
