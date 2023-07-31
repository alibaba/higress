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
