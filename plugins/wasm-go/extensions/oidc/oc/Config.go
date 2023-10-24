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
	//SkipIssuerCheck 用于特殊情况，其中调用者希望推迟对签发者的验证。
	//当启用时，调用者必须独立验证令牌的签发者是否为已知的有效值。
	//
	//
	//不匹配的签发者通常指示客户端配置错误。如果不希望发生不匹配，请检查所提供的签发者URL是否正确，而不是启用这个选项。
	SkipIssuerCheck bool
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
	Path                 string
	Clientdomain         string
	SupportedSigningAlgs []string
	SkipExpiryCheck      bool
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
