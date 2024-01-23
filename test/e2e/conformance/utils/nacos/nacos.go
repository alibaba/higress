package nacos

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

func NewClient(addr string) (config_client.IConfigClient, error) {
	cc := constant.NewClientConfig(
		constant.WithNamespaceId(""),
		constant.WithUsername(""),
		constant.WithPassword(""),
		constant.WithLogLevel("info"),
	)

	serverUrl, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}
	rawPort := serverUrl.Port()
	var port uint64
	if rawPort != "" {
		port, err = strconv.ParseUint(rawPort, 10, 0)
		if err != nil || port < 1 || port > 65535 {
			return nil, err
		}
	} else {
		port = 80
	}
	path := serverUrl.Path
	if strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}
	sc := []constant.ServerConfig{
		{
			IpAddr:      serverUrl.Hostname(),
			ContextPath: path,
			Port:        port,
			Scheme:      serverUrl.Scheme,
		},
	}

	client, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  cc,
			ServerConfigs: sc,
		},
	)
	if err != nil {
		return nil, err
	}
	return client, nil
}
