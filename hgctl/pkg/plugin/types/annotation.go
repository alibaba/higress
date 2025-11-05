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

package types

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type Annotation struct {
	Type     AnnotationType
	I18nType I18nType
	Text     string
}

type AnnotationType int

const (
	// Info
	ACategory AnnotationType = iota
	AName
	ATitle
	ADescription
	AIconUrl
	AVersion
	AContactName
	AContactUrl
	AContactEmail

	// Spec
	APhase
	APriority

	// Schema
	AScope
	AExample
	AEnd

	AUnknown
)

func str2AnnotationType(typ string) AnnotationType {
	switch strings.ToLower(typ) {
	case "@category":
		return ACategory
	case "@name":
		return AName
	case "@title":
		return ATitle
	case "@description":
		return ADescription
	case "@iconurl":
		return AIconUrl
	case "@version":
		return AVersion
	case "@contact.name":
		return AContactName
	case "@contact.url":
		return AContactUrl
	case "@contact.email":
		return AContactEmail
	case "@phase":
		return APhase
	case "@priority":
		return APriority
	case "@scope":
		return AScope
	case "@example":
		return AExample
	case "@end":
		return AEnd
	default:
		return AUnknown
	}
}

// GetAnnotations returns all annotations in the comment
func GetAnnotations(comment string) []Annotation {
	as := make([]Annotation, 0)
	cs := strings.Split(comment, "\n")
	for i := 0; i < len(cs); i++ {
		a, err := getAnnotationFrom(cs[i])
		if err != nil {
			continue
		}

		if a.Type == AExample {
			for j := i + 1; j < len(cs); j++ {
				if str2AnnotationType(strings.TrimSpace(cs[j])) == AEnd {
					break
				}
				if j == i+1 {
					a.Text = fmt.Sprintf("%s", cs[j])
				} else {
					a.Text = fmt.Sprintf("%s\n%s", a.Text, cs[j])
				}
			}
		}
		as = append(as, a)
	}
	return as
}

func getAnnotationFrom(c string) (Annotation, error) {
	// the annotation is like `@AnnotationType [I18nType] Text`

	c = strings.TrimSpace(c)
	if !strings.HasPrefix(c, "@") {
		return Annotation{}, errors.New("invalid annotation")
	}

	// first param: AnnotationType
	idx := strings.Index(c, " ")
	if idx == -1 && str2AnnotationType(c) == AUnknown { // only an invalid annotation type
		return Annotation{}, errors.New("invalid annotation")
	}
	// idx != -1 or type != unknown
	var typ AnnotationType
	if idx == -1 {
		typ = str2AnnotationType(c)
	} else {
		typ = str2AnnotationType(strings.TrimSpace(c[0:idx]))
	}
	c = strings.TrimSpace(c[idx+1:])
	a := Annotation{
		Type:     typ,
		I18nType: I18nDefault,
		Text:     c,
	}
	if a.Type != ATitle && a.Type != ADescription { // other annotation types do not define i18n
		a.I18nType = I18nUndefined
	}
	if idx == -1 && typ != AUnknown { // only a valid annotation type
		a.Text = ""
	}

	// second or/and third param: I18nType and Text
	idx = strings.Index(c, " ")
	if idx == -1 {
		return a, nil
	}
	i18n := str2I18nType(strings.TrimSpace(c[0:idx]))
	if i18n == I18nUnknown {
		return a, nil
	}
	a.I18nType = i18n
	a.Text = strings.TrimSpace(c[idx+1:])
	return a, nil
}
