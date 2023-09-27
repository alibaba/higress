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

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

var (
	errKeyNotFound               = errors.New("key is not found")
	errEmptyBody                 = errors.New("body is empty")
	errInvalidKeyFmt             = "key %q is invalid"
	errInvalidValueFmt           = "value %q is invalid"
	errInvalidValueCorrespKeyFmt = "value corresponding to key %q is invalid"
	errInvalidIdxFmt             = "idx %d must be less than max %d"
	errInvalidFieldTypeFmt       = "invalid field type: %v"
	errSetKeyValueFmt            = "failed to set key-value %v:%v"
)

// 查询指定 kv，返回 value 及其父节点
func lookup(data map[string]interface{}, dotsInKeys bool, key string) (interface{}, interface{}, error) {
	if data == nil {
		data = make(map[string]interface{})
	}

	keys := []string{key}
	if !dotsInKeys {
		keys = strings.Split(key, ".")
	}

	var par, cur interface{} = data, nil
	var parV, curV reflect.Value
	for i, k := range keys {
		parV = reflect.ValueOf(par) // par 取值为 data 或 curV.Interface，必然有效，因此不需判断 parV.IsValid()
		keyV := reflect.ValueOf(k)
		if !keyV.IsValid() {
			return nil, par, errors.Errorf(errInvalidKeyFmt, k)
		}
		switch parV.Kind() {
		case reflect.Map:
			curV = parV.MapIndex(keyV)

		case reflect.Slice, reflect.Array, reflect.String:
			ii, err := strconv.ParseInt(k, 10, 64)
			if err != nil {
				return nil, par, errors.Wrap(err, fmt.Sprintf(errInvalidKeyFmt, k))
			}
			idx := int(ii)
			if idx >= parV.Len() {
				return nil, par, errors.Errorf(errInvalidIdxFmt, idx, parV.Len())
			}
			curV = parV.Index(idx)

		default:
			return nil, par, errors.Errorf(errInvalidFieldTypeFmt, parV.Kind())
		}

		if !curV.IsValid() {
			return nil, par, errKeyNotFound
		}
		cur = curV.Interface()

		if i == len(keys)-1 { // 最后一个，par 不再前进
			break
		}
		par = cur
	}

	return cur, par, nil
}

// 设置 kv，若指定 kv 存在则覆盖
func set(data map[string]interface{}, dotsInKeys bool, key string, value interface{}) error {
	cur, par, err := lookup(data, dotsInKeys, key)
	if err != nil && !errors.Is(err, errKeyNotFound) {
		return err
	}

	keys := []string{key}
	if !dotsInKeys {
		keys = strings.Split(key, ".")
		key = keys[len(keys)-1]
	}

	var parV reflect.Value
	if cur != nil { // kv already exists
		parV = reflect.ValueOf(par)
	} else {
		for i := range keys {
			// TODO(WeixinX): 重复访问前缀字段，可考虑优化，但字段嵌套深度不大？
			curPath := strings.Join(keys[:i+1], ".")
			cur, par, err = lookup(data, dotsInKeys, curPath)
			if err != nil && !errors.Is(err, errKeyNotFound) {
				return err
			}
			if cur != nil {
				continue
			}

			parV = reflect.ValueOf(par)
			if parV.Kind() != reflect.Map {
				return errors.Errorf(errInvalidFieldTypeFmt, parV.Kind())
			}
			for j, k := range keys[i:] {
				if j == len(keys[i:])-1 { // 最后一个，不需要再创建 map
					break
				}

				curV := reflect.ValueOf(make(map[string]interface{}))
				KeyV := reflect.ValueOf(k)
				if !KeyV.IsValid() {
					return errors.Errorf(errInvalidKeyFmt, k)
				}
				parV.SetMapIndex(KeyV, curV)
				parV = curV
				par = curV.Interface()
			}
		}
	}

	if parV.Kind() != reflect.Map {
		return errors.Errorf(errInvalidFieldTypeFmt, parV.Kind())
	}
	keyV := reflect.ValueOf(key)
	if !keyV.IsValid() {
		return errors.Errorf(errInvalidKeyFmt, key)
	}
	valueV := reflect.ValueOf(value)
	if !valueV.IsValid() {
		return errors.Errorf(errInvalidValueFmt, value)
	}
	parV.SetMapIndex(keyV, valueV)

	return nil
}

// 删除指定 kv
func remove(data map[string]interface{}, dotsInKeys bool, key string) error {
	_, par, err := lookup(data, dotsInKeys, key)
	if err != nil {
		if errors.Is(err, errKeyNotFound) {
			return nil
		}
		return err
	}

	parV := reflect.ValueOf(par)
	if parV.Kind() != reflect.Map {
		return fmt.Errorf(errInvalidFieldTypeFmt, parV.Kind())
	}

	if !dotsInKeys {
		keys := strings.Split(key, ".")
		key = keys[len(keys)-1]
	}

	keyV := reflect.ValueOf(key)
	if !keyV.IsValid() {
		return errors.Errorf(errInvalidKeyFmt, keyV)
	}
	parV.SetMapIndex(keyV, reflect.Value{}) // delete

	return nil
}

// 将指定 key 改名为 newKey，若不存在则无操作
func rename(data map[string]interface{}, dotsInKeys bool, key, newKey string) error {
	_, par, err := lookup(data, dotsInKeys, key)
	if err != nil {
		if errors.Is(err, errKeyNotFound) {
			return nil
		}
		return err
	}

	parV := reflect.ValueOf(par)
	if parV.Kind() != reflect.Map {
		return fmt.Errorf(errInvalidFieldTypeFmt, parV.Kind())
	}

	if !dotsInKeys {
		keys := strings.Split(key, ".")
		key = keys[len(keys)-1]
	}

	oldKeyV := reflect.ValueOf(key)
	if !oldKeyV.IsValid() {
		return errors.Errorf(errInvalidKeyFmt, oldKeyV)
	}
	newKeyV := reflect.ValueOf(newKey)
	if !newKeyV.IsValid() {
		return errors.Errorf(errInvalidKeyFmt, newKeyV)
	}
	valueV := parV.MapIndex(oldKeyV)
	if !valueV.IsValid() {
		return errors.Errorf(errInvalidValueCorrespKeyFmt, oldKeyV)
	}
	err = set(data, dotsInKeys, newKey, valueV.Interface())
	if err != nil {
		return errors.Wrapf(err, fmt.Sprintf(errSetKeyValueFmt, newKey, valueV.Interface()))
	}
	parV.SetMapIndex(oldKeyV, reflect.Value{}) // delete

	return nil
}

// 替换指定 key 的 value 为 newValue，若不存在则无操作
func replace(data map[string]interface{}, dotsInKeys bool, key string, newValue interface{}) error {
	_, _, err := lookup(data, dotsInKeys, key)
	if err != nil {
		if errors.Is(err, errKeyNotFound) {
			return nil
		}
		return err
	}

	err = set(data, dotsInKeys, key, newValue)
	if err != nil {
		return errors.Wrapf(err, fmt.Sprintf(errSetKeyValueFmt, key, newValue))
	}

	return nil
}

// 添加 kv，若指定 key 存在则无操作
func add(data map[string]interface{}, dotsInKeys bool, key string, value interface{}) error {
	cur, _, err := lookup(data, dotsInKeys, key)
	if err != nil && !errors.Is(err, errKeyNotFound) {
		return err
	}

	if cur != nil { // kv already exists
		return nil
	}

	err = set(data, dotsInKeys, key, value)
	if err != nil {
		return errors.Wrapf(err, fmt.Sprintf(errSetKeyValueFmt, key, value))
	}

	return nil
}

// 若指定 key 存在，则将它的值和 value 聚合成新的数组；若不存在则执行 add 操作
func append_(data map[string]interface{}, dotsInKeys bool, key string, value interface{}) error {
	cur, par, err := lookup(data, dotsInKeys, key)
	if err != nil && !errors.Is(err, errKeyNotFound) {
		return err
	}

	if cur == nil { // kv does not exist
		err = set(data, dotsInKeys, key, value)
		if err != nil {
			return errors.Wrapf(err, fmt.Sprintf(errSetKeyValueFmt, key, value))
		}
		return nil
	}

	// kv already exist
	if !dotsInKeys {
		keys := strings.Split(key, ".")
		key = keys[len(keys)-1]
	}

	if value == nil {
		return errors.Errorf(errInvalidFieldTypeFmt, value)
	}
	parV := reflect.ValueOf(par)
	if parV.Kind() != reflect.Map {
		return errors.Errorf(errInvalidFieldTypeFmt, parV.Kind())
	}

	// cur != nil && value != nil && par != nil && parV.Kind() == reflect.Map
	keyV := reflect.ValueOf(key)
	if !keyV.IsValid() {
		return errors.Errorf(errInvalidKeyFmt, key)
	}
	curV := reflect.ValueOf(cur)
	valueV := reflect.ValueOf(value)
	sliceT := reflect.TypeOf([]interface{}{})
	sliceV := reflect.MakeSlice(sliceT, 0, 0)
	if isSliceOrArray(curV) {
		if isSliceOrArray(valueV) {
			if valueV.Len() == 0 {
				return nil // par[key] = [cur...]
			}

			// value.Len() != 0
			if curV.Len() == 0 {
				parV.SetMapIndex(keyV, reflect.AppendSlice(sliceV, valueV)) // par[key] = [value...]
				return nil
			}

			// value.Len() != 0 && curV.Len() != 0
			if sliceElemTypeIsSame(curV, valueV) {
				sliceV = reflect.AppendSlice(sliceV, curV)
				parV.SetMapIndex(keyV, reflect.AppendSlice(sliceV, valueV)) // par[key] = [cur..., value...]
			}
			return nil
		}

		// !isSliceOrArray(valueV)
		if curV.Len() == 0 {
			parV.SetMapIndex(keyV, valueV) // par[key] = value
			return nil
		}

		//  !isSliceOrArray(valueV) && curV.Len() != 0
		if sliceElemTypeIsSameToVal(curV, valueV) {
			sliceV = reflect.AppendSlice(sliceV, curV)
			parV.SetMapIndex(keyV, reflect.Append(sliceV, valueV)) // par[key] = [cur..., value]
		}

		return nil // par[key] = [cur...]
	}

	// !isSliceOrArray(curV)
	if isSliceOrArray(valueV) {
		if valueV.Len() == 0 || !sliceElemTypeIsSameToVal(valueV, curV) {
			return nil // par[key] = cur
		}

		// valueV.Index(0).Elem().Kind() == curV.Kind()
		sliceV = reflect.Append(sliceV, curV)
		parV.SetMapIndex(keyV, reflect.AppendSlice(sliceV, valueV)) // par[key] = [cur, value...]
		return nil
	}

	// !isSliceOrArray(curV) && !isSliceOrArray(valueV)
	if valTypeIsSame(curV, valueV) {
		parV.SetMapIndex(keyV, reflect.Append(sliceV, curV, valueV)) // par[key] = [cur, value]
	}

	return nil // par[key] = cur
}

// 若存在 kv 为 fromKey:fromValue，则将 fromValue 映射给 toKey 的值；若 fromKey 不存在则无操作
func map_(data map[string]interface{}, dotsInKeys bool, fromKey string, toKey string) error {
	cur, _, err := lookup(data, dotsInKeys, fromKey)
	if err != nil {
		if errors.Is(err, errKeyNotFound) {
			return nil
		}
		return err
	}

	var fromValue interface{}
	b, err := json.Marshal(cur)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &fromValue)
	if err != nil {
		return err
	}

	err = set(data, dotsInKeys, toKey, fromValue)
	if err != nil {
		return errors.Wrapf(err, fmt.Sprintf(errSetKeyValueFmt, toKey, fromValue))
	}

	return nil
}

// 若指定 kv 为 slice 或 array，则根据 strategy 去重；若不存在则无操作
func dedupe(data map[string]interface{}, dotsInKeys bool, key string, strategy string) error {
	cur, par, err := lookup(data, dotsInKeys, key)
	if err != nil {
		if errors.Is(err, errKeyNotFound) {
			return nil
		}
		return err
	}

	if !dotsInKeys {
		keys := strings.Split(key, ".")
		key = keys[len(keys)-1]
	}

	curV := reflect.ValueOf(cur)
	parV := reflect.ValueOf(par)
	if parV.Kind() != reflect.Map {
		return errors.Errorf(errInvalidFieldTypeFmt, parV.Kind())
	}
	keyV := reflect.ValueOf(key)
	if !keyV.IsValid() {
		return errors.Errorf(errInvalidKeyFmt, key)
	}

	var val interface{}
	if isSliceOrArray(curV) {
		if curV.Len() == 0 {
			return nil
		}
		switch strings.ToUpper(strategy) {
		case "RETAIN_UNIQUE":
			uniMap := make(map[interface{}]struct{})
			uniques := make([]interface{}, 0)
			for i := 0; i < curV.Len(); i++ {
				v := curV.Index(i).Interface()
				if _, ok := uniMap[v]; !ok {
					uniMap[v] = struct{}{}
					uniques = append(uniques, v)
				}
			}
			val = uniques
		case "RETAIN_LAST":
			val = curV.Index(curV.Len() - 1).Interface()
		case "RETAIN_FIRST":
			fallthrough
		default:
			val = curV.Index(0).Interface()
		}
	} else {
		val = curV.Interface()
	}
	parV.SetMapIndex(keyV, reflect.ValueOf(val))

	return nil
}

func isSliceOrArray(v reflect.Value) bool {
	return v.IsValid() && v.Kind() == reflect.Slice || v.Kind() == reflect.Array
}

func valTypeIsSame(a, b reflect.Value) bool {
	if !a.IsValid() || !b.IsValid() {
		return false
	}

	aKind := a.Kind()
	bKind := b.Kind()

	if aKind != reflect.Interface && bKind != reflect.Interface {
		return aKind == bKind
	}
	if aKind == reflect.Interface && bKind != reflect.Interface {
		return a.Elem().Kind() == bKind
	}
	if aKind != reflect.Interface && bKind == reflect.Interface {
		return aKind == b.Elem().Kind()
	}

	// aKind == reflect.Interface && bKind == reflect.Interface
	return a.Elem().Kind() == b.Elem().Kind()
}

func sliceElemTypeIsSame(a reflect.Value, b reflect.Value) bool {
	if !a.IsValid() || !b.IsValid() ||
		(a.Kind() != reflect.Slice && a.Kind() != reflect.Array) ||
		(b.Kind() != reflect.Slice && b.Kind() != reflect.Array) {
		return false
	}

	aElemKind := a.Type().Elem().Kind()
	bElemKind := b.Type().Elem().Kind()

	if aElemKind != reflect.Interface && bElemKind != reflect.Interface {
		return aElemKind == bElemKind
	}
	if aElemKind == reflect.Interface && bElemKind != reflect.Interface {
		return a.Len() == 0 || a.Index(0).Elem().Kind() == bElemKind
	}
	if aElemKind != reflect.Interface && bElemKind == reflect.Interface {
		return b.Len() == 0 || b.Index(0).Elem().Kind() == aElemKind
	}

	// aElemKind == reflect.Interface && bElemKind == reflect.Interface
	if a.Len() == 0 && b.Len() == 0 {
		return aElemKind == bElemKind
	}
	if a.Len() != 0 && b.Len() != 0 {
		return a.Index(0).Elem().Kind() == b.Index(0).Elem().Kind()
	}

	// a.Len() == 0 && b.Len() != 0 || a.Len() != 0 && b.Len() == 0
	return true
}

func sliceElemTypeIsSameToVal(slice, val reflect.Value) bool {
	if !slice.IsValid() || !val.IsValid() ||
		(slice.Kind() != reflect.Slice && slice.Kind() != reflect.Array) {
		return false
	}

	sliceElemKind := slice.Type().Elem().Kind()
	valKind := val.Kind()

	if sliceElemKind == reflect.Interface && valKind != reflect.Interface {
		return slice.Len() == 0 || slice.Index(0).Elem().Kind() == valKind
	}
	if sliceElemKind != reflect.Interface && valKind == reflect.Interface {
		return sliceElemKind == val.Elem().Kind()
	}
	if sliceElemKind != reflect.Interface && valKind != reflect.Interface {
		return sliceElemKind == valKind
	}

	// sliceElemKind == reflect.Interface && valKind == reflect.Interface
	return slice.Len() == 0 || slice.Index(0).Elem().Kind() == val.Elem().Kind()
}

func parseQueryByPath(path string) (map[string][]string, error) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	qs := make(map[string][]string)
	for k, vs := range u.Query() {
		qs[k] = vs
	}
	return qs, nil
}

func constructPath(path string, qs map[string][]string) (string, error) {
	u, err := url.Parse(path)
	if err != nil {
		return path, err
	}

	query := url.Values{}
	for k, vs := range qs {
		for _, v := range vs {
			query.Add(k, v)
		}
	}
	u.RawQuery = query.Encode()
	return u.String(), nil
}

// 返回值为 map[string]interface{} 或 map[string][]string，使用时断言即可
func parseBody(mediaType string, body []byte) (interface{}, error) {
	if len(body) == 0 {
		return nil, errEmptyBody
	}

	typ, params, err := mime.ParseMediaType(mediaType)
	if err != nil {
		return nil, err
	}
	switch typ {
	case "application/json":
		ret := make(map[string]interface{})
		err = json.Unmarshal(body, &ret)
		if err != nil {
			return nil, err
		}
		return ret, nil

	case "application/x-www-form-urlencoded":
		ret := make(map[string][]string)
		kvs, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, err
		}
		for k, vs := range kvs {
			ret[k] = vs
		}
		return ret, nil

	case "multipart/form-data":
		ret := make(map[string][]string)
		mr := multipart.NewReader(bytes.NewReader(body), params["boundary"])
		for {
			p, err := mr.NextPart()
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, err
			}
			formName := p.FormName()
			fileName := p.FileName()
			if formName == "" || fileName != "" {
				continue
			}
			formValue, err := io.ReadAll(p)
			if err != nil {
				return nil, err
			}
			ret[formName] = append(ret[formName], string(formValue))
		}
		return ret, nil

	default:
		return nil, errors.Errorf("unsupported media type: %s", mediaType)
	}
}

func constructBody(mediaType string, body interface{}) ([]byte, error) {
	ret := new(bytes.Buffer)
	typ, params, err := mime.ParseMediaType(mediaType)
	if err != nil {
		return nil, err
	}
	switch typ {
	case "application/json":
		bd, ok := body.(map[string]interface{})
		if !ok {
			return nil, errors.New("body type error")
		}
		b, err := json.MarshalIndent(bd, "", " ")
		if err != nil {
			return nil, err
		}
		ret.Write(b)
	case "application/x-www-form-urlencoded":
		bd, ok := body.(map[string][]string)
		if !ok {
			return nil, errors.New("body type error")
		}
		query := url.Values{}
		for k, vs := range bd {
			for _, v := range vs {
				query.Add(k, v)
			}
		}
		ret.WriteString(query.Encode())
	case "multipart/form-data":
		bd, ok := body.(map[string][]string)
		if !ok {
			return nil, errors.New("body type error")
		}
		w := multipart.NewWriter(ret)
		if err = w.SetBoundary(params["boundary"]); err != nil {
			return nil, err
		}
		for k, vs := range bd {
			for _, v := range vs {
				if err = w.WriteField(k, v); err != nil {
					return nil, err
				}
			}
		}
		if err = w.Close(); err != nil {
			return nil, err
		}
	default:
		return nil, errors.Errorf("unsupported media type: %s", mediaType)
	}

	return ret.Bytes(), nil
}

func convertByJsonType(typ string, value string) interface{} {
	var (
		ret interface{}
		err error
	)
	switch strings.ToLower(typ) {
	case "boolean":
		ret, err = strconv.ParseBool(value)
		if err != nil {
			ret = value
		}
	case "number":
		ret, err = strconv.ParseFloat(value, 64)
		if err != nil {
			ret = value
		}
	case "string":
		fallthrough
	default:
		ret = value
	}
	return ret
}

func convertHeaders(hs [][2]string) map[string][]string {
	ret := make(map[string][]string)
	for _, h := range hs {
		k, v := strings.ToLower(h[0]), h[1]
		ret[k] = append(ret[k], v)
	}
	return ret
}

func undoConvertHeaders(hs map[string][]string) [][2]string {
	var ret [][2]string
	for k, vs := range hs {
		for _, v := range vs {
			ret = append(ret, [2]string{k, v})
		}
	}
	return ret
}
