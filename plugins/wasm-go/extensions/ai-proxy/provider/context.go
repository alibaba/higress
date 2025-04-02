package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-proxy/util"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/log"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/gjson"
)

type ContextConfig struct {
	// @Title zh-CN 文件URL
	// @Description zh-CN 用于获取对话上下文的文件的URL。目前仅支持HTTP和HTTPS协议，纯文本格式文件
	fileUrl string `required:"true" yaml:"url" json:"url"`
	// @Title zh-CN 上游服务名称
	// @Description zh-CN 文件服务所对应的网关内上游服务名称
	serviceName string `required:"true" yaml:"serviceName" json:"serviceName"`
	// @Title zh-CN 上游服务端口
	// @Description zh-CN 文件服务所对应的网关内上游服务名称
	servicePort int64 `required:"true" yaml:"servicePort" json:"servicePort"`

	fileUrlObj *url.URL `yaml:"-"`
}

func (c *ContextConfig) FromJson(json gjson.Result) {
	c.fileUrl = json.Get("fileUrl").String()
	c.serviceName = json.Get("serviceName").String()
	c.servicePort = json.Get("servicePort").Int()
}

func (c *ContextConfig) Validate() error {
	if c.fileUrl == "" {
		return errors.New("missing fileUrl in context config")
	}
	if fileUrlObj, err := url.Parse(c.fileUrl); err != nil {
		return fmt.Errorf("invalid fileUrl in context config: %v", err)
	} else {
		c.fileUrlObj = fileUrlObj
	}
	if c.serviceName == "" {
		return errors.New("missing serviceName in context config")
	}
	if c.servicePort == 0 {
		return errors.New("missing servicePort in context config")
	}
	return nil
}

type contextCache struct {
	client  wrapper.HttpClient
	fileUrl *url.URL
	timeout uint32

	loaded  bool
	content string
}

type ContextInserter interface {
	insertHttpContextMessage(body []byte, content string, onlyOneSystemBeforeFile bool) ([]byte, error)
}

func (c *contextCache) GetContent(callback func(string, error)) error {
	if callback == nil {
		return errors.New("callback is nil")
	}

	if c.loaded {
		log.Debugf("context file loaded from cache")
		callback(c.content, nil)
		return nil
	}

	log.Infof("loading context file from %s", c.fileUrl.String())
	return c.client.Get(c.fileUrl.Path, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		if statusCode != http.StatusOK {
			callback("", fmt.Errorf("failed to load context file, status: %d", statusCode))
			return
		}
		c.content = string(responseBody)
		c.loaded = true
		log.Debugf("content: %s", c.content)
		callback(c.content, nil)
	}, c.timeout)
}

func createContextCache(providerConfig *ProviderConfig) *contextCache {
	contextConfig := providerConfig.context
	if contextConfig == nil {
		return nil
	}
	fileUrlObj, _ := url.Parse(contextConfig.fileUrl)
	cluster := wrapper.FQDNCluster{
		FQDN: contextConfig.serviceName,
		Port: contextConfig.servicePort,
		Host: fileUrlObj.Host,
	}
	return &contextCache{
		client:  wrapper.NewClusterClient(cluster),
		fileUrl: fileUrlObj,
		timeout: providerConfig.timeout,
	}
}

func (c *contextCache) GetContextFromFile(ctx wrapper.HttpContext, provider Provider, body []byte) error {
	if c.loaded {
		log.Debugf("context file loaded from cache")
		insertContext(provider, c.content, nil, body)
		return nil
	}

	log.Infof("loading context file from %s", c.fileUrl.String())
	return c.client.Get(c.fileUrl.Path, nil, func(statusCode int, responseHeaders http.Header, responseBody []byte) {
		if statusCode != http.StatusOK {
			insertContext(provider, "", fmt.Errorf("failed to load context file, status: %d", statusCode), nil)
			return
		}
		c.content = string(responseBody)
		c.loaded = true
		log.Debugf("content: %s", c.content)
		insertContext(provider, c.content, nil, body)
	}, c.timeout)
}

func insertContext(provider Provider, content string, err error, body []byte) {
	defer func() {
		_ = proxywasm.ResumeHttpRequest()
	}()

	typ := provider.GetProviderType()
	if err != nil {
		log.Errorf("failed to load context file: %v", err)
		util.ErrorHandler(fmt.Sprintf("ai-proxy.%s.load_ctx_failed", typ), fmt.Errorf("failed to load context file: %v", err))
	}

	if inserter, ok := provider.(ContextInserter); ok {
		body, err = inserter.insertHttpContextMessage(body, content, false)
	} else {
		body, err = defaultInsertHttpContextMessage(body, content)
	}

	if err != nil {
		util.ErrorHandler(fmt.Sprintf("ai-proxy.%s.insert_ctx_failed", typ), fmt.Errorf("failed to insert context message: %v", err))
	}
	if err := replaceRequestBody(body); err != nil {
		util.ErrorHandler(fmt.Sprintf("ai-proxy.%s.replace_request_body_failed", typ), fmt.Errorf("failed to replace request body: %v", err))
	}
}

func defaultInsertHttpContextMessage(body []byte, content string) ([]byte, error) {
	request := &chatCompletionRequest{}
	if err := json.Unmarshal(body, request); err != nil {
		return nil, fmt.Errorf("unable to unmarshal request: %v", err)
	}

	fileMessage := chatMessage{
		Role:    roleSystem,
		Content: content,
	}
	var firstNonSystemMessageIndex int
	for i, message := range request.Messages {
		if message.Role != roleSystem {
			firstNonSystemMessageIndex = i
			break
		}
	}
	if firstNonSystemMessageIndex == 0 {
		request.Messages = append([]chatMessage{fileMessage}, request.Messages...)
	} else {
		request.Messages = append(request.Messages[:firstNonSystemMessageIndex], append([]chatMessage{fileMessage}, request.Messages[firstNonSystemMessageIndex:]...)...)
	}

	return json.Marshal(request)
}
