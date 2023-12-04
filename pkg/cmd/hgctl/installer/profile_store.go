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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alibaba/higress/pkg/cmd/hgctl/helm"
	"github.com/alibaba/higress/pkg/cmd/hgctl/kubernetes"
	"github.com/alibaba/higress/pkg/cmd/hgctl/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ProfileConfigmapKey        = "profile"
	ProfileConfigmapName       = "higress-profile"
	ProfileConfigmapAnnotation = "higress.io/install"
	ProfileFilePrefix          = "install"
)

type ProfileContext struct {
	Profile        *helm.Profile
	SourceType     string
	Namespace      string
	PathOrName     string
	Install        helm.InstallMode
	HigressVersion string
}

type ProfileStore interface {
	Save(profile *helm.Profile) (string, error)
	List() ([]*ProfileContext, error)
	Delete(profile *helm.Profile) (string, error)
}

type FileDirProfileStore struct {
	profilesPath string
}

func (f *FileDirProfileStore) Save(profile *helm.Profile) (string, error) {
	namespace := profile.Global.Namespace
	install := profile.Global.Install
	var profileName = ""
	if install == helm.InstallK8s || install == helm.InstallLocalK8s {
		profileName = filepath.Join(f.profilesPath, fmt.Sprintf("%s-%s.yaml", ProfileFilePrefix, namespace))
	} else {
		profileName = filepath.Join(f.profilesPath, fmt.Sprintf("%s-%s.yaml", ProfileFilePrefix, install))
	}
	if err := util.WriteFileString(profileName, util.ToYAML(profile), 0o644); err != nil {
		return "", err
	}
	return profileName, nil
}

func (f *FileDirProfileStore) List() ([]*ProfileContext, error) {
	profileContexts := make([]*ProfileContext, 0)
	dir, err := os.ReadDir(f.profilesPath)
	if err != nil {
		return nil, err
	}
	for _, file := range dir {
		if !strings.HasSuffix(file.Name(), ".yaml") {
			continue
		}
		if file.IsDir() {
			continue
		}
		fileName := filepath.Join(f.profilesPath, file.Name())
		content, err2 := os.ReadFile(fileName)
		if err2 != nil {
			continue
		}
		profile, err3 := helm.UnmarshalProfile(string(content))
		if err3 != nil {
			continue
		}
		profileContext := &ProfileContext{
			Profile:        profile,
			Namespace:      profile.Global.Namespace,
			Install:        profile.Global.Install,
			HigressVersion: profile.HigressVersion,
			SourceType:     "file",
			PathOrName:     fileName,
		}
		profileContexts = append(profileContexts, profileContext)
	}
	return profileContexts, nil
}

func (f *FileDirProfileStore) Delete(profile *helm.Profile) (string, error) {
	namespace := profile.Global.Namespace
	install := profile.Global.Install
	var profileName = ""
	if install == helm.InstallK8s || install == helm.InstallLocalK8s {
		profileName = filepath.Join(f.profilesPath, fmt.Sprintf("%s-%s.yaml", ProfileFilePrefix, namespace))
	} else {
		profileName = filepath.Join(f.profilesPath, fmt.Sprintf("%s-%s.yaml", ProfileFilePrefix, install))
	}
	if err := os.Remove(profileName); err != nil {
		return "", err
	}
	return profileName, nil
}

func NewFileDirProfileStore(profilesPath string) (ProfileStore, error) {
	if _, err := os.Stat(profilesPath); os.IsNotExist(err) {
		if err = os.MkdirAll(profilesPath, os.ModePerm); err != nil {
			return nil, err
		}
	}

	profileStore := &FileDirProfileStore{
		profilesPath: profilesPath,
	}
	return profileStore, nil
}

type ConfigmapProfileStore struct {
	kubeCli kubernetes.CLIClient
}

func (c *ConfigmapProfileStore) Save(profile *helm.Profile) (string, error) {
	bytes, err := json.Marshal(profile)
	jsonProfile := ""
	if err == nil {
		jsonProfile = string(bytes)
	}
	annotation := make(map[string]string, 0)
	annotation[ProfileConfigmapAnnotation] = jsonProfile
	configmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   profile.Global.Namespace,
			Name:        ProfileConfigmapName,
			Annotations: annotation,
		},
	}
	configmap.Data = make(map[string]string, 0)
	configmap.Data[ProfileConfigmapKey] = util.ToYAML(profile)
	name := fmt.Sprintf("%s/%s", profile.Global.Namespace, ProfileConfigmapName)
	if err := c.applyConfigmap(configmap); err != nil {
		return "", err
	}
	return name, nil
}

func (c *ConfigmapProfileStore) List() ([]*ProfileContext, error) {
	profileContexts := make([]*ProfileContext, 0)
	configmapList, err := c.listConfigmaps(ProfileConfigmapName, "", 100)
	if err != nil {
		return profileContexts, err
	}
	for _, configmap := range configmapList.Items {
		if data, ok := configmap.Data[ProfileConfigmapKey]; ok {
			profile, err := helm.UnmarshalProfile(data)
			if err != nil {
				continue
			}
			profileContext := &ProfileContext{
				Profile:        profile,
				Namespace:      profile.Global.Namespace,
				Install:        profile.Global.Install,
				HigressVersion: profile.HigressVersion,
				SourceType:     "configmap",
				PathOrName:     fmt.Sprintf("%s/%s", profile.Global.Namespace, configmap.Name),
			}
			profileContexts = append(profileContexts, profileContext)
		}
	}
	return profileContexts, nil
}

func (c *ConfigmapProfileStore) Delete(profile *helm.Profile) (string, error) {
	configmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: profile.Global.Namespace,
			Name:      ProfileConfigmapName,
		},
	}
	name := fmt.Sprintf("%s/%s", profile.Global.Namespace, ProfileConfigmapName)
	if err := c.deleteConfigmap(configmap); err != nil {
		return "", err
	}
	return name, nil
}

func (c *ConfigmapProfileStore) listConfigmaps(name string, namespace string, size int64) (*corev1.ConfigMapList, error) {
	var result *corev1.ConfigMapList
	var err error
	if len(namespace) == 0 {
		result, err = c.kubeCli.KubernetesInterface().CoreV1().ConfigMaps("").List(context.Background(), metav1.ListOptions{Limit: size, FieldSelector: fmt.Sprintf("metadata.name=%s", name)})
	} else {
		result, err = c.kubeCli.KubernetesInterface().CoreV1().ConfigMaps(namespace).List(context.Background(), metav1.ListOptions{Limit: size, FieldSelector: fmt.Sprintf("metadata.name=%s", name)})
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (c *ConfigmapProfileStore) applyConfigmap(configmap *corev1.ConfigMap) error {
	_, err := c.kubeCli.KubernetesInterface().CoreV1().ConfigMaps(configmap.Namespace).Get(context.Background(), configmap.Name, metav1.GetOptions{})
	if err != nil && errors.IsNotFound(err) {
		_, err = c.kubeCli.KubernetesInterface().CoreV1().ConfigMaps(configmap.Namespace).Create(context.Background(), configmap, metav1.CreateOptions{})
		return err
	} else if err != nil {
		return err
	} else {
		_, err = c.kubeCli.KubernetesInterface().CoreV1().ConfigMaps(configmap.Namespace).Update(context.Background(), configmap, metav1.UpdateOptions{})
		return err
	}
}

func (c *ConfigmapProfileStore) deleteConfigmap(configmap *corev1.ConfigMap) error {
	err := c.kubeCli.KubernetesInterface().CoreV1().ConfigMaps(configmap.Namespace).Delete(context.Background(), configmap.Name, metav1.DeleteOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func NewConfigmapProfileStore(kubeCli kubernetes.CLIClient) (ProfileStore, error) {
	profileStore := &ConfigmapProfileStore{
		kubeCli: kubeCli,
	}
	return profileStore, nil
}
