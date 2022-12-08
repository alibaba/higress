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

package kube

import (
	istiokube "istio.io/istio/pkg/kube"
	"k8s.io/client-go/tools/clientcmd"

	higressclient "github.com/alibaba/higress/client/pkg/clientset/versioned"
	higressinformer "github.com/alibaba/higress/client/pkg/informers/externalversions"
)

type Client interface {
	istiokube.Client

	// Higress returns the Higress kube client.
	Higress() higressclient.Interface

	// HigressInformer returns an informer for the higress client
	HigressInformer() higressinformer.SharedInformerFactory
}

type client struct {
	istiokube.Client

	higress         higressclient.Interface
	higressInformer higressinformer.SharedInformerFactory
}

const resyncInterval = 0

func NewClient(clientConfig clientcmd.ClientConfig) (Client, error) {
	var c client
	istioClient, err := istiokube.NewClient(clientConfig)
	if err != nil {
		return nil, err
	}
	c.Client = istioClient

	c.higress, err = higressclient.NewForConfig(istioClient.RESTConfig())
	if err != nil {
		return nil, err
	}
	c.higressInformer = higressinformer.NewSharedInformerFactory(c.higress, resyncInterval)
	return &c, nil
}

func (c *client) Higress() higressclient.Interface {
	return c.higress
}

func (c *client) HigressInformer() higressinformer.SharedInformerFactory {
	return c.higressInformer
}

func (c *client) RunAndWait(stop <-chan struct{}) {
	c.Client.RunAndWait(stop)
	c.higressInformer.Start(stop)
	c.higressInformer.WaitForCacheSync(stop)
}
