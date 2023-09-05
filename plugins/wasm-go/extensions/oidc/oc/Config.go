package oc

import (
	"encoding/json"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"golang.org/x/oauth2"
	"time"
)

type Accessmod int

const (
	Access  Accessmod = 0
	SenBack Accessmod = 1
)

type IDConfig struct {
	ClientID string

	SupportedSigningAlgs []string

	SkipExpiryCheck bool

	// Time function to check Token expiry. Defaults to time.Now
	Now func() time.Time
}

type Oatuh2Config struct {
	oauth2.Config
	Issuer               string
	JwksURL              string
	Path                 string
	SupportedSigningAlgs []string
	SkipIssuerCheck      bool
	Client               wrapper.HttpClient
}
type audience []string

func (a *audience) UnmarshalJSON(b []byte) error {
	var s string
	if json.Unmarshal(b, &s) == nil {
		*a = audience{s}
		return nil
	}
	var auds []string
	if err := json.Unmarshal(b, &auds); err != nil {
		return err
	}
	*a = auds
	return nil
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

type jsonTime time.Time

func (j *jsonTime) UnmarshalJSON(b []byte) error {
	var n json.Number
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}
	var unix int64

	if t, err := n.Int64(); err == nil {
		unix = t
	} else {
		f, err := n.Float64()
		if err != nil {
			return err
		}
		unix = int64(f)
	}
	*j = jsonTime(time.Unix(unix, 0))
	return nil
}
