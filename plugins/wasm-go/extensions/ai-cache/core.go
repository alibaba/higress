package main

import (
	"encoding/json"
	"fmt"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/config"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/vector"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/resp"
)

func CheckCacheForKey(key string, ctx wrapper.HttpContext, config config.PluginConfig, log wrapper.Log, stream bool, useSimilaritySearch bool) {
	activeCacheProvider := config.GetCacheProvider()
	err := activeCacheProvider.Get(key, func(response resp.Value) {
		if err := response.Error(); err == nil && !response.IsNull() {
			log.Infof("cache hit, key: %s", key)
			ProcessCacheHit(key, response, stream, ctx, config, log)
		} else {
			log.Infof("cache miss, key: %s, error: %s", key, err.Error())
			if useSimilaritySearch {
				err = performSimilaritySearch(key, ctx, config, log, key, stream)
				if err != nil {
					log.Errorf("failed to perform similarity search for key: %s, error: %v", key, err)
					proxywasm.ResumeHttpRequest()
				}
			} else {
				proxywasm.ResumeHttpRequest()
				return
			}
		}
	})

	if err != nil {
		log.Errorf("Failed to retrieve key: %s from cache, error: %v", key, err)
		proxywasm.ResumeHttpRequest()
	}
}

func ProcessCacheHit(key string, response resp.Value, stream bool, ctx wrapper.HttpContext, config config.PluginConfig, log wrapper.Log) {
	escapedResponse, err := json.Marshal(response.String())
	if err != nil {
		proxywasm.SendHttpResponse(500, [][2]string{{"content-type", "text/plain"}}, []byte("Internal Server Error"), -1)
		return
	}
	ctx.SetContext(CACHE_KEY_CONTEXT_KEY, nil)
	if !stream {
		proxywasm.SendHttpResponse(200, [][2]string{{"content-type", "application/json; charset=utf-8"}}, []byte(fmt.Sprintf(config.ResponseTemplate, escapedResponse)), -1)
	} else {
		proxywasm.SendHttpResponse(200, [][2]string{{"content-type", "text/event-stream; charset=utf-8"}}, []byte(fmt.Sprintf(config.StreamResponseTemplate, escapedResponse)), -1)
	}
}

func performSimilaritySearch(key string, ctx wrapper.HttpContext, config config.PluginConfig, log wrapper.Log, queryString string, stream bool) error {
	activeEmbeddingProvider := config.GetEmbeddingProvider()
	err := activeEmbeddingProvider.GetEmbedding(queryString, ctx, log,
		func(emb []float64, err error) {
			if err != nil {
				log.Errorf("failed to fetch embeddings for key: %s, err: %v", key, err)
				proxywasm.ResumeHttpRequest()
				return
			}
			log.Debugf("successfully fetched embeddings for key: %s", key)
			QueryVectorDB(key, emb, ctx, config, log, stream)
		})
	return err
}

func QueryVectorDB(key string, textEmbedding []float64, ctx wrapper.HttpContext, config config.PluginConfig, log wrapper.Log, stream bool) {
	log.Debugf("starting query for key: %s", key)
	activeVectorProvider := config.GetVectorProvider()
	log.Debugf("active vector provider configuration: %+v", activeVectorProvider)

	err := activeVectorProvider.QueryEmbedding(textEmbedding, ctx, log, func(results []vector.QueryEmbeddingResult, ctx wrapper.HttpContext, log wrapper.Log, err error) {
		if err != nil {
			log.Errorf("error querying vector database: %v", err)
			proxywasm.ResumeHttpRequest()
			return
		}

		if len(results) == 0 {
			log.Warnf("no similar keys found in vector database for key: %s", key)
			uploadEmbedding(textEmbedding, key, ctx, log, activeVectorProvider)
			return
		}

		mostSimilarData := results[0]
		log.Debugf("most similar key found: %s with score: %f", mostSimilarData.Text, mostSimilarData.Score)

		if mostSimilarData.Score < activeVectorProvider.GetSimThreshold() {
			log.Infof("key accepted: %s with score: %f below threshold", mostSimilarData.Text, mostSimilarData.Score)
			CheckCacheForKey(mostSimilarData.Text, ctx, config, log, stream, false)
		} else {
			log.Infof("score too high for key: %s with score: %f above threshold", mostSimilarData.Text, mostSimilarData.Score)
			uploadEmbedding(textEmbedding, key, ctx, log, activeVectorProvider)
		}
	})

	if err != nil {
		log.Errorf("error querying vector database: %v", err)
		proxywasm.ResumeHttpRequest()
	}
}

func uploadEmbedding(textEmbedding []float64, key string, ctx wrapper.HttpContext, log wrapper.Log, provider vector.Provider) {
	provider.UploadEmbedding(textEmbedding, key, ctx, log, func(ctx wrapper.HttpContext, log wrapper.Log, err error) {
		if err != nil {
			log.Errorf("failed to upload embedding for key: %s, error: %v", key, err)
		} else {
			log.Debugf("successfully uploaded embedding for key: %s", key)
		}
		proxywasm.ResumeHttpRequest()
	})
}
