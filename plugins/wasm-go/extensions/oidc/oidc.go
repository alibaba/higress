package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	middlewareapi "oidc/pkg/apis/middleware"
	"oidc/pkg/apis/options"
	sessionsapi "oidc/pkg/apis/sessions"
	"oidc/pkg/app/redirect"
	"oidc/pkg/cookies"
	"oidc/pkg/encryption"
	"oidc/pkg/middleware"
	requestutil "oidc/pkg/requests/util"
	"oidc/pkg/sessions"
	"oidc/pkg/util"
	"oidc/providers"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/gorilla/mux"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/justinas/alice"
)

const (
	SetCookieHeader = "Set-Cookie"
	schemeHTTP      = "http"
	schemeHTTPS     = "https"
	applicationJSON = "application/json"

	oauthStartPath    = "/start"
	oauthCallbackPath = "/callback"
)

var (
	// ErrNeedsLogin means the user should be redirected to the login page
	ErrNeedsLogin = errors.New("redirect to login page")

	// ErrAccessDenied means the user should receive a 401 Unauthorized response
	ErrAccessDenied = errors.New("access denied")
)

// OAuthProxy is the main authentication proxy
type OAuthProxy struct {
	CookieOptions *options.Cookie
	Validator     func(string) bool
	Ctx           wrapper.HttpContext

	redirectURL         *url.URL // the url to receive requests at
	relativeRedirectURL bool
	whitelistDomains    []string
	provider            providers.Provider
	sessionStore        sessionsapi.SessionStore
	ProxyPrefix         string
	skipAuthPreflight   bool

	sessionChain alice.Chain
	preAuthChain alice.Chain

	serveMux          *mux.Router
	redirectValidator redirect.Validator
	appDirector       redirect.AppDirector

	passAuthorization bool
	encodeState       bool

	client wrapper.HttpClient
}

// NewOAuthProxy creates a new instance of OAuthProxy from the options provided
func NewOAuthProxy(opts *options.Options, validator func(string) bool) (*OAuthProxy, error) {
	sessionStore, err := sessions.NewSessionStore(&opts.Session, &opts.Cookie)
	if err != nil {
		return nil, fmt.Errorf("error initialising session store: %v", err)
	}

	provider, err := providers.NewProvider(opts.Providers[0])
	if err != nil {
		return nil, fmt.Errorf("error initialising provider: %v", err)
	}

	redirectURL := opts.GetRedirectURL()
	if redirectURL.Path == "" {
		redirectURL.Path = fmt.Sprintf("%s/callback", opts.ProxyPrefix)
	}

	util.Logger.Infof("OAuthProxy configured for %s Client ID: %s", provider.Data().ProviderName, opts.Providers[0].ClientID)
	refresh := "disabled"
	if opts.Cookie.Refresh != time.Duration(0) {
		refresh = fmt.Sprintf("after %s", opts.Cookie.Refresh)
	}
	util.Logger.Infof("Cookie settings: name:%s secure(https):%v httponly:%v expiry:%s domains:%s path:%s samesite:%s refresh:%s", opts.Cookie.Name, opts.Cookie.Secure, opts.Cookie.HTTPOnly, opts.Cookie.Expire, strings.Join(opts.Cookie.Domains, ","), opts.Cookie.Path, opts.Cookie.SameSite, refresh)

	serviceClient, err := opts.Service.NewService()
	if err != nil {
		return nil, err
	}

	preAuthChain, err := buildPreAuthChain(opts)
	if err != nil {
		return nil, fmt.Errorf("could not build pre-auth chain: %v", err)
	}
	sessionChain := buildSessionChain(opts, provider, sessionStore, serviceClient)

	redirectValidator := redirect.NewValidator(opts.WhitelistDomains)
	appDirector := redirect.NewAppDirector(redirect.AppDirectorOpts{
		ProxyPrefix: opts.ProxyPrefix,
		Validator:   redirectValidator,
	})

	p := &OAuthProxy{
		CookieOptions: &opts.Cookie,
		Validator:     validator,

		ProxyPrefix:         opts.ProxyPrefix,
		provider:            provider,
		sessionStore:        sessionStore,
		redirectURL:         redirectURL,
		relativeRedirectURL: opts.RelativeRedirectURL,
		whitelistDomains:    opts.WhitelistDomains,
		skipAuthPreflight:   opts.SkipAuthPreflight,

		sessionChain: sessionChain,
		preAuthChain: preAuthChain,

		redirectValidator: redirectValidator,
		appDirector:       appDirector,
		encodeState:       opts.EncodeState,
		passAuthorization: opts.PassAuthorization,

		client: serviceClient,
	}
	p.buildServeMux(opts.ProxyPrefix)

	return p, nil
}

func (p *OAuthProxy) buildServeMux(proxyPrefix string) {
	// Use the encoded path here, so we can have the option to pass it on in the upstream mux.
	// Otherwise, something like /%2F/ would be redirected to / here already.
	r := mux.NewRouter().UseEncodedPath()
	// Everything served by the router must go through the preAuthChain first.
	r.Use(p.preAuthChain.Then)

	// This will register all the paths under the proxy prefix, except the auth only path so that no cache headers
	// are not applied.
	p.buildProxySubRouter(r.PathPrefix(proxyPrefix).Subrouter())

	// Register serveHTTP last, so it catches anything that isn't already caught earlier.
	// Anything that got to this point needs to have a session loaded.
	r.PathPrefix("/").Handler(p.sessionChain.ThenFunc(p.Proxy))
	p.serveMux = r
}

func (p *OAuthProxy) buildProxySubRouter(s *mux.Router) {
	s.Use(prepareNoCacheMiddleware)

	s.Path(oauthStartPath).HandlerFunc(p.OAuthStart)
	s.Path(oauthCallbackPath).HandlerFunc(p.OAuthCallback)
}

// buildPreAuthChain constructs a chain that should process every request before
// the OAuth2 Proxy authentication logic kicks in.
// For example forcing HTTPS or health checks.
func buildPreAuthChain(opts *options.Options) (alice.Chain, error) {
	chain := alice.New(middleware.NewScope(opts.ReverseProxy, "X-Request-Id"))
	return chain, nil
}

func buildSessionChain(opts *options.Options, provider providers.Provider, sessionStore sessionsapi.SessionStore, serviceClient wrapper.HttpClient) alice.Chain {
	chain := alice.New()

	chain = chain.Append(middleware.NewStoredSessionLoader(&middleware.StoredSessionLoaderOptions{
		SessionStore:          sessionStore,
		RefreshPeriod:         opts.Cookie.Refresh,
		RefreshSession:        provider.RefreshSession,
		ValidateSession:       provider.ValidateSession,
		RefreshClient:         serviceClient,
		RefreshRequestTimeout: provider.Data().RedeemTimeout,
	}))

	return chain
}

// OAuthStart starts the OAuth2 authentication flow
func (p *OAuthProxy) OAuthStart(rw http.ResponseWriter, req *http.Request) {
	// start the flow permitting login URL query parameters to be overridden from the request URL
	p.doOAuthStart(rw, req, req.URL.Query())
}

func (p *OAuthProxy) doOAuthStart(rw http.ResponseWriter, req *http.Request, overrides url.Values) {
	extraParams := p.provider.Data().LoginURLParams(overrides)
	prepareNoCache(rw)

	var (
		err                                              error
		codeChallenge, codeVerifier, codeChallengeMethod string
	)
	if p.provider.Data().CodeChallengeMethod != "" {
		codeChallengeMethod = p.provider.Data().CodeChallengeMethod
		codeVerifier, err = encryption.GenerateRandomASCIIString(96)
		if err != nil {
			util.SendError(fmt.Sprintf("Unable to build random ASCII string for code verifier: %v", err), rw, http.StatusInternalServerError)
			return
		}

		codeChallenge, err = encryption.GenerateCodeChallenge(p.provider.Data().CodeChallengeMethod, codeVerifier)
		if err != nil {
			util.SendError(fmt.Sprintf("Error creating code challenge: %v", err), rw, http.StatusInternalServerError)
			return
		}

		extraParams.Add("code_challenge", codeChallenge)
		extraParams.Add("code_challenge_method", codeChallengeMethod)
	}

	csrf, err := cookies.NewCSRF(p.CookieOptions, codeVerifier)
	if err != nil {
		util.SendError(fmt.Sprintf("Error creating CSRF nonce: %v", err), rw, http.StatusInternalServerError)
		return
	}

	appRedirect, err := p.appDirector.GetRedirect(req)
	if err != nil {
		util.SendError(fmt.Sprintf("Error obtaining application redirect: %v", err), rw, http.StatusBadRequest)
		return
	}
	callbackRedirect := p.getOAuthRedirectURI(req)

	loginURL := p.provider.GetLoginURL(
		callbackRedirect,
		encodeState(csrf.HashOAuthState(), appRedirect, p.encodeState),
		csrf.HashOIDCNonce(),
		extraParams,
	)

	if _, err := csrf.SetCookie(rw, req); err != nil {
		util.SendError(fmt.Sprintf("Error setting CSRF cookie: %v", err), rw, http.StatusInternalServerError)
		return
	}
	redirectToLocation(rw, loginURL)
}

// getOAuthRedirectURI returns the redirectURL that the upstream OAuth Provider will
// redirect clients to once authenticated.
// This is usually the OAuthProxy callback URL.
func (p *OAuthProxy) getOAuthRedirectURI(req *http.Request) string {
	// if `p.redirectURL` already has a host, return it
	if p.relativeRedirectURL || p.redirectURL.Host != "" {
		return p.redirectURL.String()
	}

	// Otherwise figure out the scheme + host from the request
	rd := *p.redirectURL
	rd.Host = requestutil.GetRequestHost(req)
	rd.Scheme = requestutil.GetRequestProto(req)

	// If there's no scheme in the request, we should still include one
	if rd.Scheme == "" {
		rd.Scheme = schemeHTTP
	}

	// If CookieSecure is true, return `https` no matter what
	// Not all reverse proxies set X-Forwarded-Proto
	if p.CookieOptions.Secure {
		rd.Scheme = schemeHTTPS
	}
	return rd.String()
}

// OAuthCallback is the OAuth2 authentication flow callback that finishes the
// OAuth2 authentication flow
func (p *OAuthProxy) OAuthCallback(rw http.ResponseWriter, req *http.Request) {
	// finish the oauth cycle
	err := req.ParseForm()
	if err != nil {
		util.SendError(fmt.Sprintf("Error while parsing OAuth2 callback: %v", err), rw, http.StatusInternalServerError)
		return
	}
	errorString := req.Form.Get("error")
	if errorString != "" {
		util.SendError(fmt.Sprintf("Error while parsing OAuth2 callback: %s", errorString), rw, http.StatusForbidden)
		return
	}

	csrf, err := cookies.LoadCSRFCookie(req, p.CookieOptions)
	if err != nil {
		util.SendError(fmt.Sprintf("Invalid authentication via OAuth2. Error while loading CSRF cookie: %v", err), rw, http.StatusForbidden)
		return
	}

	callback := func(args ...interface{}) {
		session := args[0].(*sessionsapi.SessionState)
		csrf.ClearCookie(rw, req)
		nonce, appRedirect, err := decodeState(req.Form.Get("state"), p.encodeState)
		if err != nil {
			util.SendError(fmt.Sprintf("Error while parsing OAuth2 state: %v", err), rw, http.StatusInternalServerError)
			return
		}

		if !csrf.CheckOAuthState(nonce) {
			util.SendError("Invalid authentication via OAuth2: CSRF token mismatch, potential attack", rw, http.StatusForbidden)
			return
		}
		csrf.SetSessionNonce(session)
		if !p.provider.ValidateSession(req.Context(), session) {
			util.SendError(fmt.Sprintf("Session validation failed: %s", session), rw, http.StatusForbidden)
			return
		}
		if !p.redirectValidator.IsValidRedirect(appRedirect) {
			appRedirect = "/"
		}
		// set cookie, or deny
		authorized, err := p.provider.Authorize(req.Context(), session)
		if err != nil {
			util.Logger.Errorf("Error with authorization: %v", err)
		}
		if p.Validator(session.Email) && authorized {
			util.Logger.Infof("Authenticated successfully via OAuth2: %s", session)
			err := p.SaveSession(rw, req, session)
			if err != nil {
				util.SendError(fmt.Sprintf("Error saving session state: %v", err), rw, http.StatusInternalServerError)
				return
			}
			redirectToLocation(rw, appRedirect)
		} else {
			util.SendError("Invalid authentication via OAuth2: unauthorized", rw, http.StatusForbidden)
		}
	}

	err = p.redeemCode(req, csrf.GetCodeVerifier(), p.client, callback)
	if err != nil {
		util.SendError(fmt.Sprintf("Error redeeming code during OAuth2 callback: %v", err), rw, http.StatusInternalServerError)
		return
	}
}

// Proxy proxies the user request if the user is authenticated else it prompts
// them to authenticate
func (p *OAuthProxy) Proxy(rw http.ResponseWriter, req *http.Request) {
	session, err := p.getAuthenticatedSession(rw, req)
	switch {
	case err == nil:
		if p.passAuthorization {
			proxywasm.AddHttpRequestHeader("Authorization", fmt.Sprintf("%s %s", providers.TokenTypeBearer, session.IDToken))
		}
		if cookies, ok := rw.Header()[SetCookieHeader]; ok && len(cookies) > 0 {
			newCookieValue := strings.Join(cookies, ",")
			p.Ctx.SetContext(SetCookieHeader, newCookieValue)
			modifyRequestCookie(req, p.CookieOptions.Name, newCookieValue)
			util.Logger.Infof("Authentication and session refresh successfully .")
		} else {
			util.Logger.Infof("Authentication successfully.")
			rw.WriteHeader(http.StatusOK)
		}
	case errors.Is(err, ErrNeedsLogin):
		// we need to send the user to a login screen
		if isAjax(req) {
			util.SendError("No valid authentication in request. Access Denied.", rw, http.StatusUnauthorized)
			return
		}
		util.Logger.Infof("No valid authentication in request. Initiating login.")
		// start OAuth flow, but only with the default login URL params - do not
		// consider this request's query params as potential overrides, since
		// the user did not explicitly start the login flow
		p.doOAuthStart(rw, req, nil)
	case errors.Is(err, ErrAccessDenied):
		util.SendError("The session failed authorization checks", rw, http.StatusForbidden)
	default:
		// unknown error
		util.SendError(fmt.Sprintf("Unexpected internal error: %v", err), rw, http.StatusInternalServerError)
	}
}

// getAuthenticatedSession checks whether a user is authenticated and returns a session object and nil error if so
// Returns:
// - `nil, ErrNeedsLogin` if user needs to log in.
// - `nil, ErrAccessDenied` if the authenticated user is not authorized
// Set-Cookie headers may be set on the response as a side effect of calling this method.
func (p *OAuthProxy) getAuthenticatedSession(rw http.ResponseWriter, req *http.Request) (*sessionsapi.SessionState, error) {
	session := middlewareapi.GetRequestScope(req).Session

	// Check this after loading the session so that if a valid session exists, we can add headers from it
	if p.IsAllowedRequest(req) {
		return session, nil
	}

	if session == nil {
		return nil, ErrNeedsLogin
	}

	invalidEmail := session.Email != "" && !p.Validator(session.Email)
	authorized, err := p.provider.Authorize(req.Context(), session)
	if err != nil {
		util.Logger.Errorf("Error with authorization: %v", err)
	}

	if invalidEmail || !authorized {
		cause := "unauthorized"
		if invalidEmail {
			cause = "invalid email"
		}

		util.Logger.Errorf("Invalid authorization via session (%s): removing session", cause)
		// Invalid session, clear it
		err := p.ClearSessionCookie(rw, req)
		if err != nil {
			util.Logger.Errorf("Error clearing session cookie: %v", err)
		}
		return nil, ErrAccessDenied
	}

	return session, nil
}

// IsAllowedRequest is used to check if auth should be skipped for this request
func (p *OAuthProxy) IsAllowedRequest(req *http.Request) bool {
	isPreflightRequestAllowed := p.skipAuthPreflight && req.Method == "OPTIONS"
	return isPreflightRequestAllowed
}

// See https://developers.google.com/web/fundamentals/performance/optimizing-content-efficiency/http-caching?hl=en
var noCacheHeaders = map[string]string{
	"Expires":         time.Unix(0, 0).Format(time.RFC1123),
	"Cache-Control":   "no-cache, no-store, must-revalidate, max-age=0",
	"X-Accel-Expires": "0", // https://www.nginx.com/resources/wiki/start/topics/examples/x-accel/
}

// prepareNoCache prepares headers for preventing browser caching.
func prepareNoCache(w http.ResponseWriter) {
	// Set NoCache headers
	for k, v := range noCacheHeaders {
		w.Header().Set(k, v)
	}
}

func prepareNoCacheMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		prepareNoCache(rw)
		next.ServeHTTP(rw, req)
	})
}

// encodedState builds the OAuth state param out of our nonce and
// original application redirect
func encodeState(nonce string, redirect string, encode bool) string {
	rawString := fmt.Sprintf("%v:%v", nonce, redirect)
	if encode {
		return base64.RawURLEncoding.EncodeToString([]byte(rawString))
	}
	return rawString
}

// decodeState splits the reflected OAuth state response back into
// the nonce and original application redirect
func decodeState(state string, encode bool) (string, string, error) {
	toParse := state
	if encode {
		decoded, _ := base64.RawURLEncoding.DecodeString(state)
		toParse = string(decoded)
	}

	parsedState := strings.SplitN(toParse, ":", 2)
	if len(parsedState) != 2 {
		return "", "", errors.New("invalid length")
	}
	return parsedState[0], parsedState[1], nil
}

// SaveSession creates a new session cookie value and sets this on the response
func (p *OAuthProxy) SaveSession(rw http.ResponseWriter, req *http.Request, s *sessionsapi.SessionState) error {
	return p.sessionStore.Save(rw, req, s)
}

// ClearSessionCookie creates a cookie to unset the user's authentication cookie
// stored in the user's session
func (p *OAuthProxy) ClearSessionCookie(rw http.ResponseWriter, req *http.Request) error {
	return p.sessionStore.Clear(rw, req)
}

func (p *OAuthProxy) redeemCode(req *http.Request, codeVerifier string, client wrapper.HttpClient, callback func(args ...interface{})) error {
	code := req.Form.Get("code")
	if code == "" {
		return providers.ErrMissingCode
	}

	setEmptyVar := func(args ...interface{}) {
		s := args[0].(*sessionsapi.SessionState)
		if s.CreatedAt == nil {
			s.CreatedAtNow()
		}
		if s.ExpiresOn == nil {
			s.ExpiresIn(p.CookieOptions.Expire)
		}
	}
	combine := util.Combine(setEmptyVar, callback)
	redirectURI := p.getOAuthRedirectURI(req)
	err := p.provider.Redeem(req.Context(), redirectURI, code, codeVerifier, client, combine, p.provider.Data().RedeemTimeout)
	if err != nil {
		return err
	}

	return nil
}

// isAjax checks if a request is an ajax request
func isAjax(req *http.Request) bool {
	acceptValues := req.Header.Values("Accept")
	const ajaxReq = applicationJSON
	// Iterate over multiple Accept headers, i.e.
	// Accept: application/json
	// Accept: text/plain
	for _, mimeTypes := range acceptValues {
		// Iterate over multiple mimetypes in a single header, i.e.
		// Accept: application/json, text/plain, */*
		for _, mimeType := range strings.Split(mimeTypes, ",") {
			mimeType = strings.TrimSpace(mimeType)
			if mimeType == ajaxReq {
				return true
			}
		}
	}
	return false
}

// redirect to the specified location through proxywasm
func redirectToLocation(rw http.ResponseWriter, location string) {
	headersMap := [][2]string{{"Location", location}}
	for key, value := range rw.Header() {
		if strings.EqualFold(key, SetCookieHeader) {
			for _, value := range value {
				headersMap = append(headersMap, [2]string{SetCookieHeader, value})
			}
		} else {
			headersMap = append(headersMap, [2]string{key, strings.Join(value, ",")})
		}
	}
	proxywasm.SendHttpResponse(http.StatusFound, headersMap, nil, -1)
}

func modifyRequestCookie(req *http.Request, cookieName, newValue string) {
	var cookies []string
	found := false
	for _, cookie := range req.Cookies() {
		// find specify cookie name
		if cookie.Name == cookieName {
			found = true
			cookies = append(cookies, fmt.Sprintf("%s=%s", cookie.Name, newValue))
		} else {
			cookies = append(cookies, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
		}
	}
	if !found {
		cookies = append(cookies, fmt.Sprintf("%s=%s", cookieName, newValue))
	}
	proxywasm.ReplaceHttpRequestHeader("Cookie", strings.Join(cookies, "; "))
}
