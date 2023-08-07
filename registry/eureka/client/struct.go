package client

import "github.com/hudl/fargo"

type Applications struct {
	Apps         map[string]*fargo.Application
	HashCode     string
	VersionDelta int
}
