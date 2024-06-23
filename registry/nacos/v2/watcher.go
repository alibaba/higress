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

package v2

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"go.uber.org/atomic"
	"istio.io/api/networking/v1alpha3"
	"istio.io/pkg/log"

	apiv1 "github.com/alibaba/higress/api/networking/v1"
	"github.com/alibaba/higress/pkg/common"
	provider "github.com/alibaba/higress/registry"
	"github.com/alibaba/higress/registry/memory"
	"github.com/alibaba/higress/registry/nacos/address"
)

const (
	DefaultInitTimeout          = time.Second * 10
	DefaultNacosTimeout         = 5000
	DefaultNacosLogLevel        = "warn"
	DefaultNacosLogDir          = "/var/log/nacos/log/"
	DefaultNacosCacheDir        = "/var/log/nacos/cache/"
	DefaultNacosNotLoadCache    = true
	DefaultNacosLogRotateTime   = "24h"
	DefaultNacosLogMaxAge       = 3
	DefaultUpdateCacheWhenEmpty = true
	DefaultRefreshInterval      = time.Second * 30
	DefaultRefreshIntervalLimit = time.Second * 10
	DefaultFetchPageSize        = 50
	DefaultJoiner               = "@@"
)

type watcher struct {
	provider.BaseWatcher
	apiv1.RegistryConfig
	WatchingServices     map[string]bool              `json:"watching_services"`
	RegistryType         provider.ServiceRegistryType `json:"registry_type"`
	Status               provider.WatcherStatus       `json:"status"`
	namingClient         naming_client.INamingClient
	cache                memory.Cache
	mutex                *sync.Mutex
	stop                 chan struct{}
	isStop               bool
	addrProvider         *address.NacosAddressProvider
	updateCacheWhenEmpty bool
	nacosClientConfig     *constant.ClientConfig
	authOption           provider.AuthOption
}

type WatcherOption func(w *watcher)

func NewWatcher(cache memory.Cache, opts ...WatcherOption) (provider.Watcher, error) {
	w := &watcher{
		WatchingServices: make(map[string]bool),
		RegistryType:     provider.Nacos2,
		Status:           provider.UnHealthy,
		cache:            cache,
		mutex:            &sync.Mutex{},
		stop:             make(chan struct{}),
	}

	w.NacosRefreshInterval = int64(DefaultRefreshInterval)

	for _, opt := range opts {
		opt(w)
	}

	if w.NacosNamespace == "" {
		w.NacosNamespace = w.NacosNamespaceId
	}

	log.Infof("new nacos2 watcher with config Name:%s", w.Name)

	w.nacosClientConfig = constant.NewClientConfig(
		constant.WithTimeoutMs(DefaultNacosTimeout),
		constant.WithLogLevel(DefaultNacosLogLevel),
		constant.WithLogDir(DefaultNacosLogDir),
		constant.WithCacheDir(DefaultNacosCacheDir),
		constant.WithNotLoadCacheAtStart(DefaultNacosNotLoadCache),
		constant.WithLogRollingConfig(&constant.ClientLogRollingConfig{
			MaxAge: DefaultNacosLogMaxAge,
		}),
		constant.WithUpdateCacheWhenEmpty(w.updateCacheWhenEmpty),
		constant.WithNamespaceId(w.NacosNamespaceId),
		constant.WithAccessKey(w.NacosAccessKey),
		constant.WithSecretKey(w.NacosSecretKey),
		constant.WithUsername(w.authOption.NacosUsername),
		constant.WithPassword(w.authOption.NacosPassword),
	)

	initTimer := time.NewTimer(DefaultInitTimeout)
	if w.NacosAddressServer != "" {
		w.addrProvider = address.NewNacosAddressProvider(w.NacosAddressServer, w.NacosNamespace)
		w.Domain = ""
		select {
		case w.Domain = <-w.addrProvider.GetNacosAddress(w.Domain):
		case <-initTimer.C:
			return nil, errors.New("new nacos2 watcher timeout")
		}
		go w.updateNacosClient()
	}
	sc := []constant.ServerConfig{
		*constant.NewServerConfig(w.Domain, uint64(w.Port)),
	}

	success := make(chan struct{})
	go func() {
		namingClient, err := clients.NewNamingClient(vo.NacosClientParam{
			ClientConfig:  w.nacosClientConfig,
			ServerConfigs: sc,
		})
		if err == nil {
			w.namingClient = namingClient
			close(success)
		} else {
			log.Errorf("can not create naming client, err:%v", err)
		}
	}()

	select {
	case <-initTimer.C:
		return nil, errors.New("new nacos2 watcher timeout")
	case <-success:
		return w, nil
	}
}

func WithNacosAddressServer(nacosAddressServer string) WatcherOption {
	return func(w *watcher) {
		w.NacosAddressServer = nacosAddressServer
	}
}

func WithNacosAccessKey(nacosAccessKey string) WatcherOption {
	return func(w *watcher) {
		w.NacosAccessKey = nacosAccessKey
	}
}

func WithNacosSecretKey(nacosSecretKey string) WatcherOption {
	return func(w *watcher) {
		w.NacosSecretKey = nacosSecretKey
	}
}

func WithNacosNamespaceId(nacosNamespaceId string) WatcherOption {
	return func(w *watcher) {
		if nacosNamespaceId == "" {
			w.NacosNamespaceId = "public"
		} else {
			w.NacosNamespaceId = nacosNamespaceId
		}
	}
}

func WithNacosNamespace(nacosNamespace string) WatcherOption {
	return func(w *watcher) {
		w.NacosNamespace = nacosNamespace
	}
}

func WithNacosGroups(nacosGroups []string) WatcherOption {
	return func(w *watcher) {
		w.NacosGroups = nacosGroups
	}
}

func WithNacosRefreshInterval(refreshInterval int64) WatcherOption {
	return func(w *watcher) {
		if refreshInterval < int64(DefaultRefreshIntervalLimit) {
			refreshInterval = int64(DefaultRefreshIntervalLimit)
		}
		w.NacosRefreshInterval = refreshInterval
	}
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

func WithUpdateCacheWhenEmpty(enable bool) WatcherOption {
	return func(w *watcher) {
		w.updateCacheWhenEmpty = enable
	}
}

func WithAuthOption(authOption provider.AuthOption) WatcherOption {
	return func(w *watcher) {
		w.authOption = authOption
	}
}

func (w *watcher) Run() {
	ticker := time.NewTicker(time.Duration(w.NacosRefreshInterval))
	defer ticker.Stop()
	w.Status = provider.ProbeWatcherStatus(w.Domain, strconv.FormatUint(uint64(w.Port), 10))
	err := w.fetchAllServices()
	if err != nil {
		log.Errorf("first fetch services failed, err:%v", err)
	} else {
		w.Ready(true)
	}
	for {
		select {
		case <-ticker.C:
			err := w.fetchAllServices()
			if err != nil {
				log.Errorf("fetch services failed, err:%v", err)
			} else {
				w.Ready(true)
			}
		case <-w.stop:
			return
		}
	}
}

func (w *watcher) updateNacosClient() {
	for {
		select {
		case addr := <-w.addrProvider.GetNacosAddress(w.Domain):
			func() {
				w.mutex.Lock()
				defer w.mutex.Unlock()
				w.Domain = addr
				namingClient, err := clients.NewNamingClient(vo.NacosClientParam{
					ClientConfig: w.nacosClientConfig,
					ServerConfigs: []constant.ServerConfig{
						*constant.NewServerConfig(addr, uint64(w.Port)),
					},
				})
				if err != nil {
					log.Errorf("can not update naming client, err:%v", err)
					return
				}
				w.namingClient = namingClient
				log.Info("naming client updated")
			}()
		case <-w.stop:
			return
		}
	}
}

func (w *watcher) fetchAllServices() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.isStop {
		return nil
	}
	fetchedServices := make(map[string]bool)
	var tries int
	for _, groupName := range w.NacosGroups {
		for page := 1; ; page++ {
			ss, err := w.namingClient.GetAllServicesInfo(vo.GetAllServiceInfoParam{
				GroupName: groupName,
				PageNo:    uint32(page),
				PageSize:  DefaultFetchPageSize,
				NameSpace: w.NacosNamespace,
			})
			if err != nil {
				if tries > 10 {
					return err
				}
				if w.addrProvider != nil {
					w.addrProvider.Trigger()
				}
				log.Errorf("fetch nacos service list failed, err:%v, pageNo:%d", err, page)
				page--
				tries++
				continue
			}
			for _, serviceName := range ss.Doms {
				fetchedServices[groupName+DefaultJoiner+serviceName] = true
			}
			if len(ss.Doms) < DefaultFetchPageSize {
				break
			}
		}
	}

	for key := range w.WatchingServices {
		if _, exist := fetchedServices[key]; !exist {
			s := strings.Split(key, DefaultJoiner)
			err := w.unsubscribe(s[0], s[1])
			if err != nil {
				return err
			}
			delete(w.WatchingServices, key)
		}
	}
	wg := sync.WaitGroup{}
	subscribeFailed := atomic.NewBool(false)
	watchingKeys := make(chan string, len(fetchedServices))
	for key := range fetchedServices {
		if _, exist := w.WatchingServices[key]; !exist {
			s := strings.Split(key, DefaultJoiner)
			if !shouldSubscribe(s[1]) {
				continue
			}
			wg.Add(1)
			go func(k string) {
				err := w.subscribe(s[0], s[1])
				if err != nil {
					subscribeFailed.Store(true)
					log.Errorf("subscribe failed, err:%v, group:%s, service:%s", err, s[0], s[1])
				} else {
					watchingKeys <- k
				}
				wg.Done()
			}(key)
		}
	}
	wg.Wait()
	close(watchingKeys)
	for key := range watchingKeys {
		w.WatchingServices[key] = true
	}
	if subscribeFailed.Load() {
		return errors.New("subscribe services failed")
	}
	return nil
}

func (w *watcher) subscribe(groupName string, serviceName string) error {
	log.Debugf("subscribe service, groupName:%s, serviceName:%s", groupName, serviceName)

	err := w.namingClient.Subscribe(&vo.SubscribeParam{
		ServiceName:       serviceName,
		GroupName:         groupName,
		SubscribeCallback: w.getSubscribeCallback(groupName, serviceName),
	})

	if err != nil {
		log.Errorf("subscribe service error:%v, groupName:%s, serviceName:%s", err, groupName, serviceName)
		return err
	}

	return nil
}

func (w *watcher) unsubscribe(groupName string, serviceName string) error {
	log.Debugf("unsubscribe service, groupName:%s, serviceName:%s", groupName, serviceName)

	err := w.namingClient.Unsubscribe(&vo.SubscribeParam{
		ServiceName:       serviceName,
		GroupName:         groupName,
		SubscribeCallback: w.getSubscribeCallback(groupName, serviceName),
	})

	if err != nil {
		log.Errorf("unsubscribe service error:%v, groupName:%s, serviceName:%s", err, groupName, serviceName)
		return err
	}

	return nil
}

func (w *watcher) getSubscribeCallback(groupName string, serviceName string) func(services []model.Instance, err error) {
	suffix := strings.Join([]string{groupName, w.NacosNamespace, "nacos"}, common.DotSeparator)
	suffix = strings.ReplaceAll(suffix, common.Underscore, common.Hyphen)
	host := strings.Join([]string{serviceName, suffix}, common.DotSeparator)

	return func(services []model.Instance, err error) {
		defer w.UpdateService()

		//log.Info("callback", "serviceName", serviceName, "suffix", suffix, "details", services)

		if err != nil {
			if strings.Contains(err.Error(), "hosts is empty") {
				if w.updateCacheWhenEmpty {
					w.cache.DeleteServiceEntryWrapper(host)
				}
			} else {
				log.Errorf("callback error:%v", err)
			}
			return
		}

		if len(services) > 0 && services[0].Metadata != nil && services[0].Metadata["register-resource"] == "mcp-bridge" {
			return
		}
		serviceEntry := w.generateServiceEntry(host, services)
		w.cache.UpdateServiceEntryWrapper(host, &memory.ServiceEntryWrapper{
			ServiceName:  serviceName,
			ServiceEntry: serviceEntry,
			Suffix:       suffix,
			RegistryType: w.Type,
		})
	}
}

func (w *watcher) generateServiceEntry(host string, services []model.Instance) *v1alpha3.ServiceEntry {
	portList := make([]*v1alpha3.Port, 0)
	endpoints := make([]*v1alpha3.WorkloadEntry, 0)

	for _, service := range services {
		protocol := common.HTTP
		if service.Metadata != nil && service.Metadata["protocol"] != "" {
			protocol = common.ParseProtocol(service.Metadata["protocol"])
		}
		port := &v1alpha3.Port{
			Name:     protocol.String(),
			Number:   uint32(service.Port),
			Protocol: protocol.String(),
		}
		if len(portList) == 0 {
			portList = append(portList, port)
		}
		endpoint := &v1alpha3.WorkloadEntry{
			Address: service.Ip,
			Ports:   map[string]uint32{port.Protocol: port.Number},
			Labels:  service.Metadata,
		}
		endpoints = append(endpoints, endpoint)
	}

	se := &v1alpha3.ServiceEntry{
		Hosts:      []string{host},
		Ports:      portList,
		Location:   v1alpha3.ServiceEntry_MESH_INTERNAL,
		Resolution: v1alpha3.ServiceEntry_STATIC,
		Endpoints:  endpoints,
	}

	return se
}

func (w *watcher) Stop() {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if w.addrProvider != nil {
		w.addrProvider.Stop()
	}
	for key := range w.WatchingServices {
		s := strings.Split(key, DefaultJoiner)
		err := w.unsubscribe(s[0], s[1])
		if err == nil {
			delete(w.WatchingServices, key)
		}

		// clean the cache
		suffix := strings.Join([]string{s[0], w.NacosNamespace, "nacos"}, common.DotSeparator)
		suffix = strings.ReplaceAll(suffix, common.Underscore, common.Hyphen)
		host := strings.Join([]string{s[1], suffix}, common.DotSeparator)
		w.cache.DeleteServiceEntryWrapper(host)
	}

	w.isStop = true
	w.namingClient.CloseClient()
	close(w.stop)
	w.Ready(false)
}

func (w *watcher) IsHealthy() bool {
	return w.Status == provider.Healthy
}

func (w *watcher) GetRegistryType() string {
	return w.RegistryType.String()
}

func shouldSubscribe(serviceName string) bool {
	prefixFilters := []string{"consumers:"}
	fullFilters := []string{""}

	for _, f := range prefixFilters {
		if strings.HasPrefix(serviceName, f) {
			return false
		}
	}

	for _, f := range fullFilters {
		if serviceName == f {
			return false
		}
	}

	return true
}
