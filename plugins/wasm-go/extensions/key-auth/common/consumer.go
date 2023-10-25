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

package common

import (
	"errors"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

var (
	errConsumersNotFound = errors.New("consumers: Not Found")
	errConsumersIsEmpty  = errors.New("consumers: consumer cannot be empty")

	errConsumerNotFound = errors.New("consumer: Not Found")
	errConsumerIsEmpty  = errors.New("consumer: consumer cannot be empty")

	errDuplicateConsumer       = errors.New("consumer: duplicate consumer")
	errInvalidCredentialFormat = errors.New("credential: invalid credential format")
)

type Consumers struct {
	Consumers []Consumer `yaml:"consumers"`
}

type Consumer struct {
	Name       string `yaml:"name"`
	Credential string `yaml:"credential"`
}

func ParseConsumersConfig(json gjson.Result, config *Consumers, log wrapper.Log) error {
	consumers := json.Get("consumers")
	if !consumers.Exists() {
		return errConsumersNotFound
	}
	if len(consumers.Array()) == 0 {
		return errConsumersIsEmpty
	}

	var credentials map[string]bool

	for _, item := range consumers.Array() {
		name := item.Get("name")
		if !name.Exists() || name.String() == "" {
			return errConsumerNotFound
		}
		credential := item.Get("credential")
		if !credential.Exists() || credential.String() == "" {
			return errConsumerNotFound
		}
		if _, ok := credentials[credential.String()]; ok {
			return errDuplicateConsumer
		} else {
			credentials[credential.String()] = true
		}
		userAndPasswd := strings.Split(credential.String(), ":")
		if len(userAndPasswd) != 2 {
			return errInvalidCredentialFormat
		}

		consumer := Consumer{
			Name:       name.String(),
			Credential: credential.String(),
		}
		config.Consumers = append(config.Consumers, consumer)
	}

	return nil
}
