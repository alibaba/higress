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

package cert

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type CertType int

const (
	CACertType CertType = iota
	ServerCertType
	ClientCertType
)

const (
	// RSABits defines the bit length of the RSA private key
	RSABits = 2048
	// ValidFor defines the certificate validity period
	ValidFor = 365 * 24 * time.Hour
)

// MustGenerateCaCert must generate a CA certificate and private key.
// `certOut` and `keyOut` are PEM format buffers for certificate and private key, respectively.
// `caCert` and `caKey` are the corresponding structures.
func MustGenerateCaCert(t *testing.T) (certOut, keyOut *bytes.Buffer, caCert *x509.Certificate, caKey *rsa.PrivateKey) {
	notBefore := time.Now()
	notAfter := notBefore.Add(ValidFor)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1+int64(CACertType)), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	require.NoError(t, err, "failed to generate serial number")

	caCert = &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "default",
			Organization: []string{"Higress E2E Test"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caKey, err = rsa.GenerateKey(rand.Reader, RSABits)
	certOut, keyOut, err = GenerateCert(caCert, caKey, caCert, caKey)
	return
}

// MustGenerateCertWithCA must generate a self-signed client/server certificate and private key
// using CA certificate and private key.
// `hosts` is used when CertType == ServerCertType
func MustGenerateCertWithCA(t *testing.T, certType CertType, caCert *x509.Certificate, caKey *rsa.PrivateKey, hosts []string) (certOut, keyOut *bytes.Buffer) {
	notBefore := time.Now()
	notAfter := notBefore.Add(ValidFor)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1+int64(certType)), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	require.NoError(t, err, "failed to generate serial number")

	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "default",
			Organization: []string{"Higress E2E Test"},
		},
		NotBefore:   notBefore,
		NotAfter:    notAfter,
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	if certType == ServerCertType && hosts != nil {
		for _, h := range hosts {
			if ip := net.ParseIP(h); ip != nil {
				template.IPAddresses = append(template.IPAddresses, ip)
			} else {
				template.DNSNames = append(template.DNSNames, h)
			}
		}
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, RSABits)
	require.NoError(t, err, "failed to generate ras key")
	certOut, keyOut, err = GenerateCert(template, privateKey, caCert, caKey)
	return
}

// GenerateCert obtains the corresponding certificate and private key buffers
// using the certificate template and private key.
func GenerateCert(cert *x509.Certificate, key *rsa.PrivateKey, caCert *x509.Certificate, caKey *rsa.PrivateKey) (
	certOut, keyOut *bytes.Buffer, err error) {
	var (
		priv   = key
		pub    = &priv.PublicKey
		privPm = priv
	)
	if caKey != nil {
		privPm = caKey
	}
	certDER, err := x509.CreateCertificate(rand.Reader, cert, caCert, pub, privPm)
	if err != nil {
		err = fmt.Errorf("failed to create certificate: %w", err)
		return
	}
	certOut = new(bytes.Buffer)
	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err != nil {
		err = fmt.Errorf("failed creating cert: %w", err)
		return
	}
	keyOut = new(bytes.Buffer)
	privDER := x509.MarshalPKCS1PrivateKey(priv)
	err = pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privDER})
	if err != nil {
		err = fmt.Errorf("failed creating key: %w", err)
		return
	}
	return
}
