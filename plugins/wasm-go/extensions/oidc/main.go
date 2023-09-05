package main

import (
	"errors"
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"golang.org/x/oauth2"
	"net/http"
	"net/url"
	"oidc/oc"
	"strings"
)

const (
	console = "console.higress.io"
)

type OidcConfig struct {
	Issuer          string
	Path            string
	ClientID        string
	ClientSecret    string
	RedirectURL     string
	Scopes          []string
	SkipExpiryCheck bool
	Client          wrapper.HttpClient
}

func main() {
	wrapper.SetCtx(
		"oidc",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

func parseConfig(json gjson.Result, config *OidcConfig, log wrapper.Log) error {
	config.Issuer = json.Get("Issuer").String()
	if config.Issuer == "" {
		return errors.New("missing Issuer in config")
	}

	config.ClientID = json.Get("clientID").String()
	if config.ClientID == "" {
		return errors.New("missing clientID in config")
	}

	config.ClientSecret = json.Get("clientSecret").String()
	if config.ClientSecret == "" {
		return errors.New("missing clientSecret in config")
	}

	config.RedirectURL = json.Get("RedirectURL").String()
	if config.RedirectURL == "" {
		return errors.New("missing RedirectURL in config")
	}
	config.SkipExpiryCheck = json.Get("SkipExpiryCheck").Bool()
	for _, item := range json.Get("Scopes").Array() {
		scopes := item.String()
		config.Scopes = append(config.Scopes, scopes)
	}
	parsedURL, err := url.Parse(config.Issuer)
	if err != nil {
		return errors.New("failed to parse issuer URL")
	}
	config.Path = parsedURL.Path

	serviceSource := json.Get("serviceSource").String()
	serviceName := json.Get("serviceName").String()
	servicePort := json.Get("servicePort").Int()
	serviceHost := json.Get("serviceHost").String()
	if serviceName == "" || servicePort == 0 {
		return errors.New("invalid service config")
	}
	switch serviceSource {
	case "ip":
		config.Client = wrapper.NewClusterClient(&wrapper.StaticIpCluster{
			ServiceName: serviceName,
			Host:        serviceHost,
			Port:        servicePort,
		})

		return nil
	case "dns":
		domain := json.Get("domain").String()
		if domain == "" {
			return errors.New("missing domain in config")
		}
		config.Client = wrapper.NewClusterClient(&wrapper.DnsCluster{
			ServiceName: serviceName,
			Port:        servicePort,
			Domain:      domain,
		})

		return nil
	default:
		return errors.New("unknown service source: " + serviceSource)
	}

}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config OidcConfig, log wrapper.Log) types.Action {
	log.Debugf("path : %v host:%v ", ctx.Path(), ctx.Host())
	if ctx.Host() == console {
		return types.ActionContinue
	}
	oauth2Config := &oc.Oatuh2Config{
		Config: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.RedirectURL,
			Scopes:       config.Scopes,
		},
		Issuer:          config.Issuer,
		Path:            config.Path,
		SkipIssuerCheck: config.SkipExpiryCheck,
		Client:          config.Client,
	}

	u, err := url.Parse(ctx.Path())
	if err != nil {
		oc.SendError(&log, fmt.Sprintf("Error parsing query string : %v", err), http.StatusBadRequest)
	}

	query := u.Query()
	state, code := query.Get("state"), query.Get("code")
	rawToken, _ := proxywasm.GetHttpRequestHeader("Authorization")

	log.Debugf(" rawToken:%v state:%v code:%v ", rawToken, state, code)

	StatStr := oc.GenState()
	Nonce := oc.GenState()

	if rawToken == "" {
		if code == "" && state == "" {
			// Redirect if user is not authorized

			if err := oc.ProcessRedirect(StatStr, Nonce, oauth2Config, &log); err != nil {
				oc.SendError(&log, fmt.Sprintf("Redirect error : %v", err), http.StatusInternalServerError)
			}
		} else if strings.Contains(ctx.Path(), "oidc/callback") {
			// Handle callback
			parts := strings.Split(state, ".")
			if len(parts) != 2 {
				oc.SendError(&log, "State signature verification failed", http.StatusUnauthorized)
			}
			stateval, signature := parts[0], parts[1]

			// Verify state signature
			if !oc.VerifyState(stateval, signature) {
				oc.SendError(&log, "State signature verification failed", http.StatusUnauthorized)
			}
			if err := oc.ProcessExchangeToken(code, oauth2Config, &log, oc.SenBack); err != nil {
				oc.SendError(&log, fmt.Sprintf("ProcessExchangeToken error : %v", err), http.StatusInternalServerError)
			}
		}
	} else {
		parts := strings.Split(rawToken, " ")
		if len(parts) != 2 && parts[0] != "Bearer" {
			oc.SendError(&log, "Invalid Authorization header format", http.StatusUnauthorized)
		}

		if err := oc.ProcessVerify(parts[1], oauth2Config, &log, oc.Access); err != nil {
			oc.SendError(&log, fmt.Sprintf("ProcessVerify error : %v", err), http.StatusUnauthorized)

		}
	}
	return types.ActionPause
}
