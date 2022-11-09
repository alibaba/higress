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

package annotations

import (
	"errors"
	"strconv"
	"strings"
)

const (
	// DefaultAnnotationsPrefix defines the common prefix used in the nginx ingress controller
	DefaultAnnotationsPrefix = "nginx.ingress.kubernetes.io"

	// MSEAnnotationsPrefix defines the common prefix used in the mse ingress controller
	MSEAnnotationsPrefix = "mse.ingress.kubernetes.io"
)

var (
	// ErrMissingAnnotations the ingress rule does not contain annotations
	// This is an error only when annotations are being parsed
	ErrMissingAnnotations = errors.New("ingress rule without annotations")

	// ErrInvalidAnnotationName the ingress rule does contain an invalid
	// annotation name
	ErrInvalidAnnotationName = errors.New("invalid annotation name")

	// ErrInvalidAnnotationValue the ingress rule does contain an invalid
	// annotation value
	ErrInvalidAnnotationValue = errors.New("invalid annotation value")
)

// IsMissingAnnotations checks if the error is an error which
// indicates the ingress does not contain annotations
func IsMissingAnnotations(e error) bool {
	return e == ErrMissingAnnotations
}

type Annotations map[string]string

func (a Annotations) ParseBool(key string) (bool, error) {
	if len(a) == 0 {
		return false, ErrMissingAnnotations
	}

	val, ok := a[buildNginxAnnotationKey(key)]
	if ok {
		b, err := strconv.ParseBool(val)
		if err != nil {
			return false, ErrInvalidAnnotationValue
		}
		return b, nil
	}

	return false, ErrMissingAnnotations
}

func (a Annotations) ParseBoolForMSE(key string) (bool, error) {
	if len(a) == 0 {
		return false, ErrMissingAnnotations
	}

	val, ok := a[buildMSEAnnotationKey(key)]
	if ok {
		b, err := strconv.ParseBool(val)
		if err != nil {
			return false, ErrInvalidAnnotationValue
		}
		return b, nil
	}

	return false, ErrMissingAnnotations
}

func (a Annotations) ParseBoolASAP(key string) (bool, error) {
	if result, err := a.ParseBool(key); err == nil {
		return result, nil
	}
	return a.ParseBoolForMSE(key)
}

func (a Annotations) ParseString(key string) (string, error) {
	if len(a) == 0 {
		return "", ErrMissingAnnotations
	}

	val, ok := a[buildNginxAnnotationKey(key)]
	if ok {
		s := normalizeString(val)
		if s == "" {
			return "", ErrInvalidAnnotationValue
		}
		return s, nil
	}

	return "", ErrMissingAnnotations
}

func (a Annotations) ParseStringForMSE(key string) (string, error) {
	if len(a) == 0 {
		return "", ErrMissingAnnotations
	}

	val, ok := a[buildMSEAnnotationKey(key)]
	if ok {
		s := normalizeString(val)
		if s == "" {
			return "", ErrInvalidAnnotationValue
		}
		return s, nil
	}

	return "", ErrMissingAnnotations
}

// ParseStringASAP will first extra config from nginx annotation, then will
// try to extra config from mse annotation if the first step fails.
func (a Annotations) ParseStringASAP(key string) (string, error) {
	if result, err := a.ParseString(key); err == nil {
		return result, nil
	}
	return a.ParseStringForMSE(key)
}

func (a Annotations) ParseInt(key string) (int, error) {
	if len(a) == 0 {
		return 0, ErrMissingAnnotations
	}

	val, ok := a[buildNginxAnnotationKey(key)]
	if ok {
		i, err := strconv.Atoi(val)
		if err != nil {
			return 0, ErrInvalidAnnotationValue
		}
		return i, nil
	}
	return 0, ErrMissingAnnotations
}

func (a Annotations) ParseIntForMSE(key string) (int, error) {
	if len(a) == 0 {
		return 0, ErrMissingAnnotations
	}

	val, ok := a[buildMSEAnnotationKey(key)]
	if ok {
		i, err := strconv.Atoi(val)
		if err != nil {
			return 0, ErrInvalidAnnotationValue
		}
		return i, nil
	}
	return 0, ErrMissingAnnotations
}

func (a Annotations) ParseInt32(key string) (int32, error) {
	if len(a) == 0 {
		return 0, ErrMissingAnnotations
	}

	val, ok := a[buildNginxAnnotationKey(key)]
	if ok {
		i, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return 0, ErrInvalidAnnotationValue
		}
		return int32(i), nil
	}
	return 0, ErrMissingAnnotations
}

func (a Annotations) ParseInt32ForMSE(key string) (int32, error) {
	if len(a) == 0 {
		return 0, ErrMissingAnnotations
	}

	val, ok := a[buildMSEAnnotationKey(key)]
	if ok {
		i, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return 0, ErrInvalidAnnotationValue
		}
		return int32(i), nil
	}
	return 0, ErrMissingAnnotations
}

func (a Annotations) ParseUint32ForMSE(key string) (uint32, error) {
	if len(a) == 0 {
		return 0, ErrMissingAnnotations
	}

	val, ok := a[buildMSEAnnotationKey(key)]
	if ok {
		i, err := strconv.ParseUint(val, 10, 32)
		if err != nil {
			return 0, ErrInvalidAnnotationValue
		}
		return uint32(i), nil
	}
	return 0, ErrMissingAnnotations
}

func (a Annotations) ParseIntASAP(key string) (int, error) {
	if result, err := a.ParseInt(key); err == nil {
		return result, nil
	}
	return a.ParseIntForMSE(key)
}

func (a Annotations) ParseInt32ASAP(key string) (int32, error) {
	if result, err := a.ParseInt32(key); err == nil {
		return result, nil
	}
	return a.ParseInt32ForMSE(key)
}

func (a Annotations) Has(key string) bool {
	if len(a) == 0 {
		return false
	}

	_, exist := a[buildNginxAnnotationKey(key)]
	return exist
}

func (a Annotations) HasMSE(key string) bool {
	if len(a) == 0 {
		return false
	}

	_, exist := a[buildMSEAnnotationKey(key)]
	return exist
}

func (a Annotations) HasASAP(key string) bool {
	if a.Has(key) {
		return true
	}
	return a.HasMSE(key)
}

func buildNginxAnnotationKey(key string) string {
	return DefaultAnnotationsPrefix + "/" + key
}

func buildMSEAnnotationKey(key string) string {
	return MSEAnnotationsPrefix + "/" + key
}

func normalizeString(input string) string {
	var trimmedContent []string
	for _, line := range strings.Split(input, "\n") {
		trimmedContent = append(trimmedContent, strings.TrimSpace(line))
	}

	return strings.Join(trimmedContent, "\n")
}
