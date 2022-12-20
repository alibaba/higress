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

package zookeeper

import (
	"errors"
	"time"
)

const (
	DEFAULT_REG_TIMEOUT   = "10s"
	DUBBO                 = "/dubbo/"
	SPRING_CLOUD_SERVICES = "/services"
	DUBBO_SERVICES        = "/dubbo"
	PROVIDERS             = "/providers"
	CONFIG                = "config"
	MAPPING               = "mapping"
	METADATA              = "metadata"
	DUBBO_PROTOCOL        = "dubbo"
	HTTP_PROTOCOL         = "http"
	VERSION               = "version"
	PROTOCOL              = "protocol"
)

type ServiceType int

const (
	DubboService ServiceType = iota
	SpringCloudService
)

type EventType int

type Event struct {
	Path          string
	Action        EventType
	Content       []byte
	InterfaceName string
	ServiceType   ServiceType
}

const (
	// ConnDelay connection delay interval
	ConnDelay = 3
	// MaxFailTimes max fail times
	MaxFailTimes = 3
)

var DefaultTTL = 10 * time.Minute

type InterfaceConfig struct {
	Host        string
	Endpoints   []Endpoint
	Protocol    string
	ServiceType ServiceType
}

type Endpoint struct {
	Ip       string
	Port     string
	Metadata map[string]string
}

var ErrNilChildren = errors.New("has none children")

func WithType(t string) WatcherOption {
	return func(w *watcher) {
		w.Type = t
	}
}

func WithName(name string) WatcherOption {
	return func(w *watcher) {
		w.Name = name
	}
}

func WithDomain(domain string) WatcherOption {
	return func(w *watcher) {
		w.Domain = domain
	}
}

func WithPort(port uint32) WatcherOption {
	return func(w *watcher) {
		w.Port = port
	}
}

type DataListener interface {
	DataChange(eventType Event) bool // bool is return for interface implement is interesting
}

const (
	// EventTypeAdd means add event
	EventTypeAdd = iota
	// EventTypeDel means del event
	EventTypeDel
	// EventTypeUpdate means update event
	EventTypeUpdate
)

type ListServiceConfig struct {
	UrlIndex      string
	InterfaceName string
	Exit          chan struct{}
	ServiceType   ServiceType
}

type SpringCloudInstancePayload struct {
	Metadata map[string]string `json:"metadata"`
}

type SpringCloudInstance struct {
	Name    string                     `json:"name"`
	Address string                     `json:"address"`
	Port    int                        `json:"port"`
	Payload SpringCloudInstancePayload `json:"payload"`
}
