/*
Copyright 2022 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package http

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/alibaba/higress/test/e2e/conformance/utils/config"
	"github.com/alibaba/higress/test/e2e/conformance/utils/roundtripper"
)

type Assertion struct {
	Meta     AssertionMeta
	Request  AssertionRequest
	Response AssertionResponse
}

type AssertionMeta struct {
	// TestCaseName is the User Given TestCase name
	TestCaseName string
	// TargetBackend defines the target backend service
	TargetBackend string
	// TargetNamespace defines the target backend namespace
	TargetNamespace string
}

type AssertionRequest struct {
	// ActualRequest defines the request to make.
	ActualRequest Request

	// ExpectedRequest defines the request that
	// is expected to arrive at the backend. If
	// not specified, the backend request will be
	// expected to match Request.
	ExpectedRequest *ExpectedRequest
	RedirectRequest *roundtripper.RedirectRequest
}

type AssertionResponse struct {
	// ExpectedResponse defines what response the test case
	// should receive.
	ExpectedResponse Response
	// AdditionalResponseHeaders is a set of headers
	// the echoserver should set in its response.
	AdditionalResponseHeaders map[string]string
	// set not need to judge response has request info
	ExpectedResponseNoRequest bool
}

// Request can be used as both the request to make and a means to verify
// that echoserver received the expected request. Note that multiple header
// values can be provided, as a comma-separated value.
type Request struct {
	Host             string
	Method           string
	Path             string
	Headers          map[string]string
	UnfollowRedirect bool
	TLSConfig        *TLSConfig
}

// TLSConfig defines the TLS configuration for the client.
// When this field is set, the HTTPS protocol is used.
type TLSConfig struct {
	// MinVersion specifies the minimum TLS version,
	// e.g. tls.VersionTLS12.
	MinVersion uint16
	// MinVersion specifies the maximum TLS version,
	// e.g. tls.VersionTLS13.
	MaxVersion uint16
	// SNI is short for Server Name Indication.
	// If this field is not specified, the value will be equal to `Host`.
	SNI string
	// CipherSuites can specify multiple client cipher suites,
	// e.g. tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA.
	CipherSuites []uint16
	// Certificates defines the certificate chain
	Certificates Certificates
}

// Certificates contains CA and client certificate chain
type Certificates struct {
	CACerts        [][]byte
	ClientKeyPairs []ClientKeyPair
}

// ClientKeyPair is a pair of client certificate and private key.
type ClientKeyPair struct {
	ClientCert []byte
	ClientKey  []byte
}

// ExpectedRequest defines expected properties of a request that reaches a backend.
type ExpectedRequest struct {
	Request

	// AbsentHeaders are names of headers that are expected
	// *not* to be present on the request.
	AbsentHeaders []string
}

// Response defines expected properties of a response from a backend.
type Response struct {
	StatusCode    int
	Headers       map[string]string
	AbsentHeaders []string
}

// requiredConsecutiveSuccesses is the number of requests that must succeed in a row
// for MakeRequestAndExpectEventuallyConsistentResponse to consider the response "consistent"
// before making additional assertions on the response body. If this number is not reached within
// maxTimeToConsistency, the test will fail.
const requiredConsecutiveSuccesses = 3

// MakeRequestAndExpectEventuallyConsistentResponse makes a request with the given parameters,
// understanding that the request may fail for some amount of time.
//
// Once the request succeeds consistently with the response having the expected status code, make
// additional assertions on the response body using the provided ExpectedResponse.
func MakeRequestAndExpectEventuallyConsistentResponse(t *testing.T, r roundtripper.RoundTripper, timeoutConfig config.TimeoutConfig, gwAddr string, expected Assertion) {
	t.Helper()

	var (
		scheme    = "http"
		protocol  = "HTTP"
		tlsConfig *roundtripper.TLSConfig
	)
	if expected.Request.ActualRequest.TLSConfig != nil {
		scheme = "https"
		protocol = "HTTPS"
		clientKeyPairs := make([]roundtripper.ClientKeyPair, 0, len(expected.Request.ActualRequest.TLSConfig.Certificates.ClientKeyPairs))
		for _, keyPair := range expected.Request.ActualRequest.TLSConfig.Certificates.ClientKeyPairs {
			clientKeyPairs = append(clientKeyPairs, roundtripper.ClientKeyPair{
				ClientCert: keyPair.ClientCert,
				ClientKey:  keyPair.ClientKey,
			})
		}
		tlsConfig = &roundtripper.TLSConfig{
			MinVersion:   expected.Request.ActualRequest.TLSConfig.MinVersion,
			MaxVersion:   expected.Request.ActualRequest.TLSConfig.MaxVersion,
			SNI:          expected.Request.ActualRequest.TLSConfig.SNI,
			CipherSuites: expected.Request.ActualRequest.TLSConfig.CipherSuites,
			Certificates: roundtripper.Certificates{
				CACert:         expected.Request.ActualRequest.TLSConfig.Certificates.CACerts,
				ClientKeyPairs: clientKeyPairs,
			},
		}
		if tlsConfig.SNI == "" {
			tlsConfig.SNI = expected.Request.ActualRequest.Host
		}
	}

	if expected.Request.ActualRequest.Method == "" {
		expected.Request.ActualRequest.Method = "GET"
	}

	if expected.Response.ExpectedResponse.StatusCode == 0 {
		expected.Response.ExpectedResponse.StatusCode = 200
	}

	t.Logf("Making %s request to %s://%s%s", expected.Request.ActualRequest.Method, scheme, gwAddr, expected.Request.ActualRequest.Path)

	path, query, _ := strings.Cut(expected.Request.ActualRequest.Path, "?")

	req := roundtripper.Request{
		Method:           expected.Request.ActualRequest.Method,
		Host:             expected.Request.ActualRequest.Host,
		URL:              url.URL{Scheme: scheme, Host: gwAddr, Path: path, RawQuery: query},
		Protocol:         protocol,
		Headers:          map[string][]string{},
		UnfollowRedirect: expected.Request.ActualRequest.UnfollowRedirect,
		TLSConfig:        tlsConfig,
	}

	if expected.Request.ActualRequest.Headers != nil {
		for name, value := range expected.Request.ActualRequest.Headers {
			req.Headers[name] = []string{value}
		}
	}

	backendSetHeaders := []string{}
	for name, val := range expected.Response.AdditionalResponseHeaders {
		backendSetHeaders = append(backendSetHeaders, name+":"+val)
	}
	req.Headers["X-Echo-Set-Header"] = []string{strings.Join(backendSetHeaders, ",")}

	WaitForConsistentResponse(t, r, req, expected, requiredConsecutiveSuccesses, timeoutConfig.MaxTimeToConsistency)
}

// awaitConvergence runs the given function until it returns 'true' `threshold` times in a row.
// Each failed attempt has a 1s delay; successful attempts have no delay.
func awaitConvergence(t *testing.T, threshold int, maxTimeToConsistency time.Duration, fn func(elapsed time.Duration) bool) {
	successes := 0
	attempts := 0
	start := time.Now()
	to := time.After(maxTimeToConsistency)
	delay := time.Second
	for {
		select {
		case <-to:
			t.Fatalf("timeout while waiting after %d attempts", attempts)
		default:
		}

		completed := fn(time.Now().Sub(start))
		attempts++
		if completed {
			successes++
			if successes >= threshold {
				return
			}
			// Skip delay if we have a success
			continue
		}

		successes = 0
		select {
		// Capture the overall timeout
		case <-to:
			t.Fatalf("timeout while waiting after %d attempts, %d/%d sucessess", attempts, successes, threshold)
			// And the per-try delay
		case <-time.After(delay):
		}
	}
}

// WaitForConsistentResponse repeats the provided request until it completes with a response having
// the expected response consistently. The provided threshold determines how many times in
// a row this must occur to be considered "consistent".
func WaitForConsistentResponse(t *testing.T, r roundtripper.RoundTripper, req roundtripper.Request, expected Assertion, threshold int, maxTimeToConsistency time.Duration) {
	awaitConvergence(t, threshold, maxTimeToConsistency, func(elapsed time.Duration) bool {
		cReq, cRes, err := r.CaptureRoundTrip(req)
		if err != nil {
			t.Logf("Request failed, not ready yet: %v (after %v)", err.Error(), elapsed)
			return false
		}

		if err := CompareRequest(&req, cReq, cRes, expected); err != nil {
			t.Logf("Response expectation failed for request: %v  not ready yet: %v (after %v)", req, err, elapsed)
			return false
		}

		return true
	})
	t.Logf("Request passed")
}

func CompareRequest(req *roundtripper.Request, cReq *roundtripper.CapturedRequest, cRes *roundtripper.CapturedResponse, expected Assertion) error {
	if expected.Response.ExpectedResponse.StatusCode != cRes.StatusCode {
		return fmt.Errorf("expected status code to be %d, got %d", expected.Response.ExpectedResponse.StatusCode, cRes.StatusCode)
	}
	if cRes.StatusCode == 200 && !expected.Response.ExpectedResponseNoRequest {
		// The request expected to arrive at the backend is
		// the same as the request made, unless otherwise
		// specified.
		if expected.Request.ExpectedRequest == nil {
			expected.Request.ExpectedRequest = &ExpectedRequest{Request: expected.Request.ActualRequest}
		}

		if expected.Request.ExpectedRequest.Method == "" {
			expected.Request.ExpectedRequest.Method = "GET"
		}

		if expected.Request.ExpectedRequest.Host != "" && expected.Request.ExpectedRequest.Host != cReq.Host {
			return fmt.Errorf("expected host to be %s, got %s", expected.Request.ExpectedRequest.Host, cReq.Host)
		}

		if expected.Request.ExpectedRequest.Path != cReq.Path {
			return fmt.Errorf("expected path to be %s, got %s", expected.Request.ExpectedRequest.Path, cReq.Path)
		}
		if expected.Request.ExpectedRequest.Method != cReq.Method {
			return fmt.Errorf("expected method to be %s, got %s", expected.Request.ExpectedRequest.Method, cReq.Method)
		}
		if expected.Meta.TargetNamespace != cReq.Namespace {
			return fmt.Errorf("expected namespace to be %s, got %s", expected.Meta.TargetNamespace, cReq.Namespace)
		}
		if expected.Request.ExpectedRequest.Headers != nil {
			if cReq.Headers == nil {
				return fmt.Errorf("no headers captured, expected %v", len(expected.Request.ExpectedRequest.Headers))
			}
			for name, val := range cReq.Headers {
				cReq.Headers[strings.ToLower(name)] = val
			}
			for name, expectedVal := range expected.Request.ExpectedRequest.Headers {
				actualVal, ok := cReq.Headers[strings.ToLower(name)]
				if !ok {
					return fmt.Errorf("expected %s header to be set, actual headers: %v", name, cReq.Headers)
				} else if strings.Join(actualVal, ",") != expectedVal {
					return fmt.Errorf("expected %s header to be set to %s, got %s", name, expectedVal, strings.Join(actualVal, ","))
				}

			}
		}

		if expected.Response.ExpectedResponse.Headers != nil {
			if cRes.Headers == nil {
				return fmt.Errorf("no headers captured, expected %v", len(expected.Request.ExpectedRequest.Headers))
			}
			for name, val := range cRes.Headers {
				cRes.Headers[strings.ToLower(name)] = val
			}

			for name, expectedVal := range expected.Response.ExpectedResponse.Headers {
				actualVal, ok := cRes.Headers[strings.ToLower(name)]
				if !ok {
					return fmt.Errorf("expected %s header to be set, actual headers: %v", name, cRes.Headers)
				} else if strings.Join(actualVal, ",") != expectedVal {
					return fmt.Errorf("expected %s header to be set to %s, got %s", name, expectedVal, strings.Join(actualVal, ","))
				}
			}
		}

		if len(expected.Response.ExpectedResponse.AbsentHeaders) > 0 {
			for name, val := range cRes.Headers {
				cRes.Headers[strings.ToLower(name)] = val
			}

			for _, name := range expected.Response.ExpectedResponse.AbsentHeaders {
				val, ok := cRes.Headers[strings.ToLower(name)]
				if ok {
					return fmt.Errorf("expected %s header to not be set, got %s", name, val)
				}
			}
		}

		// Verify that headers expected *not* to be present on the
		// request are actually not present.
		if len(expected.Request.ExpectedRequest.AbsentHeaders) > 0 {
			for name, val := range cReq.Headers {
				cReq.Headers[strings.ToLower(name)] = val
			}

			for _, name := range expected.Request.ExpectedRequest.AbsentHeaders {
				val, ok := cReq.Headers[strings.ToLower(name)]
				if ok {
					return fmt.Errorf("expected %s header to not be set, got %s", name, val)
				}
			}
		}

		if !strings.HasPrefix(cReq.Pod, expected.Meta.TargetBackend) {
			return fmt.Errorf("expected pod name to start with %s, got %s", expected.Meta.TargetBackend, cReq.Pod)
		}
	} else if roundtripper.IsRedirect(cRes.StatusCode) {
		if expected.Request.RedirectRequest == nil {
			return nil
		}

		setRedirectRequestDefaults(req, cRes, &expected)

		if expected.Request.RedirectRequest.Host != cRes.RedirectRequest.Host {
			return fmt.Errorf("expected redirected hostname to be %s, got %s", expected.Request.RedirectRequest.Host, cRes.RedirectRequest.Host)
		}

		if expected.Request.RedirectRequest.Port != cRes.RedirectRequest.Port {
			return fmt.Errorf("expected redirected port to be %s, got %s", expected.Request.RedirectRequest.Port, cRes.RedirectRequest.Port)
		}

		if expected.Request.RedirectRequest.Scheme != cRes.RedirectRequest.Scheme {
			return fmt.Errorf("expected redirected scheme to be %s, got %s", expected.Request.RedirectRequest.Scheme, cRes.RedirectRequest.Scheme)
		}

		if expected.Request.RedirectRequest.Path != cRes.RedirectRequest.Path {
			return fmt.Errorf("expected redirected path to be %s, got %s", expected.Request.RedirectRequest.Path, cRes.RedirectRequest.Path)
		}
	}
	return nil
}

// Get User-defined test case name or generate from expected response to a given request.
func (er *Assertion) GetTestCaseName(i int) string {

	// If TestCase name is provided then use that or else generate one.
	if er.Meta.TestCaseName != "" {
		return er.Meta.TestCaseName
	}

	headerStr := ""
	reqStr := ""

	if er.Request.ActualRequest.Headers != nil {
		headerStr = " with headers"
	}

	reqStr = fmt.Sprintf("%d request to '%s%s'%s", i, er.Request.ActualRequest.Host, er.Request.ActualRequest.Path, headerStr)

	if er.Meta.TargetBackend != "" {
		return fmt.Sprintf("%s should go to %s", reqStr, er.Meta.TargetBackend)
	}
	return fmt.Sprintf("%s should receive a %d", reqStr, er.Response.ExpectedResponse.StatusCode)
}

func setRedirectRequestDefaults(req *roundtripper.Request, cRes *roundtripper.CapturedResponse, expected *Assertion) {
	// If the expected host is nil it means we do not test host redirect.
	// In that case we are setting it to the one we got from the response because we do not know the ip/host of the gateway.
	if expected.Request.RedirectRequest.Host == "" {
		expected.Request.RedirectRequest.Host = cRes.RedirectRequest.Host
	}

	if expected.Request.RedirectRequest.Port == "" {
		expected.Request.RedirectRequest.Port = req.URL.Port()
	}

	if expected.Request.RedirectRequest.Scheme == "" {
		expected.Request.RedirectRequest.Scheme = req.URL.Scheme
	}

	if expected.Request.RedirectRequest.Path == "" {
		expected.Request.RedirectRequest.Path = req.URL.Path
	}
}
