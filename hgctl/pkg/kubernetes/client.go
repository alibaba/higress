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
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/retry"
	ctrClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type CLIClient interface {
	// RESTConfig returns the Kubernetes rest.Config used to configure the clients.
	RESTConfig() *rest.Config

	// Pod returns the pod for the given namespaced name.
	Pod(namespacedName types.NamespacedName) (*corev1.Pod, error)

	// PodsForSelector finds pods matching selector.
	PodsForSelector(namespace string, labelSelectors ...string) (*corev1.PodList, error)

	// PodExec takes a command and the pod data to run the command in the specified pod.
	PodExec(namespacedName types.NamespacedName, container string, command string) (stdout string, stderr string, err error)

	// ApplyObject creates or updates unstructured object
	ApplyObject(obj *unstructured.Unstructured) error

	// DeleteObject delete unstructured object
	DeleteObject(obj *unstructured.Unstructured) error

	// CreateNamespace create namespace
	CreateNamespace(namespace string) error

	// KubernetesInterface get kubernetes interface
	KubernetesInterface() kubernetes.Interface
}

var _ CLIClient = &client{}

type client struct {
	config     *rest.Config
	restClient *rest.RESTClient
	kube       kubernetes.Interface
	ctrClient  ctrClient.Client
}

func NewCLIClient(clientConfig clientcmd.ClientConfig) (CLIClient, error) {
	return newClientInternal(clientConfig)
}

func newClientInternal(clientConfig clientcmd.ClientConfig) (*client, error) {
	var (
		c   client
		err error
	)

	c.config, err = clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	setRestDefaults(c.config)

	c.restClient, err = rest.RESTClientFor(c.config)
	if err != nil {
		return nil, err
	}

	c.kube, err = kubernetes.NewForConfig(c.config)
	if err != nil {
		return nil, err
	}

	c.ctrClient, err = ctrClient.New(c.config, ctrClient.Options{})
	if err != nil {
		return nil, err
	}
	return &c, err
}

func (c *client) RESTConfig() *rest.Config {
	if c.config == nil {
		return nil
	}
	cpy := *c.config
	return &cpy
}

func (c *client) PodsForSelector(namespace string, podSelectors ...string) (*corev1.PodList, error) {
	return c.kube.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: strings.Join(podSelectors, ","),
	})
}

func (c *client) Pod(namespacedName types.NamespacedName) (*corev1.Pod, error) {
	return c.kube.CoreV1().Pods(namespacedName.Namespace).Get(context.TODO(), namespacedName.Name, metav1.GetOptions{})
}

func (c *client) PodExec(namespacedName types.NamespacedName, container string, command string) (stdout string, stderr string, err error) {
	defer func() {
		if err != nil {
			if len(stderr) > 0 {
				err = fmt.Errorf("error exec into %s/%s container %s: %v\n%s",
					namespacedName.Namespace, namespacedName.Name, container, err, stderr)
			} else {
				err = fmt.Errorf("error exec into %s/%s container %s: %v",
					namespacedName.Namespace, namespacedName.Name, container, err)
			}
		}
	}()

	req := c.restClient.Post().
		Resource("pods").
		Namespace(namespacedName.Namespace).
		Name(namespacedName.Name).
		SubResource("exec").
		Param("container", container).
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   strings.Fields(command),
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, kubescheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(c.config, "POST", req.URL())
	if err != nil {
		return "", "", err
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
		Tty:    false,
	})

	stdout = stdoutBuf.String()
	stderr = stderrBuf.String()
	return
}

// DeleteObject delete unstructured object
func (c *client) DeleteObject(obj *unstructured.Unstructured) error {
	err := c.ctrClient.Delete(context.TODO(), obj, ctrClient.PropagationPolicy(metav1.DeletePropagationBackground))
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

// ApplyObject creates or updates unstructured object
func (c *client) ApplyObject(obj *unstructured.Unstructured) error {
	if obj.GetKind() == "List" {
		objList, err := obj.ToList()
		if err != nil {
			return err
		}
		for _, item := range objList.Items {
			if err := c.ApplyObject(&item); err != nil {
				return err
			}
		}
		return nil
	}

	key := ctrClient.ObjectKeyFromObject(obj)
	receiver := &unstructured.Unstructured{}
	receiver.SetGroupVersionKind(obj.GroupVersionKind())

	if err := retry.RetryOnConflict(wait.Backoff{
		Duration: time.Millisecond * 10,
		Factor:   2,
		Steps:    3,
	}, func() error {
		if err := c.ctrClient.Get(context.Background(), key, receiver); err != nil {
			if errors.IsNotFound(err) {
				if err := c.ctrClient.Create(context.Background(), obj); err != nil {
					return err
				}
			}
			return nil
		}
		if err := applyOverlay(receiver, obj); err != nil {
			return err
		}
		if err := c.ctrClient.Update(context.Background(), receiver); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

// CreateNamespace create namespace
func (c *client) CreateNamespace(namespace string) error {
	key := ctrClient.ObjectKey{
		Namespace: metav1.NamespaceSystem,
		Name:      namespace,
	}
	if err := c.ctrClient.Get(context.Background(), key, &corev1.Namespace{}); err != nil {
		if errors.IsNotFound(err) {
			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: metav1.NamespaceSystem,
					Name:      namespace,
				},
			}
			if err := c.ctrClient.Create(context.Background(), nsObj); err != nil {
				return err
			}
			return nil
		}
		return fmt.Errorf("failed to check if namespace %v exists: %v", namespace, err)
	}

	return nil
}

// KubernetesInterface get kubernetes interface
func (c *client) KubernetesInterface() kubernetes.Interface {
	return c.kube

}
