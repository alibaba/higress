// Copyright 2026 Higress Authors
// Licensed under the Apache License, Version 2.0.

package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	ociImageManifestMediaType = "application/vnd.oci.image.manifest.v1+json"
	unknownConfigMediaType    = "application/vnd.unknown.config.v1+json"
	emptyObjectDigest         = "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"
	wasmConfigMediaType       = "application/vnd.module.wasm.config.v1+json"
	wasmContentMediaType      = "application/vnd.module.wasm.content.layer.v1+wasm"
)

type pulledDescriptor struct {
	MediaType   string            `json:"mediaType"`
	Digest      string            `json:"digest"`
	Size        int64             `json:"size"`
	Annotations map[string]string `json:"annotations"`
}

type pulledPluginManifest struct {
	SchemaVersion int                `json:"schemaVersion"`
	MediaType     string             `json:"mediaType"`
	Config        pulledDescriptor   `json:"config"`
	Layers        []pulledDescriptor `json:"layers"`
	Annotations   map[string]string  `json:"annotations"`
}

func commandVerifyPulledPlugin(args []string) error {
	fs := flag.NewFlagSet("verify-pulled-plugin", flag.ContinueOnError)
	manifestPath := fs.String("manifest", "", "raw digest-pinned OCI manifest")
	configPath := fs.String("config", "", "pulled config.json layer")
	wasmPath := fs.String("wasm", "", "pulled plugin.wasm layer")
	digest := fs.String("digest", "", "expected manifest digest")
	sourceCommit := fs.String("source-commit", "", "expected source revision")
	sourceCreated := fs.String("source-created", "", "expected RFC3339 source commit time")
	version := fs.String("version", "", "expected stable plugin version")
	inputHash := fs.String("input-hash", "", "expected plugin input hash")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *manifestPath == "" || *configPath == "" || *wasmPath == "" {
		return errors.New("--manifest, --config, and --wasm are required")
	}
	return verifyPulledPlugin(*manifestPath, *configPath, *wasmPath, *digest, *sourceCommit, *sourceCreated, *version, *inputHash)
}

func verifyPulledPlugin(manifestPath, configPath, wasmPath, expectedDigest, expectedSource, expectedCreated, expectedVersion, expectedInputHash string) error {
	if !digestPattern.MatchString(expectedDigest) {
		return errors.New("--digest must be a lowercase sha256 digest")
	}
	if !commitPattern.MatchString(expectedSource) {
		return errors.New("--source-commit must be a full lowercase Git commit")
	}
	if _, err := time.Parse(time.RFC3339, expectedCreated); err != nil {
		return errors.New("--source-created must be an RFC3339 timestamp")
	}
	parsedVersion, err := parseSemver(expectedVersion)
	if err != nil || parsedVersion.prerelease != "" {
		return errors.New("--version must be stable SemVer")
	}
	if !digestPattern.MatchString(expectedInputHash) {
		return errors.New("--input-hash must be a lowercase sha256 digest")
	}

	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read pulled manifest: %w", err)
	}
	manifestSum := sha256.Sum256(manifestBytes)
	if got := "sha256:" + hex.EncodeToString(manifestSum[:]); got != expectedDigest {
		return fmt.Errorf("pulled manifest digest %s does not match expected digest %s", got, expectedDigest)
	}
	var manifest pulledPluginManifest
	if err := decodeSingleJSON(manifestBytes, &manifest); err != nil {
		return fmt.Errorf("parse pulled manifest as a canonical OCI v1.0 schema 2 image manifest: %w", err)
	}
	if manifest.SchemaVersion != 2 || manifest.MediaType != ociImageManifestMediaType {
		return errors.New("pulled manifest must be a canonical OCI v1.0 schema 2 image manifest")
	}
	if manifest.Config.MediaType != unknownConfigMediaType || manifest.Config.Digest != emptyObjectDigest || manifest.Config.Size != 2 || len(manifest.Config.Annotations) != 0 {
		return errors.New("pulled manifest OCI config descriptor must identify the canonical empty JSON object")
	}
	if len(manifest.Annotations) != 4 ||
		manifest.Annotations["org.opencontainers.image.created"] != expectedCreated ||
		manifest.Annotations["org.opencontainers.image.revision"] != expectedSource ||
		manifest.Annotations["org.opencontainers.image.version"] != expectedVersion ||
		manifest.Annotations["io.higress.plugin.input-hash"] != expectedInputHash {
		return errors.New("pulled manifest annotations do not match the deterministic source creation time, revision, version, and input hash")
	}
	if len(manifest.Layers) != 2 {
		return fmt.Errorf("pulled manifest must contain exactly two layers, got %d", len(manifest.Layers))
	}
	configDescriptor, wasmDescriptor := manifest.Layers[0], manifest.Layers[1]
	if configDescriptor.MediaType != wasmConfigMediaType || len(configDescriptor.Annotations) != 1 || configDescriptor.Annotations["org.opencontainers.image.title"] != "config.json" {
		return errors.New("pulled manifest layer 0 must be the exactly titled config.json Wasm config layer")
	}
	if wasmDescriptor.MediaType != wasmContentMediaType || len(wasmDescriptor.Annotations) != 1 || wasmDescriptor.Annotations["org.opencontainers.image.title"] != "plugin.wasm" {
		return errors.New("pulled manifest layer 1 must be the exactly titled plugin.wasm Wasm content layer")
	}

	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read pulled config.json: %w", err)
	}
	if err := verifyPulledLayer("config.json", configBytes, configDescriptor); err != nil {
		return err
	}
	var configValue any
	if err := decodeSingleJSON(configBytes, &configValue); err != nil {
		return fmt.Errorf("pulled config.json is invalid JSON: %w", err)
	}
	if _, ok := configValue.(map[string]any); !ok {
		return errors.New("pulled config.json must contain a JSON object")
	}
	if !bytes.Equal(configBytes, []byte(`{}`)) {
		return errors.New("pulled config.json must be the canonical empty JSON object")
	}

	wasmBytes, err := os.ReadFile(wasmPath)
	if err != nil {
		return fmt.Errorf("read pulled plugin.wasm: %w", err)
	}
	if err := verifyPulledLayer("plugin.wasm", wasmBytes, wasmDescriptor); err != nil {
		return err
	}
	if err := validateProxyWasmModule(wasmBytes); err != nil {
		return fmt.Errorf("pulled plugin.wasm: %w", err)
	}
	return nil
}

func decodeSingleJSON(data []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("multiple JSON values")
		}
		return err
	}
	return nil
}

func verifyPulledLayer(name string, data []byte, descriptor pulledDescriptor) error {
	if !digestPattern.MatchString(descriptor.Digest) {
		return fmt.Errorf("%s layer descriptor has an invalid digest", name)
	}
	if descriptor.Size < 0 || int64(len(data)) != descriptor.Size {
		return fmt.Errorf("%s size %d does not match layer descriptor size %d", name, len(data), descriptor.Size)
	}
	sum := sha256.Sum256(data)
	if got := "sha256:" + hex.EncodeToString(sum[:]); got != descriptor.Digest {
		return fmt.Errorf("%s digest %s does not match layer descriptor digest %s", name, got, descriptor.Digest)
	}
	return nil
}

func validateProxyWasmModule(data []byte) error {
	if len(data) < 8 || !bytes.Equal(data[:4], []byte{0x00, 0x61, 0x73, 0x6d}) {
		return errors.New("invalid WebAssembly magic")
	}
	if !bytes.Equal(data[4:8], []byte{0x01, 0x00, 0x00, 0x00}) {
		return errors.New("unsupported WebAssembly binary version")
	}
	sectionRank := map[byte]int{1: 1, 2: 2, 3: 3, 4: 4, 5: 5, 13: 6, 6: 7, 7: 8, 8: 9, 9: 10, 12: 11, 10: 12, 11: 13}
	seen := map[byte]bool{}
	lastRank := 0
	exportsSeen := false
	required := map[string]bool{"proxy_on_vm_start": false, "proxy_on_configure": false}
	abiMarker := false
	for offset := 8; offset < len(data); {
		sectionID := data[offset]
		offset++
		sectionSize, consumed, err := readWasmVarUint32(data[offset:])
		if err != nil {
			return fmt.Errorf("section %d size: %w", sectionID, err)
		}
		offset += consumed
		if uint64(sectionSize) > uint64(len(data)-offset) {
			return fmt.Errorf("section %d is truncated", sectionID)
		}
		end := offset + int(sectionSize)
		if sectionID != 0 {
			rank, ok := sectionRank[sectionID]
			if !ok {
				return fmt.Errorf("unsupported WebAssembly section id %d", sectionID)
			}
			if seen[sectionID] || rank <= lastRank {
				return fmt.Errorf("WebAssembly section %d is duplicate or out of order", sectionID)
			}
			seen[sectionID], lastRank = true, rank
		}
		if sectionID == 7 {
			exportsSeen = true
			if err := parseProxyWasmExports(data[offset:end], required, &abiMarker); err != nil {
				return err
			}
		}
		offset = end
	}
	if !exportsSeen {
		return errors.New("WebAssembly export section is missing")
	}
	for name, present := range required {
		if !present {
			return fmt.Errorf("required function export %q is missing", name)
		}
	}
	if !abiMarker {
		return errors.New("required proxy_abi_version_0_2_* function export is missing")
	}
	return nil
}

func parseProxyWasmExports(payload []byte, required map[string]bool, abiMarker *bool) error {
	count, consumed, err := readWasmVarUint32(payload)
	if err != nil {
		return fmt.Errorf("export count: %w", err)
	}
	offset := consumed
	seenNames := map[string]bool{}
	for i := uint32(0); i < count; i++ {
		nameLength, n, err := readWasmVarUint32(payload[offset:])
		if err != nil {
			return fmt.Errorf("export %d name length: %w", i, err)
		}
		offset += n
		if uint64(nameLength) > uint64(len(payload)-offset) {
			return fmt.Errorf("export %d name is truncated", i)
		}
		nameBytes := payload[offset : offset+int(nameLength)]
		if !utf8.Valid(nameBytes) {
			return fmt.Errorf("export %d name is not UTF-8", i)
		}
		name := string(nameBytes)
		offset += int(nameLength)
		if seenNames[name] {
			return fmt.Errorf("duplicate WebAssembly export %q", name)
		}
		seenNames[name] = true
		if offset >= len(payload) {
			return fmt.Errorf("export %q descriptor is truncated", name)
		}
		kind := payload[offset]
		offset++
		if kind > 4 {
			return fmt.Errorf("export %q has invalid kind %d", name, kind)
		}
		_, n, err = readWasmVarUint32(payload[offset:])
		if err != nil {
			return fmt.Errorf("export %q index: %w", name, err)
		}
		offset += n
		if _, wanted := required[name]; wanted && kind == 0 {
			required[name] = true
		}
		if strings.HasPrefix(name, "proxy_abi_version_0_2_") && len(name) > len("proxy_abi_version_0_2_") && kind == 0 {
			*abiMarker = true
		}
	}
	if offset != len(payload) {
		return errors.New("WebAssembly export section has trailing bytes")
	}
	return nil
}

func readWasmVarUint32(data []byte) (uint32, int, error) {
	var value uint32
	for i := 0; i < 5; i++ {
		if i >= len(data) {
			return 0, 0, errors.New("truncated varuint32")
		}
		b := data[i]
		if i == 4 && b&0xf0 != 0 {
			return 0, 0, errors.New("varuint32 overflow")
		}
		value |= uint32(b&0x7f) << (7 * i)
		if b&0x80 == 0 {
			return value, i + 1, nil
		}
	}
	return 0, 0, errors.New("varuint32 is too long")
}
