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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
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
}

var _ CLIClient = &client{}

type client struct {
	config     *rest.Config
	restClient *rest.RESTClient
	kube       kubernetes.Interface
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

	return &c, err
}

func setRestDefaults(config *rest.Config) *rest.Config {
	if config.GroupVersion == nil || config.GroupVersion.Empty() {
		config.GroupVersion = &corev1.SchemeGroupVersion
	}
	if len(config.APIPath) == 0 {
		if len(config.GroupVersion.Group) == 0 {
			config.APIPath = "/api"
		} else {
			config.APIPath = "/apis"
		}
	}
	if len(config.ContentType) == 0 {
		config.ContentType = runtime.ContentTypeJSON
	}
	if config.NegotiatedSerializer == nil {
		// This codec factory ensures the resources are not converted. Therefore, resources
		// will not be round-tripped through internal versions. Defaulting does not happen
		// on the client.
		config.NegotiatedSerializer = serializer.NewCodecFactory(kubescheme.Scheme).WithoutConversion()
	}

	return config
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
