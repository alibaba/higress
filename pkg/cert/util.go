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
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"hash/fnv"
	"math/rand"
	"net"
	"regexp"
	"time"
)

// parseCertsFromPEMBundle parses a certificate bundle from top to bottom and returns
// a slice of x509 certificates. This function will error if no certificates are found.
func parseCertsFromPEMBundle(bundle []byte) ([]*x509.Certificate, error) {
	var certificates []*x509.Certificate
	var certDERBlock *pem.Block
	for {
		certDERBlock, bundle = pem.Decode(bundle)
		if certDERBlock == nil {
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			cert, err := x509.ParseCertificate(certDERBlock.Bytes)
			if err != nil {
				return nil, err
			}
			certificates = append(certificates, cert)
		}
	}
	if len(certificates) == 0 {
		return nil, fmt.Errorf("no certificates found in bundle")
	}
	return certificates, nil
}

func notAfter(cert *x509.Certificate) time.Time {
	if cert == nil {
		return time.Time{}
	}
	return cert.NotAfter.Truncate(time.Second).Add(1 * time.Second)
}

func notBefore(cert *x509.Certificate) time.Time {
	if cert == nil {
		return time.Time{}
	}
	return cert.NotBefore.Truncate(time.Second).Add(1 * time.Second)
}

// hostOnly returns only the host portion of hostport.
// If there is no port or if there is an error splitting
// the port off, the whole input string is returned.
func hostOnly(hostport string) string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		return hostport // OK; probably had no port to begin with
	}
	return host
}

func rangeRandom(min, max int) (number int) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	number = r.Intn(max-min) + min
	return number
}

func ValidateEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	regExp := regexp.MustCompile(pattern)
	if regExp.MatchString(email) {
		return true
	} else {
		return false
	}
}

func fastHash(input []byte) string {
	h := fnv.New32a()
	h.Write(input)
	return fmt.Sprintf("%x", h.Sum32())
}
