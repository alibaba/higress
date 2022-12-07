package bootstrap

import (
	"context"
	"github.com/agiledragon/gomonkey/v2"
	"istio.io/istio/pilot/pkg/features"
	"istio.io/istio/pkg/keepalive"
	kubelib "istio.io/istio/pkg/kube"
	"testing"
	"time"
)

func TestStartWithNoError(t *testing.T) {
	var (
		s   *Server
		err error
	)

	mockFn := func(s *Server) error {
		s.kubeClient = kubelib.NewFakeClient()
		return nil
	}

	gomonkey.ApplyFunc((*Server).initKubeClient, mockFn)

	if s, err = NewServer(newServerArgs()); err != nil {
		t.Errorf("failed to create server: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
