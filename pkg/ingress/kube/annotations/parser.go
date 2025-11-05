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

	// HigressAnnotationsPrefix defines the common prefix used in the higress ingress controller
	HigressAnnotationsPrefix = "higress.io"
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

func (a Annotations) ParseBoolForHigress(key string) (bool, error) {
	if len(a) == 0 {
		return false, ErrMissingAnnotations
	}

	val, ok := a[buildHigressAnnotationKey(key)]
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
	return a.ParseBoolForHigress(key)
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

func (a Annotations) ParseStringForHigress(key string) (string, error) {
	if len(a) == 0 {
		return "", ErrMissingAnnotations
	}

	val, ok := a[buildHigressAnnotationKey(key)]
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
// try to extra config from Higress annotation if the first step fails.
func (a Annotations) ParseStringASAP(key string) (string, error) {
	if result, err := a.ParseString(key); err == nil {
		return result, nil
	}
	return a.ParseStringForHigress(key)
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

func (a Annotations) ParseIntForHigress(key string) (int, error) {
	if len(a) == 0 {
		return 0, ErrMissingAnnotations
	}

	val, ok := a[buildHigressAnnotationKey(key)]
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

func (a Annotations) ParseInt32ForHigress(key string) (int32, error) {
	if len(a) == 0 {
		return 0, ErrMissingAnnotations
	}

	val, ok := a[buildHigressAnnotationKey(key)]
	if ok {
		i, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return 0, ErrInvalidAnnotationValue
		}
		return int32(i), nil
	}
	return 0, ErrMissingAnnotations
}

func (a Annotations) ParseUint32ForHigress(key string) (uint32, error) {
	if len(a) == 0 {
		return 0, ErrMissingAnnotations
	}

	val, ok := a[buildHigressAnnotationKey(key)]
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
	return a.ParseIntForHigress(key)
}

func (a Annotations) ParseInt32ASAP(key string) (int32, error) {
	if result, err := a.ParseInt32(key); err == nil {
		return result, nil
	}
	return a.ParseInt32ForHigress(key)
}

func (a Annotations) Has(key string) bool {
	if len(a) == 0 {
		return false
	}

	_, exist := a[buildNginxAnnotationKey(key)]
	return exist
}

func (a Annotations) HasHigress(key string) bool {
	if len(a) == 0 {
		return false
	}

	_, exist := a[buildHigressAnnotationKey(key)]
	return exist
}

func (a Annotations) HasASAP(key string) bool {
	if a.Has(key) {
		return true
	}
	return a.HasHigress(key)
}

func buildNginxAnnotationKey(key string) string {
	return DefaultAnnotationsPrefix + "/" + key
}

func buildHigressAnnotationKey(key string) string {
	return HigressAnnotationsPrefix + "/" + key
}

func normalizeString(input string) string {
	var trimmedContent []string
	for _, line := range strings.Split(input, "\n") {
		trimmedContent = append(trimmedContent, strings.TrimSpace(line))
	}

	return strings.Join(trimmedContent, "\n")
}
