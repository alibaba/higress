package oc

import (
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/go-jose/go-jose/v3"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/gjson"
)

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
			re := regexp.MustCompile("<[^>]*>")
			cleanedBody := re.ReplaceAllString(string(responseBody), "")
			SendError(log, fmt.Sprintf("Valid failed , status : %v err : %v \n err_info: %v ", statusCode, err, string(cleanedBody)), statusCode)
			return
		}
		callback(responseBody)
	}, 2000); err != nil {
		return err
	}

	return nil
}

func (d *DefaultOAuthHandler) ProcessRedirect(log *wrapper.Log, cfg *Oatuh2Config) error {
	return ProcessHTTPCall(log, cfg, func(responseBody []byte) {
		StatStr, nonce := GenState(), GenState()

		cfg.Endpoint.AuthURL = gjson.ParseBytes(responseBody).Get("authorization_endpoint").String()
		if cfg.Endpoint.AuthURL == "" {
			SendError(log, " Miss authorization_endpoint ", http.StatusInternalServerError)
		}
		codeURL := cfg.AuthCodeURL(StatStr, SetNonce(nonce))
		err := proxywasm.SendHttpResponse(http.StatusFound, [][2]string{
			{"Location", codeURL},
		}, nil, -1)
		if err != nil {
			log.Errorf("error sending redirect response: %v", err)
			return
		}
	})
}

func (d *DefaultOAuthHandler) ProcessExchangeToken(log *wrapper.Log, cfg *Oatuh2Config) error {
	return ProcessHTTPCall(log, cfg, func(responseBody []byte) {
		PvRJson := gjson.ParseBytes(responseBody)

		cfg.Endpoint.TokenURL = PvRJson.Get("token_endpoint").String()
		if cfg.Endpoint.TokenURL == "" {
			SendError(log, " Miss token_endpoint ", http.StatusInternalServerError)
		}
		cfg.JwksURL = PvRJson.Get("jwks_uri").String()
		if cfg.JwksURL == "" {
			SendError(log, " Miss jwks uri ", http.StatusInternalServerError)
		}
		cfg.Option.AuthStyle = AuthStyle(cfg.Endpoint.AuthStyle)
		if err := d.ProcessToken(log, cfg); err != nil {
			log.Errorf("failed to process token: %v", err)
		}
	})
}

func (d *DefaultOAuthHandler) ProcessVerify(log *wrapper.Log, cfg *Oatuh2Config) error {
	return ProcessHTTPCall(log, cfg, func(responseBody []byte) {
		PvRJson := gjson.ParseBytes(responseBody)

		cfg.JwksURL = PvRJson.Get("jwks_uri").String()
		if cfg.JwksURL == "" {
			SendError(log, " Miss jwks uri ", http.StatusInternalServerError)
		}
		var algs []string
		for _, a := range PvRJson.Get("id_token_signing_alg_values_supported").Array() {
			if SupportedAlgorithms[a.String()] {
				algs = append(algs, a.String())
			}
		}
		cfg.SupportedSigningAlgs = algs
		err := d.ProcesTokenVerify(log, cfg)
		if err != nil {
			log.Errorf("failed to verify token: %v", err)
		}
	})
}

func (d *DefaultOAuthHandler) ProcessToken(log *wrapper.Log, cfg *Oatuh2Config) error {
	parsedURL, err := url.Parse(cfg.Endpoint.TokenURL)
	if err != nil {
		return fmt.Errorf("invalid TokenURL: %v", err)
	}

	var token Token
	v := ReturnURL(cfg.RedirectURL, cfg.Option.Code)
	needsAuthStyleProbe := cfg.Option.AuthStyle == AuthStyleUnknown
	if needsAuthStyleProbe {
		if style, ok := LookupAuthStyle(cfg.Endpoint.TokenURL); ok {
			cfg.Option.AuthStyle = style
		} else {
			cfg.Option.AuthStyle = AuthStyleInHeader
		}
	}

	headers, body, err := NewTokenRequest(cfg.Endpoint.TokenURL, cfg.ClientID, cfg.ClientSecret, v, cfg.Option.AuthStyle)

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
			d.ProcessToken(log, cfg)
			return
		}

		tk, err := UnmarshalToken(&token, responseHeaders, responseBody)
		if err != nil {
			SendError(log, fmt.Sprintf("UnmarshalToken error: %v", err), http.StatusInternalServerError)
			return
		}

		if needsAuthStyleProbe && err == nil {
			SetAuthStyle(cfg.Endpoint.TokenURL, cfg.Option.AuthStyle)
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
		cfg.Option.RawIdToken = rawIDToken
		err = d.ProcesTokenVerify(log, cfg)
		if err != nil {
			log.Errorf("failed to verify token: %v", err)
		}

	}

	err = cfg.Client.Post(parsedURL.Path, headers, body, cb, 2000)
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

		idtoken, err := idTokenVerify.VerifyToken(cfg.Option.RawIdToken, keySet)

		if err != nil {
			log.Errorf("VerifyToken err : %v", err)
			d.ProcessRedirect(log, cfg)
			return
		}

		//回发和放行
		if cfg.Option.Mod == Access {
			proxywasm.AddHttpRequestHeader("X-MSE-IDToken", cfg.Option.RawIdToken)
			proxywasm.ResumeHttpRequest()
			return
		}

		cookieHeader, _ := buildSecureCookieHeader(cfg.Option.RawIdToken, cfg.Clientdomain, idtoken.Expiry, cfg.SecureCookie)
		scheme := "http://"
		if cfg.SecureCookie == true {
			scheme = "https://"
		}
		log.Debugf("set cookie")
		proxywasm.SendHttpResponse(http.StatusFound, [][2]string{
			{"Set-Cookie", cookieHeader},
			{"Location", scheme + cfg.Clientdomain},
		}, nil, -1)

		return
	}

	if err := cfg.Client.Get(parsedURL.Path, nil, cb, 2000); err != nil {
		log.Errorf("client.Get error: %v", err)
		return err
	}
	return nil
}
