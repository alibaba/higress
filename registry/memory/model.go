package memory

import (
	"time"

	"istio.io/api/networking/v1alpha3"
)

type ServiceEntryWrapper struct {
	ServiceName  string
	ServiceEntry *v1alpha3.ServiceEntry
	Suffix       string
	RegistryType string
	createTime   time.Time
}

func (sew *ServiceEntryWrapper) DeepCopy() *ServiceEntryWrapper {
	return &ServiceEntryWrapper{
		ServiceEntry: sew.ServiceEntry.DeepCopy(),
		createTime:   sew.GetCreateTime(),
	}
}

func (sew *ServiceEntryWrapper) SetCreateTime(createTime time.Time) {
	sew.createTime = createTime
}

func (sew *ServiceEntryWrapper) GetCreateTime() time.Time {
	return sew.createTime
}
