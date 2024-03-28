package provider

import (
	"errors"

	"github.com/tetratelabs/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
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
	typ      string `required:"true" yaml:"type" json:"type"`
	domain   string `required:"false" yaml:"domain" json:"serviceDomain"`
	apiToken string `required:"false" yaml:"apiToken" json:"apiToken"`
	model    string `required:"false" yaml:"model" json:"model"`
	fileId   string `required:"true" yaml:"fileId" json:"fileId"`
	timeout  uint32 `required:"false" yaml:"timeout" json:"timeout"`
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
