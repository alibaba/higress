package vector

import (
	"errors"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/tidwall/gjson"
)

const (
	providerTypeDashVector = "dashvector"
	providerTypeChroma     = "chroma"
)

type providerInitializer interface {
	ValidateConfig(ProviderConfig) error
	CreateProvider(ProviderConfig) (Provider, error)
}

var (
	providerInitializers = map[string]providerInitializer{
		providerTypeDashVector: &dashVectorProviderInitializer{},
		// providerTypeChroma:     &chromaProviderInitializer{},
	}
)

// 定义通用的查询结果的结构体
type QueryEmbeddingResult struct {
	Text      string    // 相似的文本
	Embedding []float64 // 相似文本的向量
	Score     float64   // 文本的向量相似度或距离等度量
}

type Provider interface {
	GetProviderType() string
	// TODO: 考虑失败的场景
	QueryEmbedding(
		emb []float64,
		ctx wrapper.HttpContext,
		log wrapper.Log,
		callback func(results []QueryEmbeddingResult, ctx wrapper.HttpContext, log wrapper.Log))
	// TODO: 考虑失败的场景
	UploadEmbedding(
		queryEmb []float64,
		queryString string,
		ctx wrapper.HttpContext,
		log wrapper.Log,
		callback func(ctx wrapper.HttpContext, log wrapper.Log))
	GetThreshold() float64
	// ParseQueryResponse(responseBody []byte, ctx wrapper.HttpContext, log wrapper.Log) (QueryEmbeddingResult, error)
}

type ProviderConfig struct {
	// @Title zh-CN 向量存储服务提供者类型
	// @Description zh-CN 向量存储服务提供者类型，例如 DashVector、Milvus
	typ           string
	serviceName   string
	serviceDomain string
	servicePort   int64
	apiKey        string
	topK          int
	timeout       uint32
	collectionID  string

	// // @Title zh-CN Chroma 的上游服务名称
	// // @Description zh-CN Chroma 服务所对应的网关内上游服务名称
	// ChromaServiceName string `require:"true" yaml:"ChromaServiceName" json:"ChromaServiceName"`
	// // @Title zh-CN Chroma Collection ID
	// // @Description zh-CN Chroma Collection 的 ID
	// ChromaCollectionID string `require:"false" yaml:"ChromaCollectionID" json:"ChromaCollectionID"`
	// @Title zh-CN Chroma 距离阈值
	// @Description zh-CN Chroma 距离阈值，默认为 2000
	ChromaDistanceThreshold float64 `require:"false" yaml:"ChromaDistanceThreshold" json:"ChromaDistanceThreshold"`
	// // @Title zh-CN Chroma 搜索返回结果数量
	// // @Description zh-CN Chroma 搜索返回结果数量，默认为 1
	// ChromaNResult int `require:"false" yaml:"ChromaNResult" json:"ChromaNResult"`
	// // @Title zh-CN Chroma 超时设置
	// // @Description zh-CN Chroma 超时设置，默认为 10 秒
	// ChromaTimeout uint32 `require:"false" yaml:"ChromaTimeout" json:"ChromaTimeout"`
}

func (c *ProviderConfig) FromJson(json gjson.Result) {
	c.typ = json.Get("type").String()
	// DashVector
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
	// Chroma
	// c.ChromaCollectionID = json.Get("ChromaCollectionID").String()
	// c.ChromaServiceName = json.Get("ChromaServiceName").String()
	// c.ChromaDistanceThreshold = json.Get("ChromaDistanceThreshold").Float()
	// if c.ChromaDistanceThreshold == 0 {
	// 	c.ChromaDistanceThreshold = 2000
	// }
	// c.ChromaNResult = int(json.Get("ChromaNResult").Int())
	// if c.ChromaNResult == 0 {
	// 	c.ChromaNResult = 1
	// }
	// c.ChromaTimeout = uint32(json.Get("ChromaTimeout").Int())
	// if c.ChromaTimeout == 0 {
	// 	c.ChromaTimeout = 10000
	// }
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
