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
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"istio.io/istio/pilot/pkg/features"
	"istio.io/istio/pkg/keepalive"

	higresskube "github.com/alibaba/higress/pkg/kube"
)

func TestStartWithNoError(t *testing.T) {
	var (
		s   *Server
		err error
	)

	mockFn := func(s *Server) error {
		s.kubeClient = higresskube.NewFakeClient()
		return nil
	}

	gomonkey.ApplyFunc((*Server).initKubeClient, mockFn)

	if s, err = NewServer(newServerArgs()); err != nil {
		t.Errorf("failed to create server: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err = s.Start(ctx.Done()); err != nil {
		t.Errorf("failed to start the server: %v", err)
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
			DebounceAfter:     features.DebounceAfter,
			DebounceMax:       features.DebounceMax,
			EnableEDSDebounce: features.EnableEDSDebounce,
		},
	}
}
