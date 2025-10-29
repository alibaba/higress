// Copyright (c) 2025 Alibaba Group Holding Ltd.
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

package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type HigressClient struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

func NewHigressClient(baseURL, username, password string) *HigressClient {
	client := &HigressClient{
		baseURL:  baseURL,
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	return client
}

func (c *HigressClient) Get(path string) ([]byte, error) {
	return c.request("GET", path, nil)
}

func (c *HigressClient) Post(path string, data interface{}) ([]byte, error) {
	return c.request("POST", path, data)
}

func (c *HigressClient) Put(path string, data interface{}) ([]byte, error) {
	return c.request("PUT", path, data)
}

func (c *HigressClient) Delete(path string) ([]byte, error) {
	return c.request("DELETE", path, nil)
}
func (c *HigressClient) request(method, path string, data interface{}) ([]byte, error) {
	url := c.baseURL + path

	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request data: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.username, c.password)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 409 {
		return nil, fmt.Errorf("resource already exists")
	}

	if resp.StatusCode == 400 {
		return nil, fmt.Errorf("invalid resource definition")
	}

	if resp.StatusCode == 500 {
		return nil, fmt.Errorf("server internal error")
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP error %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, nil
}
