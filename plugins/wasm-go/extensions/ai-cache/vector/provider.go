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
		providerTypeChroma:     &chromaProviderInitializer{},
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
	ChromaDistanceThreshold float64 `require:"false" yaml:"ChromaDistanceThreshold" json:"ChromaDistanceThreshold"`
	// @Title zh-CN Chroma 搜索返回结果数量
	// @Description zh-CN Chroma 搜索返回结果数量，默认为 1
	ChromaNResult int `require:"false" yaml:"ChromaNResult" json:"ChromaNResult"`
	// @Title zh-CN Chroma 超时设置
	// @Description zh-CN Chroma 超时设置，默认为 10 秒
	ChromaTimeout uint32 `require:"false" yaml:"ChromaTimeout" json:"ChromaTimeout"`
}

func (c *ProviderConfig) FromJson(json gjson.Result) {
	c.typ = json.Get("vectorStoreProviderType").String()
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
	c.ChromaDistanceThreshold = json.Get("ChromaDistanceThreshold").Float()
	if c.ChromaDistanceThreshold == 0 {
		c.ChromaDistanceThreshold = 2000
	}
	c.ChromaNResult = int(json.Get("ChromaNResult").Int())
	if c.ChromaNResult == 0 {
		c.ChromaNResult = 1
	}
	c.ChromaTimeout = uint32(json.Get("ChromaTimeout").Int())
	if c.ChromaTimeout == 0 {
		c.ChromaTimeout = 10000
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
