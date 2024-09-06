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
)

type providerInitializer interface {
	ValidateConfig(ProviderConfig) error
	CreateProvider(ProviderConfig) (Provider, error)
}

var (
	providerInitializers = map[string]providerInitializer{
		providerTypeDashVector: &dashVectorProviderInitializer{},
		providerTypeChroma:     &chromaProviderInitializer{},
		providerTypeES:         &esProviderInitializer{},
		providerTypeWeaviate:   &weaviateProviderInitializer{},
	}
)

type Provider interface {
	GetProviderType() string
	QueryEmbedding(
		emb []float64,
		ctx wrapper.HttpContext,
		log wrapper.Log,
		callback func(responseBody []byte, ctx wrapper.HttpContext, log wrapper.Log))
	UploadEmbedding(
		query_emb []float64,
		queryString string,
		ctx wrapper.HttpContext,
		log wrapper.Log,
		callback func(ctx wrapper.HttpContext, log wrapper.Log))
	GetThreshold() float64
	ParseQueryResponse(responseBody []byte, ctx wrapper.HttpContext, log wrapper.Log) (QueryEmbeddingResult, error)
}

// 定义通用的查询结果的结构体
type QueryEmbeddingResult struct {
	MostSimilarData string  // 相似的文本
	Score           float64 // 文本的向量相似度或距离等度量
}

type ProviderConfig struct {
	// @Title zh-CN 向量存储服务提供者类型
	// @Description zh-CN 向量存储服务提供者类型，例如 DashVector、Milvus
	typ string `json:"vectorStoreProviderType"`
	// @Title zh-CN ElasticSearch 需要满足的查询条件阈值关系
	// @Title zh-CN ElasticSearch 需要满足的查询条件阈值关系，默认为 lt，所有条件包括 lt (less than，小于)、lte (less than or equal to，小等于)、gt (greater than，大于)、gte (greater than or equal to，大等于)
	ThresholdRelation string `require:"false" yaml:"ThresholdRelation" json:"ThresholdRelation"`
	// @Title zh-CN DashVector 阿里云向量搜索引擎
	// @Description zh-CN 调用阿里云的向量搜索引擎
	DashVectorServiceName string `require:"true" yaml:"DashVectorServiceName" json:"DashVectorServiceName"`
	// @Title zh-CN DashVector Key
	// @Description zh-CN 阿里云向量搜索引擎的 key
	DashVectorKey string `require:"true" yaml:"DashVectorKey" json:"DashVectorKey"`
	// @Title zh-CN DashVector AuthApiEnd
	// @Description zh-CN 阿里云向量搜索引擎的 AuthApiEnd
	DashVectorAuthApiEnd string `require:"true" yaml:"DashVectorEnd" json:"DashVectorEnd"`
	// @Title zh-CN DashVector Collection
	// @Description zh-CN 指定使用阿里云搜索引擎中的哪个向量集合
	DashVectorCollection string `require:"true" yaml:"DashVectorCollection" json:"DashVectorCollection"`
	// @Title zh-CN DashVector Client
	// @Description zh-CN 阿里云向量搜索引擎的 Client
	DashVectorTopK    int                `require:"false" yaml:"DashVectorTopK" json:"DashVectorTopK"`
	DashVectorTimeout uint32             `require:"false" yaml:"DashVectorTimeout" json:"DashVectorTimeout"`
	DashVectorClient  wrapper.HttpClient `yaml:"-" json:"-"`

	// @Title zh-CN Chroma 的上游服务名称
	// @Description zh-CN Chroma 服务所对应的网关内上游服务名称
	ChromaServiceName string `require:"true" yaml:"ChromaServiceName" json:"ChromaServiceName"`
	// @Title zh-CN Chroma Collection ID
	// @Description zh-CN Chroma Collection 的 ID
	ChromaCollectionID string `require:"false" yaml:"ChromaCollectionID" json:"ChromaCollectionID"`
	// @Title zh-CN Chroma 距离阈值
	// @Description zh-CN Chroma 距离阈值，默认为 2000
	ChromaThreshold float64 `require:"false" yaml:"ChromaThreshold" json:"ChromaThreshold"`
	// @Title zh-CN Chroma 搜索返回结果数量
	// @Description zh-CN Chroma 搜索返回结果数量，默认为 1
	ChromaNResult int `require:"false" yaml:"ChromaNResult" json:"ChromaNResult"`
	// @Title zh-CN Chroma 超时设置
	// @Description zh-CN Chroma 超时设置，默认为 10 秒
	ChromaTimeout uint32 `require:"false" yaml:"ChromaTimeout" json:"ChromaTimeout"`

	// @Title zh-CN ElasticSearch 的上游服务名称
	// @Description zh-CN ElasticSearch 服务所对应的网关内上游服务名称
	ESServiceName string `require:"true" yaml:"ESServiceName" json:"ESServiceName"`
	// @Title zh-CN ElasticSearch index
	// @Description zh-CN ElasticSearch 的 index 名称
	ESIndex string `require:"false" yaml:"ESIndex" json:"ESIndex"`
	// @Title zh-CN ElasticSearch 距离阈值
	// @Description zh-CN ElasticSearch 距离阈值，默认为 2000
	ESThreshold float64 `require:"false" yaml:"ESThreshold" json:"ESThreshold"`
	// @Description zh-CN ElasticSearch 搜索返回结果数量，默认为 1
	ESNResult int `require:"false" yaml:"ESNResult" json:"ESNResult"`
	// @Title zh-CN Chroma 超时设置
	// @Description zh-CN Chroma 超时设置，默认为 10 秒
	ESTimeout uint32 `require:"false" yaml:"ESTimeout" json:"ESTimeout"`
	// @Title zh-CN ElasticSearch 用户名
	// @Description zh-CN ElasticSearch 用户名，默认为 elastic
	ESUsername string `require:"false" yaml:"ESUsername" json:"ESUsername"`
	// @Title zh-CN ElasticSearch 密码
	// @Description zh-CN ElasticSearch 密码，默认为 elastic
	ESPassword string `require:"false" yaml:"ESPassword" json:"ESPassword"`

	// @Title zh-CN Weaviate 的上游服务名称
	// @Description zh-CN Weaviate 服务所对应的网关内上游服务名称
	WeaviateServiceName string `require:"true" yaml:"WeaviateServiceName" json:"WeaviateServiceName"`
	// @Title zh-CN Weaviate 的 Collection 名称
	// @Description zh-CN Weaviate Collection 的名称（class name），注意这里 weaviate 会自动把首字母进行大写
	WeaviateCollection string `require:"true" yaml:"WeaviateCollection" json:"WeaviateCollection"`
	// @Title zh-CN Weaviate 的距离阈值
	// @Description zh-CN Weaviate 距离阈值，默认为 0.5，具体见 https://weaviate.io/developers/weaviate/config-refs/distances
	WeaviateThreshold float64 `require:"false" yaml:"WeaviateThreshold" json:"WeaviateThreshold"`
	// @Title zh-CN 搜索返回结果数量
	// @Description zh-CN 搜索返回结果数量，默认为 1
	WeaviateNResult int `require:"false" yaml:"WeaviateNResult" json:"WeaviateNResult"`
	// @Title zh-CN Chroma 超时设置
	// @Description zh-CN Chroma 超时设置，默认为 10 秒
	WeaviateTimeout uint32 `require:"false" yaml:"WeaviateTimeout" json:"WeaviateTimeout"`
}

func (c *ProviderConfig) FromJson(json gjson.Result) {
	c.typ = json.Get("VectorStoreProviderType").String()
	c.ThresholdRelation = json.Get("ThresholdRelation").String()
	if c.ThresholdRelation == "" {
		c.ThresholdRelation = "lt"
	}
	// DashVector
	c.DashVectorServiceName = json.Get("DashVectorServiceName").String()
	c.DashVectorKey = json.Get("DashVectorKey").String()
	c.DashVectorAuthApiEnd = json.Get("DashVectorEnd").String()
	c.DashVectorCollection = json.Get("DashVectorCollection").String()
	c.DashVectorTopK = int(json.Get("DashVectorTopK").Int())
	if c.DashVectorTopK == 0 {
		c.DashVectorTopK = 1
	}
	c.DashVectorTimeout = uint32(json.Get("DashVectorTimeout").Int())
	if c.DashVectorTimeout == 0 {
		c.DashVectorTimeout = 10000
	}
	// Chroma
	c.ChromaCollectionID = json.Get("ChromaCollectionID").String()
	c.ChromaServiceName = json.Get("ChromaServiceName").String()
	c.ChromaThreshold = json.Get("ChromaThreshold").Float()
	if c.ChromaThreshold == 0 {
		c.ChromaThreshold = 2000
	}
	c.ChromaNResult = int(json.Get("ChromaNResult").Int())
	if c.ChromaNResult == 0 {
		c.ChromaNResult = 1
	}
	c.ChromaTimeout = uint32(json.Get("ChromaTimeout").Int())
	if c.ChromaTimeout == 0 {
		c.ChromaTimeout = 10000
	}
	// ElasticSearch
	c.ESServiceName = json.Get("ESServiceName").String()
	c.ESIndex = json.Get("ESIndex").String()
	c.ESThreshold = json.Get("ESThreshold").Float()
	if c.ESThreshold == 0 {
		c.ESThreshold = 2000
	}
	c.ESNResult = int(json.Get("ElasticSearchNResult").Int())
	if c.ESNResult == 0 {
		c.ESNResult = 1
	}
	c.ESTimeout = uint32(json.Get("ElasticSearchTimeout").Int())
	if c.ESTimeout == 0 {
		c.ESTimeout = 10000
	}
	c.ESUsername = json.Get("ESUser").String()
	if c.ESUsername == "" {
		c.ESUsername = "elastic"
	}
	c.ESPassword = json.Get("ESPassword").String()
	if c.ESPassword == "" {
		c.ESPassword = "elastic"
	}
	// Weaviate
	c.WeaviateServiceName = json.Get("WeaviateServiceName").String()
	c.WeaviateCollection = json.Get("WeaviateCollection").String()
	c.WeaviateThreshold = json.Get("WeaviateThreshold").Float()
	if c.WeaviateThreshold == 0 {
		c.WeaviateThreshold = 0.5
	}
	c.WeaviateNResult = int(json.Get("WeaviateNResult").Int())
	if c.WeaviateNResult == 0 {
		c.WeaviateNResult = 1
	}
	c.WeaviateTimeout = uint32(json.Get("WeaviateTimeout").Int())
	if c.WeaviateTimeout == 0 {
		c.WeaviateTimeout = 10000
	}
}

func (c *ProviderConfig) Validate() error {
	if c.typ == "" {
		return errors.New("[ai-cache] missing type in provider config")
	}
	initializer, has := providerInitializers[c.typ]
	if !has {
		return errors.New("unknown provider type: " + c.typ)
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
