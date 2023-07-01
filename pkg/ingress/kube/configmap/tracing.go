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

package configmap

import (
	"fmt"
	"sync/atomic"

	"github.com/alibaba/higress/pkg/ingress/kube/util"
	. "github.com/alibaba/higress/pkg/ingress/log"
	networking "istio.io/api/networking/v1alpha3"
	"istio.io/istio/pkg/config"
	"istio.io/istio/pkg/config/schema/gvk"
	listersv1 "k8s.io/client-go/listers/core/v1"
	"sigs.k8s.io/yaml"
)

type TracingMgr struct {
	Namespace               string
	HigressConfigController HigressConfigController
	HigressConfigLister     listersv1.ConfigMapNamespaceLister
	higressConfig           atomic.Value
}

func NewTracingMgr(namespace string, higressConfigController HigressConfigController, higressConfigLister listersv1.ConfigMapNamespaceLister) *TracingMgr {

	tracingMgr := &TracingMgr{
		Namespace:               namespace,
		HigressConfigController: higressConfigController,
		HigressConfigLister:     higressConfigLister,
		higressConfig:           atomic.Value{},
	}
	tracingMgr.HigressConfigController.AddEventHandler(tracingMgr.AddOrUpdateHigressConfig)
	tracingMgr.SetHigressConfig(NewDefaultHigressConfig())
	return tracingMgr
}

func (t *TracingMgr) SetHigressConfig(higressConfig *HigressConfig) {
	t.higressConfig.Store(higressConfig)
}

func (t *TracingMgr) GetHigressConfig() *HigressConfig {
	value := t.higressConfig.Load()
	if value != nil {
		if higressConfig, ok := value.(*HigressConfig); ok {
			return higressConfig
		}
	}
	return nil
}

func (t *TracingMgr) AddOrUpdateHigressConfig(name util.ClusterNamespacedName) {
	if name.Namespace != t.Namespace || name.Name != HigressConfigMapName {
		return
	}
	higressConfigmap, err := t.HigressConfigLister.Get(HigressConfigMapName)
	if err != nil {
		IngressLog.Errorf("higress-config configmap is not found, namespace:%s, name:%s",
			name.Namespace, name.Name)
		return
	}

	if _, ok := higressConfigmap.Data[HigressConfigMapKey]; !ok {
		return
	}

	newHigressConfig := NewDefaultHigressConfig()
	if err = yaml.Unmarshal([]byte(higressConfigmap.Data[HigressConfigMapKey]), newHigressConfig); err != nil {
		IngressLog.Errorf("data:%s,  convert to higressconfig error, error: %+v", higressConfigmap.Data[HigressConfigMapKey], err)
		return
	}

	if err = ValidTracing(newHigressConfig); err != nil {
		IngressLog.Errorf("data:%s,  convert to higress config map, error: %+v", higressConfigmap.Data[HigressConfigMapKey], err)
		return
	}

	result, _ := CompareTracing(t.GetHigressConfig(), newHigressConfig)

	switch result {
	case ResultReplace:
		t.SetHigressConfig(newHigressConfig)
		IngressLog.Infof("AddOrUpdate Higress config")
	case ResultDelete:
		t.SetHigressConfig(NewDefaultHigressConfig())
		IngressLog.Infof("Delete Higress config")
	}

}

func (t *TracingMgr) ConstructTracingEnvoyFilter() (*config.Config, error) {
	higressConfig := t.GetHigressConfig()
	namespace := t.Namespace
	if higressConfig == nil {
		return nil, nil
	}
	if higressConfig.Tracing.Enable == false {
		return nil, nil
	}
	tracingConfig := t.constructTracingTracer(higressConfig, namespace)
	if len(tracingConfig) == 0 {
		return nil, nil
	}

	return &config.Config{
		Meta: config.Meta{
			GroupVersionKind: gvk.EnvoyFilter,
			Name:             HigressTracingEnvoyFilterName,
			Namespace:        namespace,
		},
		Spec: &networking.EnvoyFilter{
			ConfigPatches: []*networking.EnvoyFilter_EnvoyConfigObjectPatch{
				{
					ApplyTo: networking.EnvoyFilter_NETWORK_FILTER,
					Match: &networking.EnvoyFilter_EnvoyConfigObjectMatch{
						Context: networking.EnvoyFilter_GATEWAY,
						ObjectTypes: &networking.EnvoyFilter_EnvoyConfigObjectMatch_Listener{
							Listener: &networking.EnvoyFilter_ListenerMatch{
								FilterChain: &networking.EnvoyFilter_ListenerMatch_FilterChainMatch{
									Filter: &networking.EnvoyFilter_ListenerMatch_FilterMatch{
										Name: "envoy.filters.network.http_connection_manager",
									},
								},
							},
						},
					},
					Patch: &networking.EnvoyFilter_Patch{
						Operation: networking.EnvoyFilter_Patch_MERGE,
						Value:     util.BuildPatchStruct(tracingConfig),
					},
				},
				{
					ApplyTo: networking.EnvoyFilter_HTTP_FILTER,
					Match: &networking.EnvoyFilter_EnvoyConfigObjectMatch{
						Context: networking.EnvoyFilter_GATEWAY,
						ObjectTypes: &networking.EnvoyFilter_EnvoyConfigObjectMatch_Listener{
							Listener: &networking.EnvoyFilter_ListenerMatch{
								FilterChain: &networking.EnvoyFilter_ListenerMatch_FilterChainMatch{
									Filter: &networking.EnvoyFilter_ListenerMatch_FilterMatch{
										Name: "envoy.filters.network.http_connection_manager",
										SubFilter: &networking.EnvoyFilter_ListenerMatch_SubFilterMatch{
											Name: "envoy.filters.http.router",
										},
									},
								},
							},
						},
					},
					Patch: &networking.EnvoyFilter_Patch{
						Operation: networking.EnvoyFilter_Patch_MERGE,
						Value: util.BuildPatchStruct(`{
							"name":"envoy.filters.http.router",
							"typed_config":{
								"@type": "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router",
								"start_child_span": true
							}
						}`),
					},
				},
			},
		},
	}, nil
}

func (t *TracingMgr) constructTracingTracer(higressConfig *HigressConfig, namespace string) string {
	tracingConfig := ""
	timeout := float32(higressConfig.Tracing.Timeout) / 1000
	if higressConfig.Tracing.Skywalking != nil {
		skywalking := higressConfig.Tracing.Skywalking
		tracingConfig = fmt.Sprintf(`{
	"name": "envoy.filters.network.http_connection_manager",
	"typed_config": {
		"@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
		"tracing": {
			"provider": {
				"name": "envoy.tracers.skywalking",
				"typed_config": {
					"@type": "type.googleapis.com/envoy.config.trace.v3.SkyWalkingConfig",
					"client_config": {
						"service_name": "higress-gateway.%s",
                        "backend_token": "%s"
					},
					"grpc_service": {
						"envoy_grpc": {
							"cluster_name": "outbound|%s||%s"
						},
						"timeout": "%.3fs"
					}
				}
			},
			"random_sampling": {
				"value": %.1f
			}
		}
	}
}`, namespace, skywalking.AccessToken, skywalking.Port, skywalking.Service, timeout, higressConfig.Tracing.Sampling)
	}

	if higressConfig.Tracing.Zipkin != nil {
		zipkin := higressConfig.Tracing.Zipkin
		tracingConfig = fmt.Sprintf(`{
	"name": "envoy.filters.network.http_connection_manager",
	"typed_config": {
		"@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
		"tracing": {
			"provider": {
				"name": "envoy.tracers.zipkin",
				"typed_config": {
					"@type": "type.googleapis.com/envoy.config.trace.v3.ZipkinConfig",
                    "collector_cluster": "outbound|%s||%s",
                    "collector_endpoint": "/api/v2/spans",
                    "collector_hostname": "higress-gateway",
                    "collector_endpoint_version": "HTTP_JSON",
                    "split_spans_for_request": true
				}
			},
			"random_sampling": {
				"value": %.1f
			}
		}
	}
}`, zipkin.Port, zipkin.Service, higressConfig.Tracing.Sampling)
	}

	if higressConfig.Tracing.OpenTelemetry != nil {
		opentelemetry := higressConfig.Tracing.OpenTelemetry
		tracingConfig = fmt.Sprintf(`{
	"name": "envoy.filters.network.http_connection_manager",
	"typed_config": {
		"@type": "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager",
		"tracing": {
			"provider": {
				"name": "envoy.tracers.opentelemetry",
				"typed_config": {
					"@type": "type.googleapis.com/envoy.config.trace.v3.OpenTelemetryConfig",
					"service_name": "higress-gateway.%s"
					"grpc_service": {
						"envoy_grpc": {
							"cluster_name": "outbound|%s||%s"
						},
						"timeout": "%.3fs"
					}
				}
			},
			"random_sampling": {
				"value": %.1f
			}
		}
	}
}`, namespace, opentelemetry.Port, opentelemetry.Service, timeout, higressConfig.Tracing.Sampling)
	}
	return tracingConfig
}
