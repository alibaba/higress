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
	"strconv"
	"strings"

	jsonpatch "github.com/evanphx/json-patch/v5"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
)

// applyOverlay applies an overlay using JSON patch strategy over the current Object in place.
func applyOverlay(current, overlay *unstructured.Unstructured) error {
	cj, err := runtime.Encode(unstructured.UnstructuredJSONScheme, current)
	if err != nil {
		return err
	}

	overlayUpdated := overlay.DeepCopy()
	if strings.EqualFold(current.GetKind(), "service") {
		if err := saveClusterIP(current, overlayUpdated); err != nil {
			return err
		}

		saveNodePorts(current, overlayUpdated)
	}

	if current.GetKind() == "PersistentVolumeClaim" {
		if err := savePersistentVolumeClaim(current, overlayUpdated); err != nil {
			return err
		}
	}

	uj, err := runtime.Encode(unstructured.UnstructuredJSONScheme, overlayUpdated)
	if err != nil {
		return err
	}
	merged, err := jsonpatch.MergePatch(cj, uj)
	if err != nil {
		return err
	}
	return runtime.DecodeInto(unstructured.UnstructuredJSONScheme, merged, current)
}

// createPortMap returns a map, mapping the value of the port and value of the nodePort
func createPortMap(current *unstructured.Unstructured) map[string]uint32 {
	portMap := make(map[string]uint32)
	svc := &corev1.Service{}
	if err := scheme.Scheme.Convert(current, svc, nil); err != nil {
		return portMap
	}
	for _, p := range svc.Spec.Ports {
		portMap[strconv.Itoa(int(p.Port))] = uint32(p.NodePort)
	}
	return portMap
}

// savePersistentVolumeClaim copies the storageClassName from the current cluster into the overlay
func savePersistentVolumeClaim(current, overlay *unstructured.Unstructured) error {
	// Save the value of spec.storageClassName set by the cluster
	if storageClassName, found, err := unstructured.NestedString(current.Object, "spec",
		"storageClassName"); err != nil {
		return err
	} else if found {
		if _, _, err2 := unstructured.NestedString(overlay.Object, "spec",
			"storageClassName"); err2 != nil {
			// override when overlay storageClassName property is not existed
			if err3 := unstructured.SetNestedField(overlay.Object, storageClassName, "spec",
				"storageClassName"); err3 != nil {
				return err3
			}
		}
	}
	return nil
}

// saveNodePorts transfers the port values from the current cluster into the overlay
func saveNodePorts(current, overlay *unstructured.Unstructured) {
	portMap := createPortMap(current)
	ports, _, _ := unstructured.NestedFieldNoCopy(overlay.Object, "spec", "ports")
	portList, ok := ports.([]any)
	if !ok {
		return
	}
	for _, port := range portList {
		m, ok := port.(map[string]any)
		if !ok {
			continue
		}
		if nodePortNum, ok := m["nodePort"]; ok && fmt.Sprintf("%v", nodePortNum) == "0" {
			if portNum, ok := m["port"]; ok {
				if v, ok := portMap[fmt.Sprintf("%v", portNum)]; ok {
					m["nodePort"] = v
				}
			}
		}
	}
}

// saveClusterIP copies the cluster IP from the current cluster into the overlay
func saveClusterIP(current, overlay *unstructured.Unstructured) error {
	// Save the value of spec.clusterIP set by the cluster
	if clusterIP, found, err := unstructured.NestedString(current.Object, "spec",
		"clusterIP"); err != nil {
		return err
	} else if found {
		if err := unstructured.SetNestedField(overlay.Object, clusterIP, "spec",
			"clusterIP"); err != nil {
			return err
		}
	}
	return nil
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
