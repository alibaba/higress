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

package kubernetes

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type DynamicClient struct {
	config *rest.Config
	client dynamic.Interface
}

func NewDynamicClient(clientConfig clientcmd.ClientConfig) (*DynamicClient, error) {
	var (
		c   DynamicClient
		err error
	)

	c.config, err = clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	c.client, err = dynamic.NewForConfig(c.config)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func (c DynamicClient) Get(gvr schema.GroupVersionResource, namespace, name string) (*unstructured.Unstructured, error) {
	return c.client.Resource(gvr).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

func (c DynamicClient) List(gvr schema.GroupVersionResource, namespace string) (*unstructured.UnstructuredList, error) {
	return c.client.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
}

func (c DynamicClient) Create(gvr schema.GroupVersionResource, namespace string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return c.client.Resource(gvr).Namespace(namespace).Create(context.TODO(), obj, metav1.CreateOptions{})
}

func (c DynamicClient) Delete(gvr schema.GroupVersionResource, namespace, name string) (*unstructured.Unstructured, error) {
	ctx := context.TODO()
	result, err := c.client.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	err = c.client.Resource(gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c DynamicClient) Update(gvr schema.GroupVersionResource, namespace string, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return c.client.Resource(gvr).Namespace(namespace).Update(context.TODO(), obj, metav1.UpdateOptions{})
}
