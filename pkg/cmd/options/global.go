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

package options

import (
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var DefaultConfigFlags = genericclioptions.NewConfigFlags(true)

func AddKubeConfigFlags(flags *pflag.FlagSet) {
	flags.StringVar(DefaultConfigFlags.KubeConfig, "kubeconfig", *DefaultConfigFlags.KubeConfig,
		"Path to the kubeconfig file to use for CLI requests.")
	flags.StringVar(DefaultConfigFlags.Context, "context", *DefaultConfigFlags.Context,
		"The name of the kubeconfig context to use.")
}
