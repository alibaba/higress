package main

import (
	"errors"
	"fmt"

	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/config"
	"github.com/alibaba/higress/plugins/wasm-go/extensions/ai-cache/vector"
	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/tidwall/resp"
)

// CheckCacheForKey checks if the key is in the cache, or triggers similarity search if not found.
func CheckCacheForKey(key string, ctx wrapper.HttpContext, config config.PluginConfig, log wrapper.Log, stream bool, useSimilaritySearch bool) error {
	activeCacheProvider := config.GetCacheProvider()
	if activeCacheProvider == nil {
		log.Debug("No cache provider configured, performing similarity search")
		return performSimilaritySearch(key, ctx, config, log, key, stream)
	}

	queryKey := activeCacheProvider.GetCacheKeyPrefix() + key
	log.Debugf("Querying cache with key: %s", queryKey)

	err := activeCacheProvider.Get(queryKey, func(response resp.Value) {
		handleCacheResponse(key, response, ctx, log, stream, config, useSimilaritySearch)
	})

	if err != nil {
		log.Errorf("Failed to retrieve key: %s from cache, error: %v", key, err)
		return err
	}

	return nil
}

// handleCacheResponse processes cache response and handles cache hits and misses.
func handleCacheResponse(key string, response resp.Value, ctx wrapper.HttpContext, log wrapper.Log, stream bool, config config.PluginConfig, useSimilaritySearch bool) {
	if err := response.Error(); err == nil && !response.IsNull() {
		log.Infof("Cache hit for key: %s", key)
		processCacheHit(key, response.String(), stream, ctx, config, log)
		return
	}

	log.Infof("Cache miss for key: %s", key)
	if err := response.Error(); err != nil {
		log.Errorf("Error retrieving key: %s from cache, error: %v", key, err)
	}

	if useSimilaritySearch && config.EnableSemanticCache {
		if err := performSimilaritySearch(key, ctx, config, log, key, stream); err != nil {
			log.Errorf("Failed to perform similarity search for key: %s, error: %v", key, err)
			proxywasm.ResumeHttpRequest()
		}
	} else {
		proxywasm.ResumeHttpRequest()
	}
}

// processCacheHit handles a successful cache hit.
func processCacheHit(key string, response string, stream bool, ctx wrapper.HttpContext, config config.PluginConfig, log wrapper.Log) {
	if stream {
		log.Debug("streaming response is not supported for cache hit yet")
		stream = false
	}
	// escapedResponse, err := json.Marshal(response)
	// log.Debugf("Cached response for key %s: %s", key, escapedResponse)

	// if err != nil {
	// 	handleInternalError(err, "Failed to marshal cached response", log)
	// 	return
	// }
	log.Debugf("Cached response for key %s: %s", key, response)

	ctx.SetContext(CACHE_KEY_CONTEXT_KEY, nil)

	// proxywasm.SendHttpResponseWithDetail(200, "ai-cache.hit", [][2]string{{"content-type", "application/json; charset=utf-8"}}, []byte(fmt.Sprintf(config.ResponseTemplate, escapedResponse)), -1)
	proxywasm.SendHttpResponseWithDetail(200, "ai-cache.hit", [][2]string{{"content-type", "application/json; charset=utf-8"}}, []byte(fmt.Sprintf(config.ResponseTemplate, response)), -1)
}

// performSimilaritySearch determines the appropriate similarity search method to use.
func performSimilaritySearch(key string, ctx wrapper.HttpContext, config config.PluginConfig, log wrapper.Log, queryString string, stream bool) error {
	activeVectorProvider := config.GetVectorProvider()
	if activeVectorProvider == nil {
		return errors.New("no vector provider configured for similarity search")
	}

	// Check if the active vector provider implements the StringQuerier interface.
	if _, ok := activeVectorProvider.(vector.StringQuerier); ok {
		return performStringQuery(key, queryString, ctx, config, log, stream)
	}

	// Check if the active vector provider implements the EmbeddingQuerier interface.
	if _, ok := activeVectorProvider.(vector.EmbeddingQuerier); ok {
		return performEmbeddingQuery(key, ctx, config, log, stream)
	}

	return errors.New("no suitable querier or embedding provider available for similarity search")
}

// performStringQuery executes the string-based similarity search.
func performStringQuery(key string, queryString string, ctx wrapper.HttpContext, config config.PluginConfig, log wrapper.Log, stream bool) error {
	stringQuerier, ok := config.GetVectorProvider().(vector.StringQuerier)
	if !ok {
		return logAndReturnError(log, "active vector provider does not implement StringQuerier interface")
	}

	return stringQuerier.QueryString(queryString, ctx, log, func(results []vector.QueryResult, ctx wrapper.HttpContext, log wrapper.Log, err error) {
		handleQueryResults(key, results, ctx, log, stream, config, err)
	})
}

// performEmbeddingQuery executes the embedding-based similarity search.
func performEmbeddingQuery(key string, ctx wrapper.HttpContext, config config.PluginConfig, log wrapper.Log, stream bool) error {
	embeddingQuerier, ok := config.GetVectorProvider().(vector.EmbeddingQuerier)
	if !ok {
		return logAndReturnError(log, "active vector provider does not implement EmbeddingQuerier interface")
	}

	activeEmbeddingProvider := config.GetEmbeddingProvider()
	if activeEmbeddingProvider == nil {
		return logAndReturnError(log, "no embedding provider configured for similarity search")
	}

	return activeEmbeddingProvider.GetEmbedding(key, ctx, log, func(textEmbedding []float64, err error) {
		if err != nil {
			handleInternalError(err, fmt.Sprintf("Error getting embedding for key: %s", key), log)
			return
		}
		ctx.SetContext(CACHE_KEY_EMBEDDING_KEY, textEmbedding)

		err = embeddingQuerier.QueryEmbedding(textEmbedding, ctx, log, func(results []vector.QueryResult, ctx wrapper.HttpContext, log wrapper.Log, err error) {
			handleQueryResults(key, results, ctx, log, stream, config, err)
		})
		if err != nil {
			handleInternalError(err, fmt.Sprintf("Error querying vector database for key: %s", key), log)
		}
	})
}

// handleQueryResults processes the results of similarity search and determines next actions.
func handleQueryResults(key string, results []vector.QueryResult, ctx wrapper.HttpContext, log wrapper.Log, stream bool, config config.PluginConfig, err error) {
	if err != nil {
		handleInternalError(err, fmt.Sprintf("Error querying vector database for key: %s", key), log)
		return
	}

	if len(results) == 0 {
		log.Warnf("No similar keys found for key: %s", key)
		proxywasm.ResumeHttpRequest()
		return
	}

	mostSimilarData := results[0]
	log.Debugf("For key: %s, the most similar key found: %s with score: %f", key, mostSimilarData.Text, mostSimilarData.Score)

	simThresholdProvider, ok := config.GetVectorProvider().(vector.SimilarityThresholdProvider)
	if !ok {
		handleInternalError(nil, "Active vector provider does not implement SimilarityThresholdProvider interface", log)
		return
	}

	simThreshold := simThresholdProvider.GetSimilarityThreshold()
	if mostSimilarData.Score < simThreshold {
		log.Infof("Key accepted: %s with score: %f below threshold", mostSimilarData.Text, mostSimilarData.Score)
		if mostSimilarData.Answer != "" {
			// direct return the answer if available
			processCacheHit(key, mostSimilarData.Answer, stream, ctx, config, log)
		} else {
			// // otherwise, continue to check cache for the most similar key
			// err = CheckCacheForKey(mostSimilarData.Text, ctx, config, log, stream, false)
			// if err != nil {
			// 	log.Errorf("check cache for key: %s failed, error: %v", mostSimilarData.Text, err)
			// 	proxywasm.ResumeHttpRequest()
			// }

			// Otherwise, do not check the cache, directly return
			log.Warnf("No cache hit for key: %s, however, no answer found in vector database", mostSimilarData.Text)
			proxywasm.ResumeHttpRequest()
		}
	} else {
		log.Infof("Score too high for key: %s with score: %f above threshold", mostSimilarData.Text, mostSimilarData.Score)
		proxywasm.ResumeHttpRequest()
	}
}

// logAndReturnError logs an error and returns it.
func logAndReturnError(log wrapper.Log, message string) error {
	log.Errorf(message)
	return errors.New(message)
}

// handleInternalError logs an error and resumes the HTTP request.
func handleInternalError(err error, message string, log wrapper.Log) {
	if err != nil {
		log.Errorf("%s: %v", message, err)
	} else {
		log.Errorf(message)
	}
	// proxywasm.SendHttpResponse(500, [][2]string{{"content-type", "text/plain"}}, []byte("Internal Server Error"), -1)
	proxywasm.ResumeHttpRequest()
}

// Caches the response value
func cacheResponse(ctx wrapper.HttpContext, config config.PluginConfig, key string, value string, log wrapper.Log) {
	activeCacheProvider := config.GetCacheProvider()
	if activeCacheProvider != nil {
		queryKey := activeCacheProvider.GetCacheKeyPrefix() + key
		log.Infof("[onHttpResponseBody] setting cache to redis, key: %s, value: %s", queryKey, value)
		_ = activeCacheProvider.Set(queryKey, value, nil)
	}
}

// Handles embedding upload if available
func uploadEmbeddingAndAnswer(ctx wrapper.HttpContext, config config.PluginConfig, key string, value string, log wrapper.Log) {
	embedding := ctx.GetContext(CACHE_KEY_EMBEDDING_KEY)
	if embedding == nil {
		return
	}

	emb, ok := embedding.([]float64)
	if !ok {
		log.Errorf("[onHttpResponseBody] embedding is not of expected type []float64")
		return
	}

	activeVectorProvider := config.GetVectorProvider()
	if activeVectorProvider == nil {
		log.Debug("[onHttpResponseBody] no vector provider configured for uploading embedding")
		return
	}

	// Attempt to upload answer embedding first
	if ansEmbUploader, ok := activeVectorProvider.(vector.AnswerAndEmbeddingUploader); ok {
		log.Infof("[onHttpResponseBody] uploading answer embedding for key: %s", key)
		err := ansEmbUploader.UploadAnswerAndEmbedding(key, emb, value, ctx, log, nil)
		if err != nil {
			log.Warnf("[onHttpResponseBody] failed to upload answer embedding for key: %s, error: %v", key, err)
		} else {
			return // If successful, return early
		}
	}

	// If answer embedding upload fails, attempt normal embedding upload
	if embUploader, ok := activeVectorProvider.(vector.EmbeddingUploader); ok {
		log.Infof("[onHttpResponseBody] uploading embedding for key: %s", key)
		err := embUploader.UploadEmbedding(key, emb, ctx, log, nil)
		if err != nil {
			log.Warnf("[onHttpResponseBody] failed to upload embedding for key: %s, error: %v", key, err)
		}
	}
}
