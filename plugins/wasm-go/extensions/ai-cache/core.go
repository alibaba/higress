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
func CheckCacheForKey(key string, ctx wrapper.HttpContext, c config.PluginConfig, log wrapper.Log, stream bool, useSimilaritySearch bool) error {
	activeCacheProvider := c.GetCacheProvider()
	if activeCacheProvider == nil {
		log.Debug("No cache provider configured, performing similarity search")
		return performSimilaritySearch(key, ctx, c, log, key, stream)
	}

	queryKey := activeCacheProvider.GetCacheKeyPrefix() + key
	log.Debugf("Querying cache with key: %s", queryKey)

	err := activeCacheProvider.Get(queryKey, func(response resp.Value) {
		handleCacheResponse(key, response, ctx, log, stream, c, useSimilaritySearch)
	})

	if err != nil {
		log.Errorf("Failed to retrieve key: %s from cache, error: %v", key, err)
		return err
	}

	return nil
}

// handleCacheResponse processes cache response and handles cache hits and misses.
func handleCacheResponse(key string, response resp.Value, ctx wrapper.HttpContext, log wrapper.Log, stream bool, c config.PluginConfig, useSimilaritySearch bool) {
	if err := response.Error(); err == nil && !response.IsNull() {
		log.Infof("Cache hit for key: %s", key)
		processCacheHit(key, response.String(), stream, ctx, c, log)
		return
	}

	log.Infof("Cache miss for key: %s", key)
	if err := response.Error(); err != nil {
		log.Errorf("Error retrieving key: %s from cache, error: %v", key, err)
	}

	if useSimilaritySearch && c.EnableSemanticCache {
		if err := performSimilaritySearch(key, ctx, c, log, key, stream); err != nil {
			log.Errorf("Failed to perform similarity search for key: %s, error: %v", key, err)
			proxywasm.ResumeHttpRequest()
		}
	} else {
		proxywasm.ResumeHttpRequest()
	}
}

// processCacheHit handles a successful cache hit.
func processCacheHit(key string, response string, stream bool, ctx wrapper.HttpContext, c config.PluginConfig, log wrapper.Log) {
	log.Debugf("Cached response for key %s: %s", key, response)

	ctx.SetContext(CACHE_KEY_CONTEXT_KEY, nil)

	if stream {
		proxywasm.SendHttpResponseWithDetail(200, "ai-cache.hit", [][2]string{{"content-type", "application/json; charset=utf-8"}}, []byte(fmt.Sprintf(c.StreamResponseTemplate, response)), -1)
	} else {
		proxywasm.SendHttpResponseWithDetail(200, "ai-cache.hit", [][2]string{{"content-type", "application/json; charset=utf-8"}}, []byte(fmt.Sprintf(c.ResponseTemplate, response)), -1)
	}
}

// performSimilaritySearch determines the appropriate similarity search method to use.
func performSimilaritySearch(key string, ctx wrapper.HttpContext, c config.PluginConfig, log wrapper.Log, queryString string, stream bool) error {
	activeVectorProvider := c.GetVectorProvider()
	if activeVectorProvider == nil {
		return errors.New("no vector provider configured for similarity search")
	}

	// Check if the active vector provider implements the StringQuerier interface.
	if _, ok := activeVectorProvider.(vector.StringQuerier); ok {
		log.Debugf("[%s] [performSimilaritySearch] active vector provider implements StringQuerier interface, performing string query", PLUGIN_NAME)
		return performStringQuery(key, queryString, ctx, c, log, stream)
	}

	// Check if the active vector provider implements the EmbeddingQuerier interface.
	if _, ok := activeVectorProvider.(vector.EmbeddingQuerier); ok {
		log.Debugf("[%s] [performSimilaritySearch] active vector provider implements EmbeddingQuerier interface, performing embedding query", PLUGIN_NAME)
		return performEmbeddingQuery(key, ctx, c, log, stream)
	}

	return errors.New("no suitable querier or embedding provider available for similarity search")
}

// performStringQuery executes the string-based similarity search.
func performStringQuery(key string, queryString string, ctx wrapper.HttpContext, c config.PluginConfig, log wrapper.Log, stream bool) error {
	stringQuerier, ok := c.GetVectorProvider().(vector.StringQuerier)
	if !ok {
		return logAndReturnError(log, "active vector provider does not implement StringQuerier interface")
	}

	return stringQuerier.QueryString(queryString, ctx, log, func(results []vector.QueryResult, ctx wrapper.HttpContext, log wrapper.Log, err error) {
		handleQueryResults(key, results, ctx, log, stream, c, err)
	})
}

// performEmbeddingQuery executes the embedding-based similarity search.
func performEmbeddingQuery(key string, ctx wrapper.HttpContext, c config.PluginConfig, log wrapper.Log, stream bool) error {
	embeddingQuerier, ok := c.GetVectorProvider().(vector.EmbeddingQuerier)
	if !ok {
		return logAndReturnError(log, "active vector provider does not implement EmbeddingQuerier interface")
	}

	activeEmbeddingProvider := c.GetEmbeddingProvider()
	if activeEmbeddingProvider == nil {
		return logAndReturnError(log, "no embedding provider configured for similarity search")
	}

	return activeEmbeddingProvider.GetEmbedding(key, ctx, log, func(textEmbedding []float64, err error) {
		log.Debugf("[%s] [performEmbeddingQuery] GetEmbedding success, length of embedding: %d, error: %v", PLUGIN_NAME, len(textEmbedding), err)
		if err != nil {
			handleInternalError(err, fmt.Sprintf("Error getting embedding for key: %s", key), log)
			return
		}
		ctx.SetContext(CACHE_KEY_EMBEDDING_KEY, textEmbedding)

		err = embeddingQuerier.QueryEmbedding(textEmbedding, ctx, log, func(results []vector.QueryResult, ctx wrapper.HttpContext, log wrapper.Log, err error) {
			handleQueryResults(key, results, ctx, log, stream, c, err)
		})
		if err != nil {
			handleInternalError(err, fmt.Sprintf("Error querying vector database for key: %s", key), log)
		}
	})
}

// handleQueryResults processes the results of similarity search and determines next actions.
func handleQueryResults(key string, results []vector.QueryResult, ctx wrapper.HttpContext, log wrapper.Log, stream bool, c config.PluginConfig, err error) {
	if err != nil {
		handleInternalError(err, fmt.Sprintf("Error querying vector database for key: %s", key), log)
		return
	}

	if len(results) == 0 {
		log.Warnf("[%s] [handleQueryResults] No similar keys found for key: %s", PLUGIN_NAME, key)
		proxywasm.ResumeHttpRequest()
		return
	}

	mostSimilarData := results[0]
	log.Debugf("[%s] [handleQueryResults] For key: %s, the most similar key found: %s with score: %f", PLUGIN_NAME, key, mostSimilarData.Text, mostSimilarData.Score)
	simThreshold := c.GetVectorProviderConfig().Threshold
	simThresholdRelation := c.GetVectorProviderConfig().ThresholdRelation
	if compare(simThresholdRelation, mostSimilarData.Score, simThreshold) {
		log.Infof("[%s] [handleQueryResults] Key accepted: %s with score: %f below threshold", PLUGIN_NAME, mostSimilarData.Text, mostSimilarData.Score)
		if mostSimilarData.Answer != "" {
			// direct return the answer if available
			processCacheHit(key, mostSimilarData.Answer, stream, ctx, c, log)
		} else {
			// // otherwise, continue to check cache for the most similar key
			// err = CheckCacheForKey(mostSimilarData.Text, ctx, config, log, stream, false)
			// if err != nil {
			// 	log.Errorf("check cache for key: %s failed, error: %v", mostSimilarData.Text, err)
			// 	proxywasm.ResumeHttpRequest()
			// }

			// Otherwise, do not check the cache, directly return
			log.Warnf("[%s] [handleQueryResults] Cache hit for key: %s, but no corresponding answer found in the vector database", PLUGIN_NAME, mostSimilarData.Text)
			proxywasm.ResumeHttpRequest()
		}
	} else {
		log.Infof("[%s] [handleQueryResults] Score not meet the threshold %f: %s with score %f", PLUGIN_NAME, simThreshold, mostSimilarData.Text, mostSimilarData.Score)
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
func cacheResponse(ctx wrapper.HttpContext, c config.PluginConfig, key string, value string, log wrapper.Log) {
	activeCacheProvider := c.GetCacheProvider()
	if activeCacheProvider != nil {
		queryKey := activeCacheProvider.GetCacheKeyPrefix() + key
		log.Infof("[%s] [cacheResponse] setting cache to redis, key: %s, value: %s", PLUGIN_NAME, queryKey, value)
		_ = activeCacheProvider.Set(queryKey, value, nil)
	}
}

// Handles embedding upload if available
func uploadEmbeddingAndAnswer(ctx wrapper.HttpContext, c config.PluginConfig, key string, value string, log wrapper.Log) {
	embedding := ctx.GetContext(CACHE_KEY_EMBEDDING_KEY)
	if embedding == nil {
		return
	}

	emb, ok := embedding.([]float64)
	if !ok {
		log.Errorf("[%s] [uploadEmbeddingAndAnswer] embedding is not of expected type []float64", PLUGIN_NAME)
		return
	}

	activeVectorProvider := c.GetVectorProvider()
	if activeVectorProvider == nil {
		log.Debugf("[%s] [uploadEmbeddingAndAnswer] no vector provider configured for uploading embedding", PLUGIN_NAME)
		return
	}

	// Attempt to upload answer embedding first
	if ansEmbUploader, ok := activeVectorProvider.(vector.AnswerAndEmbeddingUploader); ok {
		log.Infof("[%s] [uploadEmbeddingAndAnswer] uploading answer embedding for key: %s", PLUGIN_NAME, key)
		err := ansEmbUploader.UploadAnswerAndEmbedding(key, emb, value, ctx, log, nil)
		if err != nil {
			log.Warnf("[%s] [uploadEmbeddingAndAnswer] failed to upload answer embedding for key: %s, error: %v", PLUGIN_NAME, key, err)
		} else {
			return // If successful, return early
		}
	}

	// If answer embedding upload fails, attempt normal embedding upload
	if embUploader, ok := activeVectorProvider.(vector.EmbeddingUploader); ok {
		log.Infof("[%s] [uploadEmbeddingAndAnswer] uploading embedding for key: %s", PLUGIN_NAME, key)
		err := embUploader.UploadEmbedding(key, emb, ctx, log, nil)
		if err != nil {
			log.Warnf("[uploadEmbeddingAndAnswer] failed to upload embedding for key: %s, error: %v",key, err)
		}
	}
}

// 主要用于相似度/距离/点积判断
// 余弦相似度度量的是两个向量在方向上的相似程度。相似度越高，两个向量越接近。
// 距离度量的是两个向量在空间上的远近程度。距离越小，两个向量越接近。
// compare 函数根据操作符进行判断并返回结果
func compare(operator string, value1 float64, value2 float64) bool {
	switch operator {
	case "gt":
		return value1 > value2
	case "gte":
		return value1 >= value2
	case "lt":
		return value1 < value2
	case "lte":
		return value1 <= value2
	default:
		return false
	}
}
