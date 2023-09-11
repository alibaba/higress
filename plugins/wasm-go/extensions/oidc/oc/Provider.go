package oc

import (
	"errors"
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/gjson"
	"net/http"
	"net/url"
	"time"
)

type ProviderType string

const (
	IssuerGithubAccounts = "github.com"
)

var defaultEndpoints = map[string]struct {
	AuthURL  string
	TokenURL string
	apiURL   string
}{
	IssuerGithubAccounts: {
		AuthURL:  "https://github.com/login/oauth/authorize",
		TokenURL: "https://github.com/login/oauth/access_token",
	},
}

func setDefaultsBasedOnIssuer(Oatuh2Config *Oatuh2Config) {
	if endpoint, exists := defaultEndpoints[Oatuh2Config.Issuer]; exists {
		Oatuh2Config.Endpoint.AuthURL = endpoint.AuthURL
		Oatuh2Config.Endpoint.TokenURL = endpoint.TokenURL
	}
}

func NewGithubProvider(Oatuh2Config *Oatuh2Config) *GithubProvider {
	setDefaultsBasedOnIssuer(Oatuh2Config)
	return &GithubProvider{
		Cfg:         Oatuh2Config,
		AccessToken: "",
		Token_type:  "",
	}

}

type GithubProvider struct {
	Cfg         *Oatuh2Config
	AccessToken string
	expires_in  time.Time
	Token_type  string
}

func (g *GithubProvider) ProcessVerify(rawAccessToken string) (bool, error) {
	if rawAccessToken == "" {
		return false, errors.New("miss token")
	}

	exits, err := checkAccessTokenValidity(GithubSharedDataKey, rawAccessToken)
	if err != nil {
		return false, err
	}
	return exits, errors.New("Token has expired or error token")

}

func (g *GithubProvider) ProcessRedirect(stateStr, nonce string, cfg *Oatuh2Config, log *wrapper.Log) error {

	if cfg.Endpoint.AuthURL == "" {
		SendError(log, " Miss authorization_endpoint ", http.StatusInternalServerError)
	}

	codeURL := cfg.AuthCodeURL(stateStr, SetNonce(nonce))
	err := proxywasm.SendHttpResponse(http.StatusFound, [][2]string{
		{"Location", codeURL},
	}, nil, -1)
	if err != nil {
		log.Errorf("error sending redirect response: %v", err)
		return err
	}
	return nil
}

func (g *GithubProvider) ProcessExchangeToken(code string, cfg *Oatuh2Config, log *wrapper.Log) error {
	if cfg.Endpoint.TokenURL == "" {
		SendError(log, " Miss token_endpoint ", http.StatusInternalServerError)
	}

	authStyle := AuthStyle(cfg.Endpoint.AuthStyle)

	parsedURL, err := url.Parse(cfg.Endpoint.TokenURL)
	if err != nil {
		return fmt.Errorf("invalid TokenURL: %v", err)
	}

	v := ReturnURL(cfg.RedirectURL, code)
	needsAuthStyleProbe := authStyle == AuthStyleUnknown
	if needsAuthStyleProbe {
		if style, ok := LookupAuthStyle(cfg.Endpoint.TokenURL); ok {
			authStyle = style
		} else {
			authStyle = AuthStyleInHeader
		}
	}
	cfg.Scopes = []string{"user:email"}
	headers, body, err := NewTokenRequest(cfg.Endpoint.TokenURL, cfg.ClientID, cfg.ClientSecret, v, authStyle)

	if err != nil {
		return fmt.Errorf("failed to create token request: %v", err)
	}

	cb := func(statusCode int, responseHeaders http.Header, responseBody []byte) {

		if statusCode != http.StatusOK {
			SendError(log, fmt.Sprintf("http call failed, status: %d", statusCode), statusCode)
			return
		}
		if !gjson.ValidBytes(responseBody) {
			SendError(log, "invalid JSON format in response body", http.StatusInternalServerError)
			return
		}

		result := gjson.ParseBytes(responseBody)
		g.AccessToken = result.Get("access_token").String()
		if g.AccessToken == "" {
			SendError(log, "miss AccessToken", http.StatusUnauthorized)
			return
		}
		g.Token_type = result.Get("token_type").String()
		if g.Token_type == "" {
			SendError(log, "miss tokeb_type", http.StatusUnauthorized)
			return
		}

		expirtime := result.Get("expires_in").Int()
		if expirtime == 0 {
			SendError(log, "miss expires_in data", http.StatusUnauthorized)
			return
		}

		g.expires_in = time.Now().Add(time.Second * time.Duration(expirtime))
		err = setSharedData(GithubSharedDataKey, g.AccessToken, g.expires_in)
		if err != nil {
			SendError(log, fmt.Sprintf("set acctoken err : %v ", err), http.StatusInternalServerError)
			return
		}
		proxywasm.SendHttpResponse(http.StatusOK, nil, []byte(g.AccessToken), -1)
		proxywasm.ResumeHttpRequest()
		return

	}

	err = cfg.Client.Post(parsedURL.Path, headers, body, cb, 2000)
	if err != nil {
		return fmt.Errorf("HTTP POST error: %v", err)
	}

	return nil

}
