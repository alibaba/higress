package main

import (
	"fmt"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/config"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/vector"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/resp"
)

func RedisSearchHandler(key string, ctx wrapper.HttpContext, config config.PluginConfig, log wrapper.Log, stream bool, ifUseEmbedding bool) {
	activeCacheProvider := config.GetCacheProvider()
	key = activeCacheProvider.GetCacheKeyPrefix() + ":" + key
	log.Debugf("activeCacheProvider:%v", activeCacheProvider)
	activeCacheProvider.Get(key, func(response resp.Value) {
		if err := response.Error(); err == nil && !response.IsNull() {
			log.Debugf("cache hit, key:%s", key)
			HandleCacheHit(key, response, stream, ctx, config, log)
		} else {
			log.Debugf("cache miss, key:%s", key)
			if ifUseEmbedding {
				HandleCacheMiss(key, err, response, ctx, config, log, key, stream)
			} else {
				proxywasm.ResumeHttpRequest()
				return
			}
		}
	})
}

func HandleCacheHit(key string, response resp.Value, stream bool, ctx wrapper.HttpContext, config config.PluginConfig, log wrapper.Log) {
	ctx.SetContext(CACHE_KEY_CONTEXT_KEY, nil)
	if !stream {
		proxywasm.SendHttpResponse(200, [][2]string{{"content-type", "application/json; charset=utf-8"}}, []byte(fmt.Sprintf(config.ResponseTemplate, response.String())), -1)
	} else {
		proxywasm.SendHttpResponse(200, [][2]string{{"content-type", "text/event-stream; charset=utf-8"}}, []byte(fmt.Sprintf(config.StreamResponseTemplate, response.String())), -1)
	}
}

func HandleCacheMiss(key string, err error, response resp.Value, ctx wrapper.HttpContext, config config.PluginConfig, log wrapper.Log, queryString string, stream bool) {
	if err != nil {
		log.Warnf("redis get key:%s failed, err:%v", key, err)
	}
	if response.IsNull() {
		log.Warnf("cache miss, key:%s", key)
	}
	FetchAndProcessEmbeddings(key, ctx, config, log, queryString, stream)
}

func FetchAndProcessEmbeddings(key string, ctx wrapper.HttpContext, config config.PluginConfig, log wrapper.Log, queryString string, stream bool) {
	activeEmbeddingProvider := config.GetEmbeddingProvider()
	activeEmbeddingProvider.GetEmbedding(queryString, ctx, log,
		func(emb []float64) {
			log.Debugf("Successfully fetched embeddings for key: %s", key)
			QueryVectorDB(key, emb, ctx, config, log, stream)
		})
}

func QueryVectorDB(key string, textEmbedding []float64, ctx wrapper.HttpContext, config config.PluginConfig, log wrapper.Log, stream bool) {
	log.Debugf("QueryVectorDB key: %s", key)
	activeVectorProvider := config.GetVectorProvider()
	log.Debugf("activeVectorProvider: %+v", activeVectorProvider)
	activeVectorProvider.QueryEmbedding(textEmbedding, ctx, log,
		func(results []vector.QueryEmbeddingResult, ctx wrapper.HttpContext, log wrapper.Log) {
			// The baisc logic is to compare the similarity of the embedding with the most similar key in the database
			if len(results) == 0 {
				log.Warnf("Failed to query vector database, no similar key found")
				activeVectorProvider.UploadEmbedding(textEmbedding, key, ctx, log,
					func(ctx wrapper.HttpContext, log wrapper.Log) {
						proxywasm.ResumeHttpRequest()
					})
				return
			}

			mostSimilarData := results[0]
			log.Infof("most similar key: %s", mostSimilarData.Text)
			if mostSimilarData.Score < activeVectorProvider.GetThreshold() {
				log.Infof("accept most similar key: %s, score: %f", mostSimilarData.Text, mostSimilarData.Score)
				// ctx.SetContext(embedding.CacheKeyContextKey, nil)
				RedisSearchHandler(mostSimilarData.Text, ctx, config, log, stream, false)
			} else {
				log.Infof("the most similar key's score is too high, key: %s, score: %f", mostSimilarData.Text, mostSimilarData.Score)
				activeVectorProvider.UploadEmbedding(textEmbedding, key, ctx, log,
					func(ctx wrapper.HttpContext, log wrapper.Log) {
						proxywasm.ResumeHttpRequest()
					})
				return
			}
		},
	)
}
