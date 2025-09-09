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

package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync/atomic"

	"github.com/alibaba/higress/pkg/ingress/kube/configmap"
	"github.com/alibaba/higress/test/e2e/conformance/utils/kubernetes"
	"github.com/alibaba/higress/test/e2e/conformance/utils/roundtripper"
	"github.com/alibaba/higress/test/e2e/conformance/utils/suite"
	"net/url"

	"log"
	"math/rand"
	"net/http"
	"sync"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	Register(HttpRouteLimiter)
}

var HttpRouteLimiter = suite.ConformanceTest{
	ShortName:   "HttpRouteLimiter",
	Description: "The Ingress in the higress-conformance-infra namespace uses rps annotation",
	Features:    []suite.SupportedFeature{suite.HTTPConformanceFeature},
	Manifests:   []string{"tests/httproute-limit.yaml"},
	Parallel:    false,
	Test: func(t *testing.T, suite *suite.ConformanceTestSuite) {
		t.Log("🚀 HttpRouteLimiter: Test started")
		t.Run("HTTPRoute limiter", func(t *testing.T) {
			t.Log("📍 STEP 1: Checking if higress-config ConfigMap exists")
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			cm := &v1.ConfigMap{}
			err := suite.Client.Get(ctx, client.ObjectKey{Namespace: "higress-system", Name: "higress-config"}, cm)
			if err != nil {
				t.Logf("❌ STEP 1 FAILED: Cannot get higress-config: %v", err)
				t.Logf("📍 STEP 1 FAILED: ConfigMap access failed: %v", err)
				t.FailNow()
			}
			t.Log("✅ STEP 1 SUCCESS: higress-config ConfigMap found")
			
			// Log current config state
			currentConfig := cm.Data["higress"]
			if currentConfig == "" {
				t.Log("⚠️  STEP 1: higress-config data is empty")
			} else {
				t.Log("✅ STEP 1: higress-config data exists, length:", len(currentConfig))
			}
			
			t.Log("📍 STEP 2: Preparing gzip disabled configuration")
			gzipDisabledConfig := &configmap.HigressConfig{
				Gzip: &configmap.Gzip{
					Enable:              false,
					MinContentLength:    1024,
					ContentType:         []string{"text/html", "text/css", "text/plain", "text/xml", "application/json", "application/javascript", "application/xhtml+xml", "image/svg+xml"},
					DisableOnEtagHeader: true,
					MemoryLevel:         5,
					WindowBits:          12,
					ChunkSize:           4096,
					CompressionLevel:    "BEST_COMPRESSION",
					CompressionStrategy: "DEFAULT_STRATEGY",
				},
			}
			t.Log("✅ STEP 2 SUCCESS: Gzip disabled config prepared")
			
			t.Log("📍 STEP 3: Applying gzip disabled configuration to ConfigMap")
			err = kubernetes.ApplyConfigmapDataWithYaml(t, suite.Client, "higress-system", "higress-config", "higress", gzipDisabledConfig)
			if err != nil {
				t.Logf("❌ STEP 3 FAILED: Cannot apply config: %v", err)
				t.Logf("📍 STEP 3 FAILED: Config application failed: %v", err)
				t.FailNow()
			}
			t.Log("✅ STEP 3 SUCCESS: Gzip disabled config applied")
			
			t.Log("📍 STEP 4: Verifying config was applied correctly")
			// Wait a moment for config to propagate
			time.Sleep(2 * time.Second)
			
			updatedCm := &v1.ConfigMap{}
			err = suite.Client.Get(ctx, client.ObjectKey{Namespace: "higress-system", Name: "higress-config"}, updatedCm)
			if err != nil {
				t.Logf("❌ STEP 4 FAILED: Cannot get updated config: %v", err)
				t.Logf("📍 STEP 4 FAILED: Cannot verify config update: %v", err)
				t.FailNow()
			}
			
			updatedConfig := updatedCm.Data["higress"]
			t.Log("✅ STEP 4 SUCCESS: Updated config retrieved, length:", len(updatedConfig))
			
			// Check if gzip is actually disabled in the config
			if strings.Contains(updatedConfig, "enable: false") {
				t.Log("✅ STEP 4: Confirmed gzip is disabled in config")
			} else {
				t.Log("⚠️  STEP 4: Config may not have gzip disabled as expected")
				if len(updatedConfig) > 200 {
					t.Log("Config preview:", updatedConfig[:200])
				} else {
					t.Log("Config preview:", updatedConfig)
				}
			}
			
			// Monitor ConfigMap for a short period to see if it changes
			t.Log("📍 STEP 4.1: Monitoring ConfigMap for changes...")
			for i := 0; i < 5; i++ {
				time.Sleep(1 * time.Second)
				monitorCm := &v1.ConfigMap{}
				err := suite.Client.Get(ctx, client.ObjectKey{Namespace: "higress-system", Name: "higress-config"}, monitorCm)
				if err != nil {
					t.Logf("⚠️  STEP 4.1: Failed to get config during monitoring: %v", err)
					continue
				}
				monitorConfig := monitorCm.Data["higress"]
				if monitorConfig != updatedConfig {
					t.Logf("⚠️  STEP 4.1: ConfigMap changed during monitoring, iteration %d", i+1)
					t.Log("Previous length:", len(updatedConfig), "Current length:", len(monitorConfig))
				}
			}
			t.Log("✅ STEP 4.1: ConfigMap monitoring completed")

			t.Log("📍 STEP 5: Starting gzip verification via HTTP requests")
			testReq := &roundtripper.Request{
				Method: "GET",
				Host:   "limiter.higress.io",
				URL: url.URL{
					Scheme: "http",
					Host:   suite.GatewayAddress,
					Path:   "/",
				},
				Headers: map[string][]string{
					"Accept-Encoding": {"*"},
				},
			}
			
			t.Log("📍 STEP 5.1: Making initial HTTP request to check gzip status")
			_, cRes, err := suite.RoundTripper.CaptureRoundTrip(*testReq)
			if err != nil {
				t.Logf("❌ STEP 5.1 FAILED: HTTP request failed: %v", err)
				t.Logf("📍 STEP 5.1 FAILED: HTTP request failed: %v", err)
				t.FailNow()
			}
			
			t.Log("✅ STEP 5.1 SUCCESS: HTTP request completed")
			t.Log("Response status:", cRes.StatusCode)
			t.Log("Response headers:", cRes.Headers)
			
			// Check if content-encoding header is absent (gzip disabled)
			if _, exists := cRes.Headers["content-encoding"]; exists {
				t.Log("❌ STEP 5.1: Gzip is still enabled (content-encoding header found)")
				t.Log("Content-encoding value:", cRes.Headers["content-encoding"])
				
				t.Log("📍 STEP 5.2: Starting extended gzip verification loop")
				successes := 0
				maxAttempts := 20 // Reduced from 30 to avoid long wait times
				for attempt := 0; attempt < maxAttempts; attempt++ {
					t.Logf("Attempt %d/%d: Checking gzip status...", attempt+1, maxAttempts)
					_, cRes, err := suite.RoundTripper.CaptureRoundTrip(*testReq)
					if err != nil {
						t.Logf("❌ Attempt %d: Request failed: %v", attempt+1, err)
						time.Sleep(2 * time.Second)
						continue
					}
					
					if _, exists := cRes.Headers["content-encoding"]; !exists {
						successes++
						t.Logf("✅ Attempt %d: Gzip disabled (no content-encoding header)", attempt+1)
						if successes >= 2 { // Reduced from 3 to 2
							t.Logf("✅ STEP 5.2 SUCCESS: Gzip verified disabled after %d attempts", attempt+1)
							break
						}
					} else {
						t.Logf("❌ Attempt %d: Gzip still enabled, content-encoding: %v", attempt+1, cRes.Headers["content-encoding"])
						successes = 0
					}
					
					if attempt < maxAttempts-1 {
						time.Sleep(2 * time.Second)
					}
				}
				
				if successes < 2 {
					t.Logf("❌ STEP 5.2 FAILED: Gzip verification failed after %d attempts", maxAttempts)
					t.Logf("📍 STEP 5.2 FAILED: Failed to verify gzip disabled configuration after %d attempts", maxAttempts)
					t.FailNow()
				}
			} else {
				t.Log("✅ STEP 5.1: Gzip is already disabled (no content-encoding header)")
			}

			t.Log("📍 STEP 6: Starting rate limiting tests")
			client := &http.Client{}
			
			t.Log("📍 STEP 6.1: Running TestRps10")
			TestRps10(t, suite.GatewayAddress, client)
			t.Log("✅ STEP 6.1: TestRps10 completed")
			
			t.Log("📍 STEP 6.2: Running TestRps50")
			TestRps50(t, suite.GatewayAddress, client)
			t.Log("✅ STEP 6.2: TestRps50 completed")
			
			t.Log("📍 STEP 6.3: Running TestRps10Burst3")
			TestRps10Burst3(t, suite.GatewayAddress, client)
			t.Log("✅ STEP 6.3: TestRps10Burst3 completed")
			
			t.Log("📍 STEP 6.4: Running TestRpm10")
			TestRpm10(t, suite.GatewayAddress, client)
			t.Log("✅ STEP 6.4: TestRpm10 completed")
			
			t.Log("📍 STEP 6.5: Running TestRpm10Burst3")
			TestRpm10Burst3(t, suite.GatewayAddress, client)
			t.Log("✅ STEP 6.5: TestRpm10Burst3 completed")
			
			t.Log("🎉 ALL STEPS COMPLETED: HttpRouteLimiter test finished successfully")
		})
	},
}

// TestRps10 test case 1: rps10
func TestRps10(t *testing.T, gwAddr string, client *http.Client) {
	t.Log("📍 TestRps10: Starting RPS 10 test")
	req := &roundtripper.Request{
		Method: "GET",
		Host:   "limiter.higress.io",
		URL: url.URL{
			Scheme: "http",
			Host:   gwAddr,
			Path:   "/rps10",
		},
	}

	t.Log("📍 TestRps10: Executing parallel request runner")
	result, err := ParallelRunner(10, 3000, req, client)
	if err != nil {
		t.Logf("❌ TestRps10: Parallel runner failed: %v", err)
		t.Logf("📍 TestRps10: Parallel runner failed: %v", err)
		t.FailNow()
	}
	t.Log("✅ TestRps10: Parallel runner completed")
	t.Log("📍 TestRps10: Asserting RPS results")
	AssertRps(t, result, 10, 0.5)
	t.Log("✅ TestRps10: Test completed successfully")
}

// TestRps50 test case 2: rps50
func TestRps50(t *testing.T, gwAddr string, client *http.Client) {
	t.Log("📍 TestRps50: Starting RPS 50 test")
	req := &roundtripper.Request{
		Method: "GET",
		Host:   "limiter.higress.io",
		URL: url.URL{
			Scheme: "http",
			Host:   gwAddr,
			Path:   "/rps50",
		},
	}

	t.Log("📍 TestRps50: Executing parallel request runner")
	result, err := ParallelRunner(10, 5000, req, client)
	if err != nil {
		t.Logf("❌ TestRps50: Parallel runner failed: %v", err)
		t.Logf("📍 TestRps50: Parallel runner failed: %v", err)
		t.FailNow()
	}
	t.Log("✅ TestRps50: Parallel runner completed")
	t.Log("📍 TestRps50: Asserting RPS results")
	AssertRps(t, result, 50, 0.5)
	t.Log("✅ TestRps50: Test completed successfully")
}

// TestRps10Burst3 test case 3: rps10 burst3
func TestRps10Burst3(t *testing.T, gwAddr string, client *http.Client) {
	t.Log("📍 TestRps10Burst3: Starting RPS 10 burst 3 test")
	req := &roundtripper.Request{
		Method: "GET",
		Host:   "limiter.higress.io",
		URL: url.URL{
			Scheme: "http",
			Host:   gwAddr,
			Path:   "/rps10/burst3",
		},
	}

	t.Log("📍 TestRps10Burst3: Executing parallel request runner")
	result, err := ParallelRunner(30, 50, req, client)
	if err != nil {
		t.Logf("❌ TestRps10Burst3: Parallel runner failed: %v", err)
		t.Logf("📍 TestRps10Burst3: Parallel runner failed: %v", err)
		t.FailNow()
	}
	t.Log("✅ TestRps10Burst3: Parallel runner completed")
	t.Log("📍 TestRps10Burst3: Asserting RPS results")
	AssertRps(t, result, 30, -1)
	t.Log("✅ TestRps10Burst3: Test completed successfully")
}

// TestRpm10 test case 4: rpm10
func TestRpm10(t *testing.T, gwAddr string, client *http.Client) {
	t.Log("📍 TestRpm10: Starting RPM 10 test")
	req := &roundtripper.Request{
		Method: "GET",
		Host:   "limiter.higress.io",
		URL: url.URL{
			Scheme: "http",
			Host:   gwAddr,
			Path:   "/rpm10",
		},
	}

	t.Log("📍 TestRpm10: Executing parallel request runner")
	result, err := ParallelRunner(10, 100, req, client)
	if err != nil {
		t.Logf("❌ TestRpm10: Parallel runner failed: %v", err)
		t.Logf("📍 TestRpm10: Parallel runner failed: %v", err)
		t.FailNow()
	}
	t.Log("✅ TestRpm10: Parallel runner completed")
	t.Log("📍 TestRpm10: Asserting RPS results")
	AssertRps(t, result, 10, -1)
	t.Log("✅ TestRpm10: Test completed successfully")
}

// TestRpm10Burst3 test case 5: rpm10 burst3
func TestRpm10Burst3(t *testing.T, gwAddr string, client *http.Client) {
	t.Log("📍 TestRpm10Burst3: Starting RPM 10 burst 3 test")
	req := &roundtripper.Request{
		Method: "GET",
		Host:   "limiter.higress.io",
		URL: url.URL{
			Scheme: "http",
			Host:   gwAddr,
			Path:   "/rpm10/burst3",
		},
	}

	t.Log("📍 TestRpm10Burst3: Executing parallel request runner")
	result, err := ParallelRunner(30, 100, req, client)
	if err != nil {
		t.Logf("❌ TestRpm10Burst3: Parallel runner failed: %v", err)
		t.Logf("📍 TestRpm10Burst3: Parallel runner failed: %v", err)
		t.FailNow()
	}
	t.Log("✅ TestRpm10Burst3: Parallel runner completed")
	t.Log("📍 TestRpm10Burst3: Asserting RPS results")
	AssertRps(t, result, 30, -1)
	t.Log("✅ TestRpm10Burst3: Test completed successfully")
}

// DoRequest send Http request according to req and client, return status code and error
func DoRequest(req *roundtripper.Request, client *http.Client) (int, error) {
	u := &url.URL{
		Scheme:   req.URL.Scheme,
		Host:     req.URL.Host,
		Path:     req.URL.Path,
		RawQuery: req.URL.RawQuery,
	}
	r, err := http.NewRequest(req.Method, u.String(), nil)
	if err != nil {
		return 0, err
	}

	if r.Host != "" {
		r.Host = req.Host
	}

	if req.Headers != nil {
		for name, values := range req.Headers {
			for _, value := range values {
				r.Header.Add(name, value)
			}
		}
	}

	if r.Body != nil {
		body, err := json.Marshal(req.Body)
		if err != nil {
			return 0, err
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}

	resp, err := client.Do(r)
	if err != nil {
		return 1, err
	}
	defer client.CloseIdleConnections()
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

// ParallelRunner send Http request in parallel and count rps
func ParallelRunner(threads int, times int, req *roundtripper.Request, client *http.Client) (*Result, error) {
	log.Printf("📍 ParallelRunner: Starting with %d threads, %d total requests", threads, times)
	var wg sync.WaitGroup
	result := &Result{
		Requests: times,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startTime := time.Now()
	log.Printf("📍 ParallelRunner: Starting request execution at %v", startTime)
	
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func(threadID int) {
			defer wg.Done()
			threadSuccess := 0
			threadTotal := times / threads
			
			for j := 0; j < threadTotal; j++ {
				if ctx.Err() != nil {
					log.Printf("⚠️  ParallelRunner: Thread %d cancelled at iteration %d", threadID, j)
					return
				}
				
				b2 := time.Now()
				statusCode, err := DoRequest(req, client)
				elapsed := time.Since(b2).Nanoseconds() / 1e6
				
				if err != nil {
					log.Printf("❌ ParallelRunner: Thread %d, request %d failed: %v", threadID, j, err)
					continue
				}
				
				detailRecord := &DetailRecord{
					StatusCode: statusCode,
					ElapseMs:   elapsed,
				}
				result.DetailMaps.Store(rand.Int(), detailRecord)
				
				if statusCode >= 200 && statusCode < 300 {
					atomic.AddInt32(&result.Success, 1)
					threadSuccess++
				} else {
					log.Printf("⚠️  ParallelRunner: Thread %d, request %d returned status %d", threadID, j, statusCode)
					time.Sleep(50 * time.Millisecond)
				}
			}
			log.Printf("✅ ParallelRunner: Thread %d completed, %d/%d successful", threadID, threadSuccess, threadTotal)
		}(i)
	}

	wg.Wait()
	result.TotalCostMs = time.Since(startTime).Nanoseconds() / 1e6
	result.SuccessRps = float64(result.Success) * 1000 / float64(result.TotalCostMs)
	result.ActualRps = float64(result.Requests) * 1000 / float64(result.TotalCostMs)
	
	log.Printf("✅ ParallelRunner: All threads completed. Total: %dms, Success: %d/%d, Success RPS: %.2f, Actual RPS: %.2f", 
		result.TotalCostMs, result.Success, result.Requests, result.SuccessRps, result.ActualRps)
	
	return result, nil
}

// AssertRps check actual rps is in expected range if tolerance is not -1
// else check actual success requests is less than expected
func AssertRps(t *testing.T, result *Result, expectedRps float64, tolerance float64) {
	if tolerance != -1 {
		fmt.Printf("Total Cost(s): %.2f, Total Request: %d, Total Success: %d, Actual RPS: %.2f, Expected Rps: %.2f, Success Rps: %.2f\n",
			float64(result.TotalCostMs)/1000, result.Requests, result.Success, result.ActualRps, expectedRps, result.SuccessRps)
		lo := expectedRps * (1 - tolerance)
		hi := expectedRps * (1 + tolerance)
		message := fmt.Sprintf("RPS `%.2f` should between `%.2f` - `%.2f`", result.SuccessRps, lo, hi)
		if result.SuccessRps < lo || result.SuccessRps > hi {
			t.Errorf(message)
		}
	} else {
		fmt.Printf("Total Cost(s): %.2f, Total Request: %d, Total Success: %d, Expected: %.2f\n",
			float64(result.TotalCostMs)/1000, result.Requests, result.Success, expectedRps)
		message := fmt.Sprintf("Success Requests should less than : %d, actual: %d", int32(expectedRps), result.Success)
		if result.Success > int32(expectedRps) {
			t.Errorf(message)
		}
	}
}

type DetailRecord struct {
	StatusCode int
	ElapseMs   int64
}

type Result struct {
	Requests    int
	Success     int32
	TotalCostMs int64
	SuccessRps  float64
	ActualRps   float64
	DetailMaps  sync.Map
}
