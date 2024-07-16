package main

import (
	"errors"
	"ext-auth/expr"
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
	"net/http"
	"strings"
)

const (
	DefaultStatusOnError uint32 = http.StatusForbidden

	DefaultHttpServiceTimeout uint32 = 200
)

type ExtAuthConfig struct {
	httpService               HttpService
	failureModeAllow          bool
	failureModeAllowHeaderAdd bool
	statusOnError             uint32
}

type HttpService struct {
	client                wrapper.HttpClient
	requestMethod         string
	path                  string
	timeout               uint32
	authorizationRequest  AuthorizationRequest
	authorizationResponse AuthorizationResponse
}

type AuthorizationRequest struct {
	// allowedHeaders In addition to the user’s supplied matchers,
	// Host, Method, Path, Content-Length, and Authorization are automatically included to the list.
	allowedHeaders  expr.Matcher
	headersToAdd    map[string]string
	withRequestBody bool
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
	err := parseHttpServiceConfig(httpServiceConfig, config)
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

	statusOnError := json.Get("status_on_error")
	if statusOnError.Exists() {
		config.statusOnError = uint32(statusOnError.Uint())
	} else {
		config.statusOnError = DefaultStatusOnError
	}

	return nil
}

func parseHttpServiceConfig(json gjson.Result, config *ExtAuthConfig) error {
	var httpService HttpService

	if err := parseEndpointConfig(json, &httpService); err != nil {
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

func parseEndpointConfig(json gjson.Result, httpService *HttpService) error {
	endpointConfig := json.Get("endpoint")
	if !endpointConfig.Exists() {
		return errors.New("missing endpoint in config")
	}

	serviceSource := endpointConfig.Get("service_source").String()
	serviceName := endpointConfig.Get("service_name").String()
	servicePort := endpointConfig.Get("service_port").Int()
	if serviceName == "" || servicePort == 0 {
		return errors.New("invalid service config")
	}
	switch serviceSource {
	case "k8s":
		namespace := json.Get("namespace").String()
		httpService.client = wrapper.NewClusterClient(wrapper.K8sCluster{
			ServiceName: serviceName,
			Namespace:   namespace,
			Port:        servicePort,
		})
		return nil
	case "nacos":
		namespace := json.Get("namespace").String()
		httpService.client = wrapper.NewClusterClient(wrapper.NacosCluster{
			ServiceName: serviceName,
			NamespaceID: namespace,
			Port:        servicePort,
		})
		return nil
	case "ip":
		httpService.client = wrapper.NewClusterClient(wrapper.StaticIpCluster{
			ServiceName: serviceName,
			Port:        servicePort,
		})
	case "dns":
		domain := endpointConfig.Get("domain").String()
		httpService.client = wrapper.NewClusterClient(wrapper.DnsCluster{
			ServiceName: serviceName,
			Port:        servicePort,
			Domain:      domain,
		})
	default:
		return errors.New("unknown service source: " + serviceSource)
	}

	requestMethodConfig := endpointConfig.Get("request_method")
	if !requestMethodConfig.Exists() {
		httpService.requestMethod = http.MethodGet
	} else {
		httpService.requestMethod = strings.ToUpper(requestMethodConfig.String())
	}

	pathConfig := endpointConfig.Get("path")
	if !pathConfig.Exists() {
		return errors.New("missing path in config")
	}
	httpService.path = pathConfig.String()

	return nil
}

func parseAuthorizationRequestConfig(json gjson.Result, httpService *HttpService) error {
	authorizationRequestConfig := json.Get("authorization_request")
	if authorizationRequestConfig.Exists() {
		var authorizationRequest AuthorizationRequest

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

		allowedHeaders := authorizationRequestConfig.Get("allowed_headers")
		if allowedHeaders.Exists() {
			result, err := expr.BuildRepeatedStringMatcherIgnoreCase(allowedHeaders.Array())
			if err != nil {
				return err
			}
			authorizationRequest.allowedHeaders = result
		}

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
