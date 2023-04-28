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
	"errors"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/alibaba/higress/pkg/ingress/kube/util"
	. "github.com/alibaba/higress/pkg/ingress/log"
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

	authConfig := &AuthConfig{
		AuthType: defaultAuthType,
	}

	// Check auth type
	authType, err := annotations.ParseStringASAP(authType)
	if err != nil {
		IngressLog.Errorf("Parse auth type error %v within ingress %/%s", err, config.Namespace, config.Name)
		return nil
	}
	if authType != defaultAuthType {
		IngressLog.Errorf("Auth type %s within ingress %/%s is not supported yet.", authType, config.Namespace, config.Name)
		return nil
	}

	secretName, _ := annotations.ParseStringASAP(authSecretAnn)
	namespaced := util.SplitNamespacedName(secretName)
	if namespaced.Name == "" {
		IngressLog.Errorf("Auth secret name within ingress %s/%s is invalid", config.Namespace, config.Name)
		return nil
	}
	if namespaced.Namespace == "" {
		namespaced.Namespace = config.Namespace
	}

	configKey := util.ClusterNamespacedName{
		NamespacedName: namespaced,
		ClusterId:      config.ClusterId,
	}
	authConfig.AuthSecret = configKey

	// Subscribe secret
	globalContext.WatchedSecrets.Insert(configKey.String())

	secretType := authFileAuthSecretType
	if rawSecretType, err := annotations.ParseStringASAP(authSecretTypeAnn); err == nil {
		resultAuthSecretType := authSecretType(rawSecretType)
		if resultAuthSecretType == authFileAuthSecretType || resultAuthSecretType == authMapAuthSecretType {
			secretType = resultAuthSecretType
		}
	}

	authConfig.AuthRealm, _ = annotations.ParseStringASAP(authRealm)

	// Process credentials.
	secretLister, exist := globalContext.ClusterSecretLister[config.ClusterId]
	if !exist {
		IngressLog.Errorf("secret lister of cluster %s doesn't exist", config.ClusterId)
		return nil
	}
	authSecret, err := secretLister.Secrets(namespaced.Namespace).Get(namespaced.Name)
	if err != nil {
		IngressLog.Errorf("Secret %s within ingress %s/%s is not found",
			namespaced.String(), config.Namespace, config.Name)
		return nil
	}
	credentials, err := convertCredentials(secretType, authSecret)
	if err != nil {
		IngressLog.Errorf("Parse auth secret fail, err %v", err)
		return nil
	}
	authConfig.Credentials = credentials

	config.Auth = authConfig
	return nil
}

func convertCredentials(secretType authSecretType, secret *corev1.Secret) ([]string, error) {
	var result []string
	switch secretType {
	case authFileAuthSecretType:
		users, exist := secret.Data[authFileKey]
		if !exist {
			return nil, errors.New("the auth file type must has auth key in secret data")
		}
		userList := strings.Split(string(users), "\n")
		for _, item := range userList {
			if !strings.Contains(item, ":") {
				continue
			}
			result = append(result, item)
		}
	case authMapAuthSecretType:
		for name, password := range secret.Data {
			result = append(result, name+":"+string(password))
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i] < result[j]
	})

	return result, nil
}

func needAuthConfig(annotations Annotations) bool {
	return annotations.HasASAP(authType) &&
		annotations.HasASAP(authSecretAnn)
}
