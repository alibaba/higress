package common

import "strings"

type Protocol string

const (
	TCP         Protocol = "TCP"
	HTTP        Protocol = "HTTP"
	GRPC        Protocol = "GRPC"
	Dubbo       Protocol = "Dubbo"
	Unsupported Protocol = "UnsupportedProtocol"
)

func ParseProtocol(s string) Protocol {
	switch strings.ToLower(s) {
	case "tcp":
		return TCP
	case "http":
		return HTTP
	case "grpc":
		return GRPC
	case "dubbo":
		return Dubbo
	}
	return Unsupported
}

func (p Protocol) IsTCP() bool {
	switch p {
	case TCP:
		return true
	default:
		return false
	}
}

func (p Protocol) IsHTTP() bool {
	switch p {
	case HTTP, GRPC:
		return true
	default:
		return false
	}
}

func (p Protocol) IsGRPC() bool {
	switch p {
	case GRPC:
		return true
	default:
		return false
	}
}

func (p Protocol) IsDubbo() bool {
	return p == Dubbo
}

func (p Protocol) IsUnsupported() bool {
	return p == Unsupported
}

func (p Protocol) String() string {
	return string(p)
}
