// Copyright (c) 2023 Alibaba Group Holding Ltd.
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

package annotations

import (
	. "github.com/alibaba/higress/pkg/ingress/log"
)

const (
	http2rpcKey        = "http2rpc-name"
	rpcDestinationName = "rpc-destination-name"
)

// help to conform http2rpc implements method of Parse
var _ Parser = http2rpc{}

type Http2RpcConfig struct {
	Name string
}

type http2rpc struct{}

func (a http2rpc) Parse(annotations Annotations, config *Ingress, _ *GlobalContext) error {
	if !needHttp2RpcConfig(annotations) {
		return nil
	}
	value, err := annotations.ParseStringForHigress(rpcDestinationName)
	IngressLog.Infof("Parse http2rpc ingress name %s", value)
	if err != nil {
		IngressLog.Errorf("parse http2rpc error %v within ingress %s/%s", err, config.Namespace, config.Name)
		return nil
	}
	config.Http2Rpc = &Http2RpcConfig{
		Name: value,
	}
	return nil
}

func needHttp2RpcConfig(annotations Annotations) bool {
	return annotations.HasHigress(rpcDestinationName)
}
