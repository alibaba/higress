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

package suite

import (
	"fmt"
	"testing"

	"github.com/alibaba/higress/test/e2e/conformance/utils/config"
	"github.com/alibaba/higress/test/e2e/conformance/utils/kubernetes"
	"github.com/alibaba/higress/test/e2e/conformance/utils/roundtripper"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ConformanceTestSuite defines the test suite used to run Gateway API
// conformance tests.
type ConformanceTestSuite struct {
	Client           client.Client
	RoundTripper     roundtripper.RoundTripper
	GatewayAddress   string
	IngressClassName string
	Debug            bool
	Cleanup          bool
	BaseManifests    string
	Applier          kubernetes.Applier
	TimeoutConfig    config.TimeoutConfig
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

	// CleanupBaseResources indicates whether or not the base test
	// resources such as Gateways should be cleaned up after the run.
	CleanupBaseResources bool
	TimeoutConfig        config.TimeoutConfig
}

// New returns a new ConformanceTestSuite.
func New(s Options) *ConformanceTestSuite {
	config.SetupTimeoutConfig(&s.TimeoutConfig)

	roundTripper := s.RoundTripper
	if roundTripper == nil {
		roundTripper = &roundtripper.DefaultRoundTripper{Debug: s.Debug, TimeoutConfig: s.TimeoutConfig}
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
			NamespaceLabels: s.NamespaceLabels,
		},
		TimeoutConfig: s.TimeoutConfig,
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

	suite.Applier.IngressClass = suite.IngressClassName

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
		"nacos-standlone-rc3",
		"dubbo-demo-provider",
	}
	kubernetes.NamespacesMustBeAccepted(t, suite.Client, suite.TimeoutConfig, namespaces)
}

// Run runs the provided set of conformance tests.
func (suite *ConformanceTestSuite) Run(t *testing.T, tests []ConformanceTest) {

	t.Logf("Start Running %d Test Cases: \n\n%s", len(tests), globalConformanceTestsListInfo(tests))
	for _, test := range tests {
		t.Run(test.ShortName, func(t *testing.T) {
			test.Run(t, suite)
		})
	}
}

func globalConformanceTestsListInfo(tests []ConformanceTest) string {
	var cases string
	for index, test := range tests {
		cases += fmt.Sprintf("CaseNum: %d\nCaseName: %s\nScenario: %s\n\n", index+1, test.ShortName, test.Description)
	}

	return cases
}

// ConformanceTest is used to define each individual conformance test.
type ConformanceTest struct {
	ShortName   string
	Description string
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

	for _, manifestLocation := range test.Manifests {
		t.Logf("Applying %s", manifestLocation)
		suite.Applier.MustApplyWithCleanup(t, suite.Client, suite.TimeoutConfig, manifestLocation, true)
	}

	test.Test(t, suite)
}
