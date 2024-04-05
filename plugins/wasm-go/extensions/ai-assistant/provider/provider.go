package provider

import (
	"errors"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)

const (
	providerTypeMoonshot = "moonshot"

	defaultTimeout = 2 * 60 * 1000 // ms

	chatResponseTemplate = `
{
	"message": "%s"
}
`
)

type Provider interface {
	ProcessChatRequest(ctx wrapper.HttpContext, content string, log wrapper.Log) (types.Action, error)
}

type ProviderConfig struct {
	// @Title zh-CN AI服务提供商
	// @Description zh-CN AI服务提供商类型，目前支持的取值为："moonshot"
	typ string `required:"true" yaml:"type" json:"type"`
	// @Title zh-CN AI服务域名
	// @Description zh-CN AI服务提供商接口所使用的域名
	domain string `required:"false" yaml:"domain" json:"serviceDomain"`
	// @Title zh-CN API Token
	// @Description zh-CN 在请求AI服务时用于认证的API Token。不同的AI服务提供商可能有不同的名称。例Moonshot AI的API Token称为API Key
	apiToken string `required:"false" yaml:"apiToken" json:"apiToken"`
	// @Title zh-CN 模型名称
	// @Description zh-CN AI服务提供商的模型名称，用于指定使用的模型。具体取值请参考AI服务提供商的文档
	model string `required:"false" yaml:"model" json:"model"`
	// @Title zh-CN
	// @Description zh-CN
	fileId string `required:"true" yaml:"fileId" json:"fileId"`
	// @Title zh-CN 请求超时
	// @Description zh-CN 请求AI服务的超时时间，单位为毫秒。默认值为120000，即2分钟
	timeout uint32 `required:"false" yaml:"timeout" json:"timeout"`
}

func (c *ProviderConfig) FromJson(json gjson.Result) {
	c.typ = json.Get("type").String()
	c.domain = json.Get("domain").String()
	c.apiToken = json.Get("apiToken").String()
	c.model = json.Get("model").String()
	c.fileId = json.Get("fileId").String()
	c.timeout = uint32(json.Get("timeout").Uint())
}

func (c *ProviderConfig) Validate() error {
	if c.typ == "" {
		return errors.New("missing type in provider config")
	}
	if !isKnownProviderType(c.typ) {
		return errors.New("unsupported type in provider config")
	}

	if c.fileId == "" {
		return errors.New("missing fileId in config")
	}
	if c.timeout < 0 {
		return errors.New("invalid timeout in config")
	}
	return nil
}

func CreateProvider(pc ProviderConfig) (Provider, error) {
	client, err := createClient(pc)
	if err != nil {
		return nil, err
	}
	switch pc.typ {
	case providerTypeMoonshot:
		return &moonshotProvider{
			config: pc,
			client: client,
		}, nil
	default:
		return nil, errors.New("unknown provider type: " + pc.typ)
	}
}

func createClient(config ProviderConfig) (wrapper.HttpClient, error) {
	return wrapper.NewClusterClient(wrapper.RouteCluster{
		Host: config.domain,
	}), nil
}

func isKnownProviderType(typ string) bool {
	switch typ {
	case providerTypeMoonshot:
		return true
	default:
		return false
	}
}
