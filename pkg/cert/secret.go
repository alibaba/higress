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
	"strconv"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type SecretMgr struct {
	client    kubernetes.Interface
	namespace string
}

func NewSecretMgr(namespace string, client kubernetes.Interface) (*SecretMgr, error) {
	secretMgr := &SecretMgr{
		namespace: namespace,
		client:    client,
	}

	return secretMgr, nil
}

func (s *SecretMgr) Update(domain string, secretName string, privateKey []byte, certificate []byte, notBefore time.Time, notAfter time.Time, isRenew bool) error {
	name := secretName
	namespace := s.namespace
	namespaceP, secretP := ParseTLSSecret(secretName)
	if namespaceP != "" {
		namespace = namespaceP
		name = secretP
	}

	secret := s.constructSecret(domain, name, namespace, privateKey, certificate, notBefore, notAfter, isRenew)
	_, err := s.client.CoreV1().Secrets(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// create secret
			_, err2 := s.client.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
			return err2
		}
		return err
	}
	// check secret annotations
	if _, ok := secret.Annotations["higress.io/cert-domain"]; !ok {
		return fmt.Errorf("the secret name %s is not automatic https secret name for the domain:%s, please rename it in config", secretName, domain)
	}
	_, err1 := s.client.CoreV1().Secrets(namespace).Update(context.Background(), secret, metav1.UpdateOptions{})
	if err1 != nil {
		return err1
	}

	return nil
}

func (s *SecretMgr) constructSecret(domain string, name string, namespace string, privateKey []byte, certificate []byte, notBefore time.Time, notAfter time.Time, isRenew bool) *v1.Secret {
	annotationMap := make(map[string]string, 0)
	annotationMap["higress.io/cert-domain"] = domain
	annotationMap["higress.io/cert-notAfter"] = notAfter.Format("2006-01-02 15:04:05")
	annotationMap["higress.io/cert-notBefore"] = notBefore.Format("2006-01-02 15:04:05")
	annotationMap["higress.io/cert-renew"] = strconv.FormatBool(isRenew)
	if isRenew {
		annotationMap["higress.io/cert-renew-time"] = time.Now().Format("2006-01-02 15:04:05")
	}
	// Required fields:
	// - Secret.Data["tls.key"] - TLS private key.
	//   Secret.Data["tls.crt"] - TLS certificate.
	dataMap := make(map[string][]byte, 0)
	dataMap["tls.key"] = privateKey
	dataMap["tls.crt"] = certificate
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotationMap,
		},
		Type: v1.SecretTypeTLS,
		Data: dataMap,
	}
	return secret
}
