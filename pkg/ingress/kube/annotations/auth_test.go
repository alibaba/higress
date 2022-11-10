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
	"context"
	"reflect"
	"testing"
	"time"

	"istio.io/istio/pilot/pkg/model"
	"istio.io/istio/pilot/pkg/util/sets"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/alibaba/higress/pkg/ingress/kube/util"
)

func TestAuthParse(t *testing.T) {
	auth := auth{}
	inputCases := []struct {
		input         map[string]string
		secret        *v1.Secret
		expect        *AuthConfig
		watchedSecret string
	}{
		{
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
				},
				Data: map[string][]byte{
					"auth": []byte("A:a\nB:b"),
				},
			},
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(authType): "digest",
			},
			expect: nil,
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
				},
				Data: map[string][]byte{
					"auth": []byte("A:a\nB:b"),
				},
			},
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(authType):        defaultAuthType,
				buildHigressAnnotationKey(authSecretAnn): "foo/bar",
			},
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
				},
				Data: map[string][]byte{
					"auth": []byte("A:a\nB:b"),
				},
			},
			expect: &AuthConfig{
				AuthType: defaultAuthType,
				AuthSecret: util.ClusterNamespacedName{
					NamespacedName: model.NamespacedName{
						Namespace: "foo",
						Name:      "bar",
					},
					ClusterId: "cluster",
				},
				Credentials: []string{"A:a", "B:b"},
			},
			watchedSecret: "cluster/foo/bar",
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(authType):          defaultAuthType,
				buildHigressAnnotationKey(authSecretAnn):   "foo/bar",
				buildNginxAnnotationKey(authSecretTypeAnn): string(authMapAuthSecretType),
			},
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "foo",
				},
				Data: map[string][]byte{
					"A": []byte("a"),
					"B": []byte("b"),
				},
			},
			expect: &AuthConfig{
				AuthType: defaultAuthType,
				AuthSecret: util.ClusterNamespacedName{
					NamespacedName: model.NamespacedName{
						Namespace: "foo",
						Name:      "bar",
					},
					ClusterId: "cluster",
				},
				Credentials: []string{"A:a", "B:b"},
			},
			watchedSecret: "cluster/foo/bar",
		},
		{
			input: map[string]string{
				buildNginxAnnotationKey(authType):          defaultAuthType,
				buildHigressAnnotationKey(authSecretAnn):   "bar",
				buildNginxAnnotationKey(authSecretTypeAnn): string(authFileAuthSecretType),
			},
			secret: &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "bar",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"auth": []byte("A:a\nB:b"),
				},
			},
			expect: &AuthConfig{
				AuthType: defaultAuthType,
				AuthSecret: util.ClusterNamespacedName{
					NamespacedName: model.NamespacedName{
						Namespace: "default",
						Name:      "bar",
					},
					ClusterId: "cluster",
				},
				Credentials: []string{"A:a", "B:b"},
			},
			watchedSecret: "cluster/default/bar",
		},
	}

	for _, inputCase := range inputCases {
		t.Run("", func(t *testing.T) {
			config := &Ingress{
				Meta: Meta{
					Namespace: "default",
					ClusterId: "cluster",
				},
			}

			globalContext, cancel := initGlobalContext(inputCase.secret)
			defer cancel()

			_ = auth.Parse(inputCase.input, config, globalContext)
			if !reflect.DeepEqual(inputCase.expect, config.Auth) {
				t.Fatal("Should be equal")
			}

			if inputCase.watchedSecret != "" {
				if !globalContext.WatchedSecrets.Contains(inputCase.watchedSecret) {
					t.Fatalf("Should watch secret %s", inputCase.watchedSecret)
				}
			}
		})
	}
}

func initGlobalContext(secret *v1.Secret) (*GlobalContext, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	client := fake.NewSimpleClientset(secret)
	informerFactory := informers.NewSharedInformerFactory(client, time.Hour)
	secretInformer := informerFactory.Core().V1().Secrets()
	go secretInformer.Informer().Run(ctx.Done())
	cache.WaitForCacheSync(ctx.Done(), secretInformer.Informer().HasSynced)

	return &GlobalContext{
		WatchedSecrets: sets.NewSet(),
		ClusterSecretLister: map[string]listerv1.SecretLister{
			"cluster": secretInformer.Lister(),
		},
	}, cancel
}
