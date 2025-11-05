// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package util

import (
	"istio.io/istio/pilot/pkg/util/informermetric"
	"istio.io/istio/pkg/config/schema/kubeclient"
	"istio.io/istio/pkg/kube/informerfactory"
	ktypes "istio.io/istio/pkg/kube/kubetypes"
	"istio.io/istio/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

func GetInformerFiltered(
	c kubeclient.ClientGetter,
	opts ktypes.InformerOptions,
	g schema.GroupVersionResource,
	exampleObject runtime.Object,
	l func(options metav1.ListOptions) (runtime.Object, error),
	w func(options metav1.ListOptions) (watch.Interface, error),
) informerfactory.StartableInformer {
	return c.Informers().InformerFor(g, opts, func() cache.SharedIndexInformer {
		inf := cache.NewSharedIndexInformer(
			&cache.ListWatch{
				ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
					options.FieldSelector = opts.FieldSelector
					options.LabelSelector = opts.LabelSelector
					return l(options)
				},
				WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
					options.FieldSelector = opts.FieldSelector
					options.LabelSelector = opts.LabelSelector
					return w(options)
				},
			},
			exampleObject,
			0,
			cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc},
		)
		setupInformer(opts, inf)
		return inf
	})
}

func setupInformer(opts ktypes.InformerOptions, inf cache.SharedIndexInformer) {
	// It is important to set this in the newFunc rather than after InformerFor to avoid
	// https://github.com/kubernetes/kubernetes/issues/117869
	if opts.ObjectTransform != nil {
		_ = inf.SetTransform(opts.ObjectTransform)
	} else {
		_ = inf.SetTransform(stripUnusedFields)
	}
	if err := inf.SetWatchErrorHandler(informermetric.ErrorHandlerForCluster(opts.Cluster)); err != nil {
		log.Debugf("failed to set watch handler, informer may already be started: %v", err)
	}
}

// stripUnusedFields is the transform function for shared informers,
// it removes unused fields from objects before they are stored in the cache to save memory.
func stripUnusedFields(obj any) (any, error) {
	t, ok := obj.(metav1.ObjectMetaAccessor)
	if !ok {
		// shouldn't happen
		return obj, nil
	}
	// ManagedFields is large and we never use it
	t.GetObjectMeta().SetManagedFields(nil)
	return obj, nil
}
