package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"oidc/oc"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"golang.org/x/oauth2"
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
	ClientDomain    string
	Timeout         uint32
	Scopes          []string
	SkipExpiryCheck bool
	SkipIssuerCheck bool
	SecuceCookie    bool
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
	config.Issuer = json.Get("issuer").String()
	if config.Issuer == "" {
		return errors.New("missing Issuer in config")
	}

	config.ClientID = json.Get("clientId").String()
	if config.ClientID == "" {
		return errors.New("missing clientID in config")
	}

	config.ClientSecret = json.Get("clientSecret").String()
	if config.ClientSecret == "" {
		return errors.New("missing clientSecret in config")
	}

	config.RedirectURL = json.Get("redirectUrl").String()
	if config.RedirectURL == "" {
		return errors.New("missing RedirectURL in config")
	}
	config.SkipExpiryCheck = json.Get("skipExpiryCheck ").Bool()
	config.SkipExpiryCheck = json.Get("skipIssuerCheck").Bool()
	for _, item := range json.Get("scopes").Array() {
		scopes := item.String()
		config.Scopes = append(config.Scopes, scopes)
	}
	parsedURL, err := url.Parse(config.Issuer)
	if err != nil {
		return errors.New("failed to parse issuer URL")
	}
	config.Path = parsedURL.Path

	code := json.Get("timeOut").Int()
	if code != 0 {
		config.Timeout = uint32(code)
	} else {
		config.Timeout = 500
	}

	config.ClientDomain = json.Get("clientDomain").String()
	if config.ClientDomain == "" {
		return errors.New("missing ClientDomain in config")
	}

	config.SkipExpiryCheck = json.Get("secureCookie").Bool()
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
		log.Debugf("%v %v %v", serviceName, serviceHost, servicePort)
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
	if ctx.Host() == console {
		return types.ActionContinue
	}

	DefaultHandler := oc.NewDefaultOAuthHandler()
	cookieString, _ := proxywasm.GetHttpRequestHeader("cookie")
	oidcCookieValue, code, state := oc.GetParams(cookieString, ctx.Path())

	cfg := &oc.Oatuh2Config{
		Config: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.RedirectURL,
			Scopes:       config.Scopes,
		},
		Issuer:          config.Issuer,
		Path:            config.Path,
		Clientdomain:    config.ClientDomain,
		SkipExpiryCheck: config.SkipExpiryCheck,
		SkipIssuerCheck: config.SkipIssuerCheck,
		SecureCookie:    config.SecuceCookie,
		Timeout:         config.Timeout,
		Client:          config.Client,
		Option:          &oc.OidcOption{},
	}
	log.Debugf("path :%v host :%v state :%v code :%v cookie :%v", ctx.Path(), ctx.Host(), state, code, oidcCookieValue)

	if oidcCookieValue == "" {

		if code == "" {
			if err := DefaultHandler.ProcessRedirect(&log, cfg); err != nil {
				oc.SendError(&log, fmt.Sprintf("Redirect error : %v", err), http.StatusInternalServerError)
			}
		}

		if strings.Contains(ctx.Path(), "oidc/callback") {
			parts := strings.Split(state, ".")
			if len(parts) != 2 {
				oc.SendError(&log, "State signature verification failed", http.StatusUnauthorized)
			}

			stateval, signature := parts[0], parts[1]
			if !oc.VerifyState(stateval, signature) {
				oc.SendError(&log, "State signature verification failed", http.StatusUnauthorized)
			}

			cfg.Option.Code = code
			cfg.Option.Mod = oc.SenBack
			if err := DefaultHandler.ProcessExchangeToken(&log, cfg); err != nil {
				oc.SendError(&log, fmt.Sprintf("ProcessExchangeToken error : %v", err), http.StatusInternalServerError)
			}
		}
	} else {
		log.Debugf("verify token")
		cfg.Option.Mod = oc.Access
		cfg.Option.RawIdToken = oidcCookieValue
		if err := DefaultHandler.ProcessVerify(&log, cfg); err != nil {
			oc.SendError(&log, fmt.Sprintf("ProcessVerify error : %v", err), http.StatusUnauthorized)
		}
	}

	return types.ActionPause
}
