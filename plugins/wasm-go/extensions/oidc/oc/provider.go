// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package oc

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/go-jose/go-jose/v3"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/gjson"
	"golang.org/x/oauth2"
)

var re = regexp.MustCompile("<[^>]*>")

// OidcHandler 定义了处理 OpenID Connect（OIDC）认证流程的方法集合。
// OIDC 是一个基于 OAuth 2.0 协议的身份验证和授权协议。
type OidcHandler interface {
	// ProcessRedirect 负责处理来自 OIDC 身份提供者的重定向响应。
	// 该方法会从openid-configuration中获取 authorization_endpoint，
	// 并确保其中的状态以及任何可能的错误代码都得到正确处理。
	ProcessRedirect(log *wrapper.Log, cfg *Oatuh2Config) error

	// ProcessExchangeToken 负责执行令牌交换过程。
	// 该方法会从 openid-configuration 中获取 token_endpoint 和 jwks_uri，
	// 然后使用授权码来交换 access token 和 ID token。
	ProcessExchangeToken(log *wrapper.Log, cfg *Oatuh2Config) error

	// ProcessVerify 负责验证 ID 令牌的有效性。
	// 通过使用 openid-configuration 中的获取的 jwks_uri 配置信息来验证 ID 令牌的签名和有效性。
	ProcessVerify(log *wrapper.Log, cfg *Oatuh2Config) error
}
type DefaultOAuthHandler struct {
}

func NewDefaultOAuthHandler() OidcHandler {
	return &DefaultOAuthHandler{}
}

func ProcessHTTPCall(log *wrapper.Log, cfg *Oatuh2Config, callback func(responseBody []byte)) error {
	wellKnownPath := strings.TrimSuffix(cfg.Path, "/") + "/.well-known/openid-configuration"
	if err := cfg.Client.Get(wellKnownPath, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		if err := ValidateHTTPResponse(statusCode, responseHeaders, responseBody); err != nil {
			cleanedBody := re.ReplaceAllString(string(responseBody), "")
			SendError(log, fmt.Sprintf("ValidateHTTPResponse failed , status : %v err : %v  err_info: %v ", statusCode, err, cleanedBody), statusCode, "oidc.bad_well_known_response")
			return
		}
		callback(responseBody)

	}, uint32(cfg.Timeout)); err != nil {
		return err
	}

	return nil
}

func (d *DefaultOAuthHandler) ProcessRedirect(log *wrapper.Log, cfg *Oatuh2Config) error {
	return ProcessHTTPCall(log, cfg, func(responseBody []byte) {
		state, _ := Nonce(32)
		statStr := GenState(state, cfg.ClientSecret, cfg.RedirectURL)
		cfg.Endpoint.AuthURL = gjson.ParseBytes(responseBody).Get("authorization_endpoint").String()
		if cfg.Endpoint.AuthURL == "" {
			SendError(log, "Missing 'authorization_endpoint' in the OpenID configuration response.", http.StatusInternalServerError, "oidc.auth_endpoint_missing")
			return
		}

		var opts oauth2.AuthCodeOption
		if !cfg.SkipNonceCheck {
			opts = SetNonce(string(cfg.CookieData.Nonce))
		}
		codeURL := cfg.AuthCodeURL(statStr, opts)
		proxywasm.SendHttpResponseWithDetail(http.StatusFound, "oidc.authed", [][2]string{
			{"Location", codeURL},
		}, nil, -1)
		return
	})
}

func (d *DefaultOAuthHandler) ProcessExchangeToken(log *wrapper.Log, cfg *Oatuh2Config) error {
	return ProcessHTTPCall(log, cfg, func(responseBody []byte) {

		PvRJson := gjson.ParseBytes(responseBody)
		cfg.Endpoint.TokenURL = PvRJson.Get("token_endpoint").String()
		if cfg.Endpoint.TokenURL == "" {
			SendError(log, "Missing 'token_endpoint' in the OpenID configuration response.", http.StatusInternalServerError, "oidc.token_endpoint_missing")
			return
		}
		cfg.JwksURL = PvRJson.Get("jwks_uri").String()
		if cfg.JwksURL == "" {
			SendError(log, "Missing 'jwks_uri' in the OpenID configuration response.", http.StatusInternalServerError, "oidc.jwks_uri_missing")
			return
		}
		cfg.Option.AuthStyle = AuthStyle(cfg.Endpoint.AuthStyle)

		if err := processToken(log, cfg); err != nil {
			SendError(log, fmt.Sprintf("ProcessToken failed : err %v", err), http.StatusInternalServerError, "oidc.process_token_failed")
			return
		}
	})
}

func (d *DefaultOAuthHandler) ProcessVerify(log *wrapper.Log, cfg *Oatuh2Config) error {
	return ProcessHTTPCall(log, cfg, func(responseBody []byte) {
		PvRJson := gjson.ParseBytes(responseBody)

		cfg.JwksURL = PvRJson.Get("jwks_uri").String()
		if cfg.JwksURL == "" {
			SendError(log, "Missing 'token_endpoint' in the OpenID configuration response.", http.StatusInternalServerError, "oidc.token_endpoint_missing")
			return
		}
		var algs []string
		for _, a := range PvRJson.Get("id_token_signing_alg_values_supported").Array() {
			if SupportedAlgorithms[a.String()] {
				algs = append(algs, a.String())
			}
		}
		cfg.SupportedSigningAlgs = algs
		if err := processTokenVerify(log, cfg); err != nil {
			SendError(log, fmt.Sprintf("failed to verify token: %v", err), http.StatusInternalServerError, "oidc.verify_token_failed")
			return
		}
	})
}

func processToken(log *wrapper.Log, cfg *Oatuh2Config) error {
	parsedURL, err := url.Parse(cfg.Endpoint.TokenURL)
	if err != nil {
		return fmt.Errorf("invalid TokenURL: %v", err)
	}

	var token Token
	urlVales := ReturnURL(cfg.RedirectURL, cfg.Option.Code)
	needsAuthStyleProbe := cfg.Option.AuthStyle == AuthStyleUnknown
	if needsAuthStyleProbe {
		if style, ok := LookupAuthStyle(cfg.Endpoint.TokenURL); ok {
			cfg.Option.AuthStyle = style
		} else {
			cfg.Option.AuthStyle = AuthStyleInHeader
		}
	}

	headers, body, err := NewTokenRequest(cfg.Endpoint.TokenURL, cfg.ClientID, cfg.ClientSecret, urlVales, cfg.Option.AuthStyle)
	cb := func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		if err := ValidateHTTPResponse(statusCode, responseHeaders, responseBody); err != nil {
			cleanedBody := re.ReplaceAllString(string(responseBody), "")
			SendError(log, fmt.Sprintf("Valid failed , status : %v err : %v  err_info: %v ", statusCode, err, cleanedBody), statusCode, "oidc.bad_token_response")
			return
		}

		tk, err := UnmarshalToken(&token, responseHeaders, responseBody)
		if err != nil {
			SendError(log, fmt.Sprintf("UnmarshalToken error: %v", err), http.StatusInternalServerError, "oidc.extract_token_failed")
			return
		}

		if tk != nil && token.RefreshToken == "" {
			token.RefreshToken = urlVales.Get("refresh_token")
		}

		betoken := TokenFromInternal(tk)

		rawIDToken, ok := betoken.Extra("id_token").(string)
		if !ok {
			SendError(log, fmt.Sprintf("No id_token field in oauth2 token."), http.StatusInternalServerError, "oidc.id_token_missing")
			return
		}
		cfg.Option.RawIdToken = rawIDToken

		err = processTokenVerify(log, cfg)
		if err != nil {
			SendError(log, fmt.Sprintf("failed to verify token: %v", err), http.StatusInternalServerError, "oidc.verify_token_failed")
			return
		}

	}

	err = cfg.Client.Post(parsedURL.Path, headers, body, cb, uint32(cfg.Timeout))
	if err != nil {
		return fmt.Errorf("HTTP POST error: %v", err)
	}

	return nil
}

func processTokenVerify(log *wrapper.Log, cfg *Oatuh2Config) error {
	keySet := jose.JSONWebKeySet{}
	idTokenVerify := cfg.Verifier(&IDConfig{
		ClientID:             cfg.ClientID,
		SupportedSigningAlgs: cfg.SupportedSigningAlgs,
		SkipExpiryCheck:      cfg.SkipExpiryCheck,
		SkipNonceCheck:       cfg.SkipNonceCheck,
	})

	defaultHandlerForRedirect := NewDefaultOAuthHandler()
	parsedURL, err := url.Parse(cfg.JwksURL)
	if err != nil {
		log.Errorf("JwksURL is invalid  err : %v", err)
		return err
	}
	cb := func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		if err := ValidateHTTPResponse(statusCode, responseHeaders, responseBody); err != nil {
			cleanedBody := re.ReplaceAllString(string(responseBody), "")
			SendError(log, fmt.Sprintf("Valid failed , status : %v err : %v  err_info: %v ", statusCode, err, cleanedBody), statusCode, "oidc.bad_validate_response")
			return
		}

		res := gjson.ParseBytes(responseBody)
		for _, val := range res.Get("keys").Array() {
			jsw, err := GenJswkey(val)
			if err != nil {
				log.Errorf("err: %v", err)
				SendError(log, fmt.Sprintf("GenJswkey error:%v", err), http.StatusInternalServerError, "oidc.gen_jsw_key_failed")
				return
			}
			keySet.Keys = append(keySet.Keys, *jsw)
		}
		idtoken, err := idTokenVerify.VerifyToken(cfg.Option.RawIdToken, keySet)

		if err != nil {
			log.Errorf("VerifyToken err : %v ", err)
			defaultHandlerForRedirect.ProcessRedirect(log, cfg)
			return
		}
		if !cfg.SkipNonceCheck && Access == cfg.Option.Mod {
			err := verifyNonce(idtoken, cfg)
			if err != nil {
				log.Error("VerifyNonce failed")
				defaultHandlerForRedirect.ProcessRedirect(log, cfg)
				return
			}
		}

		//回发和放行
		if cfg.Option.Mod == Access {
			proxywasm.AddHttpRequestHeader("Authorization", "Bearer "+cfg.Option.RawIdToken)
			proxywasm.ResumeHttpRequest()
			return
		}

		cfg.CookieOption.Expire = idtoken.Expiry
		cfg.CookieData.IDToken = cfg.Option.RawIdToken
		cfg.CookieData.ExpiresOn = idtoken.Expiry
		cfg.CookieData.Secret = cfg.CookieOption.Secret

		cookieHeader, err := SerializeAndEncryptCookieData(cfg.CookieData, cfg.CookieOption.Secret, cfg.CookieOption)
		if err != nil {
			SendError(log, fmt.Sprintf("SerializeAndEncryptCookieData failed : %v", err), http.StatusInternalServerError, "oidc.gen_cookie_failed")
			return
		}
		proxywasm.SendHttpResponseWithDetail(http.StatusFound, "oidc.token_verified", [][2]string{
			{"Location", cfg.ClientUrl},
			{"Set-Cookie", cookieHeader},
		}, nil, -1)

		return
	}

	if err := cfg.Client.Get(parsedURL.Path, nil, cb, uint32(cfg.Timeout)); err != nil {
		log.Errorf("client.Get error: %v", err)
		return err
	}
	return nil
}

func verifyNonce(idtoken *IDToken, cfg *Oatuh2Config) error {
	parts := strings.Split(idtoken.Nonce, ".")
	if len(parts) != 2 {
		return errors.New("nonce format err expect 2 parts")
	}
	stateval, signature := parts[0], parts[1]
	return VerifyState(stateval, signature, cfg.ClientSecret, cfg.RedirectURL)
}
