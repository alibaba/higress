package main

import (
	"fmt"
	"net/http"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/config"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/embedding"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/resp"
)

func RedisSearchHandler(key string, ctx wrapper.HttpContext, config config.PluginConfig, log wrapper.Log, stream bool, ifUseEmbedding bool) {
	activeCacheProvider := config.GetCacheProvider()
	log.Debugf("activeCacheProvider:%v", activeCacheProvider)
	activeCacheProvider.Get(embedding.CacheKeyPrefix+key, func(response resp.Value) {
		if err := response.Error(); err == nil && !response.IsNull() {
			log.Warnf("cache hit, key:%s", key)
			HandleCacheHit(key, response, stream, ctx, config, log)
		} else {
			log.Warnf("cache miss, key:%s", key)
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
	ctx.SetContext(embedding.CacheKeyContextKey, nil)
	if !stream {
		proxywasm.SendHttpResponse(200, [][2]string{{"content-type", "application/json; charset=utf-8"}}, []byte(fmt.Sprintf(config.ReturnResponseTemplate, "[Test, this is cache]"+response.String())), -1)
	} else {
		proxywasm.SendHttpResponse(200, [][2]string{{"content-type", "text/event-stream; charset=utf-8"}}, []byte(fmt.Sprintf(config.ReturnStreamResponseTemplate, "[Test, this is cache]"+response.String())), -1)
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
		func(emb []float64, statusCode int, responseHeaders http.Header, responseBody []byte) {
			if statusCode != 200 {
				log.Errorf("Failed to fetch embeddings, statusCode: %d, responseBody: %s", statusCode, string(responseBody))
			} else {
				log.Debugf("Successfully fetched embeddings for key: %s", key)
				QueryVectorDB(key, emb, ctx, config, log, stream)
			}
		})
}

func QueryVectorDB(key string, text_embedding []float64, ctx wrapper.HttpContext, config config.PluginConfig, log wrapper.Log, stream bool) {
	log.Debugf("QueryVectorDB key: %s", key)
	activeVectorDatabaseProvider := config.GetvectorProvider()
	log.Debugf("activeVectorDatabaseProvider: %+v", activeVectorDatabaseProvider)
	activeVectorDatabaseProvider.QueryEmbedding(text_embedding, ctx, log,
		func(responseBody []byte, ctx wrapper.HttpContext, log wrapper.Log) {
			resp, err := activeVectorDatabaseProvider.ParseQueryResponse(responseBody, ctx, log)
			if err != nil {
				log.Errorf("Failed to query vector database, err: %v", err)
				proxywasm.ResumeHttpRequest()
				return
			}

			if len(resp.MostSimilarData) == 0 {
				log.Warnf("Failed to query vector database, no most similar key found")
				activeVectorDatabaseProvider.UploadEmbedding(text_embedding, key, ctx, log,
					func(ctx wrapper.HttpContext, log wrapper.Log) {
						proxywasm.ResumeHttpRequest()
					})
				return
			}

			log.Infof("most similar key: %s", resp.MostSimilarData)
			if resp.Score < activeVectorDatabaseProvider.GetThreshold() {
				log.Infof("accept most similar key: %s, score: %f", resp.MostSimilarData, resp.Score)
				// ctx.SetContext(embedding.CacheKeyContextKey, nil)
				RedisSearchHandler(resp.MostSimilarData, ctx, config, log, stream, false)
			} else {
				log.Infof("the most similar key's score is too high, key: %s, score: %f", resp.MostSimilarData, resp.Score)
				activeVectorDatabaseProvider.UploadEmbedding(text_embedding, key, ctx, log,
					func(ctx wrapper.HttpContext, log wrapper.Log) {
						proxywasm.ResumeHttpRequest()
					})
				return
			}
		},
	)
}
