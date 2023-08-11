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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	HigressExtGroup      = "extensions.higress.io"
	HigressExtVersion    = "v1alpha1"
	HigressExtAPIVersion = HigressExtGroup + "/" + HigressExtVersion
	HigressNamespace     = "higress-system"

	WasmPluginKind     = "WasmPlugin"
	WasmPluginResource = "wasmplugins"
)

var (
	CustomHigressNamespace = "higress-system" // default
	WasmPluginRes          = schema.GroupVersionResource{Group: HigressExtGroup, Version: HigressExtVersion, Resource: WasmPluginResource}
)

func GetWasmPlugin(c *DynamicClient, name string) (*unstructured.Unstructured, error) {
	return c.Get(WasmPluginRes, CustomHigressNamespace, name)
}

func ListWasmPlugins(c *DynamicClient) (*unstructured.UnstructuredList, error) {
	return c.List(WasmPluginRes, CustomHigressNamespace)
}

func CreateWasmPlugin(c *DynamicClient, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return c.Create(WasmPluginRes, CustomHigressNamespace, obj)
}

func DeleteWasmPlugin(c *DynamicClient, name string) (*unstructured.Unstructured, error) {
	return c.Delete(WasmPluginRes, CustomHigressNamespace, name)
}

func UpdateWasmPlugin(c *DynamicClient, obj *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	return c.Update(WasmPluginRes, CustomHigressNamespace, obj)
}
