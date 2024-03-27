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

package app

import (
	"context"
	"flag"

	"github.com/alibaba/higress/cmd/cert/app/options"
	"github.com/alibaba/higress/pkg/cert"
	"github.com/spf13/cobra"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
)

// NewManagerCommand creates a *cobra.Command object with default parameters
func NewManagerCommand(ctx context.Context) *cobra.Command {
	opts := options.NewOptions()

	cmd := &cobra.Command{
		Use:  "serve",
		Long: `Higress certificate manager for automatic https`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := opts.Complete(); err != nil {
				klog.Exit(err)
			}
			if errs := opts.Validate(); len(errs) != 0 {
				klog.Exit(errs)
			}

			if err := Run(ctx, opts); err != nil {
				klog.Exit(err)
			}
		},
	}

	cmd.Flags().AddGoFlagSet(flag.CommandLine)
	opts.AddFlags(cmd.Flags())
	return cmd
}

// Run runs with options. This should never exit.
func Run(ctx context.Context, opts *options.Options) error {
	config := ctrl.GetConfigOrDie()
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	configMgr := cert.NewConfigMgr(opts.WatchNamespace, clientSet)
	// init config if there is not existed
	defaultConfig, err := configMgr.InitConfig(opts.Email)
	if err != nil {
		return err
	}

	// init server
	server, _ := cert.NewServer(opts.WatchNamespace, clientSet, configMgr)
	// config and start
	server.InitConfig(defaultConfig)
	server.InitHttpServer()

	// configmap informer list and watch
	kubeInformerFactory := informers.NewSharedInformerFactoryWithOptions(clientSet, 0, informers.WithNamespace(opts.WatchNamespace))
	configmapInformer := kubeInformerFactory.Core().V1().ConfigMaps()
	controller := cert.NewController(server, clientSet, opts.WatchNamespace, configmapInformer, configMgr)
	kubeInformerFactory.Start(ctx.Done())
	go controller.Run(ctx.Done())

	// run server
	if err := server.Run(ctx); err != nil {
		return err
	}

	return nil
}
