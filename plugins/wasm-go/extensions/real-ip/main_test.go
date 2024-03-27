package main

import (
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/gjson"
)

func TestGetRealIp(t *testing.T) {
	tests := []struct {
		name              string
		args              RealIpConfig
		realIpHeaderValue string
		want              string
		wantErr           bool
	}{
		{
			name:              "pass",
			args:              buildRealIpConfig(gjson.Parse(`{"real_ip_from":["172.18.0.1/24","127.1.1.1"],"real_ip_header":"X-Forwarded-For","recursive":true}`)),
			realIpHeaderValue: "127.0.2.1:9090,127.1.1.1,172.18.0.123,172.18.0.1",
			want:              "127.0.2.1:9090",
			wantErr:           false,
		},
		{
			name:              "contains untrusted service pass",
			args:              buildRealIpConfig(gjson.Parse(`{"real_ip_from":["172.18.0.1/24"],"real_ip_header":"X-Forwarded-For","recursive":true}`)),
			realIpHeaderValue: "127.0.2.1:9090,127.1.1.1,172.18.0.123,172.18.0.1",
			want:              "127.1.1.1",
			wantErr:           false,
		},
		{
			name:              "non-recursive pass",
			args:              buildRealIpConfig(gjson.Parse(`{"real_ip_from":["172.18.0.1/24"],"real_ip_header":"X-Forwarded-For","recursive":false}`)),
			realIpHeaderValue: "127.0.2.1:9090,127.1.1.1,172.18.0.123,172.18.0.1",
			want:              "172.18.0.1",
			wantErr:           false,
		},
		{
			name:              "X-Real-IP header pass",
			args:              buildRealIpConfig(gjson.Parse(`{"real_ip_from":["172.18.0.1/24"],"real_ip_header":"X-Real-IP","recursive":true}`)),
			realIpHeaderValue: "172.18.0.1",
			want:              "172.18.0.1",
			wantErr:           false,
		},
		{
			name:              "empty pass",
			args:              buildRealIpConfig(gjson.Parse(`{"real_ip_from":["172.18.0.1/24"],"real_ip_header":"X-Real-IP","recursive":true}`)),
			realIpHeaderValue: "",
			want:              "",
			wantErr:           false,
		},
	}
	for _, tt := range tests {
		var p = gomonkey.ApplyFunc(proxywasm.GetHttpRequestHeader, func(key string) (string, error) {
			return tt.realIpHeaderValue, nil
		})
		t.Run(tt.name, func(t *testing.T) {
			got, err := getRealIp(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("getRealIp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getRealIp() = %v, want %v", got, tt.want)
			}
		})
		defer p.Reset()
	}
}

func buildRealIpConfig(json gjson.Result) RealIpConfig {
	config := &RealIpConfig{}
	parseConfig(json, config, wrapper.Log{})
	return *config
}
