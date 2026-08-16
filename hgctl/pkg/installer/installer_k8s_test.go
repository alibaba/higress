package installer

import (
	"bytes"
	"errors"
	"testing"

	"github.com/alibaba/higress/hgctl/pkg/helm"
)

type failingHelmOwnershipChecker struct{ err error }

func (f failingHelmOwnershipChecker) IsHigressInstalled() (bool, error) { return false, f.err }

type recordingComponent struct{ run bool }

func (c *recordingComponent) ComponentName() ComponentName    { return Higress }
func (c *recordingComponent) Namespace() string               { return "higress-system" }
func (c *recordingComponent) Enabled() bool                   { return true }
func (c *recordingComponent) Run() error                      { c.run = true; return nil }
func (c *recordingComponent) RenderManifest() (string, error) { return "", nil }

type recordingProfileStore struct{ saved bool }

func (s *recordingProfileStore) Save(*helm.Profile) (string, error)   { s.saved = true; return "", nil }
func (s *recordingProfileStore) List() ([]*ProfileContext, error)     { return nil, nil }
func (s *recordingProfileStore) Delete(*helm.Profile) (string, error) { return "", nil }

func TestK8sInstallerOwnershipCheckFailureStopsMutation(t *testing.T) {
	for _, upgrade := range []bool{false, true} {
		t.Run(map[bool]string{false: "install", true: "upgrade"}[upgrade], func(t *testing.T) {
			component := &recordingComponent{}
			store := &recordingProfileStore{}
			installer := &K8sInstaller{
				profile:      &helm.Profile{Global: helm.ProfileGlobal{Install: helm.InstallK8s}},
				writer:       &bytes.Buffer{},
				components:   map[ComponentName]Component{Higress: component},
				profileStore: store,
				helmChecker:  failingHelmOwnershipChecker{err: errors.New("helm unavailable")},
			}

			var err error
			if upgrade {
				err = installer.Upgrade()
			} else {
				err = installer.Install()
			}
			if err == nil {
				t.Fatal("Install() error = nil, want ownership-check failure")
			}
			if component.run {
				t.Fatal("component Run() was called after ownership-check failure")
			}
			if store.saved {
				t.Fatal("ProfileStore.Save() was called after ownership-check failure")
			}
		})
	}
}
