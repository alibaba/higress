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
	"fmt"

	"github.com/alibaba/higress/pkg/cert"
	"github.com/spf13/pflag"
)

type Options struct {
	WatchNamespace string
	Email          string
}

// NewOptions builds an empty options.
func NewOptions() *Options {
	return &Options{}
}

func (c *Options) AddFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&c.WatchNamespace, "watch-namespace", "", "higress-system", "watch configmap namespace, default is higress-system")
	flags.StringVarP(&c.Email, "email", "", "", "acme email account")
}

// Complete completes all the required options.
func (o *Options) Complete() error {
	return nil
}

// Validate all required options.
func (o *Options) Validate() []error {
	var errors []error
	if len(o.Email) > 0 {
		if !cert.ValidateEmail(o.Email) {
			errors = append(errors, fmt.Errorf("%s is not a valid email address", o.Email))
		}
	}
	return errors
}
