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

package address

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func setUpServer(status int, body []byte) (string, func()) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(status)
		rw.Write(body)
	}))
	return server.URL, func() {
		server.Close()
	}
}

func setUpServerWithBodyPtr(status int, body *[]byte) (string, func()) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(status)
		rw.Write(*body)
	}))
	return server.URL, func() {
		server.Close()
	}
}
func TestGetNacosAddress(t *testing.T) {
	goodURL, goodTearDown := setUpServer(200, []byte("1.1.1.1\n 2.2.2.2"))
	defer goodTearDown()
	badURL, badTearDown := setUpServer(200, []byte("abc\n 2.2.2.2"))
	defer badTearDown()
	errURL, errTearDown := setUpServer(503, []byte("1.1.1.1\n 2.2.2.2"))
	defer errTearDown()
	tests := []struct {
		name       string
		serverAddr string
		want       []string
	}{
		{
			"good",
			goodURL,
			[]string{"1.1.1.1", "2.2.2.2"},
		},
		{
			"bad",
			badURL,
			[]string{},
		},
		{
			"err",
			errURL,
			[]string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewNacosAddressProvider(tt.serverAddr, "")
			timeout := time.NewTicker(1 * time.Second)
			var got string
			if len(tt.want) == 0 {
				select {
				case got = <-provider.GetNacosAddress(""):
					t.Errorf("GetNacosAddress() = %v, want empty", got)
				case <-timeout.C:
					return
				}
			}
			select {
			case got = <-provider.GetNacosAddress(""):
			case <-timeout.C:
				t.Error("GetNacosAddress timeout")
			}
			for _, value := range tt.want {
				if got == value {
					return
				}
			}
			t.Errorf("GetNacosAddress() = %v, want %v", got, tt.want)
		})
	}
}

func TestTrigger(t *testing.T) {
	body := []byte("1.1.1.1 ")
	url, tearDown := setUpServerWithBodyPtr(200, &body)
	defer tearDown()
	provider := NewNacosAddressProvider(url, "xxxx")
	address := <-provider.GetNacosAddress("")
	if address != "1.1.1.1" {
		t.Errorf("got %s, want %s", address, "1.1.1.1")
	}
	body = []byte(" 2.2.2.2 ")
	tests := []struct {
		name    string
		trigger bool
		want    string
	}{
		{
			"no trigger",
			false,
			"1.1.1.1",
		},
		{
			"trigger",
			true,
			"2.2.2.2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.trigger {
				provider.Trigger()
			}
			timeout := time.NewTicker(1 * time.Second)
			select {
			case <-provider.GetNacosAddress("1.1.1.1"):
			case <-timeout.C:
			}
			if provider.nacosAddr != tt.want {
				t.Errorf("got %s, want %s", provider.nacosAddr, tt.want)
			}
		})
	}
}

func TestBackup(t *testing.T) {
	body := []byte("1.1.1.1 ")
	url, tearDown := setUpServerWithBodyPtr(200, &body)
	defer tearDown()
	provider := NewNacosAddressProvider(url, "xxxx")
	address := <-provider.GetNacosAddress("")
	if address != "1.1.1.1" {
		t.Errorf("got %s, want %s", address, "1.1.1.1")
	}
	tests := []struct {
		name       string
		oldaddr    string
		newaddr    string
		triggerNum int
		want       string
	}{
		{
			"case1",
			"1.1.1.1",
			"1.1.1.1\n2.2.2.2",
			1,
			"2.2.2.2",
		},
		{
			"case2",
			"1.1.1.1",
			"3.3.3.3 1.1.1.1",
			1,
			"3.3.3.3",
		},
		{
			"case3",
			"1.1.1.1",
			"3.3.3.3 1.1.1.1",
			2,
			"1.1.1.1",
		},
		{
			"case4",
			"1.1.1.1",
			"3.3.3.3\n 1.1.1.1",
			3,
			"3.3.3.3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider.nacosAddr = tt.oldaddr
			body = []byte(tt.newaddr)
			provider.addressDiscovery()
			for i := 0; i < tt.triggerNum; i++ {
				provider.Trigger()
			}
			timeout := time.NewTicker(1 * time.Second)
			var newAddr string
			select {
			case newAddr = <-provider.GetNacosAddress(""):
			case <-timeout.C:
			}
			if newAddr != tt.want {
				t.Errorf("got %s, want %s", newAddr, tt.want)
			}
		})
	}
}

func TestKeepIp(t *testing.T) {
	body := []byte("1.1.1.1")
	url, tearDown := setUpServerWithBodyPtr(200, &body)
	defer tearDown()
	provider := NewNacosAddressProvider(url, "xxxx")
	address := <-provider.GetNacosAddress("")
	if address != "1.1.1.1" {
		t.Errorf("got %s, want %s", address, "1.1.1.1")
	}
	tests := []struct {
		name    string
		newAddr []byte
		want    string
	}{
		{
			"add ip",
			[]byte("1.1.1.1\n 2.2.2.2"),
			"1.1.1.1",
		},
		{
			"remove ip",
			[]byte("2.2.2.2"),
			"2.2.2.2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body = tt.newAddr
			provider.addressDiscovery()
			timeout := time.NewTicker(1 * time.Second)
			select {
			case <-provider.GetNacosAddress("1.1.1.1"):
			case <-timeout.C:
			}
			if provider.nacosAddr != tt.want {
				t.Errorf("got %s, want %s", provider.nacosAddr, tt.want)
			}
		})
	}
}

func TestMultiClient(t *testing.T) {
	body := []byte("1.1.1.1")
	url, tearDown := setUpServerWithBodyPtr(200, &body)
	defer tearDown()
	provider := NewNacosAddressProvider(url, "xxxx")
	address := <-provider.GetNacosAddress("")
	if address != "1.1.1.1" {
		t.Errorf("got %s, want %s", address, "1.1.1.1")
	}
	body = []byte("2.2.2.2")
	tests := []struct {
		name     string
		oldAddrs []string
		want     []string
	}{
		{
			"case1",
			[]string{"1.1.1.1", "1.1.1.1"},
			[]string{"2.2.2.2", "2.2.2.2"},
		},
		{
			"case2",
			[]string{"2.2.2.2", "1.1.1.1"},
			[]string{"", "2.2.2.2"},
		},
		{
			"case3",
			[]string{"1.1.1.1", "2.2.2.2"},
			[]string{"2.2.2.2", ""},
		},
		{
			"case4",
			[]string{"2.2.2.2", "2.2.2.2"},
			[]string{"", ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider.addressDiscovery()
			for i := 0; i < len(tt.oldAddrs); i++ {
				timeout := time.NewTicker(1 * time.Second)
				var newaddr string
				select {
				case newaddr = <-provider.GetNacosAddress(tt.oldAddrs[i]):
				case <-timeout.C:
				}
				if newaddr != tt.want[i] {
					t.Errorf("got %s, want %s", newaddr, tt.want[i])
				}
			}
		})
	}
}
