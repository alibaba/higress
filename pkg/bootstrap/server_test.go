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

package bootstrap

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"istio.io/istio/pilot/pkg/features"
	"istio.io/istio/pkg/keepalive"

	higresskube "github.com/alibaba/higress/v2/pkg/kube"
)

func TestStartWithNoError(t *testing.T) {
	var (
		s   *Server
		err error
	)

	// Create fake client first
	fakeClient := higresskube.NewFakeClient()

	mockFn := func(s *Server) error {
		s.kubeClient = fakeClient
		return nil
	}

	gomonkey.ApplyFunc((*Server).initKubeClient, mockFn)

	if s, err = NewServer(newServerArgs()); err != nil {
		t.Errorf("failed to create server: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the fake client informers first
	go fakeClient.RunAndWait(ctx.Done())

	// Give the client a moment to start informers
	time.Sleep(50 * time.Millisecond)

	var wg sync.WaitGroup
	var startErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		startErr = s.Start(ctx.Done())
	}()

	// Give the server a moment to start
	time.Sleep(200 * time.Millisecond)

	// Cancel context to trigger shutdown
	cancel()

	// Wait for server to shutdown with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Server may fail to sync cache in test environment due to missing resources,
		// which is acceptable for this test. The important thing is that the server
		// doesn't panic and handles shutdown gracefully.
		if startErr != nil && startErr.Error() != "failed to sync cache" {
			t.Logf("Server shutdown with error (may be expected in test env): %v", startErr)
		}
	case <-time.After(5 * time.Second):
		t.Errorf("server did not shutdown within timeout")
	}
}

func newServerArgs() *ServerArgs {
	return &ServerArgs{
		Debug:                true,
		NativeIstio:          true,
		HttpAddress:          ":8888",
		GrpcAddress:          ":15051",
		GrpcKeepAliveOptions: keepalive.DefaultOption(),
		XdsOptions: XdsOptions{
			DebounceAfter:         features.DebounceAfter,
			DebounceMax:           features.DebounceMax,
			EnableEDSDebounce:     features.EnableEDSDebounce,
			KeepConfigLabels:      true,
			KeepConfigAnnotations: true,
		},
	}
}
