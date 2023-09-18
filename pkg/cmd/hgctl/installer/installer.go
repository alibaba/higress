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

package installer

import (
	"errors"
	"fmt"
	"io"

	"github.com/alibaba/higress/pkg/cmd/hgctl/helm"
	"github.com/alibaba/higress/pkg/cmd/hgctl/helm/object"
	"github.com/alibaba/higress/pkg/cmd/hgctl/kubernetes"
)

type Installer struct {
	started    bool
	components map[ComponentName]Component
	kubeCli    kubernetes.CLIClient
	profile    *helm.Profile
	writer     io.Writer
}

// Run must be invoked before invoking other functions.
func (o *Installer) Run() error {
	for name, component := range o.components {
		if !component.Enabled() {
			continue
		}
		if err := component.Run(); err != nil {
			return fmt.Errorf("component %s run failed, err: %s", name, err)
		}
	}
	o.started = true
	return nil
}

// RenderManifests renders component manifests specified by profile.
func (o *Installer) RenderManifests() (map[ComponentName]string, error) {
	if !o.started {
		return nil, errors.New("HigressOperator is not running")
	}
	res := make(map[ComponentName]string)
	for name, component := range o.components {
		if !component.Enabled() {
			continue
		}
		manifest, err := component.RenderManifest()
		if err != nil {
			return nil, fmt.Errorf("component %s RenderManifest err: %v", name, err)
		}
		res[name] = manifest
	}
	return res, nil
}

// ApplyManifests apply component manifests to k8s cluster
func (o *Installer) ApplyManifests(manifestMap map[ComponentName]string) error {
	if o.kubeCli == nil {
		return errors.New("no injected k8s cli into Installer")
	}
	for name, manifest := range manifestMap {
		namespace := o.components[name].Namespace()
		if err := o.applyManifest(manifest, namespace); err != nil {
			return fmt.Errorf("component %s ApplyManifest err: %v", name, err)
		}
	}
	return nil
}

func (o *Installer) applyManifest(manifest string, ns string) error {
	if err := o.kubeCli.CreateNamespace(ns); err != nil {
		return err
	}
	objs, err := object.ParseK8sObjectsFromYAMLManifest(manifest)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		// check namespaced object if namespace property has been existed
		if obj.Namespace == "" && o.isNamespacedObject(obj) {
			obj.Namespace = ns
			obj.UnstructuredObject().SetNamespace(ns)
		}
		if o.isNamespacedObject(obj) {
			fmt.Fprintf(o.writer, "start to apply object kind: %s, object name: %s on namespace: %s ......\n", obj.Kind, obj.Name, obj.Namespace)
		} else {
			fmt.Fprintf(o.writer, "start to apply object kind: %s, object name: %s  ......\n", obj.Kind, obj.Name)
		}
		if err := o.kubeCli.ApplyObject(obj.UnstructuredObject()); err != nil {
			return err
		}
	}
	return nil
}

// DeleteManifests delete component manifests to k8s cluster
func (o *Installer) DeleteManifests(manifestMap map[ComponentName]string) error {
	if o.kubeCli == nil {
		return errors.New("no injected k8s cli into Installer")
	}
	for name, manifest := range manifestMap {
		namespace := o.components[name].Namespace()
		if err := o.deleteManifest(manifest, namespace); err != nil {
			return fmt.Errorf("component %s DeleteManifest err: %v", name, err)
		}
	}
	return nil
}

// deleteManifest delete manifest to certain namespace
func (o *Installer) deleteManifest(manifest string, ns string) error {
	objs, err := object.ParseK8sObjectsFromYAMLManifest(manifest)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		// check namespaced object if namespace property has been existed
		if obj.Namespace == "" && o.isNamespacedObject(obj) {
			obj.Namespace = ns
			obj.UnstructuredObject().SetNamespace(ns)
		}
		if o.isNamespacedObject(obj) {
			fmt.Fprintf(o.writer, "start to delete object kind: %s, object name: %s on namespace: %s ......\n", obj.Kind, obj.Name, obj.Namespace)
		} else {
			fmt.Fprintf(o.writer, "start to delete object kind: %s, object name: %s  ......\n", obj.Kind, obj.Name)
		}
		if err := o.kubeCli.DeleteObject(obj.UnstructuredObject()); err != nil {
			return err
		}
	}

	return nil
}

func (o *Installer) isNamespacedObject(obj *object.K8sObject) bool {
	if obj.Kind != "CustomResourceDefinition" && obj.Kind != "ClusterRole" && obj.Kind != "ClusterRoleBinding" {
		return true
	}

	return false
}

func NewInstaller(profile *helm.Profile, cli kubernetes.CLIClient, writer io.Writer) (*Installer, error) {
	if profile == nil {
		return nil, errors.New("Install profile is empty")
	}
	// initialize components
	components := make(map[ComponentName]Component)
	higressComponent, err := NewHigressComponent(profile, writer,
		WithComponentNamespace(profile.Global.Namespace),
		WithComponentChartPath(profile.InstallPackagePath),
		WithComponentVersion(profile.Charts.Higress.Version),
		WithComponentRepoURL(profile.Charts.Higress.Url),
		WithComponentChartName(profile.Charts.Higress.Name),
	)
	if err != nil {
		return nil, fmt.Errorf("NewHigressComponent failed, err: %s", err)
	}
	components[Higress] = higressComponent

	if profile.IstioEnabled() {
		istioCRDComponent, err := NewIstioCRDComponent(profile, writer,
			WithComponentNamespace(profile.Global.IstioNamespace),
			WithComponentChartPath(profile.InstallPackagePath),
			WithComponentVersion(profile.Charts.Istio.Version),
			WithComponentRepoURL(profile.Charts.Istio.Url),
			WithComponentChartName(profile.Charts.Istio.Name),
		)
		if err != nil {
			return nil, fmt.Errorf("NewIstioCRDComponent failed, err: %s", err)
		}
		components[Istio] = istioCRDComponent
	}
	op := &Installer{
		profile:    profile,
		components: components,
		kubeCli:    cli,
		writer:     writer,
	}
	return op, nil
}
