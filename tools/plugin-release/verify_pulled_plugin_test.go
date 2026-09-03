// Copyright 2026 Higress Authors
// Licensed under the Apache License, Version 2.0.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	pulledSource  = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	pulledCreated = "2026-01-02T03:04:05Z"
	pulledInput   = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	pulledVer     = "1.2.3"
)

func TestVerifyPulledPluginAcceptsStrictTwoLayerProxyWasm(t *testing.T) {
	manifest, config, wasm, digest := writePulledPluginFixture(t)
	if err := verifyPulledPlugin(manifest, config, wasm, digest, pulledSource, pulledCreated, pulledVer, pulledInput); err != nil {
		t.Fatalf("valid pulled plugin was rejected: %v", err)
	}
}

func TestVerifyPulledPluginRejectsBadManifestBlobsAndWasm(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(t *testing.T, manifestPath, configPath, wasmPath string)
		want   string
	}{
		{
			name: "incomplete-manifest",
			mutate: func(t *testing.T, manifestPath, _, _ string) {
				mutatePulledManifest(t, manifestPath, func(manifest map[string]any) {
					manifest["layers"] = manifest["layers"].([]any)[:1]
				})
			},
			want: "exactly two layers",
		},
		{
			name: "wrong-layer-order",
			mutate: func(t *testing.T, manifestPath, _, _ string) {
				mutatePulledManifest(t, manifestPath, func(manifest map[string]any) {
					layers := manifest["layers"].([]any)
					layers[0], layers[1] = layers[1], layers[0]
				})
			},
			want: "layer 0",
		},
		{
			name: "provenance-mismatch",
			mutate: func(t *testing.T, manifestPath, _, _ string) {
				mutatePulledManifest(t, manifestPath, func(manifest map[string]any) {
					manifest["annotations"].(map[string]any)["org.opencontainers.image.version"] = "9.9.9"
				})
			},
			want: "manifest annotations",
		},
		{
			name: "unexpected-manifest-annotation",
			mutate: func(t *testing.T, manifestPath, _, _ string) {
				mutatePulledManifest(t, manifestPath, func(manifest map[string]any) {
					manifest["annotations"].(map[string]any)["unexpected"] = "value"
				})
			},
			want: "manifest annotations",
		},
		{
			name: "oci-v1.1-artifact-type",
			mutate: func(t *testing.T, manifestPath, _, _ string) {
				mutatePulledManifest(t, manifestPath, func(manifest map[string]any) {
					manifest["artifactType"] = "application/vnd.unknown.artifact.v1"
				})
			},
			want: "canonical OCI v1.0",
		},
		{
			name: "noncanonical-oci-config",
			mutate: func(t *testing.T, manifestPath, _, _ string) {
				mutatePulledManifest(t, manifestPath, func(manifest map[string]any) {
					manifest["config"].(map[string]any)["mediaType"] = "application/vnd.oci.image.config.v1+json"
				})
			},
			want: "canonical empty JSON object",
		},
		{
			name: "unexpected-layer-descriptor-field",
			mutate: func(t *testing.T, manifestPath, _, _ string) {
				mutatePulledManifest(t, manifestPath, func(manifest map[string]any) {
					manifest["layers"].([]any)[1].(map[string]any)["artifactType"] = "application/wasm"
				})
			},
			want: "unknown field",
		},
		{
			name: "config-is-not-object",
			mutate: func(t *testing.T, manifestPath, configPath, _ string) {
				writePulledLayerAndDescriptor(t, manifestPath, configPath, []byte(`[]`), 0)
			},
			want: "JSON object",
		},
		{
			name: "config-is-not-canonical-empty-object",
			mutate: func(t *testing.T, manifestPath, configPath, _ string) {
				writePulledLayerAndDescriptor(t, manifestPath, configPath, []byte(`{ }`), 0)
			},
			want: "canonical empty JSON object",
		},
		{
			name: "config-digest-mismatch",
			mutate: func(t *testing.T, _, configPath, _ string) {
				if err := os.WriteFile(configPath, []byte(`[]`), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			want: "digest",
		},
		{
			name: "wasm-size-mismatch",
			mutate: func(t *testing.T, _, _, wasmPath string) {
				if err := os.WriteFile(wasmPath, []byte("different"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
			want: "size",
		},
		{
			name: "wasm-digest-mismatch",
			mutate: func(t *testing.T, _, _, wasmPath string) {
				wasm, err := os.ReadFile(wasmPath)
				if err != nil {
					t.Fatal(err)
				}
				wasm[len(wasm)-1] ^= 0x01
				if err := os.WriteFile(wasmPath, wasm, 0o600); err != nil {
					t.Fatal(err)
				}
			},
			want: "digest",
		},
		{
			name: "truncated-wasm-section",
			mutate: func(t *testing.T, manifestPath, _, wasmPath string) {
				writePulledLayerAndDescriptor(t, manifestPath, wasmPath, append(proxyWasmFixture(), 7, 10, 1), 1)
			},
			want: "truncated",
		},
		{
			name: "missing-required-export",
			mutate: func(t *testing.T, manifestPath, _, wasmPath string) {
				writePulledLayerAndDescriptor(t, manifestPath, wasmPath, proxyWasmWithExports("proxy_on_vm_start", "proxy_abi_version_0_2_1"), 1)
			},
			want: "proxy_on_configure",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			manifest, config, wasm, _ := writePulledPluginFixture(t)
			tc.mutate(t, manifest, config, wasm)
			manifestBytes, err := os.ReadFile(manifest)
			if err != nil {
				t.Fatal(err)
			}
			digest := digestBytes(manifestBytes)
			err = verifyPulledPlugin(manifest, config, wasm, digest, pulledSource, pulledCreated, pulledVer, pulledInput)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("invalid pulled artifact was accepted or returned the wrong error: %v", err)
			}
		})
	}
}

func writePulledPluginFixture(t *testing.T) (manifestPath, configPath, wasmPath, digest string) {
	t.Helper()
	root := t.TempDir()
	manifestPath = filepath.Join(root, "manifest.json")
	configPath = filepath.Join(root, "config.json")
	wasmPath = filepath.Join(root, "plugin.wasm")
	config := []byte(`{}`)
	wasm := proxyWasmFixture()
	if err := os.WriteFile(configPath, config, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(wasmPath, wasm, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := map[string]any{
		"schemaVersion": 2,
		"mediaType":     ociImageManifestMediaType,
		"config": map[string]any{
			"mediaType": "application/vnd.unknown.config.v1+json",
			"digest":    digestBytes([]byte(`{}`)),
			"size":      2,
		},
		"annotations": map[string]any{
			"org.opencontainers.image.created":  pulledCreated,
			"org.opencontainers.image.revision": pulledSource,
			"org.opencontainers.image.version":  pulledVer,
			"io.higress.plugin.input-hash":      pulledInput,
		},
		"layers": []any{
			pulledLayer(wasmConfigMediaType, "config.json", config),
			pulledLayer(wasmContentMediaType, "plugin.wasm", wasm),
		},
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return manifestPath, configPath, wasmPath, digestBytes(data)
}

func mutatePulledManifest(t *testing.T, path string, mutate func(map[string]any)) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	mutate(manifest)
	data, err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func writePulledLayerAndDescriptor(t *testing.T, manifestPath, layerPath string, data []byte, layerIndex int) {
	t.Helper()
	if err := os.WriteFile(layerPath, data, 0o600); err != nil {
		t.Fatal(err)
	}
	mutatePulledManifest(t, manifestPath, func(manifest map[string]any) {
		layer := manifest["layers"].([]any)[layerIndex].(map[string]any)
		layer["digest"] = digestBytes(data)
		layer["size"] = len(data)
	})
}

func pulledLayer(mediaType, title string, data []byte) map[string]any {
	return map[string]any{
		"mediaType": mediaType,
		"digest":    digestBytes(data),
		"size":      len(data),
		"annotations": map[string]any{
			"org.opencontainers.image.title": title,
		},
	}
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func proxyWasmFixture() []byte {
	return proxyWasmWithExports("proxy_on_vm_start", "proxy_on_configure", "proxy_abi_version_0_2_1")
}

func proxyWasmWithExports(names ...string) []byte {
	module := []byte{0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00}
	typePayload := []byte{0x01, 0x60, 0x00, 0x00}
	module = appendWasmSection(module, 1, typePayload)
	functionPayload := append(wasmVarUint(uint32(len(names))), make([]byte, len(names))...)
	module = appendWasmSection(module, 3, functionPayload)
	exportPayload := wasmVarUint(uint32(len(names)))
	for index, name := range names {
		exportPayload = append(exportPayload, wasmVarUint(uint32(len(name)))...)
		exportPayload = append(exportPayload, []byte(name)...)
		exportPayload = append(exportPayload, 0x00)
		exportPayload = append(exportPayload, wasmVarUint(uint32(index))...)
	}
	module = appendWasmSection(module, 7, exportPayload)
	codePayload := wasmVarUint(uint32(len(names)))
	for range names {
		codePayload = append(codePayload, 0x02, 0x00, 0x0b)
	}
	return appendWasmSection(module, 10, codePayload)
}

func appendWasmSection(module []byte, id byte, payload []byte) []byte {
	module = append(module, id)
	module = append(module, wasmVarUint(uint32(len(payload)))...)
	return append(module, payload...)
}

func wasmVarUint(value uint32) []byte {
	var out []byte
	for {
		current := byte(value & 0x7f)
		value >>= 7
		if value != 0 {
			current |= 0x80
		}
		out = append(out, current)
		if value == 0 {
			return out
		}
	}
}
