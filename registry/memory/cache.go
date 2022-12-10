package memory

import (
	"sort"
	"strconv"
	"sync"
	"time"

	"istio.io/api/networking/v1alpha3"
	"istio.io/pkg/log"

	"github.com/alibaba/higress/pkg/common"
)

type Cache interface {
	UpdateServiceEntryWrapper(service string, data *ServiceEntryWrapper)
	DeleteServiceEntryWrapper(service string)
	UpdateServiceEntryEnpointWrapper(service, ip, regionId, zoneId, protocol string, labels map[string]string)
	GetServiceByEndpoints(requestVersions, endpoints map[string]bool, versionKey string, protocol common.Protocol) map[string][]string
	GetAllServiceEntry() []*v1alpha3.ServiceEntry
	GetAllServiceEntryWrapper() []*ServiceEntryWrapper
	GetIncrementalServiceEntryWrapper() (updatedList []*ServiceEntryWrapper, deletedList []*ServiceEntryWrapper)
	RemoveEndpointByIp(ip string)
}

func NewCache() Cache {
	return &store{
		mux:         &sync.RWMutex{},
		sew:         make(map[string]*ServiceEntryWrapper),
		toBeUpdated: make([]*ServiceEntryWrapper, 0),
		toBeDeleted: make([]*ServiceEntryWrapper, 0),
		ip2services: make(map[string]map[string]bool),
	}
}

type store struct {
	mux         *sync.RWMutex
	sew         map[string]*ServiceEntryWrapper
	toBeUpdated []*ServiceEntryWrapper
	toBeDeleted []*ServiceEntryWrapper
	ip2services map[string]map[string]bool
}

func (s *store) UpdateServiceEntryEnpointWrapper(service, ip, regionId, zoneId, protocol string, labels map[string]string) {
	s.mux.Lock()
	defer s.mux.Unlock()
	if se, exist := s.sew[service]; exist {
		idx := -1
		for i, ep := range se.ServiceEntry.Endpoints {
			if ep.Address == ip {
				idx = i
				if len(regionId) != 0 {
					ep.Locality = regionId
					if len(zoneId) != 0 {
						ep.Locality = regionId + "/" + zoneId
					}
				}
				if labels != nil {
					for k, v := range labels {
						if protocol == common.Dubbo.String() && k == "version" {
							ep.Labels["appversion"] = v
							continue
						}
						ep.Labels[k] = v
					}
				}

				if idx != -1 {
					se.ServiceEntry.Endpoints[idx] = ep
				}
				return
			}

		}

	}
	return
}

func (s *store) UpdateServiceEntryWrapper(service string, data *ServiceEntryWrapper) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if old, exist := s.sew[service]; exist {
		data.SetCreateTime(old.GetCreateTime())
	} else {
		data.SetCreateTime(time.Now())
	}

	log.Debugf("mcp service entry update, name:%s, data:%v", service, data)

	s.toBeUpdated = append(s.toBeUpdated, data)
	s.sew[service] = data
}

func (s *store) DeleteServiceEntryWrapper(service string) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if data, exist := s.sew[service]; exist {
		s.toBeDeleted = append(s.toBeDeleted, data)
	}
	delete(s.sew, service)
}

// GetServiceByEndpoints get the list of services of which "address:port" contained by the endpoints
// and the version of the service contained by the requestVersions. The result format is as below:
// key: serviceName + "#@" + suffix
// values: ["v1", "v2"] which has removed duplication
func (s *store) GetServiceByEndpoints(requestVersions, endpoints map[string]bool, versionKey string, protocol common.Protocol) map[string][]string {
	s.mux.RLock()
	defer s.mux.RUnlock()

	result := make(map[string][]string)
	for _, serviceEntryWrapper := range s.sew {
		for _, workload := range serviceEntryWrapper.ServiceEntry.Endpoints {
			port, exist := workload.Ports[protocol.String()]
			if !exist {
				continue
			}

			endpoint := workload.Address + common.ColonSeparator + strconv.Itoa(int(port))
			if _, hit := endpoints[endpoint]; hit {
				if version, has := workload.Labels[versionKey]; has {
					if _, in := requestVersions[version]; in {
						key := serviceEntryWrapper.ServiceName + common.SpecialSeparator + serviceEntryWrapper.Suffix
						result[key] = append(result[key], version)
					}
				}
			}
		}
	}

	// remove duplication
	for key, versions := range result {
		sort.Strings(versions)
		i := 0
		for j := 1; j < len(versions); j++ {
			if versions[j] != versions[i] {
				i++
				versions[i] = versions[j]
			}
		}
		result[key] = versions[:i+1]
	}

	return result
}

// GetAllServiceEntry get all ServiceEntry in the store for xds push
func (s *store) GetAllServiceEntry() []*v1alpha3.ServiceEntry {
	s.mux.RLock()
	defer s.mux.RUnlock()

	seList := make([]*v1alpha3.ServiceEntry, 0)
	for _, serviceEntryWrapper := range s.sew {
		if len(serviceEntryWrapper.ServiceEntry.Hosts) == 0 {
			continue
		}
		seList = append(seList, serviceEntryWrapper.ServiceEntry.DeepCopy())
	}
	sort.Slice(seList, func(i, j int) bool {
		return seList[i].Hosts[0] > seList[j].Hosts[0]
	})
	return seList
}

// GetAllServiceEntryWrapper get all ServiceEntryWrapper in the store for xds push
func (s *store) GetAllServiceEntryWrapper() []*ServiceEntryWrapper {
	s.mux.RLock()
	defer s.mux.RUnlock()
	defer s.cleanUpdateAndDeleteArray()

	sewList := make([]*ServiceEntryWrapper, 0)
	for _, serviceEntryWrapper := range s.sew {
		sewList = append(sewList, serviceEntryWrapper.DeepCopy())
	}
	return sewList
}

// GetIncrementalServiceEntryWrapper get incremental ServiceEntryWrapper in the store for xds push
func (s *store) GetIncrementalServiceEntryWrapper() ([]*ServiceEntryWrapper, []*ServiceEntryWrapper) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	defer s.cleanUpdateAndDeleteArray()

	updatedList := make([]*ServiceEntryWrapper, 0)
	for _, serviceEntryWrapper := range s.toBeUpdated {
		updatedList = append(updatedList, serviceEntryWrapper.DeepCopy())
	}

	deletedList := make([]*ServiceEntryWrapper, 0)
	for _, serviceEntryWrapper := range s.toBeDeleted {
		deletedList = append(deletedList, serviceEntryWrapper.DeepCopy())
	}

	return updatedList, deletedList
}

func (s *store) cleanUpdateAndDeleteArray() {
	s.toBeUpdated = nil
	s.toBeDeleted = nil
}

func (s *store) updateIpMap(service string, data *ServiceEntryWrapper) {
	for _, ep := range data.ServiceEntry.Endpoints {
		if s.ip2services[ep.Address] == nil {
			s.ip2services[ep.Address] = make(map[string]bool)
		}
		s.ip2services[ep.Address][service] = true
	}
}

func (s *store) RemoveEndpointByIp(ip string) {
	s.mux.Lock()
	defer s.mux.Unlock()

	services, has := s.ip2services[ip]
	if !has {
		return
	}
	delete(s.ip2services, ip)

	for service := range services {
		if data, exist := s.sew[service]; exist {
			idx := -1
			for i, ep := range data.ServiceEntry.Endpoints {
				if ep.Address == ip {
					idx = i
					break
				}
			}
			if idx != -1 {
				data.ServiceEntry.Endpoints = append(data.ServiceEntry.Endpoints[:idx], data.ServiceEntry.Endpoints[idx+1:]...)
			}
			s.toBeUpdated = append(s.toBeUpdated, data)
		}
	}
}
