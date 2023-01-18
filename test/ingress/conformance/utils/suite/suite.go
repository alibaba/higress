// Copyright (c) 2022 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package suite

import (
	"testing"

	"github.com/alibaba/higress/test/ingress/conformance/utils/config"
	"github.com/alibaba/higress/test/ingress/conformance/utils/kubernetes"
	"github.com/alibaba/higress/test/ingress/conformance/utils/roundtripper"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SupportedFeature allows opting in to additional conformance tests at an
// individual feature granularity.
type SupportedFeature string

const (
	// This option indicates support for TLSRoute (extended conformance).
	SupportTLSRoute SupportedFeature = "TLSRoute"

	// This option indicates support for HTTPRoute query param matching (extended conformance).
	SupportHTTPRouteQueryParamMatching SupportedFeature = "HTTPRouteQueryParamMatching"

	// This option indicates support for HTTPRoute method matching (extended conformance).
	SupportHTTPRouteMethodMatching SupportedFeature = "HTTPRouteMethodMatching"

	// This option indicates support for HTTPRoute response header modification (extended conformance).
	SupportHTTPResponseHeaderModification SupportedFeature = "HTTPResponseHeaderModification"

	// This option indicates support for Destination Port matching (extended conformance).
	SupportRouteDestinationPortMatching SupportedFeature = "RouteDestinationPortMatching"

	// This option indicates support for HTTPRoute port redirect (extended conformance).
	SupportHTTPRoutePortRedirect SupportedFeature = "HTTPRoutePortRedirect"

	// This option indicates support for HTTPRoute scheme redirect (extended conformance).
	SupportHTTPRouteSchemeRedirect SupportedFeature = "HTTPRouteSchemeRedirect"

	// This option indicates support for HTTPRoute path redirect (experimental conformance).
	SupportHTTPRoutePathRedirect SupportedFeature = "HTTPRoutePathRedirect"

	// This option indicates support for HTTPRoute host rewrite (experimental conformance)
	SupportHTTPRouteHostRewrite SupportedFeature = "HTTPRouteHostRewrite"

	// This option indicates support for HTTPRoute path rewrite (experimental conformance)
	SupportHTTPRoutePathRewrite SupportedFeature = "HTTPRoutePathRewrite"
)

// StandardCoreFeatures are the features that are required to be conformant with
// the Core API features that are part of the Standard release channel.
var StandardCoreFeatures = map[SupportedFeature]bool{}

// ConformanceTestSuite defines the test suite used to run Gateway API
// conformance tests.
type ConformanceTestSuite struct {
	Client            client.Client
	RoundTripper      roundtripper.RoundTripper
	GatewayAddress    string
	IngressClassName  string
	ControllerName    string
	Debug             bool
	Cleanup           bool
	BaseManifests     string
	Applier           kubernetes.Applier
	SupportedFeatures map[SupportedFeature]bool
	TimeoutConfig     config.TimeoutConfig
}

// Options can be used to initialize a ConformanceTestSuite.
type Options struct {
	Client           client.Client
	GatewayAddress   string
	IngressClassName string
	Debug            bool
	RoundTripper     roundtripper.RoundTripper
	BaseManifests    string
	NamespaceLabels  map[string]string
	// ValidUniqueListenerPorts maps each listener port of each Gateway in the
	// manifests to a valid, unique port. There must be as many
	// ValidUniqueListenerPorts as there are listeners in the set of manifests.
	// For example, given two Gateways, each with 2 listeners, there should be
	// four ValidUniqueListenerPorts.
	// If empty or nil, ports are not modified.
	ValidUniqueListenerPorts []int

	// CleanupBaseResources indicates whether or not the base test
	// resources such as Gateways should be cleaned up after the run.
	CleanupBaseResources bool
	SupportedFeatures    map[SupportedFeature]bool
	TimeoutConfig        config.TimeoutConfig
}

// New returns a new ConformanceTestSuite.
func New(s Options) *ConformanceTestSuite {
	config.SetupTimeoutConfig(&s.TimeoutConfig)

	roundTripper := s.RoundTripper
	if roundTripper == nil {
		roundTripper = &roundtripper.DefaultRoundTripper{Debug: s.Debug, TimeoutConfig: s.TimeoutConfig}
	}

	if s.SupportedFeatures == nil {
		s.SupportedFeatures = StandardCoreFeatures
	} else {
		for feature, val := range StandardCoreFeatures {
			if _, ok := s.SupportedFeatures[feature]; !ok {
				s.SupportedFeatures[feature] = val
			}
		}
	}

	suite := &ConformanceTestSuite{
		Client:           s.Client,
		RoundTripper:     roundTripper,
		IngressClassName: s.IngressClassName,
		Debug:            s.Debug,
		Cleanup:          s.CleanupBaseResources,
		BaseManifests:    s.BaseManifests,
		GatewayAddress:   s.GatewayAddress,
		Applier: kubernetes.Applier{
			NamespaceLabels:          s.NamespaceLabels,
			ValidUniqueListenerPorts: s.ValidUniqueListenerPorts,
		},
		SupportedFeatures: s.SupportedFeatures,
		TimeoutConfig:     s.TimeoutConfig,
	}

	// apply defaults
	if suite.BaseManifests == "" {
		suite.BaseManifests = "base/manifests.yaml"
	}

	return suite
}

// Setup ensures the base resources required for conformance tests are installed
// in the cluster. It also ensures that all relevant resources are ready.
func (suite *ConformanceTestSuite) Setup(t *testing.T) {
	t.Logf("Test Setup: Ensuring IngressClass has been accepted")
	suite.ControllerName = suite.IngressClassName

	suite.Applier.IngressClass = suite.IngressClassName
	suite.Applier.ControllerName = suite.ControllerName

	t.Logf("Test Setup: Applying base manifests")
	suite.Applier.MustApplyWithCleanup(t, suite.Client, suite.TimeoutConfig, suite.BaseManifests, suite.Cleanup)

	t.Logf("Test Setup: Applying programmatic resources")
	secret := kubernetes.MustCreateSelfSignedCertSecret(t, "higress-conformance-web-backend", "certificate", []string{"*"})
	suite.Applier.MustApplyObjectsWithCleanup(t, suite.Client, suite.TimeoutConfig, []client.Object{secret}, suite.Cleanup)
	secret = kubernetes.MustCreateSelfSignedCertSecret(t, "higress-conformance-infra", "tls-validity-checks-certificate", []string{"*"})
	suite.Applier.MustApplyObjectsWithCleanup(t, suite.Client, suite.TimeoutConfig, []client.Object{secret}, suite.Cleanup)

	t.Logf("Test Setup: Ensuring Pods from base manifests are ready")
	namespaces := []string{
		"higress-conformance-infra",
		"higress-conformance-app-backend",
		"higress-conformance-web-backend",
	}
	kubernetes.NamespacesMustBeAccepted(t, suite.Client, suite.TimeoutConfig, namespaces)
}

// Run runs the provided set of conformance tests.
func (suite *ConformanceTestSuite) Run(t *testing.T, tests []ConformanceTest) {
	for _, test := range tests {
		t.Run(test.ShortName, func(t *testing.T) {
			test.Run(t, suite)
		})
	}
}

// ConformanceTest is used to define each individual conformance test.
type ConformanceTest struct {
	ShortName   string
	Description string
	Features    []SupportedFeature
	Manifests   []string
	Slow        bool
	Parallel    bool
	Test        func(*testing.T, *ConformanceTestSuite)
}

// Run runs an individual tests, applying and cleaning up the required manifests
// before calling the Test function.
func (test *ConformanceTest) Run(t *testing.T, suite *ConformanceTestSuite) {
	if test.Parallel {
		t.Parallel()
	}

	// Check that all features exercised by the test have been opted into by
	// the suite.
	for _, feature := range test.Features {
		if supported, ok := suite.SupportedFeatures[feature]; !ok || !supported {
			t.Skipf("Skipping %s: suite does not support %s", test.ShortName, feature)
		}
	}

	for _, manifestLocation := range test.Manifests {
		t.Logf("Applying %s", manifestLocation)
		suite.Applier.MustApplyWithCleanup(t, suite.Client, suite.TimeoutConfig, manifestLocation, true)
	}

	test.Test(t, suite)
}
