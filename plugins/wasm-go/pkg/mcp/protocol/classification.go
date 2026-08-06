// Copyright (c) 2022 Alibaba Group Holding Ltd.
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

package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
)

type classificationPath uint8

const (
	classificationOther classificationPath = iota
	classificationRoot
	classificationParams
	classificationMeta
)

// hasStructuredModernMetadata scans JSON tokens only up to the legacy ingress
// cap and recognizes reserved keys exclusively as direct params._meta members.
// Marker text inside argument strings or unrelated objects is never identity.
func hasStructuredModernMetadata(body []byte) bool {
	if len(body) > int(LegacyMaxBodyBytes) {
		body = body[:LegacyMaxBodyBytes]
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	found, _ := scanClassificationValue(decoder, classificationRoot)
	return found
}

func scanClassificationValue(decoder *json.Decoder, path classificationPath) (bool, error) {
	token, err := decoder.Token()
	if err != nil {
		return false, err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return false, nil
	}
	switch delimiter {
	case '{':
		for decoder.More() {
			keyToken, tokenErr := decoder.Token()
			if tokenErr != nil {
				return false, tokenErr
			}
			key, ok := keyToken.(string)
			if !ok {
				return false, errors.New("object key is not a string")
			}
			if path == classificationMeta && isModernMetadataKey(key) {
				return true, nil
			}
			childPath := classificationOther
			switch {
			case path == classificationRoot && key == "params":
				childPath = classificationParams
			case path == classificationParams && key == "_meta":
				childPath = classificationMeta
			}
			found, childErr := scanClassificationValue(decoder, childPath)
			if found || childErr != nil {
				return found, childErr
			}
		}
		_, err = decoder.Token()
		return false, err
	case '[':
		for decoder.More() {
			found, childErr := scanClassificationValue(decoder, classificationOther)
			if found || childErr != nil {
				return found, childErr
			}
		}
		_, err = decoder.Token()
		return false, err
	default:
		return false, nil
	}
}

func isModernMetadataKey(key string) bool {
	return key == MetaProtocolVersion || key == MetaClientCapabilities || key == MetaClientInfo
}
