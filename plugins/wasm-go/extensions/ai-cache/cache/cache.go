// TODO: 在这里写缓存的具体逻辑, 将textEmbeddingPrvider和vectorStoreProvider作为逻辑中的一个函数调用
package cache

import (
	"errors"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

type RedisConfig struct {
	// @Title zh-CN redis 服务名称
	// @Description zh-CN 带服务类型的完整 FQDN 名称，例如 my-redis.dns、redis.my-ns.svc.cluster.local
	RedisServiceName string `required:"true" yaml:"serviceName" json:"serviceName"`
	// @Title zh-CN redis 服务端口
	// @Description zh-CN 默认值为6379
	RedisServicePort int `required:"false" yaml:"servicePort" json:"servicePort"`
	// @Title zh-CN 用户名
	// @Description zh-CN 登陆 redis 的用户名，非必填
	RedisUsername string `required:"false" yaml:"username" json:"username"`
	// @Title zh-CN 密码
	// @Description zh-CN 登陆 redis 的密码，非必填，可以只填密码
	RedisPassword string `required:"false" yaml:"password" json:"password"`
	// @Title zh-CN 请求超时
	// @Description zh-CN 请求 redis 的超时时间，单位为毫秒。默认值是1000，即1秒
	RedisTimeout uint32 `required:"false" yaml:"timeout" json:"timeout"`

	RedisHost string `required:"false" yaml:"host" json:"host"`
}

func CreateProvider(cf RedisConfig, log wrapper.Log) (Provider, error) {
	log.Warnf("redis config: %v", cf)
	rp := redisProvider{
		config: cf,
		client: wrapper.NewRedisClusterClient(wrapper.FQDNCluster{
			FQDN: cf.RedisServiceName,
			Host: cf.RedisHost,
			Port: int64(cf.RedisServicePort)}),
	}
	err := rp.Init(cf.RedisUsername, cf.RedisPassword, cf.RedisTimeout)
	return &rp, err
}

func (c *RedisConfig) FromJson(json gjson.Result) {
	c.RedisUsername = json.Get("username").String()
	c.RedisPassword = json.Get("password").String()
	c.RedisTimeout = uint32(json.Get("timeout").Int())
	c.RedisServiceName = json.Get("serviceName").String()
	c.RedisServicePort = int(json.Get("servicePort").Int())
	if c.RedisServicePort == 0 {
		c.RedisServicePort = 6379
	}
}

func (c *RedisConfig) Validate() error {
	if len(c.RedisServiceName) == 0 {
		return errors.New("serviceName is required")
	}
	if c.RedisTimeout <= 0 {
		return errors.New("timeout must be greater than 0")
	}
	if c.RedisServicePort <= 0 {
		c.RedisServicePort = 6379
	}
	if len(c.RedisUsername) == 0 {
		// return errors.New("redis.username is required")
		c.RedisUsername = ""
	}
	if len(c.RedisPassword) == 0 {
		c.RedisPassword = ""
	}
	return nil
}

type Provider interface {
	GetProviderType() string
	Init(username string, password string, timeout uint32) error
	Get(key string, cb wrapper.RedisResponseCallback)
	Set(key string, value string, cb wrapper.RedisResponseCallback)
}

type redisProvider struct {
	config RedisConfig
	client wrapper.RedisClient
}

func (rp *redisProvider) GetProviderType() string {
	return "redis"
}

func (rp *redisProvider) Init(username string, password string, timeout uint32) error {
	return rp.client.Init(rp.config.RedisUsername, rp.config.RedisPassword, int64(rp.config.RedisTimeout))
}

func (rp *redisProvider) Get(key string, cb wrapper.RedisResponseCallback) {
	rp.client.Get(key, cb)
}

func (rp *redisProvider) Set(key string, value string, cb wrapper.RedisResponseCallback) {
	rp.client.Set(key, value, cb)
}
