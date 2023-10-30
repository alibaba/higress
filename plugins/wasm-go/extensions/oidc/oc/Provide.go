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
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/go-jose/go-jose/v3"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/gjson"
	"golang.org/x/oauth2"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

var re = regexp.MustCompile("<[^>]*>")

type OidcHandler interface {
	ProcessRedirect(log *wrapper.Log, cfg *Oatuh2Config) error
	ProcessExchangeToken(log *wrapper.Log, cfg *Oatuh2Config) error
	ProcessVerify(log *wrapper.Log, cfg *Oatuh2Config) error
	ProcessToken(log *wrapper.Log, cfg *Oatuh2Config) error
	ProcesTokenVerify(log *wrapper.Log, cfg *Oatuh2Config) error
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
			SendError(log, fmt.Sprintf("Valid failed , status : %v err : %v  err_info: %v ", statusCode, err, cleanedBody), statusCode)
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
			SendError(log, "Missing 'authorization_endpoint' in the OpenID configuration response.", http.StatusInternalServerError)
			return
		}

		var opts oauth2.AuthCodeOption
		if !cfg.SkipNonceCheck {
			opts = SetNonce(string(cfg.CookieData.Nonce))
		}
		codeURL := cfg.AuthCodeURL(statStr, opts)
		proxywasm.SendHttpResponse(http.StatusFound, [][2]string{
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
			SendError(log, "Missing 'token_endpoint' in the OpenID configuration response.", http.StatusInternalServerError)
			return
		}
		cfg.JwksURL = PvRJson.Get("jwks_uri").String()
		if cfg.JwksURL == "" {
			SendError(log, "Missing 'jwks_uri' in the OpenID configuration response.", http.StatusInternalServerError)
			return
		}
		cfg.Option.AuthStyle = AuthStyle(cfg.Endpoint.AuthStyle)

		if err := d.ProcessToken(log, cfg); err != nil {
			SendError(log, fmt.Sprintf("ProcessToken failed : err %v", err), http.StatusInternalServerError)
			return
		}
	})
}

func (d *DefaultOAuthHandler) ProcessVerify(log *wrapper.Log, cfg *Oatuh2Config) error {
	return ProcessHTTPCall(log, cfg, func(responseBody []byte) {
		PvRJson := gjson.ParseBytes(responseBody)

		cfg.JwksURL = PvRJson.Get("jwks_uri").String()
		if cfg.JwksURL == "" {
			SendError(log, "Missing 'token_endpoint' in the OpenID configuration response.", http.StatusInternalServerError)
			return
		}
		var algs []string
		for _, a := range PvRJson.Get("id_token_signing_alg_values_supported").Array() {
			if SupportedAlgorithms[a.String()] {
				algs = append(algs, a.String())
			}
		}
		cfg.SupportedSigningAlgs = algs
		if err := d.ProcesTokenVerify(log, cfg); err != nil {
			log.Errorf("failed to verify token: %v", err)
			return
		}
	})
}

func (d *DefaultOAuthHandler) ProcessToken(log *wrapper.Log, cfg *Oatuh2Config) error {
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
			SendError(log, fmt.Sprintf("Valid failed , status : %v err : %v  err_info: %v ", statusCode, err, cleanedBody), statusCode)
			return
		}

		if err != nil && needsAuthStyleProbe {
			log.Errorf("Incorrect invocation, retrying with different auth style")
			d.ProcessToken(log, cfg)
			return
		}

		tk, err := UnmarshalToken(&token, responseHeaders, responseBody)
		if err != nil {
			SendError(log, fmt.Sprintf("UnmarshalToken error: %v", err), http.StatusInternalServerError)
			return
		}

		if tk != nil && token.RefreshToken == "" {
			token.RefreshToken = urlVales.Get("refresh_token")
		}

		betoken := TokenFromInternal(tk)

		rawIDToken, ok := betoken.Extra("id_token").(string)
		if !ok {
			SendError(log, fmt.Sprintf("No id_token field in oauth2 token."), http.StatusInternalServerError)
			return
		}
		cfg.Option.RawIdToken = rawIDToken
		//todo
		err = d.ProcesTokenVerify(log, cfg)
		if err != nil {
			log.Errorf("failed to verify token: %v", err)
			return
		}

	}

	err = cfg.Client.Post(parsedURL.Path, headers, body, cb, uint32(cfg.Timeout))
	if err != nil {
		return fmt.Errorf("HTTP POST error: %v", err)
	}

	return nil
}

func (d *DefaultOAuthHandler) ProcesTokenVerify(log *wrapper.Log, cfg *Oatuh2Config) error {
	keySet := jose.JSONWebKeySet{}
	idTokenVerify := cfg.Verifier(&IDConfig{
		ClientID:             cfg.ClientID,
		SupportedSigningAlgs: cfg.SupportedSigningAlgs,
		SkipExpiryCheck:      cfg.SkipExpiryCheck,
		SkipNonceCheck:       cfg.SkipNonceCheck,
	})
	parsedURL, err := url.Parse(cfg.JwksURL)
	if err != nil {
		log.Errorf("JwksURL is invalid  err : %v", err)
		return err
	}

	cb := func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		if err := ValidateHTTPResponse(statusCode, responseHeaders, responseBody); err != nil {
			cleanedBody := re.ReplaceAllString(string(responseBody), "")
			SendError(log, fmt.Sprintf("Valid failed , status : %v err : %v  err_info: %v ", statusCode, err, cleanedBody), statusCode)
			return
		}

		res := gjson.ParseBytes(responseBody)
		for _, val := range res.Get("keys").Array() {
			jws, err := GenJswkey(val)
			if err != nil {
				log.Errorf("err: %v", err)
				SendError(log, fmt.Sprintf("GenJswkey error:%v", err), http.StatusInternalServerError)
				return
			}
			keySet.Keys = append(keySet.Keys, *jws)
		}
		idtoken, err := idTokenVerify.VerifyToken(cfg.Option.RawIdToken, keySet)

		if err != nil {
			log.Errorf("VerifyToken err : %v", err)
			d.ProcessRedirect(log, cfg)
			return
		}
		if !cfg.SkipNonceCheck && Access == cfg.Option.Mod {
			parts := strings.Split(idtoken.Nonce, ".")
			if len(parts) != 2 {
				SendError(log, "Nonce format err expect 2 parts", http.StatusUnauthorized)
				return
			}
			stateval, signature := parts[0], parts[1]
			err := VerifyState(stateval, signature, cfg.ClientSecret, cfg.RedirectURL)
			if err != nil {
				log.Errorf(" VerifyNonce failed : %v", err)
				d.ProcessRedirect(log, cfg)
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
			SendError(log, fmt.Sprintf("SerializeAndEncryptCookieData failed : %v", err), http.StatusInternalServerError)
			return
		}
		proxywasm.SendHttpResponse(http.StatusFound, [][2]string{
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
