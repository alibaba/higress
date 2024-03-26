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
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

func LocalAvailablePort(localAddress string) (int, error) {
	l, err := net.Listen("tcp", fmt.Sprintf("%s:0", localAddress))
	if err != nil {
		return 0, err
	}

	return l.Addr().(*net.TCPAddr).Port, l.Close()
}

type PortForwarder interface {
	Start() error

	Stop()

	// Address returns the address of the local forwarded address.
	Address() string

	// WaitForStop blocks until connection closed (e.g. control-C interrupt)
	WaitForStop()
}

var _ PortForwarder = &localForwarder{}

type localForwarder struct {
	types.NamespacedName
	CLIClient

	localPort    int
	podPort      int
	localAddress string

	stopCh chan struct{}
}

func NewLocalPortForwarder(client CLIClient, namespacedName types.NamespacedName, localPort, podPort int, bindAddress string) (PortForwarder, error) {
	f := &localForwarder{
		stopCh:         make(chan struct{}),
		CLIClient:      client,
		NamespacedName: namespacedName,
		localPort:      localPort,
		podPort:        podPort,
		localAddress:   bindAddress,
	}
	if f.localPort == 0 {
		// get a random port
		p, err := LocalAvailablePort(bindAddress)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get a local available port")
		}
		f.localPort = p
	}

	return f, nil
}

func (f *localForwarder) Start() error {
	errCh := make(chan error, 1)
	readyCh := make(chan struct{}, 1)
	go func() {
		for {
			select {
			case <-f.stopCh:
				return
			default:
			}

			fw, err := f.buildKubernetesPortForwarder(readyCh)
			if err != nil {
				errCh <- err
				return
			}

			if err := fw.ForwardPorts(); err != nil {
				errCh <- err
				return
			}

			readyCh = nil
		}

	}()

	select {
	case err := <-errCh:
		return errors.Wrap(err, "failed to start port forwarder")
	case <-readyCh:
		return nil
	}
}

func (f *localForwarder) buildKubernetesPortForwarder(readyCh chan struct{}) (*portforward.PortForwarder, error) {
	restClient, err := rest.RESTClientFor(f.RESTConfig())
	if err != nil {
		return nil, err
	}

	req := restClient.Post().Resource("pods").Namespace(f.Namespace).Name(f.Name).SubResource("portforward")
	serverURL := req.URL()

	roundTripper, upgrader, err := spdy.RoundTripperFor(f.RESTConfig())
	if err != nil {
		return nil, fmt.Errorf("failure creating roundtripper: %v", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: roundTripper}, http.MethodPost, serverURL)
	fw, err := portforward.NewOnAddresses(dialer,
		[]string{f.localAddress},
		[]string{fmt.Sprintf("%d:%d", f.localPort, f.podPort)},
		f.stopCh,
		readyCh,
		io.Discard,
		os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("failed establishing portforward: %v", err)
	}

	return fw, nil
}

func (f *localForwarder) Stop() {
	close(f.stopCh)
}

func (f *localForwarder) Address() string {
	return fmt.Sprintf("%s:%d", f.localAddress, f.localPort)
}

func (f *localForwarder) WaitForStop() {
	<-f.stopCh
}
