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

package main

import (
	"errors"
	"fmt"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
)

func main() {
	wrapper.SetCtx(
		"custom-span-attribute",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

type CustomSpanAttributeConfig struct {
	HeaderKey string
}

func setSpanAttribute(key string, value interface{}, log wrapper.Log) {
	if value != "" {
		traceSpanTag := wrapper.TraceSpanTagPrefix + key
		if e := proxywasm.SetProperty([]string{traceSpanTag}, []byte(fmt.Sprint(value))); e != nil {
			log.Warnf("failed to set %s in filter state: %v", traceSpanTag, e)
		}
	} else {
		log.Debugf("failed to write span attribute [%s], because it's value is empty")
	}
}

/*
Example configuration:

```yaml
headerKey: foo
```
*/
func parseConfig(configJson gjson.Result, config *CustomSpanAttributeConfig, log wrapper.Log) error {
	config.HeaderKey = configJson.Get("headerKey").String()
	if config.HeaderKey == "" {
		return errors.New("no header key specified")
	}
	return nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config CustomSpanAttributeConfig, log wrapper.Log) types.Action {
	headerValue, _ := proxywasm.GetHttpRequestHeader(config.HeaderKey)
	setSpanAttribute(config.HeaderKey, headerValue, log)
	return types.ActionContinue
}
