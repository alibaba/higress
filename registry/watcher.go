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
	GetRegistryType() string
	AppendServiceUpdateHandler(f func())
	ReadyHandler(f func(bool))
}

type BaseWatcher struct{}

func (w *BaseWatcher) Run()                                {}
func (w *BaseWatcher) Stop()                               {}
func (w *BaseWatcher) IsHealthy() bool                     { return true }
func (w *BaseWatcher) GetRegistryType() string             { return "" }
func (w *BaseWatcher) AppendServiceUpdateHandler(f func()) {}
func (w *BaseWatcher) ReadyHandler(f func(bool))           {}

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
