package vector

import (
	"errors"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

const (
	providerTypeDashVector = "dashvector"
	providerTypeChroma     = "chroma"
	providerTypeES         = "elasticsearch"
	providerTypeWeaviate   = "weaviate"
	providerTypePinecone   = "pinecone"
	providerTypeQdrant     = "qdrant"
)

type providerInitializer interface {
	ValidateConfig(ProviderConfig) error
	CreateProvider(ProviderConfig) (Provider, error)
}

var (
	providerInitializers = map[string]providerInitializer{
		providerTypeDashVector: &dashVectorProviderInitializer{},
		providerTypeChroma:     &chromaProviderInitializer{},
		providerTypeWeaviate:   &weaviateProviderInitializer{},
		providerTypeES:         &esProviderInitializer{},
		providerTypePinecone:   &pineconeProviderInitializer{},
		providerTypeQdrant:     &qdrantProviderInitializer{},
	}
)

// QueryResult 定义通用的查询结果的结构体
type QueryResult struct {
	Text      string    // 相似的文本
	Embedding []float64 // 相似文本的向量
	Score     float64   // 文本的向量相似度或距离等度量
	Answer    string    // 相似文本对应的LLM生成的回答
}

type Provider interface {
	GetProviderType() string
}

type EmbeddingQuerier interface {
	QueryEmbedding(
		emb []float64,
		ctx wrapper.HttpContext,
		log wrapper.Log,
		callback func(results []QueryResult, ctx wrapper.HttpContext, log wrapper.Log, err error)) error
}

type EmbeddingUploader interface {
	UploadEmbedding(
		queryString string,
		queryEmb []float64,
		ctx wrapper.HttpContext,
		log wrapper.Log,
		callback func(ctx wrapper.HttpContext, log wrapper.Log, err error)) error
}

type AnswerAndEmbeddingUploader interface {
	UploadAnswerAndEmbedding(
		queryString string,
		queryEmb []float64,
		answer string,
		ctx wrapper.HttpContext,
		log wrapper.Log,
		callback func(ctx wrapper.HttpContext, log wrapper.Log, err error)) error
}

type StringQuerier interface {
	QueryString(
		queryString string,
		ctx wrapper.HttpContext,
		log wrapper.Log,
		callback func(results []QueryResult, ctx wrapper.HttpContext, log wrapper.Log, err error)) error
}

type SimilarityThresholdProvider interface {
	GetSimilarityThreshold() float64
}

type ProviderConfig struct {
	// @Title zh-CN 向量存储服务提供者类型
	// @Description zh-CN 向量存储服务提供者类型，例如 DashVector、Milvus
	typ string
	// @Title zh-CN 向量存储服务名称
	// @Description zh-CN 向量存储服务名称
	serviceName string
	// @Title zh-CN 向量存储服务域名
	// @Description zh-CN 向量存储服务域名
	serviceDomain string
	// @Title zh-CN 向量存储服务端口
	// @Description zh-CN 向量存储服务端口
	servicePort int64
	// @Title zh-CN 向量存储服务 API Key
	// @Description zh-CN 向量存储服务 API Key
	apiKey string
	// @Title zh-CN 返回TopK结果
	// @Description zh-CN 返回TopK结果，默认为 1
	topK int
	// @Title zh-CN 请求超时
	// @Description zh-CN 请求向量存储服务的超时时间，单位为毫秒。默认值是10000，即10秒
	timeout uint32
	// @Title zh-CN DashVector 向量存储服务 Collection ID
	// @Description zh-CN DashVector 向量存储服务 Collection ID
	collectionID string

	// ES 配置
	// @Title zh-CN ES 用户名
	// @Description zh-CN ES 用户名
	esUsername string
	// @Title zh-CN ES 密码
	// @Description zh-CN ES 密码
	esPassword string
}

func (c *ProviderConfig) GetProviderType() string {
	return c.typ
}

func (c *ProviderConfig) FromJson(json gjson.Result) {
	c.typ = json.Get("type").String()
	c.serviceName = json.Get("serviceName").String()
	c.serviceDomain = json.Get("serviceDomain").String()
	c.servicePort = int64(json.Get("servicePort").Int())
	if c.servicePort == 0 {
		c.servicePort = 443
	}
	c.apiKey = json.Get("apiKey").String()
	c.collectionID = json.Get("collectionID").String()
	c.topK = int(json.Get("topK").Int())
	if c.topK == 0 {
		c.topK = 1
	}
	c.timeout = uint32(json.Get("timeout").Int())
	if c.timeout == 0 {
		c.timeout = 10000
	}

	// ES
	c.esUsername = json.Get("esUsername").String()
	c.esPassword = json.Get("esPassword").String()
}

func (c *ProviderConfig) Validate() error {
	if c.typ == "" {
		return errors.New("vector database service is required")
	}
	initializer, has := providerInitializers[c.typ]
	if !has {
		return errors.New("unknown vector database service provider type: " + c.typ)
	}
	if err := initializer.ValidateConfig(*c); err != nil {
		return err
	}
	return nil
}

func CreateProvider(pc ProviderConfig) (Provider, error) {
	initializer, has := providerInitializers[pc.typ]
	if !has {
		return nil, errors.New("unknown provider type: " + pc.typ)
	}
	return initializer.CreateProvider(pc)
}
