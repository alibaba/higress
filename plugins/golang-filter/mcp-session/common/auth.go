package common

import (
	"context"
)

// contextKey is the type for context keys to avoid collisions
type authContextKey string

const (
	// authHeaderKey stores the Authorization header value (for API authentication)
	authHeaderKey authContextKey = "auth_header"
	// istiodTokenKey stores the Istiod token value (for Istio debug API authentication)
	istiodTokenKey authContextKey = "istiod_token"
)

// WithAuthHeader adds the Authorization header to context
// This is typically used for authenticating with Higress Console API
func WithAuthHeader(ctx context.Context, authHeader string) context.Context {
	if authHeader == "" {
		return ctx
	}
	return context.WithValue(ctx, authHeaderKey, authHeader)
}

// GetAuthHeader retrieves the Authorization header from context
// Returns the header value and true if found, empty string and false otherwise
func GetAuthHeader(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	authHeader, ok := ctx.Value(authHeaderKey).(string)
	return authHeader, ok
}

// WithIstiodToken adds the Istiod authentication token to context
// This is typically used for authenticating with Istiod debug endpoints
func WithIstiodToken(ctx context.Context, token string) context.Context {
	if token == "" {
		return ctx
	}
	return context.WithValue(ctx, istiodTokenKey, token)
}

// GetIstiodToken retrieves the Istiod token from context
// Returns the token value and true if found, empty string and false otherwise
func GetIstiodToken(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	token, ok := ctx.Value(istiodTokenKey).(string)
	return token, ok
}
