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
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/mholt/acmez"
	"github.com/mholt/acmez/acme"
	v1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	IngressClassName   = "higress"
	IngressServiceName = "higress-controller"
	IngressNamePefix   = "higress-http-solver-"
	IngressPathPrefix  = "/.well-known/acme-challenge/"
	IngressServicePort = 8889
)

type IngressSolver struct {
	client       kubernetes.Interface
	acmeIssuer   *certmagic.ACMEIssuer
	solversMu    sync.Mutex
	namespace    string
	ingressDelay time.Duration
}

func NewIngressSolver(namespace string, client kubernetes.Interface, acmeIssuer *certmagic.ACMEIssuer) (acmez.Solver, error) {
	solver := &IngressSolver{
		namespace:    namespace,
		client:       client,
		acmeIssuer:   acmeIssuer,
		ingressDelay: 5 * time.Second,
	}
	return solver, nil
}

func (s *IngressSolver) Present(_ context.Context, challenge acme.Challenge) error {
	CertLog.Infof("ingress solver present challenge:%+v", challenge)
	s.solversMu.Lock()
	defer s.solversMu.Unlock()
	ingressName := s.getIngressName(challenge)
	ingress := s.constructIngress(challenge)
	CertLog.Infof("update ingress name:%s, ingress:%v", ingressName, ingress)
	_, err := s.client.NetworkingV1().Ingresses(s.namespace).Get(context.Background(), ingressName, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// create ingress
			_, err2 := s.client.NetworkingV1().Ingresses(s.namespace).Create(context.Background(), ingress, metav1.CreateOptions{})
			return err2
		}
		return err
	}
	_, err1 := s.client.NetworkingV1().Ingresses(s.namespace).Update(context.Background(), ingress, metav1.UpdateOptions{})
	if err1 != nil {
		return err1
	}
	return nil
}

func (s *IngressSolver) Wait(ctx context.Context, challenge acme.Challenge) error {
	CertLog.Infof("ingress solver wait challenge:%+v", challenge)
	// wait for ingress ready
	if s.ingressDelay > 0 {
		select {
		case <-time.After(s.ingressDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	CertLog.Infof("ingress solver wait challenge done")
	return nil
}

func (s *IngressSolver) CleanUp(_ context.Context, challenge acme.Challenge) error {
	CertLog.Infof("ingress solver cleanup challenge:%+v", challenge)
	s.solversMu.Lock()
	defer s.solversMu.Unlock()
	ingressName := s.getIngressName(challenge)
	CertLog.Infof("cleanup ingress name:%s", ingressName)
	err := s.client.NetworkingV1().Ingresses(s.namespace).Delete(context.Background(), ingressName, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (s *IngressSolver) Delete(_ context.Context, challenge acme.Challenge) error {
	s.solversMu.Lock()
	defer s.solversMu.Unlock()
	err := s.client.NetworkingV1().Ingresses(s.namespace).Delete(context.Background(), s.getIngressName(challenge), metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}

func (s *IngressSolver) getIngressName(challenge acme.Challenge) string {
	return IngressNamePefix + strings.ReplaceAll(challenge.Identifier.Value, ".", "-")
}

func (s *IngressSolver) constructIngress(challenge acme.Challenge) *v1.Ingress {
	ingressClassName := IngressClassName
	ingressDomain := challenge.Identifier.Value
	ingressPath := IngressPathPrefix + challenge.Token
	ingress := v1.Ingress{}
	ingress.Name = s.getIngressName(challenge)
	ingress.Namespace = s.namespace
	pathType := v1.PathTypePrefix
	ingress.Spec = v1.IngressSpec{
		IngressClassName: &ingressClassName,
		Rules: []v1.IngressRule{
			{
				Host: ingressDomain,
				IngressRuleValue: v1.IngressRuleValue{
					HTTP: &v1.HTTPIngressRuleValue{
						Paths: []v1.HTTPIngressPath{
							{
								Path:     ingressPath,
								PathType: &pathType,
								Backend: v1.IngressBackend{
									Service: &v1.IngressServiceBackend{
										Name: IngressServiceName,
										Port: v1.ServiceBackendPort{
											Number: IngressServicePort,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return &ingress
}
