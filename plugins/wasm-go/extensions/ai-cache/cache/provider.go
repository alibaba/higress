package cache

import (
	"errors"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

const (
	providerTypeRedis = "redis"
)

type providerInitializer interface {
	ValidateConfig(ProviderConfig) error
	CreateProvider(ProviderConfig) (Provider, error)
}

var (
	providerInitializers = map[string]providerInitializer{
		providerTypeRedis: &redisProviderInitializer{},
	}
)

type ProviderConfig struct {
	// @Title zh-CN redis 服务名称
	// @Description zh-CN 带服务类型的完整 FQDN 名称，例如 my-redis.dns、redis.my-ns.svc.cluster.local
	serviceName string `required:"true" yaml:"serviceName" json:"serviceName"`
	// @Title zh-CN redis 服务端口
	// @Description zh-CN 默认值为6379
	servicePort int `required:"false" yaml:"servicePort" json:"servicePort"`
	// @Title zh-CN redis 服务地址
	// @Description zh-CN redis 服务地址，非必填
	serviceHost string `required:"false" yaml:"serviceHost" json:"servicehost"`
	// @Title zh-CN 用户名
	// @Description zh-CN 登陆 redis 的用户名，非必填
	userName string `required:"false" yaml:"username" json:"username"`
	// @Title zh-CN 密码
	// @Description zh-CN 登陆 redis 的密码，非必填，可以只填密码
	password string `required:"false" yaml:"password" json:"password"`
	// @Title zh-CN 请求超时
	// @Description zh-CN 请求 redis 的超时时间，单位为毫秒。默认值是1000，即1秒
	timeout uint32 `required:"false" yaml:"timeout" json:"timeout"`
}

func (c *ProviderConfig) FromJson(json gjson.Result) {
	c.serviceName = json.Get("serviceName").String()
	c.servicePort = int(json.Get("servicePort").Int())
	if c.servicePort <= 0 {
		c.servicePort = 6379
	}
	c.serviceHost = json.Get("serviceHost").String()
	c.userName = json.Get("username").String()
	if len(c.userName) == 0 {
		c.userName = ""
	}
	c.password = json.Get("password").String()
	if len(c.password) == 0 {
		c.password = ""
	}
	c.timeout = uint32(json.Get("timeout").Int())
	if c.timeout == 0 {
		c.timeout = 1000
	}

}

func (c *ProviderConfig) Validate() error {
	if len(c.serviceName) == 0 {
		return errors.New("serviceName is required")
	}
	if c.timeout <= 0 {
		return errors.New("timeout must be greater than 0")
	}
	return nil
}

func CreateProvider(pc ProviderConfig) (Provider, error) {
	initializer, has := providerInitializers[providerTypeRedis]
	if !has {
		return nil, errors.New("unknown provider type: " + providerTypeRedis)
	}
	return initializer.CreateProvider(pc)
}

type Provider interface {
	GetProviderType() string
	Init(username string, password string, timeout uint32) error
	Get(key string, cb wrapper.RedisResponseCallback) error
	Set(key string, value string, cb wrapper.RedisResponseCallback) error
}

