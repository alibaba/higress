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

package registry

import (
	"net"
	"time"
)

const (
	Zookeeper ServiceRegistryType = "zookeeper"
	Eureka    ServiceRegistryType = "eureka"
	Consul    ServiceRegistryType = "consul"
	Nacos     ServiceRegistryType = "nacos"
	Nacos2    ServiceRegistryType = "nacos2"
	Static    ServiceRegistryType = "static"
	DNS       ServiceRegistryType = "dns"
	Healthy   WatcherStatus       = "healthy"
	UnHealthy WatcherStatus       = "unhealthy"

	DefaultDialTimeout = time.Second * 3
)

type ServiceRegistryType string

func (srt *ServiceRegistryType) String() string {
	return string(*srt)
}

type WatcherStatus string

func (ws *WatcherStatus) String() string {
	return string(*ws)
}

type Watcher interface {
	Run()
	Stop()
	IsHealthy() bool
	IsReady() bool
	GetRegistryType() string
	AppendServiceUpdateHandler(f func())
	ReadyHandler(f func(bool))
}

type BaseWatcher struct {
	UpdateService ServiceUpdateHandler
	Ready         ReadyHandler
	ReadyStatus   bool
}

func (w *BaseWatcher) Run()                    {}
func (w *BaseWatcher) Stop()                   {}
func (w *BaseWatcher) IsHealthy() bool         { return true }
func (w *BaseWatcher) IsReady() bool           { return w.ReadyStatus }
func (w *BaseWatcher) GetRegistryType() string { return "" }
func (w *BaseWatcher) AppendServiceUpdateHandler(f func()) {
	w.UpdateService = f
}
func (w *BaseWatcher) ReadyHandler(f func(isReady bool)) {
	w.Ready = func(isReady bool) {
		w.ReadyStatus = isReady
		f(isReady)
	}
}

type ServiceUpdateHandler func()
type ReadyHandler func(bool)

func ProbeWatcherStatus(host string, port string) WatcherStatus {
	address := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", address, DefaultDialTimeout)
	if err != nil || conn == nil {
		return UnHealthy
	}
	_ = conn.Close()
	return Healthy
}
