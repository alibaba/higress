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
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"golang.org/x/oauth2"
)

type OidcConfig struct {
	Issuer          string
	Path            string
	ClientID        string
	ClientSecret    string
	RedirectURL     string
	ClientUrl       string
	Timeout         int
	CookieName      string
	clientDomain    string
	CookieSecret    string
	CookieDomain    string
	CookiePath      string
	CookieSameSite  string
	CookieSecure    bool
	CookieHTTPOnly  bool
	Scopes          []string
	SkipExpiryCheck bool
	SkipNonceCheck  bool
	SecuceCookie    bool
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
		return errors.New("missing Issuer in config")
	}

	config.ClientID = json.Get("clientId").String()
	if config.ClientID == "" {
		return errors.New("missing clientID in config")
	}

	config.ClientSecret = json.Get("clientSecret").String()
	if config.ClientSecret == "" {
		return errors.New("missing clientSecret in config")
	}
	config.ClientUrl = json.Get("clientUrl").String()
	_, err := url.ParseRequestURI(config.ClientUrl)
	if err != nil {
		return errors.New("missing clientUrl in config or err format")
	}

	oc.IsValidRedirect(json.Get("redirectUrl").String())
	if err != nil {
		return err
	}
	config.RedirectURL = json.Get("redirectUrl").String()

	config.SkipExpiryCheck = json.Get("skipExpiryCheck ").Bool()
	config.SkipNonceCheck = json.Get("skipNonceCheck").Bool()
	for _, item := range json.Get("scopes").Array() {
		scopes := item.String()
		config.Scopes = append(config.Scopes, scopes)
	}
	parsedURL, err := url.Parse(config.Issuer)
	if err != nil {
		return errors.New("failed to parse issuer URL")
	}
	config.Path = parsedURL.Path

	timeout := json.Get("timeOut").Int()
	if timeout <= 0 {
		config.Timeout = 500
	} else {
		config.Timeout = int(timeout)
	}

	//cookie

	config.CookieSecret = oc.Set32Bytes(config.ClientSecret)
	config.CookieName = json.Get("CookieName").String()
	if config.CookieName == "" {
		config.CookieName = "_oauth2_wasm"
	}
	config.CookieDomain = json.Get("cookieDomain").String()
	if config.CookieDomain == "" {
		return errors.New("missing CookieDomain in config or err format")
	}
	config.CookiePath = json.Get("cookiePath").String()
	if config.CookiePath == "" {
		config.CookiePath = "/"
	}
	config.CookieSecure = json.Get("cookieSecure").Bool()
	config.CookieSecure = json.Get("cookieHttponly").Bool()

	config.CookieSameSite = json.Get("cookieSamesite").String()
	if config.CookieSameSite == "" {
		config.CookieSameSite = "Lax"
	}

	serviceSource := json.Get("serviceSource").String()
	serviceName := json.Get("serviceName").String()
	servicePort := json.Get("servicePort").Int()
	serviceHost := json.Get("serviceHost").String()
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
		domain := json.Get("domain").String()
		if domain == "" {
			return errors.New("missing domain in config")
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

	DefaultHandler := oc.NewDefaultOAuthHandler()
	cookieString, _ := proxywasm.GetHttpRequestHeader("cookie")
	oidcCookieValue, code, state, err := oc.GetParams(config.CookieName, cookieString, ctx.Path(), config.CookieSecret)
	if err != nil {
		oc.SendError(&log, fmt.Sprintf("GetParams err : %v", err), http.StatusBadRequest)
		return types.ActionPause
	}
	nonce, _ := oc.Nonce(32)
	nonceStr := oc.GenState(nonce, config.ClientSecret, config.RedirectURL)
	tm := time.Now()
	cfg := &oc.Oatuh2Config{
		Config: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.RedirectURL,
			Scopes:       config.Scopes,
		},
		Issuer:          config.Issuer,
		ClientUrl:       config.ClientUrl,
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
			Secure:   config.CookieSecure,
			HTTPOnly: config.CookieHTTPOnly,
			SameSite: config.CookieSameSite,
		},
		CookieData: &oc.CookieData{
			Nonce:     []byte(nonceStr),
			CreatedAt: tm,
		},
	}
	log.Debugf("path :%v host :%v state :%v code :%v cookie :%v", ctx.Path(), ctx.Host(), state, code, oidcCookieValue)

	if oidcCookieValue == "" {

		if code == "" {
			if err := DefaultHandler.ProcessRedirect(&log, cfg); err != nil {
				oc.SendError(&log, fmt.Sprintf("Redirect error : %v", err), http.StatusInternalServerError)
				return types.ActionPause
			}
		}

		if strings.Contains(ctx.Path(), "oauth2/callback") {
			parts := strings.Split(state, ".")
			if len(parts) != 2 {
				oc.SendError(&log, "State signature verification failed", http.StatusUnauthorized)
				return types.ActionPause

			}

			stateval, signature := parts[0], parts[1]
			if err := oc.VerifyState(stateval, signature, cfg.ClientSecret, cfg.RedirectURL); err != nil {
				oc.SendError(&log, fmt.Sprintf("State signature verification failed : %v", err), http.StatusUnauthorized)
				return types.ActionPause

			}

			cfg.Option.Code = code
			cfg.Option.Mod = oc.SenBack
			if err := DefaultHandler.ProcessExchangeToken(&log, cfg); err != nil {
				oc.SendError(&log, fmt.Sprintf("ProcessExchangeToken error : %v", err), http.StatusInternalServerError)
				return types.ActionPause
			}
		}
	} else {

		cookiedata, err := oc.DeserializedeCookieData(oidcCookieValue)
		if err != nil {
			log.Errorf("DeserializedeCookieData err : %v", err)
			if err := DefaultHandler.ProcessRedirect(&log, cfg); err != nil {
				oc.SendError(&log, fmt.Sprintf("Redirect error : %v", err), http.StatusInternalServerError)
				return types.ActionPause
			}
		}

		cfg.CookieData = &oc.CookieData{
			IDToken:   cookiedata.IDToken,
			Secret:    cfg.CookieOption.Secret,
			Nonce:     cookiedata.Nonce,
			CreatedAt: cookiedata.CreatedAt,
			ExpiresOn: cookiedata.ExpiresOn,
		}
		cfg.Option.RawIdToken = cfg.CookieData.IDToken
		cfg.Option.Mod = oc.Access
		if err := DefaultHandler.ProcessVerify(&log, cfg); err != nil {
			oc.SendError(&log, fmt.Sprintf("ProcessVerify error : %v", err), http.StatusUnauthorized)
			return types.ActionPause
		}
	}

	return types.ActionPause
}
