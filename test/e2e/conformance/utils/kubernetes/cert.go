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

package kubernetes

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"io"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	// ensure auth plugins are loaded
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/alibaba/higress/test/e2e/conformance/utils/cert"
)

// MustCreateSelfSignedCertSecret creates a self-signed SSL certificate and stores it in a secret
func MustCreateSelfSignedCertSecret(t *testing.T, namespace, secretName string, hosts []string) *corev1.Secret {
	require.Greater(t, len(hosts), 0, "require a non-empty hosts for Subject Alternate Name values")

	var serverKey, serverCert bytes.Buffer
	host := strings.Join(hosts, ",")
	require.NoError(t, generateRSACert(host, &serverKey, &serverCert), "failed to generate RSA certificate")

	return ConstructTLSSecret(namespace, secretName, serverCert.Bytes(), serverKey.Bytes())
}

// ConstructTLSSecret constructs a secret of type "kubernetes.io/tls"
func ConstructTLSSecret(namespace, secretName string, cert, key []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      secretName,
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			corev1.TLSCertKey:       cert,
			corev1.TLSPrivateKeyKey: key,
		},
	}
}

// ConstructCASecret construct a CA secret of type "Opaque"
func ConstructCASecret(namespace, secretName string, cert []byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      secretName,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			corev1.ServiceAccountRootCAKey: cert,
		},
	}
}

// generateRSACert generates a basic self signed certificate valid for a year
func generateRSACert(host string, keyOut, certOut io.Writer) error {
	notBefore := time.Now()
	notAfter := notBefore.Add(cert.ValidFor)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "default",
			Organization: []string{"Higress E2E Test"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	hosts := strings.Split(host, ",")
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}

	priv, err := rsa.GenerateKey(rand.Reader, cert.RSABits)
	if err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}
	certOut, keyOut, err = cert.GenerateCert(template, priv, template, nil)
	if err != nil {
		return fmt.Errorf("failed to generate rsa certificate: %w", err)
	}

	return nil
}
