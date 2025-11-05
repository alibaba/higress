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

package annotations

import (
	"github.com/alibaba/higress/v2/pkg/ingress/kube/util"
	. "github.com/alibaba/higress/v2/pkg/ingress/log"
)

const (
	authType          = "auth-type"
	authRealm         = "auth-realm"
	authSecretAnn     = "auth-secret"
	authSecretTypeAnn = "auth-secret-type"

	defaultAuthType = "basic"
	authFileKey     = "auth"
)

type authSecretType string

const (
	authFileAuthSecretType authSecretType = "auth-file"
	authMapAuthSecretType  authSecretType = "auth-map"
)

var _ Parser = auth{}

type AuthConfig struct {
	AuthType    string
	AuthRealm   string
	Credentials []string
	AuthSecret  util.ClusterNamespacedName
}

type auth struct{}

func (a auth) Parse(annotations Annotations, config *Ingress, globalContext *GlobalContext) error {
	if !needAuthConfig(annotations) {
		return nil
	}
	IngressLog.Error("The annotation nginx.ingress.kubernetes.io/auth-type is no longer supported after version 2.0.0, please use the higress wasm plugin (e.g., basic-auth) as an alternative.")
	return nil
}

func needAuthConfig(annotations Annotations) bool {
	return annotations.HasASAP(authType) &&
		annotations.HasASAP(authSecretAnn)
}
