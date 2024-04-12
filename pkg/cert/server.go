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

package cert

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/caddyserver/certmagic"
	"k8s.io/client-go/kubernetes"
)

type Option struct {
	Namespace     string
	ServerAddress string
	Email         string
}

type Server struct {
	httpServer *http.Server
	opts       *Option
	clientSet  kubernetes.Interface
	controller *Controller
	certMgr    *CertMgr
}

func NewServer(clientSet kubernetes.Interface, opts *Option) (*Server, error) {
	server := &Server{
		clientSet: clientSet,
		opts:      opts,
	}
	return server, nil
}

func (s *Server) InitDefaultConfig() error {
	configMgr, _ := NewConfigMgr(s.opts.Namespace, s.clientSet)
	// init config if there is not existed
	_, err := configMgr.InitConfig(s.opts.Email)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) InitServer() error {
	configMgr, _ := NewConfigMgr(s.opts.Namespace, s.clientSet)
	// init config if there is not existed
	defaultConfig, err := configMgr.InitConfig(s.opts.Email)
	if err != nil {
		return err
	}
	// init certmgr
	certMgr, err := InitCertMgr(s.opts, s.clientSet, defaultConfig) // config and start
	s.certMgr = certMgr
	// init controller
	controller, err := NewController(s.clientSet, s.opts.Namespace, certMgr, configMgr)
	s.controller = controller
	// init http server
	s.initHttpServer()
	return nil
}

func (s *Server) initHttpServer() error {
	CertLog.Infof("server init http server")
	ctx := context.Background()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Lookit my cool website over HTTPS!")
	})
	httpServer := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       5 * time.Second,
		Addr:              s.opts.ServerAddress,
		BaseContext:       func(listener net.Listener) context.Context { return ctx },
	}
	cfg := s.certMgr.cfg
	if len(cfg.Issuers) > 0 {
		if am, ok := cfg.Issuers[0].(*certmagic.ACMEIssuer); ok {
			httpServer.Handler = am.HTTPChallengeHandler(mux)
		}
	} else {
		httpServer.Handler = mux
	}
	s.httpServer = httpServer
	return nil
}

func (s *Server) Run(stopCh <-chan struct{}) error {
	go s.controller.Run(stopCh)
	CertLog.Infof("server run")
	go func() {
		<-stopCh
		CertLog.Infof("server http server shutdown now...")
		s.httpServer.Shutdown(context.Background())
	}()
	err := s.httpServer.ListenAndServe()
	return err
}
