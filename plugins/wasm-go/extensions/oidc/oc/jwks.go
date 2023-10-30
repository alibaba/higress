/*-
 * Copyright 2014 Square Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package oc

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"reflect"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	jose "github.com/go-jose/go-jose/v3"
	"github.com/tidwall/gjson"
)

const (
	RS256 = "RS256" // RSASSA-PKCS-v1.5 using SHA-256
	RS384 = "RS384" // RSASSA-PKCS-v1.5 using SHA-384
	RS512 = "RS512" // RSASSA-PKCS-v1.5 using SHA-512
	ES256 = "ES256" // ECDSA using P-256 and SHA-256
	ES384 = "ES384" // ECDSA using P-384 and SHA-384
	ES512 = "ES512" // ECDSA using P-521 and SHA-512
	PS256 = "PS256" // RSASSA-PSS using SHA256 and MGF1-SHA256
	PS384 = "PS384" // RSASSA-PSS using SHA384 and MGF1-SHA384
	PS512 = "PS512" // RSASSA-PSS using SHA512 and MGF1-SHA512
	EdDSA = "EdDSA" // Ed25519 using SHA-512
)

var SupportedAlgorithms = map[string]bool{
	RS256: true,
	RS384: true,
	RS512: true,
	ES256: true,
	ES384: true,
	ES512: true,
	PS256: true,
	PS384: true,
	PS512: true,
	EdDSA: true,
}

type rawJSONWebKey struct {
	Use string      `json:"use,omitempty"`
	Kty string      `json:"kty,omitempty"`
	Kid string      `json:"kid,omitempty"`
	Crv string      `json:"crv,omitempty"`
	Alg string      `json:"alg,omitempty"`
	K   *byteBuffer `json:"k,omitempty"`
	X   *byteBuffer `json:"x,omitempty"`
	Y   *byteBuffer `json:"y,omitempty"`
	N   *byteBuffer `json:"n,omitempty"`
	E   *byteBuffer `json:"e,omitempty"`
	// -- Following fields are only used for private keys --
	// RSA uses D, P and Q, while ECDSA uses only D. Fields Dp, Dq, and Qi are
	// completely optional. Therefore for RSA/ECDSA, D != nil is a contract that
	// we have a private key whereas D == nil means we have only a public key.
	D  *byteBuffer `json:"d,omitempty"`
	P  *byteBuffer `json:"p,omitempty"`
	Q  *byteBuffer `json:"q,omitempty"`
	Dp *byteBuffer `json:"dp,omitempty"`
	Dq *byteBuffer `json:"dq,omitempty"`
	Qi *byteBuffer `json:"qi,omitempty"`
	// Certificates
	X5c       []string `json:"x5c,omitempty"`
	X5u       string   `json:"x5u,omitempty"`
	X5tSHA1   string   `json:"x5t,omitempty"`
	X5tSHA256 string   `json:"x5t#S256,omitempty"`
}
type JSONWebKey struct {
	// Cryptographic key, can be a symmetric or asymmetric key.
	Key interface{}
	// Key identifier, parsed from `kid` header.
	KeyID string
	// Key algorithm, parsed from `alg` header.
	Algorithm string
	// Key use, parsed from `use` header.
	Use string

	// X.509 certificate chain, parsed from `x5c` header.
	Certificates []*x509.Certificate
	// X.509 certificate URL, parsed from `x5u` header.
	CertificatesURL *url.URL
	// X.509 certificate thumbprint (SHA-1), parsed from `x5t` header.
	CertificateThumbprintSHA1 []byte
	// X.509 certificate thumbprint (SHA-256), parsed from `x5t#S256` header.
	CertificateThumbprintSHA256 []byte
}
type byteBuffer struct {
	data []byte
}

func base64URLDecode(value string) ([]byte, error) {
	value = strings.TrimRight(value, "=")
	return base64.RawURLEncoding.DecodeString(value)
}

func newBuffer(data []byte) *byteBuffer {
	if data == nil {
		return nil
	}
	return &byteBuffer{
		data: data,
	}
}

func parseCertificateChain(chain []string) ([]*x509.Certificate, error) {

	out := make([]*x509.Certificate, len(chain))
	for i, cert := range chain {
		raw, err := base64.StdEncoding.DecodeString(cert)
		if err != nil {
			var log wrapper.Log
			log.Errorf("base64.StdEncoding.DecodeString(cert) err :")
			return nil, err
		}
		out[i], err = x509.ParseCertificate(raw)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
func (b byteBuffer) bigInt() *big.Int {
	return new(big.Int).SetBytes(b.data)
}

func (b byteBuffer) toInt() int {
	return int(b.bigInt().Int64())
}

func (key rawJSONWebKey) ecPublicKey() (*ecdsa.PublicKey, error) {
	var curve elliptic.Curve
	switch key.Crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("go-jose/go-jose: unsupported elliptic curve '%s'", key.Crv)
	}

	if key.X == nil || key.Y == nil {
		return nil, errors.New("go-jose/go-jose: invalid EC key, missing x/y values")
	}

	// The length of this octet string MUST be the full size of a coordinate for
	// the curve specified in the "crv" parameter.
	// https://tools.ietf.org/html/rfc7518#section-6.2.1.2
	if curveSize(curve) != len(key.X.data) {
		return nil, fmt.Errorf("go-jose/go-jose: invalid EC public key, wrong length for x")
	}

	if curveSize(curve) != len(key.Y.data) {
		return nil, fmt.Errorf("go-jose/go-jose: invalid EC public key, wrong length for y")
	}

	x := key.X.bigInt()
	y := key.Y.bigInt()

	if !curve.IsOnCurve(x, y) {
		return nil, errors.New("go-jose/go-jose: invalid EC key, X/Y are not on declared curve")
	}

	return &ecdsa.PublicKey{
		Curve: curve,
		X:     x,
		Y:     y,
	}, nil
}
func (key rawJSONWebKey) rsaPrivateKey() (*rsa.PrivateKey, error) {
	var missing []string
	switch {
	case key.N == nil:
		missing = append(missing, "N")
	case key.E == nil:
		missing = append(missing, "E")
	case key.D == nil:
		missing = append(missing, "D")
	case key.P == nil:
		missing = append(missing, "P")
	case key.Q == nil:
		missing = append(missing, "Q")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("go-jose/go-jose: invalid RSA private key, missing %s value(s)", strings.Join(missing, ", "))
	}

	rv := &rsa.PrivateKey{
		PublicKey: rsa.PublicKey{
			N: key.N.bigInt(),
			E: key.E.toInt(),
		},
		D: key.D.bigInt(),
		Primes: []*big.Int{
			key.P.bigInt(),
			key.Q.bigInt(),
		},
	}

	if key.Dp != nil {
		rv.Precomputed.Dp = key.Dp.bigInt()
	}
	if key.Dq != nil {
		rv.Precomputed.Dq = key.Dq.bigInt()
	}
	if key.Qi != nil {
		rv.Precomputed.Qinv = key.Qi.bigInt()
	}

	err := rv.Validate()
	return rv, err
}

func (key rawJSONWebKey) rsaPublicKey() (*rsa.PublicKey, error) {
	if key.N == nil || key.E == nil {
		return nil, fmt.Errorf("go-jose/go-jose: invalid RSA key, missing n/e values")
	}

	return &rsa.PublicKey{
		N: key.N.bigInt(),
		E: key.E.toInt(),
	}, nil
}
func (b *byteBuffer) bytes() []byte {
	// Handling nil here allows us to transparently handle nil slices when serializing.
	if b == nil {
		return nil
	}
	return b.data
}
func (key rawJSONWebKey) symmetricKey() ([]byte, error) {
	if key.K == nil {
		return nil, fmt.Errorf("go-jose/go-jose: invalid OCT (symmetric) key, missing k value")
	}
	return key.K.bytes(), nil
}
func (key rawJSONWebKey) edPrivateKey() (ed25519.PrivateKey, error) {
	var missing []string
	switch {
	case key.D == nil:
		missing = append(missing, "D")
	case key.X == nil:
		missing = append(missing, "X")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("go-jose/go-jose: invalid Ed25519 private key, missing %s value(s)", strings.Join(missing, ", "))
	}

	privateKey := make([]byte, ed25519.PrivateKeySize)
	copy(privateKey[0:32], key.D.bytes())
	copy(privateKey[32:], key.X.bytes())
	rv := ed25519.PrivateKey(privateKey)
	return rv, nil
}
func (key rawJSONWebKey) edPublicKey() (ed25519.PublicKey, error) {
	if key.X == nil {
		return nil, fmt.Errorf("go-jose/go-jose: invalid Ed key, missing x value")
	}
	publicKey := make([]byte, ed25519.PublicKeySize)
	copy(publicKey[0:32], key.X.bytes())
	rv := ed25519.PublicKey(publicKey)
	return rv, nil
}

func GenJswkey(parseBytes gjson.Result) (*jose.JSONWebKey, error) {
	var raw rawJSONWebKey
	var log wrapper.Log
	selClom(&raw, parseBytes)

	//
	certs, err := parseCertificateChain(raw.X5c)
	if err != nil {
		log.Errorf("err : %v", err)
	}
	var key interface{}
	var certPub interface{}
	var keyPub interface{}

	if len(certs) > 0 {
		// We need to check that leaf public key matches the key embedded in this
		// JWK, as required by the standard (see RFC 7517, Section 4.7). Otherwise
		// the JWK parsed could be semantically invalid. Technically, should also
		// check key usage fields and other extensions on the cert here, but the
		// standard doesn't exactly explain how they're supposed to map from the
		// JWK representation to the X.509 extensions.
		certPub = certs[0].PublicKey
	}

	switch raw.Kty {
	case "EC":
		if raw.D != nil {
			key, err = raw.ecPrivateKey()
			if err == nil {
				keyPub = key.(*ecdsa.PrivateKey).Public()
			}
		} else {
			key, err = raw.ecPublicKey()
			keyPub = key
		}
	case "RSA":
		if raw.D != nil {
			key, err = raw.rsaPrivateKey()
			if err == nil {
				keyPub = key.(*rsa.PrivateKey).Public()
			}
		} else {
			key, err = raw.rsaPublicKey()
			keyPub = key
		}
	case "oct":
		if certPub != nil {
			return nil, errors.New("go-jose/go-jose: invalid JWK, found 'oct' (symmetric) key with cert chain")
		}
		key, err = raw.symmetricKey()
	case "OKP":
		if raw.Crv == "Ed25519" && raw.X != nil {
			if raw.D != nil {
				key, err = raw.edPrivateKey()
				if err == nil {
					keyPub = key.(ed25519.PrivateKey).Public()
				}
			} else {
				key, err = raw.edPublicKey()
				keyPub = key
			}
		} else {
			err = fmt.Errorf("go-jose/go-jose: unknown curve %s'", raw.Crv)
		}
	default:
		err = fmt.Errorf("go-jose/go-jose: unknown json web key type '%s'", raw.Kty)
	}

	if err != nil {
		return nil, err
	}

	if certPub != nil && keyPub != nil {

		if !reflect.DeepEqual(certPub, keyPub) {
			return nil, errors.New("go-jose/go-jose: invalid JWK, public keys in key and x5c fields do not match")
		}
	}

	k := &jose.JSONWebKey{Key: key, KeyID: raw.Kid, Algorithm: raw.Alg, Use: raw.Use, Certificates: certs}

	if raw.X5u != "" {
		k.CertificatesURL, err = url.Parse(raw.X5u)
		if err != nil {
			return nil, fmt.Errorf("go-jose/go-jose: invalid JWK, x5u header is invalid URL: %w", err)
		}
	}

	// x5t parameters are base64url-encoded SHA thumbprints
	// See RFC 7517, Section 4.8, https://tools.ietf.org/html/rfc7517#section-4.8
	x5tSHA1bytes, err := base64URLDecode(raw.X5tSHA1)
	if err != nil {
		return nil, errors.New("go-jose/go-jose: invalid JWK, x5t header has invalid encoding")
	}

	// RFC 7517, Section 4.8 is ambiguous as to whether the digest output should be byte or hex,
	// for this reason, after base64 decoding, if the size is sha1.Size it's likely that the value is a byte encoded
	// checksum so we skip this. Otherwise if the checksum was hex encoded we expect a 40 byte sized array so we'll
	// try to hex decode it. When Marshalling this value we'll always use a base64 encoded version of byte format checksum.
	if len(x5tSHA1bytes) == 2*sha1.Size {
		hx, err := hex.DecodeString(string(x5tSHA1bytes))
		if err != nil {
			return nil, fmt.Errorf("go-jose/go-jose: invalid JWK, unable to hex decode x5t: %v", err)

		}
		x5tSHA1bytes = hx
	}

	k.CertificateThumbprintSHA1 = x5tSHA1bytes

	x5tSHA256bytes, err := base64URLDecode(raw.X5tSHA256)
	if err != nil {
		return nil, errors.New("go-jose/go-jose: invalid JWK, x5t#S256 header has invalid encoding")
	}

	if len(x5tSHA256bytes) == 2*sha256.Size {
		hx256, err := hex.DecodeString(string(x5tSHA256bytes))
		if err != nil {
			return nil, fmt.Errorf("go-jose/go-jose: invalid JWK, unable to hex decode x5t#S256: %v", err)
		}
		x5tSHA256bytes = hx256
	}

	k.CertificateThumbprintSHA256 = x5tSHA256bytes

	x5tSHA1Len := len(k.CertificateThumbprintSHA1)
	x5tSHA256Len := len(k.CertificateThumbprintSHA256)
	if x5tSHA1Len > 0 && x5tSHA1Len != sha1.Size {
		return nil, errors.New("go-jose/go-jose: invalid JWK, x5t header is of incorrect size")
	}
	if x5tSHA256Len > 0 && x5tSHA256Len != sha256.Size {
		return nil, errors.New("go-jose/go-jose: invalid JWK, x5t#S256 header is of incorrect size")
	}

	// If certificate chain *and* thumbprints are set, verify correctness.
	if len(k.Certificates) > 0 {
		leaf := k.Certificates[0]
		sha1sum := sha1.Sum(leaf.Raw)
		sha256sum := sha256.Sum256(leaf.Raw)

		if len(k.CertificateThumbprintSHA1) > 0 && !bytes.Equal(sha1sum[:], k.CertificateThumbprintSHA1) {
			return nil, errors.New("go-jose/go-jose: invalid JWK, x5c thumbprint does not match x5t value")
		}

		if len(k.CertificateThumbprintSHA256) > 0 && !bytes.Equal(sha256sum[:], k.CertificateThumbprintSHA256) {
			return nil, errors.New("go-jose/go-jose: invalid JWK, x5c thumbprint does not match x5t#S256 value")
		}
	}

	return k, nil
}

func curveSize(crv elliptic.Curve) int {
	bits := crv.Params().BitSize

	div := bits / 8
	mod := bits % 8

	if mod == 0 {
		return div
	}

	return div + 1
}
func dSize(curve elliptic.Curve) int {
	order := curve.Params().P
	bitLen := order.BitLen()
	size := bitLen / 8
	if bitLen%8 != 0 {
		size++
	}
	return size
}

func (key rawJSONWebKey) ecPrivateKey() (*ecdsa.PrivateKey, error) {
	var curve elliptic.Curve
	switch key.Crv {
	case "P-256":
		curve = elliptic.P256()
	case "P-384":
		curve = elliptic.P384()
	case "P-521":
		curve = elliptic.P521()
	default:
		return nil, fmt.Errorf("go-jose/go-jose: unsupported elliptic curve '%s'", key.Crv)
	}

	if key.X == nil || key.Y == nil || key.D == nil {
		return nil, fmt.Errorf("go-jose/go-jose: invalid EC private key, missing x/y/d values")
	}

	// The length of this octet string MUST be the full size of a coordinate for
	// the curve specified in the "crv" parameter.
	// https://tools.ietf.org/html/rfc7518#section-6.2.1.2
	if curveSize(curve) != len(key.X.data) {
		return nil, fmt.Errorf("go-jose/go-jose: invalid EC private key, wrong length for x")
	}

	if curveSize(curve) != len(key.Y.data) {
		return nil, fmt.Errorf("go-jose/go-jose: invalid EC private key, wrong length for y")
	}

	// https://tools.ietf.org/html/rfc7518#section-6.2.2.1
	if dSize(curve) != len(key.D.data) {
		return nil, fmt.Errorf("go-jose/go-jose: invalid EC private key, wrong length for d")
	}

	x := key.X.bigInt()
	y := key.Y.bigInt()

	if !curve.IsOnCurve(x, y) {
		return nil, errors.New("go-jose/go-jose: invalid EC key, X/Y are not on declared curve")
	}

	return &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		},
		D: key.D.bigInt(),
	}, nil
}

func selClom(raw *rawJSONWebKey, parseBytes gjson.Result) {

	raw.Use = parseBytes.Get("use").String()

	raw.Kty = parseBytes.Get("kty").String()
	raw.Kid = parseBytes.Get("kid").String()
	raw.Crv = parseBytes.Get("crv").String()
	raw.Alg = parseBytes.Get("alg").String()

	for _, item := range parseBytes.Get("x5c").Array() {
		scopes := item.String()
		raw.X5c = append(raw.X5c, scopes)
	}

	raw.X5u = parseBytes.Get("x5u").String()
	raw.X5tSHA1 = parseBytes.Get("x5t").String()

	raw.X5tSHA256 = parseBytes.Get("x5t#S256").String()

	//k
	if k := parseBytes.Get("k").Exists(); k {
		decode, err := base64URLDecode(parseBytes.Get("k").String())
		if err != nil {
			return
		}
		raw.K = newBuffer(decode)
	}
	//x
	if x := parseBytes.Get("x").Exists(); x {
		decode, err := base64URLDecode(parseBytes.Get("x").String())
		if err != nil {
			return
		}
		raw.X = newBuffer(decode)
	}
	//y
	if y := parseBytes.Get("y").Exists(); y {
		decode, err := base64URLDecode(parseBytes.Get("y").String())
		if err != nil {
			return
		}
		raw.Y = newBuffer(decode)
	}
	//n
	if n := parseBytes.Get("n").Exists(); n {
		decode, err := base64URLDecode(parseBytes.Get("n").String())
		if err != nil {
			return
		}
		raw.N = newBuffer(decode)
	}
	//e
	if e := parseBytes.Get("e").Exists(); e {
		decode, err := base64URLDecode(parseBytes.Get("e").String())
		if err != nil {
			return
		}
		raw.E = newBuffer(decode)
	}
	//d
	if d := parseBytes.Get("d").Exists(); d {
		decode, err := base64URLDecode(parseBytes.Get("d").String())
		if err != nil {
			return
		}
		raw.D = newBuffer(decode)
	}
	//p
	if p := parseBytes.Get("p").Exists(); p {
		decode, err := base64URLDecode(parseBytes.Get("p").String())
		if err != nil {
			return
		}
		raw.P = newBuffer(decode)
	}
	//q
	if q := parseBytes.Get("q").Exists(); q {
		decode, err := base64URLDecode(parseBytes.Get("q").String())
		if err != nil {
			return
		}
		raw.Q = newBuffer(decode)
	}
	//dp
	if dp := parseBytes.Get("dp").Exists(); dp {
		decode, err := base64URLDecode(parseBytes.Get("dp").String())
		if err != nil {
			return
		}
		raw.Dp = newBuffer(decode)

	}
	//dq
	if dq := parseBytes.Get("dq").Exists(); dq {
		decode, err := base64URLDecode(parseBytes.Get("dq").String())
		if err != nil {
			return
		}
		raw.Dq = newBuffer(decode)
	}
	//qi
	if qi := parseBytes.Get("qi").Exists(); qi {
		decode, err := base64URLDecode(parseBytes.Get("qi").String())
		if err != nil {
			return
		}
		raw.Qi = newBuffer(decode)
	}

}
