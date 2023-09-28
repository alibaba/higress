package oc

import (
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	jose "github.com/go-jose/go-jose/v3"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/gjson"
	"net/http"
	"net/url"
	"strings"
)

// ProcessHTTPCall
func ProcessHTTPCall(cfg *Oatuh2Config, log *wrapper.Log, callback func(responseBody []byte)) error {
	wellKnownPath := strings.TrimSuffix(cfg.Path, "/") + "/.well-known/openid-configuration"

	if err := cfg.Client.Get(wellKnownPath, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		if err := ValidateHTTPResponse(statusCode, responseHeaders, responseBody); err != nil {
			SendError(log, fmt.Sprintf("HTTP response validation failed : %v %v ", statusCode, err), statusCode)
			return
		}

		callback(responseBody)
	}, cfg.Timeout); err != nil {
		return err
	}

	return nil
}

// ProcessRedirect
func ProcessRedirect(stateStr, nonce string, cfg *Oatuh2Config, log *wrapper.Log) error {
	return ProcessHTTPCall(cfg, log, func(responseBody []byte) {

		cfg.Endpoint.AuthURL = gjson.ParseBytes(responseBody).Get("authorization_endpoint").String()
		if cfg.Endpoint.AuthURL == "" {
			SendError(log, " Miss authorization_endpoint ", http.StatusInternalServerError)
		}
		codeURL := cfg.AuthCodeURL(stateStr, SetNonce(nonce))
		err := proxywasm.SendHttpResponse(http.StatusFound, [][2]string{
			{"Location", codeURL},
		}, nil, -1)
		if err != nil {
			log.Errorf("error sending redirect response: %v", err)
			return
		}
	})

}

// ProcessExchangeToken
func ProcessExchangeToken(code string, cfg *Oatuh2Config, log *wrapper.Log, mod Accessmod) error {
	return ProcessHTTPCall(cfg, log, func(responseBody []byte) {
		PvRJson := gjson.ParseBytes(responseBody)

		cfg.Endpoint.TokenURL = PvRJson.Get("token_endpoint").String()
		if cfg.Endpoint.TokenURL == "" {
			SendError(log, " Miss token_endpoint ", http.StatusInternalServerError)
		}
		cfg.JwksURL = PvRJson.Get("jwks_uri").String()
		if cfg.JwksURL == "" {
			SendError(log, " Miss jwks_uri ", http.StatusInternalServerError)
		}
		authStyle := AuthStyle(cfg.Endpoint.AuthStyle)
		if err := ProcessToken(code, cfg, log, authStyle, mod); err != nil {
			log.Errorf("failed to process token: %v", err)
		}
	})

}

// ProcessVerify
func ProcessVerify(rawToken string, cfg *Oatuh2Config, log *wrapper.Log, mod Accessmod) error {
	return ProcessHTTPCall(cfg, log, func(responseBody []byte) {
		PvRJson := gjson.ParseBytes(responseBody)

		cfg.JwksURL = PvRJson.Get("jwks_uri").String()
		if cfg.JwksURL == "" {
			SendError(log, " Miss jwks_uri ", http.StatusInternalServerError)
		}

		var algs []string
		for _, a := range PvRJson.Get("id_token_signing_alg_values_supported").Array() {
			if SupportedAlgorithms[a.String()] {
				algs = append(algs, a.String())
			}
		}
		cfg.SupportedSigningAlgs = algs
		err := ProcesTokenVerify(rawToken, cfg, log, mod)
		if err != nil {
			log.Errorf("failed to verify token: %v", err)
		}
	})

}

func ProcessToken(code string, cfg *Oatuh2Config, log *wrapper.Log, authStyle AuthStyle, mod Accessmod) error {
	parsedURL, err := url.Parse(cfg.Endpoint.TokenURL)
	if err != nil {
		return fmt.Errorf("invalid TokenURL: %v", err)
	}

	var token Token
	v := ReturnURL(cfg.RedirectURL, code)
	needsAuthStyleProbe := authStyle == AuthStyleUnknown
	if needsAuthStyleProbe {
		if style, ok := LookupAuthStyle(cfg.Endpoint.TokenURL); ok {
			authStyle = style
		} else {
			authStyle = AuthStyleInHeader
		}
	}

	headers, body, err := NewTokenRequest(cfg.Endpoint.TokenURL, cfg.ClientID, cfg.ClientSecret, v, authStyle)

	if err != nil {
		return fmt.Errorf("failed to create token request: %v", err)
	}

	cb := func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		err = ValidateHTTPResponse(statusCode, responseHeaders, responseBody)
		if err != nil {
			log.Errorf("validateHTTPResponse err: %v", err)
		}
		log.Debugf("body %v", string(responseBody))
		if err != nil && needsAuthStyleProbe {
			log.Errorf("Incorrect invocation, retrying with different auth style")
			ProcessToken(code, cfg, log, AuthStyleInParams, mod)
			return
		}

		tk, err := UnmarshalToken(&token, responseHeaders, responseBody)
		if err != nil {
			SendError(log, fmt.Sprintf("UnmarshalToken error: %v", err), http.StatusInternalServerError)
			return
		}

		if needsAuthStyleProbe && err == nil {
			SetAuthStyle(cfg.Endpoint.TokenURL, authStyle)
		}

		if tk != nil && token.RefreshToken == "" {
			token.RefreshToken = v.Get("refresh_token")
		}

		betoken := TokenFromInternal(tk)

		rawIDToken, ok := betoken.Extra("id_token").(string)
		if !ok {
			log.Errorf("No id_token field in oauth2 token.")

			return
		}
		err = ProcesTokenVerify(rawIDToken, cfg, log, mod)
		if err != nil {
			log.Errorf("failed to verify token: %v", err)
		}

	}

	err = cfg.Client.Post(parsedURL.Path, headers, body, cb, cfg.Timeout)
	if err != nil {
		return fmt.Errorf("HTTP POST error: %v", err)
	}

	return nil
}

// ProcesTokenVerify
func ProcesTokenVerify(rawIdToken string, cfg *Oatuh2Config, log *wrapper.Log, mod Accessmod) error {
	keySet := jose.JSONWebKeySet{}

	idTokenVerify := cfg.Verifier(&IDConfig{
		ClientID:             cfg.ClientID,
		SupportedSigningAlgs: cfg.SupportedSigningAlgs,
		SkipExpiryCheck:      cfg.SkipIssuerCheck,
	})

	parsedURL, err := url.Parse(cfg.JwksURL)
	if err != nil {
		log.Errorf("JwksURL is invalid  err : %v", err)
		return err
	}
	cb := func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		if err := ValidateHTTPResponse(statusCode, responseHeaders, responseBody); err != nil {
			errMsg := fmt.Sprintf("HTTP response validation failed: %v", err)
			log.Errorf(errMsg)
			proxywasm.SendHttpResponse(uint32(statusCode), nil, []byte(errMsg), -1)
			return
		}

		res := gjson.ParseBytes(responseBody)
		for _, val := range res.Get("keys").Array() {
			jws, err := GenJswkey(val)
			if err != nil {
				log.Errorf("err: %v", err)
				return
			}
			keySet.Keys = append(keySet.Keys, *jws)
		}

		idtoken, err := idTokenVerify.VerifyToken(rawIdToken, keySet, log)

		if err != nil {
			log.Errorf("VerifyToken err : %v", err)
			state := GenState()
			Nonce := GenState()
			ProcessRedirect(state, Nonce, cfg, log)
			return
		}

		//回发和放行
		if mod == Access {
			proxywasm.AddHttpRequestHeader("X-Authorization-IDToken", rawIdToken)
			proxywasm.ResumeHttpRequest()
			return
		} else if mod == SenBack {
			cookieHeader, _ := buildSecureCookieHeader(rawIdToken, cfg.Clientdomain, idtoken.Expiry, cfg.SecureCookie)
			scheme := "http://"
			if cfg.SecureCookie == true {
				scheme = "https://"
			}
			proxywasm.SendHttpResponse(http.StatusFound, [][2]string{
				{"Set-Cookie", cookieHeader},
				{"Location", scheme + cfg.Clientdomain},
			}, nil, -1)

			return
		}

	}

	if err := cfg.Client.Get(parsedURL.Path, nil, cb, cfg.Timeout); err != nil {
		log.Errorf("client.Get error: %v", err)
		return err
	}
	return nil
}
