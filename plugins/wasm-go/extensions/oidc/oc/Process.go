package oc

import (
	"errors"
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
		if err := validateHTTPResponse(statusCode, responseHeaders, responseBody); err != nil {
			SendError(log, fmt.Sprintf("HTTP response validation failed : %v %v ", statusCode, err), statusCode)
			return
		}

		callback(responseBody)
	}, 2000); err != nil {
		return err
	}

	return nil
}

// validateHTTPResponse
func validateHTTPResponse(statusCode int, headers http.Header, body []byte) error {
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
			if supportedAlgorithms[a.String()] {
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
		if style, ok := lookupAuthStyle(cfg.Endpoint.TokenURL); ok {
			authStyle = style
		} else {
			authStyle = AuthStyleInHeader
		}
	}

	headers, body, err := newTokenRequest(cfg.Endpoint.TokenURL, cfg.ClientID, cfg.ClientSecret, v, authStyle)

	if err != nil {
		return fmt.Errorf("failed to create token request: %v", err)
	}

	cb := func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		err = validateHTTPResponse(statusCode, responseHeaders, responseBody)
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
			setAuthStyle(cfg.Endpoint.TokenURL, authStyle)
		}

		if tk != nil && token.RefreshToken == "" {
			token.RefreshToken = v.Get("refresh_token")
		}

		betoken := tokenFromInternal(tk)

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

	err = cfg.Client.Post(parsedURL.Path, headers, body, cb, 2000)
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
		if err := validateHTTPResponse(statusCode, responseHeaders, responseBody); err != nil {
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

		_, err := idTokenVerify.VerifyToken(rawIdToken, keySet, log)

		if err != nil {
			log.Errorf("VerifyToken err : %v ", err)
			StatStr := GenState()
			nonce := GenState()
			ProcessRedirect(StatStr, nonce, cfg, log)
			return
		}
		//回发和放行
		if mod == Access {
			proxywasm.ResumeHttpRequest()
			return
		} else if mod == SenBack {
			proxywasm.SendHttpResponse(http.StatusOK, nil, []byte(rawIdToken), -1)
			proxywasm.ResumeHttpRequest()
			return
		}

	}

	if err := cfg.Client.Get(parsedURL.Path, nil, cb, 2000); err != nil {
		log.Errorf("client.Get error: %v", err)
		return err
	}
	return nil
}

func SendError(log *wrapper.Log, errMsg string, status int) {
	log.Errorf(errMsg)
	proxywasm.SendHttpResponse(uint32(status), nil, []byte(errMsg), -1)
}
