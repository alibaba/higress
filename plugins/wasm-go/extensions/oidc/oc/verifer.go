/*
	Copyright 2023 go-oidc

*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at

* http://www.apache.org/licenses/LICENSE-2.0

* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*/
package oc

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/go-jose/go-jose/v3"
)

// IDTokenVerifierConfig
type IDTokenVerifier struct {
	config *IDConfig
	issuer string
}

const (
	issuerGoogleAccounts         = "https://accounts.google.com"
	issuerGoogleAccountsNoScheme = "accounts.google.com"

	LEEWAY = 5 * time.Minute
)

type IDToken struct {
	Issuer            string
	Audience          []string
	Subject           string
	Expiry            time.Time
	IssuedAt          time.Time
	Nonce             string
	AccessTokenHash   string
	sigAlgorithm      string
	claims            []byte
	distributedClaims map[string]claimSource
}
type TokenExpiredError struct {
	Expiry time.Time
}

func (e *TokenExpiredError) Error() string {
	return fmt.Sprintf("oidc: token is expired (Token Expiry: %v)", e.Expiry)
}
func (i *IDToken) Claims(v interface{}) error {
	if i.claims == nil {
		return errors.New("oidc: claims not set")
	}
	return json.Unmarshal(i.claims, v)
}

func (v *IDTokenVerifier) VerifyToken(rawIDToken string, keySet jose.JSONWebKeySet) (*IDToken, error) {
	var log wrapper.Log
	payload, err := parseJWT(rawIDToken)
	if err != nil {
		return nil, fmt.Errorf(" malformed jwt: %v", err)
	}
	var token idToken
	if err := json.Unmarshal(payload, &token); err != nil {
		log.Errorf("idToken Unmarshal error : %v ", err)
		return nil, fmt.Errorf("failed to unmarshal claims: %v", err)
	}

	distributedClaims := make(map[string]claimSource)

	//step through the token to map claim names to claim sources
	for cn, src := range token.ClaimNames {
		if src == "" {
			return nil, fmt.Errorf("failed to obtain source from claim name")
		}
		s, ok := token.ClaimSources[src]
		if !ok {
			return nil, fmt.Errorf("source does not exist")
		}
		distributedClaims[cn] = s
	}

	t := &IDToken{
		Issuer:            token.Issuer,
		Subject:           token.Subject,
		Audience:          []string(token.Audience),
		Expiry:            time.Time(token.Expiry),
		IssuedAt:          time.Time(token.IssuedAt),
		Nonce:             token.Nonce,
		AccessTokenHash:   token.AtHash,
		claims:            payload,
		distributedClaims: distributedClaims,
	}

	// Check issuer.
	if !v.config.SkipIssuerCheck && t.Issuer != v.issuer {
		// Google sometimes returns "accounts.google.com" as the issuer claim instead of
		// the required "https://accounts.google.com". Detect this case and allow it only
		// for Google.
		//
		// We will not add hooks to let other providers go off spec like this.
		if !(v.issuer == issuerGoogleAccounts && t.Issuer == issuerGoogleAccountsNoScheme) {
			return nil, fmt.Errorf("oidc: id token issued by a different provider, expected %q got %q", v.issuer, t.Issuer)
		}
	}

	if v.config.ClientID != "" {
		if !contains(t.Audience, v.config.ClientID) {
			return nil, fmt.Errorf("oidc: expected audience %q got %q", v.config.ClientID, t.Audience)
		}
	}

	// If a SkipExpiryCheck is false, make sure token is not expired.
	if !v.config.SkipExpiryCheck {
		now := time.Now
		if v.config.Now != nil {
			now = v.config.Now
		}
		nowTime := now()

		if t.Expiry.Before(nowTime) {
			return nil, &TokenExpiredError{Expiry: t.Expiry}
		}

		// If nbf claim is provided in token, ensure that it is indeed in the past.
		if token.NotBefore != nil {
			nbfTime := time.Time(*token.NotBefore)
			// Set to 5 minutes since this is what other OpenID Connect providers do to deal with clock skew.
			// https://github.com/AzureAD/azure-activedirectory-identitymodel-extensions-for-dotnet/blob/6.12.2/src/Microsoft.IdentityModel.Tokens/TokenValidationParameters.cs#L149-L153

			if nowTime.Add(LEEWAY).Before(nbfTime) {
				return nil, fmt.Errorf("oidc: current time %v before the nbf (not before) time: %v", nowTime, nbfTime)
			}
		}
	}

	jws, err := jose.ParseSigned(rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("oidc: malformed jwt: %v", err)
	}

	switch len(jws.Signatures) {
	case 0:
		return nil, fmt.Errorf("oidc: id token not signed")
	case 1:
	default:
		return nil, fmt.Errorf("oidc: multiple signatures on id token not supported")
	}

	sig := jws.Signatures[0]
	supportedSigAlgs := v.config.SupportedSigningAlgs

	if len(supportedSigAlgs) == 0 {
		supportedSigAlgs = []string{RS256}
	}

	if !contains(supportedSigAlgs, sig.Header.Algorithm) {
		return nil, fmt.Errorf("oidc: id token signed with unsupported algorithm, expected %q got %q", supportedSigAlgs, sig.Header.Algorithm)
	}

	t.sigAlgorithm = sig.Header.Algorithm

	keyID := ""
	for _, sig := range jws.Signatures {
		keyID = sig.Header.KeyID
		break
	}

	for _, key := range keySet.Keys {
		if keyID == "" || key.KeyID == keyID {
			if gotPayload, err := jws.Verify(&key); err == nil {
				if !bytes.Equal(gotPayload, payload) {
					return nil, errors.New("oidc: internal error, payload parsed did not match previous payload")
				}
			}
		}
	}

	return t, nil
}
func contains(sli []string, ele string) bool {
	for _, s := range sli {
		if s == ele {
			return true
		}
	}
	return false
}
func parseJWT(p string) ([]byte, error) {
	parts := strings.Split(p, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("oidc: malformed jwt, expected 3 parts got %d", len(parts))
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("oidc: malformed jwt payload: %v", err)
	}
	return payload, nil
}
func (cfg *Oatuh2Config) Verifier(config *IDConfig) *IDTokenVerifier {
	return &IDTokenVerifier{
		config: config,
		issuer: cfg.Issuer,
	}
}
