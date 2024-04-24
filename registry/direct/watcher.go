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

package direct

import (
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"istio.io/api/networking/v1alpha3"
	"istio.io/istio/pkg/config/protocol"
	"istio.io/pkg/log"

	apiv1 "github.com/alibaba/higress/api/networking/v1"
	"github.com/alibaba/higress/pkg/common"
	"github.com/alibaba/higress/registry"
	provider "github.com/alibaba/higress/registry"
	"github.com/alibaba/higress/registry/memory"
)

type watcher struct {
	provider.BaseWatcher
	apiv1.RegistryConfig
	cache memory.Cache
	mutex sync.Mutex
}

type WatcherOption func(w *watcher)

func NewWatcher(cache memory.Cache, opts ...WatcherOption) (provider.Watcher, error) {
	w := &watcher{
		cache: cache,
	}
	for _, opt := range opts {
		opt(w)
	}
	return w, nil
}

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

func (w *watcher) Run() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	host := strings.Join([]string{w.Name, w.Type}, common.DotSeparator)
	serviceEntry := w.generateServiceEntry(host)
	if serviceEntry != nil {
		w.cache.UpdateServiceEntryWrapper(host, &memory.ServiceEntryWrapper{
			ServiceName:  w.Name,
			ServiceEntry: serviceEntry,
			Suffix:       w.Type,
			RegistryType: w.Type,
		})
		w.UpdateService()
	}
	w.Ready(true)
}

func (w *watcher) Stop() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	host := strings.Join([]string{w.Name, w.Type}, common.DotSeparator)
	w.cache.DeleteServiceEntryWrapper(host)
	w.Ready(false)
}

var domainRegex = regexp.MustCompile(`^(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,6}$`)

func (w *watcher) generateServiceEntry(host string) *v1alpha3.ServiceEntry {
	endpoints := make([]*v1alpha3.WorkloadEntry, 0)
	for _, ep := range strings.Split(w.Domain, common.CommaSeparator) {
		var endpoint *v1alpha3.WorkloadEntry
		if w.Type == string(registry.Static) {
			pair := strings.Split(ep, common.ColonSeparator)
			if len(pair) != 2 {
				log.Errorf("invalid endpoint:%s with static type", ep)
				return nil
			}
			port, err := strconv.ParseUint(pair[1], 10, 32)
			if err != nil {
				log.Errorf("invalid port:%s of endpoint:%s", pair[1], ep)
				return nil
			}
			if net.ParseIP(pair[0]) == nil {
				log.Errorf("invalid ip:%s of endpoint:%s", pair[0], ep)
				return nil
			}
			endpoint = &v1alpha3.WorkloadEntry{
				Address: pair[0],
				Ports:   map[string]uint32{"http": uint32(port)},
			}
		} else if w.Type == string(registry.DNS) {
			if !domainRegex.MatchString(ep) {
				log.Errorf("invalid domain format:%s", ep)
				return nil
			}
			endpoint = &v1alpha3.WorkloadEntry{
				Address: ep,
			}
		} else {
			log.Errorf("unknown direct service type:%s", w.Type)
			return nil
		}
		endpoints = append(endpoints, endpoint)
	}
	if len(endpoints) == 0 {
		log.Errorf("empty endpoints will not be pushed, host:%s", host)
		return nil
	}
	var ports []*v1alpha3.Port
	ports = append(ports, &v1alpha3.Port{
		Number:   w.Port,
		Name:     "http",
		Protocol: string(protocol.HTTP),
	})
	se := &v1alpha3.ServiceEntry{
		Hosts:     []string{host},
		Ports:     ports,
		Location:  v1alpha3.ServiceEntry_MESH_INTERNAL,
		Endpoints: endpoints,
	}
	if w.Type == string(registry.Static) {
		se.Resolution = v1alpha3.ServiceEntry_STATIC
	} else if w.Type == string(registry.DNS) {
		se.Resolution = v1alpha3.ServiceEntry_DNS
	}
	return se
}

func (w *watcher) GetRegistryType() string {
	return w.RegistryConfig.Type
}
