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
	CustomHigressNamespace string // default "higress-system"
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
