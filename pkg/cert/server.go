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
	"sync"
	"time"

	"github.com/caddyserver/certmagic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

const (
	EventCertObtained = "cert_obtained"
)

type Server struct {
	cfg             *certmagic.Config
	httpServer      *http.Server
	client          kubernetes.Interface
	namespace       string
	mux             sync.RWMutex
	storage         certmagic.Storage
	cache           *certmagic.Cache
	myACME          *certmagic.ACMEIssuer
	ingressResolver *IngressSolver
	configMgr       *ConfigMgr
	secretMgr       *SecretMgr
}

func NewServer(namespace string, client kubernetes.Interface, configMgr *ConfigMgr) (*Server, error) {
	server := &Server{
		client:    client,
		namespace: namespace,
		configMgr: configMgr,
	}
	return server, nil
}

func (s *Server) InitConfig(config *Config) error {
	klog.Infof("server init config: %+v", config)
	// Init certmagic config
	// First make a pointer to a Cache as we need to reference the same Cache in
	// GetConfigForCert below.
	var cache *certmagic.Cache
	var storage certmagic.Storage
	storage, _ = NewConfigmapStorage(s.namespace, s.client)
	magicConfig := certmagic.Config{
		RenewalWindowRatio: config.RenewalWindowRatio,
		Storage:            storage,
		OnEvent:            s.OnEvent,
	}
	cache = certmagic.NewCache(certmagic.CacheOptions{
		GetConfigForCert: func(cert certmagic.Certificate) (*certmagic.Config, error) {
			// Here we use New to get a valid Config associated with the same cache.
			// The provided Config is used as a template and will be completed with
			// any defaults that are set in the Default config.
			return certmagic.New(cache, magicConfig), nil
		},
	})
	// init certmagic
	cfg := certmagic.New(cache, magicConfig)

	// Init certmagic acme
	myACME := certmagic.NewACMEIssuer(cfg, certmagic.ACMEIssuer{
		CA:                      certmagic.LetsEncryptStagingCA,
		Email:                   config.Email,
		Agreed:                  true,
		DisableHTTPChallenge:    false,
		DisableTLSALPNChallenge: true,
	})
	// inject http01 solver
	ingressSolver, _ := NewIngressResolver(s.namespace, s.client, myACME)
	myACME.Http01Solver = ingressSolver
	// init issuers
	cfg.Issuers = []certmagic.Issuer{myACME}

	// hold in server for easy access
	s.myACME = myACME
	s.cfg = cfg
	s.cache = cache
	s.storage = storage

	return nil
}

func (s *Server) Reconcile(ctx context.Context, oldConfig *Config, newConfig *Config) error {
	klog.Infof("sever reconcile old config:%+v to new config:%+v", oldConfig, newConfig)
	// sync email
	if oldConfig != nil && oldConfig.Email != newConfig.Email {
		// TODO before sync email, maybe need to clean up cache and account
		// s.CleanSync(cdontext.Background(), oldConfig.Domains)
	}

	// sync domains
	newDomains := make([]string, 0)
	newDomainsMap := make(map[string]string, 0)
	removeDomains := make([]string, 0)

	for _, newDomain := range newConfig.Domains {
		newDomains = append(newDomains, newDomain)
		newDomainsMap[newDomain] = newDomain
	}

	if oldConfig != nil {
		for _, oldDomain := range oldConfig.Domains {
			if _, ok := newDomainsMap[oldDomain]; !ok {
				removeDomains = append(removeDomains, oldDomain)
			}
		}
	}

	if newConfig.AutomaticHttps == true {
		// clean up  unused domains
		s.CleanSync(context.Background(), removeDomains)
		// sync email
		s.myACME.Email = newConfig.Email
		// sync RenewalWindowRatio
		s.cfg.RenewalWindowRatio = newConfig.RenewalWindowRatio
		// start cache
		s.cache.Start()
		// sync domains
		s.ManageSync(context.Background(), newDomains)
		s.configMgr.SetConfig(newConfig)
	} else {
		// stop cache  maintainAssets
		s.cache.Stop()
		s.configMgr.SetConfig(newConfig)
	}

	return nil
}

func (s *Server) ManageSync(ctx context.Context, domainNames []string) error {
	klog.Infof("server manage sync domains:%v", domainNames)
	return s.cfg.ManageSync(ctx, domainNames)
}

func (s *Server) CleanSync(ctx context.Context, domainNames []string) error {
	//TODO implement clean up domains
	klog.Infof("server clean sync domains:%v", domainNames)
	return nil
}

func (s *Server) InitHttpServer() error {
	klog.Infof("server init http server")
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
		Addr:              ":80",
		BaseContext:       func(listener net.Listener) context.Context { return ctx },
	}
	if len(s.cfg.Issuers) > 0 {
		if am, ok := s.cfg.Issuers[0].(*certmagic.ACMEIssuer); ok {
			httpServer.Handler = am.HTTPChallengeHandler(mux)
		}
	} else {
		httpServer.Handler = mux
	}
	s.httpServer = httpServer
	return nil
}

//func (s *Server) HTTPChallengeHandler(am *certmagic.ACMEIssuer, h http.Handler) http.Handler {
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		if am.HandleHTTPChallenge(w, r) {
//			// clean up ingress route here
//			identifier := hostOnly(r.Host)
//			chalData, ok := certmagic.GetACMEChallenge(identifier)
//			if ok {
//				err := s.ingressResolver.Delete(context.Background(), chalData.Challenge)
//				if err != nil {
//				}
//			}
//			return
//		}
//		h.ServeHTTP(w, r)
//	})
//}

func (s *Server) OnEvent(ctx context.Context, event string, data map[string]any) error {
	klog.Infof("server receive event:% data:%+v", event, data)
	/**
	event: cert_obtained
	cfg.emit(ctx, "cert_obtained", map[string]any{
		"renewal":          true,
		"remaining":        timeLeft,
		"identifier":       name,
		"issuer":           issuerKey,
		"storage_path":     StorageKeys.CertsSitePrefix(issuerKey, certKey),
		"private_key_path": StorageKeys.SitePrivateKey(issuerKey, certKey),
		"certificate_path": StorageKeys.SiteCert(issuerKey, certKey),
		"metadata_path":    StorageKeys.SiteMeta(issuerKey, certKey),
	})
	*/
	if event == EventCertObtained {
		// obtain certificate and update secret
		domain := data["identifier"].(string)
		isRenew := data["renewal"].(bool)
		privateKeyPath := data["private_key_path"].(string)
		certificatePath := data["certificate_path"].(string)
		privateKey, err := s.cfg.Storage.Load(context.Background(), privateKeyPath)
		certificate, err := s.cfg.Storage.Load(context.Background(), certificatePath)
		certChain, err := parseCertsFromPEMBundle(certificate)
		if err != nil {
			return err
		}
		notAfterTime := notAfter(certChain[0])
		notBeforeTime := notBefore(certChain[0])
		return s.secretMgr.Update(domain, privateKey, certificate, notBeforeTime, notAfterTime, isRenew)
	}
	return nil
}

func (s *Server) Run(ctx context.Context) error {
	klog.Infof("server run")
	go func() {
		<-ctx.Done()
		klog.Infof("server http server shutdown now...")
		s.httpServer.Shutdown(context.Background())
	}()
	err := s.httpServer.ListenAndServe()
	return err
}
