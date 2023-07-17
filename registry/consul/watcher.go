package consul

import (
	"strconv"
	"strings"
	"sync"
	"time"

	apiv1 "github.com/alibaba/higress/api/networking/v1"
	"github.com/alibaba/higress/pkg/common"
	provider "github.com/alibaba/higress/registry"
	"github.com/alibaba/higress/registry/memory"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"istio.io/api/networking/v1alpha3"
	"istio.io/pkg/log"
)

const (
	ConuslHealthPassing         = "passing"
	DefaultRefreshInterval      = time.Second * 30
	DefaultRefreshIntervalLimit = time.Second * 10
)

type watcher struct {
	provider.BaseWatcher
	apiv1.RegistryConfig
	serverAddress        string
	consulClient         *consulapi.Client
	consulCatalog        *consulapi.Catalog
	WatchingServices     map[string]bool
	watchers             map[string]*watch.Plan
	RegistryType         provider.ServiceRegistryType
	Status               provider.WatcherStatus
	cache                memory.Cache
	mutex                *sync.Mutex
	stop                 chan struct{}
	isStop               bool
	updateCacheWhenEmpty bool
}

type WatcherOption func(w *watcher)

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

func WithDatacenter(dataCenter string) WatcherOption {
	return func(w *watcher) {
		w.ConsulDatacenter = dataCenter
	}
}

func WithAuthToken(authToken string) WatcherOption {
	return func(w *watcher) {
		w.ConsulAuthToken = authToken
	}
}

func WithServiceTag(serviceTag string) WatcherOption {
	return func(w *watcher) {
		w.ConsulServiceTag = strings.ToLower(strings.TrimSpace(serviceTag))
	}
}

func WithRefreshInterval(refreshInterval int64) WatcherOption {
	return func(w *watcher) {
		if refreshInterval < int64(DefaultRefreshIntervalLimit) {
			refreshInterval = int64(DefaultRefreshIntervalLimit)
		}
		w.ConsulRefreshInterval = refreshInterval
	}
}

func NewWatcher(cache memory.Cache, opts ...WatcherOption) (provider.Watcher, error) {
	w := &watcher{
		WatchingServices: make(map[string]bool),
		RegistryType:     provider.Consul,
		Status:           provider.UnHealthy,
		cache:            cache,
		mutex:            &sync.Mutex{},
		stop:             make(chan struct{}),
	}

	// Set default
	w.ConsulRefreshInterval = int64(DefaultRefreshInterval)

	// Set option
	for _, opt := range opts {
		opt(w)
	}

	// Init consul client
	w.serverAddress = w.Domain + ":" + strconv.Itoa(int(w.Port))
	config := consulapi.DefaultConfig()
	config.Address = w.serverAddress
	config.Token = w.ConsulAuthToken
	client, err := consulapi.NewClient(config)
	if err != nil {
		log.Errorf("[NewWatcher] NewWatcher consul, err:%v, consul address:%s", err, w.serverAddress)
		return nil, err
	}
	w.consulClient = client
	w.consulCatalog = client.Catalog()
	return w, nil
}

func (w *watcher) fetchAllServices() error {
	log.Infof("consul start to fetch services")
	w.mutex.Lock()
	defer w.mutex.Unlock()
	if w.isStop {
		return nil
	}

	fetchedServices := make(map[string]bool)
	q := &consulapi.QueryOptions{}
	q.Datacenter = w.ConsulDatacenter
	q.Token = w.ConsulAuthToken
	services, _, err := w.consulCatalog.Services(q)

	if err != nil {
		log.Errorf("consul fetch all services error:%v", err)
		return err
	}

	for serviceName, tags := range services {
		log.Infof("consul fetch service:%s, tags:%v", serviceName, tags)
		if w.filterTags(w.ConsulServiceTag, tags) {
			log.Infof("consul find match service:%s, tags:%v", serviceName, tags)
			fetchedServices[serviceName] = true
		}
	}

	for serviceName := range w.WatchingServices {
		if _, exist := fetchedServices[serviceName]; !exist {
			err := w.unsubscribe(serviceName)
			if err == nil {
				delete(w.WatchingServices, serviceName)
			}
		}
	}

	for serviceName := range fetchedServices {
		if _, exist := w.WatchingServices[serviceName]; !exist {
			if !w.shouldSubscribe(serviceName) {
				continue
			}
			err := w.subscribe(serviceName)
			if err == nil {
				w.WatchingServices[serviceName] = true
			}
		}
	}

	return nil
}

func (w *watcher) filterTags(consulTag string, tags []string) bool {
	if len(consulTag) == 0 {
		return true
	}

	if len(tags) == 0 {
		return false
	}

	for _, tag := range tags {
		if strings.ToLower(tag) == consulTag {
			return true
		}
	}

	return false
}

func (w *watcher) Run() {
	ticker := time.NewTicker(time.Duration(w.ConsulRefreshInterval))
	defer ticker.Stop()
	w.Status = provider.ProbeWatcherStatus(w.Domain, strconv.FormatUint(uint64(w.Port), 10))
	w.fetchAllServices()
	w.Ready(true)
	for {
		select {
		case <-ticker.C:
			w.fetchAllServices()
		case <-w.stop:
			return
		}
	}
}

func (w *watcher) Stop() {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	for serviceName := range w.WatchingServices {
		err := w.unsubscribe(serviceName)
		if err == nil {
			delete(w.WatchingServices, serviceName)
		}

		// clean the cache
		suffix := strings.Join([]string{serviceName, w.ConsulDatacenter, w.Type}, common.DotSeparator)
		host := strings.ReplaceAll(suffix, common.Underscore, common.Hyphen)
		w.cache.DeleteServiceEntryWrapper(host)
	}
	w.isStop = true
	close(w.stop)
	w.Ready(false)
}

func (w *watcher) IsHealthy() bool {
	return w.Status == provider.Healthy
}

func (w *watcher) GetRegistryType() string {
	return w.RegistryType.String()
}

func (w *watcher) unsubscribe(serviceName string) error {
	log.Debugf("consul unsubscribe service, serviceName:%s", serviceName)
	if plan, ok := w.watchers[serviceName]; ok {
		plan.Stop()
		w.mutex.Lock()
		delete(w.watchers, serviceName)
		w.mutex.Unlock()
	}
	return nil
}

func (w *watcher) shouldSubscribe(serviceName string) bool {
	return true
}

func (w *watcher) subscribe(serviceName string) error {
	log.Infof("consul  subscribe service, serviceName:%s", serviceName)
	plan, err := watch.Parse(map[string]interface{}{
		"type":    "service",
		"service": serviceName,
	})

	if err != nil {
		return err
	}
	plan.Handler = w.getSubscribeCallback(serviceName)
	plan.Token = w.ConsulAuthToken
	plan.Datacenter = w.ConsulDatacenter
	go plan.Run(w.serverAddress)

	w.mutex.Lock()
	w.watchers[serviceName] = plan
	w.mutex.Unlock()
	return nil
}

func (w *watcher) getSubscribeCallback(serviceName string) func(idx uint64, data interface{}) {
	suffix := strings.Join([]string{serviceName, w.ConsulDatacenter, w.Type}, common.DotSeparator)
	host := strings.ReplaceAll(suffix, common.Underscore, common.Hyphen)

	return func(idx uint64, data interface{}) {
		log.Infof("consul subscribe callback service, host:%s, serviceName:%s", host, serviceName)
		switch services := data.(type) {
		case []*consulapi.ServiceEntry:
			for _, entry := range services {
				log.Infof("consul subscribe callback service changed, service:%s, status:%s", entry.Service.Service, entry.Checks.AggregatedStatus())
			}
			defer w.UpdateService()
			serviceEntry := w.generateServiceEntry(host, services)
			log.Infof("consul subscribe callback generate ServiceEntry:%v", serviceEntry)
			if serviceEntry != nil {
				log.Infof("consul update cache:%s", host)
				w.cache.UpdateServiceEntryWrapper(host, &memory.ServiceEntryWrapper{
					ServiceEntry: serviceEntry,
					ServiceName:  serviceName,
					Suffix:       suffix,
					RegistryType: w.Type,
				})
			} else {
				log.Infof("consul delete cache:%s", host)
				w.cache.DeleteServiceEntryWrapper(host)
			}
		}
	}
}

func (w *watcher) generateServiceEntry(host string, services []*consulapi.ServiceEntry) *v1alpha3.ServiceEntry {
	portList := make([]*v1alpha3.Port, 0)
	endpoints := make([]*v1alpha3.WorkloadEntry, 0)

	for _, service := range services {
		protocol := common.HTTP
		// service status: maintenance > critical > warning > passing
		if service.Checks.AggregatedStatus() != ConuslHealthPassing {
			continue
		}

		metaData := make(map[string]string, 0)
		if service.Service.Meta != nil {
			metaData = service.Service.Meta
		}

		port := &v1alpha3.Port{
			Name:     protocol.String(),
			Number:   uint32(service.Service.Port),
			Protocol: protocol.String(),
		}

		if len(portList) == 0 {
			portList = append(portList, port)
		}

		endpoint := v1alpha3.WorkloadEntry{
			Address: service.Service.Address,
			Ports:   map[string]uint32{port.Protocol: port.Number},
			Labels:  metaData,
		}
		endpoints = append(endpoints, &endpoint)
	}

	if len(endpoints) == 0 {
		return nil
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
