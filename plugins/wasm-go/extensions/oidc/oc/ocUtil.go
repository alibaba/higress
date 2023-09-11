package oc

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/gjson"
	"net/http"
	"strings"
	"time"
)

const (
	GithubSharedDataKey = "GithubSharedDataKey"
	GithubAccessVal     = ""
)

func ValidateHTTPResponse(statusCode int, headers http.Header, body []byte) error {
	contentType := headers.Get("Content-Type")
	if statusCode != http.StatusOK {
		return errors.New("call failed with status code")
	}
	if !strings.Contains(contentType, "application/json") {
		return fmt.Errorf("expected Content-Type = application/json or application/json;charset=UTF-8, but got %s", contentType)
	}
	if !gjson.ValidBytes(body) {
		return errors.New("invalid JSON format in response body")
	}

	return nil
}
func SendError(log *wrapper.Log, errMsg string, status int) {
	log.Errorf(errMsg)
	proxywasm.SendHttpResponse(uint32(status), nil, []byte(errMsg), -1)
}

type GithubSharedData struct {
	AccessToken string    `json:"access_token"`
	ExpiresIn   time.Time `json:"expires_in"`
}

func setSharedData(key, token string, expires time.Time) error {
	oldData, cas, err := proxywasm.GetSharedData(key)

	var log wrapper.Log
	var dataList []GithubSharedData
	if len(oldData) > 0 {
		if err := json.Unmarshal(oldData, &dataList); err != nil {
			log.Errorf("error unmarshalling shared data: %v", err)
			return err
		}
	}

	data := GithubSharedData{
		AccessToken: token,
		ExpiresIn:   expires,
	}

	dataList = append(dataList, data)

	dataBytes, err := json.Marshal(dataList)
	if err != nil {
		log.Errorf("error marshalling shared data: %v", err)
		return err
	}

	if err := proxywasm.SetSharedData(key, dataBytes, cas); err != nil {
		log.Errorf("error setting shared data: %v", err)
		return err
	}

	return nil
}

func checkAccessTokenValidity(key, token string) (bool, error) {
	value, _, err := proxywasm.GetSharedData(key)

	if err != nil {
		proxywasm.LogWarnf("error getting shared data: %v", err)
		return false, err
	}

	var dataList []GithubSharedData
	if err := json.Unmarshal(value, &dataList); err != nil {
		proxywasm.LogWarnf("error unmarshalling shared data: %v", err)
		return false, err
	}

	valid := false
	updatedDataList := dataList[:0]
	for _, data := range dataList {
		if data.AccessToken == token {
			if time.Now().Before(data.ExpiresIn) {
				valid = true
				updatedDataList = append(updatedDataList, data)
			}
		} else {
			updatedDataList = append(updatedDataList, data)
		}
	}

	if len(updatedDataList) != len(dataList) {
		dataBytes, err := json.Marshal(updatedDataList)
		if err != nil {
			proxywasm.LogWarnf("error marshalling shared data: %v", err)
			return false, err
		}
		if err := proxywasm.SetSharedData(key, dataBytes, 0); err != nil {
			proxywasm.LogWarnf("error setting shared data: %v", err)
			return false, err
		}
	}

	return valid, nil
}
