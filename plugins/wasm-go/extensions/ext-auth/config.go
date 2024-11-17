package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"ext-auth/expr"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

const (
	DefaultStatusOnError uint32 = http.StatusForbidden

	DefaultHttpServiceTimeout uint32 = 1000

	DefaultMaxRequestBodyBytes uint32 = 10 * 1024 * 1024

	EndpointModeEnvoy = "envoy"

	EndpointModeForwardAuth = "forward_auth"
)

type ExtAuthConfig struct {
	httpService               HttpService
	failureModeAllow          bool
	failureModeAllowHeaderAdd bool
	statusOnError             uint32
}

type HttpService struct {
	endpointMode string
	client       wrapper.HttpClient
	// pathPrefix is only used when endpoint_mode is envoy
	pathPrefix string
	// requestMethod is only used when endpoint_mode is forward_auth
	requestMethod string
	// path is only used when endpoint_mode is forward_auth
	path                  string
	timeout               uint32
	authorizationRequest  AuthorizationRequest
	authorizationResponse AuthorizationResponse
}

type AuthorizationRequest struct {
	// allowedHeaders In addition to the user’s supplied matchers,
	// Authorization are automatically included to the list.
	// When the endpoint_mode is set to forward_auth,
	// the original request's path is set in the X-Original-Uri header,
	// and the original request's HTTP method is set in the X-Original-Method header.
	allowedHeaders      expr.Matcher
	headersToAdd        map[string]string
	withRequestBody     bool
	maxRequestBodyBytes uint32
}

type AuthorizationResponse struct {
	allowedUpstreamHeaders expr.Matcher
	allowedClientHeaders   expr.Matcher
}

func parseConfig(json gjson.Result, config *ExtAuthConfig, log wrapper.Log) error {
	httpServiceConfig := json.Get("http_service")
	if !httpServiceConfig.Exists() {
		return errors.New("missing http_service in config")
	}
	err := parseHttpServiceConfig(httpServiceConfig, config, log)
	if err != nil {
		return err
	}

	failureModeAllow := json.Get("failure_mode_allow")
	if failureModeAllow.Exists() {
		config.failureModeAllow = failureModeAllow.Bool()
	}

	failureModeAllowHeaderAdd := json.Get("failure_mode_allow_header_add")
	if failureModeAllowHeaderAdd.Exists() {
		config.failureModeAllowHeaderAdd = failureModeAllowHeaderAdd.Bool()
	}

	statusOnError := uint32(json.Get("status_on_error").Uint())
	if statusOnError == 0 {
		statusOnError = DefaultStatusOnError
	}
	config.statusOnError = statusOnError

	return nil
}

func parseHttpServiceConfig(json gjson.Result, config *ExtAuthConfig, log wrapper.Log) error {
	var httpService HttpService

	if err := parseEndpointConfig(json, &httpService, log); err != nil {
		return err
	}

	timeout := uint32(json.Get("timeout").Uint())
	if timeout == 0 {
		timeout = DefaultHttpServiceTimeout
	}
	httpService.timeout = timeout

	if err := parseAuthorizationRequestConfig(json, &httpService); err != nil {
		return err
	}

	if err := parseAuthorizationResponseConfig(json, &httpService); err != nil {
		return err
	}

	config.httpService = httpService

	return nil
}

func parseEndpointConfig(json gjson.Result, httpService *HttpService, log wrapper.Log) error {
	endpointMode := json.Get("endpoint_mode").String()
	if endpointMode == "" {
		endpointMode = EndpointModeEnvoy
	} else if endpointMode != EndpointModeEnvoy && endpointMode != EndpointModeForwardAuth {
		return errors.New(fmt.Sprintf("endpoint_mode %s is not supported", endpointMode))
	}
	httpService.endpointMode = endpointMode

	endpointConfig := json.Get("endpoint")
	if !endpointConfig.Exists() {
		return errors.New("missing endpoint in config")
	}

	serviceName := endpointConfig.Get("service_name").String()
	if serviceName == "" {
		return errors.New("endpoint service name must not be empty")
	}
	servicePort := endpointConfig.Get("service_port").Int()
	if servicePort == 0 {
		servicePort = 80
	}
	serviceHost := endpointConfig.Get("service_host").String()

	httpService.client = wrapper.NewClusterClient(wrapper.FQDNCluster{
		FQDN: serviceName,
		Port: servicePort,
		Host: serviceHost,
	})

	switch endpointMode {
	case EndpointModeEnvoy:
		pathPrefixConfig := endpointConfig.Get("path_prefix")
		if !pathPrefixConfig.Exists() {
			return errors.New("when endpoint_mode is envoy, endpoint path_prefix must not be empty")
		}
		httpService.pathPrefix = pathPrefixConfig.String()

		if endpointConfig.Get("request_method").Exists() || endpointConfig.Get("path").Exists() {
			log.Warn("when endpoint_mode is envoy, endpoint request_method and path will be ignored")
		}
	case EndpointModeForwardAuth:
		requestMethodConfig := endpointConfig.Get("request_method")
		if !requestMethodConfig.Exists() {
			httpService.requestMethod = http.MethodGet
		} else {
			httpService.requestMethod = strings.ToUpper(requestMethodConfig.String())
		}

		pathConfig := endpointConfig.Get("path")
		if !pathConfig.Exists() {
			return errors.New("when endpoint_mode is forward_auth, endpoint path must not be empty")
		}
		httpService.path = pathConfig.String()

		if endpointConfig.Get("path_prefix").Exists() {
			log.Warn("when endpoint_mode is forward_auth, endpoint path_prefix will be ignored")
		}
	}
	return nil
}

func parseAuthorizationRequestConfig(json gjson.Result, httpService *HttpService) error {
	authorizationRequestConfig := json.Get("authorization_request")
	if authorizationRequestConfig.Exists() {
		var authorizationRequest AuthorizationRequest

		allowedHeaders := authorizationRequestConfig.Get("allowed_headers")
		if allowedHeaders.Exists() {
			result, err := expr.BuildRepeatedStringMatcherIgnoreCase(allowedHeaders.Array())
			if err != nil {
				return err
			}
			authorizationRequest.allowedHeaders = result
		}

		headersToAdd := map[string]string{}
		headersToAddConfig := authorizationRequestConfig.Get("headers_to_add")
		if headersToAddConfig.Exists() {
			for key, value := range headersToAddConfig.Map() {
				headersToAdd[key] = value.Str
			}
		}
		authorizationRequest.headersToAdd = headersToAdd

		withRequestBody := authorizationRequestConfig.Get("with_request_body")
		if withRequestBody.Exists() {
			// withRequestBody is true and the request method is GET, OPTIONS or HEAD
			if withRequestBody.Bool() &&
				(httpService.requestMethod == http.MethodGet || httpService.requestMethod == http.MethodOptions || httpService.requestMethod == http.MethodHead) {
				return errors.New(fmt.Sprintf("requestMethod %s does not support with_request_body set to true", httpService.requestMethod))
			}
			authorizationRequest.withRequestBody = withRequestBody.Bool()
		}

		maxRequestBodyBytes := uint32(authorizationRequestConfig.Get("max_request_body_bytes").Uint())
		if maxRequestBodyBytes == 0 {
			maxRequestBodyBytes = DefaultMaxRequestBodyBytes
		}
		authorizationRequest.maxRequestBodyBytes = maxRequestBodyBytes

		httpService.authorizationRequest = authorizationRequest
	}
	return nil
}

func parseAuthorizationResponseConfig(json gjson.Result, httpService *HttpService) error {
	authorizationResponseConfig := json.Get("authorization_response")
	if authorizationResponseConfig.Exists() {
		var authorizationResponse AuthorizationResponse

		allowedUpstreamHeaders := authorizationResponseConfig.Get("allowed_upstream_headers")
		if allowedUpstreamHeaders.Exists() {
			result, err := expr.BuildRepeatedStringMatcherIgnoreCase(allowedUpstreamHeaders.Array())
			if err != nil {
				return err
			}
			authorizationResponse.allowedUpstreamHeaders = result
		}

		allowedClientHeaders := authorizationResponseConfig.Get("allowed_client_headers")
		if allowedClientHeaders.Exists() {
			result, err := expr.BuildRepeatedStringMatcherIgnoreCase(allowedClientHeaders.Array())
			if err != nil {
				return err
			}
			authorizationResponse.allowedClientHeaders = result
		}

		httpService.authorizationResponse = authorizationResponse
	}
	return nil
}
