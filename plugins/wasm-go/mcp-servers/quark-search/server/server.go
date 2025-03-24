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

package server

import (
	"encoding/json"
	"errors"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

type QuarkMCPServer struct {
	ApiKey string `json:"apiKey"`
}

func (s QuarkMCPServer) ConfigHasError() error {
	if s.ApiKey == "" {
		return errors.New("missing api key")
	}
	return nil
}

func ParseFromConfig(configBytes []byte, server *QuarkMCPServer) error {
	return json.Unmarshal(configBytes, server)
}

func ParseFromRequest(ctx wrapper.HttpContext, server *QuarkMCPServer) error {
	return ctx.ParseMCPServerConfig(server)
}
