package oc

import (
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"golang.org/x/oauth2"
)

type Accessmod int

const (
	Access  Accessmod = 0
	SenBack Accessmod = 1
)

type IDConfig struct {
	ClientID             string
	SupportedSigningAlgs []string
	SkipExpiryCheck      bool
	Now                  func() time.Time
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
	Path                 string
	Clientdomain         string
	SupportedSigningAlgs []string
	SkipIssuerCheck      bool
	SecureCookie         bool
	Timeout              uint32
	Client               wrapper.HttpClient

	Option *OidcOption
}

type OidcOption struct {
	StateStr   string
	Nonce      string
	Code       string
	Mod        Accessmod
	RawIdToken string
	AuthStyle  AuthStyle
}
