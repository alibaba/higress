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

package common

import (
	"strings"
)

type ProxyType string

const (
	ProxyType_Unknown ProxyType = "Unknown"
	ProxyType_HTTP    ProxyType = "HTTP"
	ProxyType_HTTPS   ProxyType = "HTTPS"
	ProxyType_SOCKS4  ProxyType = "SOCKS4"
	ProxyType_SOCKS5  ProxyType = "SOCKS5"
)

func ParseProxyType(s string) ProxyType {
	switch strings.ToLower(s) {
	case "http":
		return ProxyType_HTTP
	case "https":
		return ProxyType_HTTPS
	case "socks4":
		return ProxyType_SOCKS4
	case "socks5":
		return ProxyType_SOCKS5
	}
	return ProxyType_Unknown
}

func (p ProxyType) GetTransportProtocol() Protocol {
	switch p {
	case ProxyType_HTTP:
		return HTTP
	case ProxyType_HTTPS:
		return HTTPS
	case ProxyType_SOCKS4, ProxyType_SOCKS5:
		return TCP
	}
	return Unsupported
}

func (p ProxyType) String() string {
	return string(p)
}
