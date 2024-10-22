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

package memory

import (
	"time"

	"istio.io/api/networking/v1alpha3"

	"github.com/alibaba/higress/pkg/ingress/kube/common"
)

type ServiceWrapper struct {
	ServiceName            string
	ServiceEntry           *v1alpha3.ServiceEntry
	DestinationRuleWrapper *common.WrapperDestinationRule
	Suffix                 string
	RegistryType           string
	RegistryName           string
	createTime             time.Time
}

func (sew *ServiceWrapper) DeepCopy() *ServiceWrapper {
	res := &ServiceWrapper{}
	res = sew
	res.ServiceEntry = sew.ServiceEntry.DeepCopy()
	res.createTime = sew.GetCreateTime()

	if sew.DestinationRuleWrapper != nil {
		res.DestinationRuleWrapper = sew.DestinationRuleWrapper
		res.DestinationRuleWrapper.DestinationRule = sew.DestinationRuleWrapper.DestinationRule.DeepCopy()
	}
	return res
}

func (sew *ServiceWrapper) SetCreateTime(createTime time.Time) {
	sew.createTime = createTime
}

func (sew *ServiceWrapper) GetCreateTime() time.Time {
	return sew.createTime
}
