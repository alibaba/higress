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

package util

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/types"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	"istio.io/istio/pilot/pkg/model"
)

const DefaultDomainSuffix = "cluster.local"

var domainSuffix = os.Getenv("DOMAIN_SUFFIX")

type ClusterNamespacedName struct {
	model.NamespacedName
	ClusterId string
}

func (c ClusterNamespacedName) String() string {
	return c.ClusterId + "/" + c.NamespacedName.String()
}

func SplitNamespacedName(name string) model.NamespacedName {
	nsName := strings.Split(name, "/")
	if len(nsName) == 2 {
		return model.NamespacedName{
			Namespace: nsName[0],
			Name:      nsName[1],
		}
	}

	return model.NamespacedName{
		Name: nsName[0],
	}
}

// CreateDestinationRuleName create the same format of DR name with ops.
func CreateDestinationRuleName(istioCluster, namespace, name string) string {
	format := path.Join(istioCluster, namespace, name)
	hash := md5.Sum([]byte(format))
	return hex.EncodeToString(hash[:])
}

func MessageToGoGoStruct(msg proto.Message) (*types.Struct, error) {
	if msg == nil {
		return nil, errors.New("nil message")
	}

	buf := &bytes.Buffer{}
	if err := (&jsonpb.Marshaler{OrigName: true}).Marshal(buf, msg); err != nil {
		return nil, err
	}

	pbs := &types.Struct{}
	if err := jsonpb.Unmarshal(buf, pbs); err != nil {
		return nil, err
	}

	return pbs, nil
}

func CreateServiceFQDN(namespace, name string) string {
	if domainSuffix == "" {
		domainSuffix = DefaultDomainSuffix
	}
	return fmt.Sprintf("%s.%s.svc.%s", name, namespace, domainSuffix)
}

func BuildPatchStruct(config string) *types.Struct {
	val := &types.Struct{}
	_ = jsonpb.Unmarshal(strings.NewReader(config), val)
	return val
}

type ServiceInfo struct {
	model.NamespacedName
	Port uint32
}

// convertToPort converts a port string to a uint32.
func convertToPort(v string) (uint32, error) {
	p, err := strconv.ParseUint(v, 10, 32)
	if err != nil || p > 65535 {
		return 0, fmt.Errorf("invalid port %s: %v", v, err)
	}
	return uint32(p), nil
}

func ParseServiceInfo(service string, ingressNamespace string) (ServiceInfo, error) {
	parts := strings.Split(service, ":")
	namespacedName := SplitNamespacedName(parts[0])

	if namespacedName.Name == "" {
		return ServiceInfo{}, errors.New("service name can not be empty")
	}

	if namespacedName.Namespace == "" {
		namespacedName.Namespace = ingressNamespace
	}

	var port uint32
	if len(parts) == 2 {
		// If port parse fail, we ignore port and pick the first one.
		port, _ = convertToPort(parts[1])
	}

	return ServiceInfo{
		NamespacedName: namespacedName,
		Port:           port,
	}, nil
}
