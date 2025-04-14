package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/internal"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

type MCPRatelimitHandler struct {
	redisClient *internal.RedisClient
	callbacks   api.FilterCallbackHandler
	limit       int64 // Maximum requests allowed per window
	window      int64 // Time window in seconds
}

// MCPRatelimitConfig is the configuration for the rate limit handler
type MCPRatelimitConfig struct {
	Limit  int64 `json:"limit"`
	Window int64 `json:"window"`
}

// NewMCPRatelimitHandler creates a new rate limit handler
func NewMCPRatelimitHandler(redisClient *internal.RedisClient, callbacks api.FilterCallbackHandler, conf *MCPRatelimitConfig) *MCPRatelimitHandler {
	if conf == nil {
		conf = &MCPRatelimitConfig{
			Limit:  100,
			Window: int64(24 * time.Hour), // 24 hours in seconds
		}
	}
	return &MCPRatelimitHandler{
		redisClient: redisClient,
		callbacks:   callbacks,
		limit:       conf.Limit,
		window:      conf.Window,
	}
}

const (
	// Lua script for rate limiting
	LimitScript = `
        local ttl = redis.call('ttl', KEYS[1])
        if ttl < 0 then
            redis.call('set', KEYS[1], ARGV[1] - 1, 'EX', ARGV[2])
            return {ARGV[1], ARGV[1] - 1, ARGV[2]}
        end
        local remaining = redis.call('incrby', KEYS[1], -1)
        return {ARGV[1], remaining, ttl}
    `
)

type LimitContext struct {
	Count     int64 // Current request count
	Remaining int64 // Remaining requests allowed
	Reset     int64 // Time until reset in seconds
}

func (h *MCPRatelimitHandler) HandleRatelimit(path string, method string, body []byte) bool {
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return false
	}
	serverName := parts[1]
	uid := parts[2]

	// Build rate limit key using serverName, uid, window and limit
	limitKey := fmt.Sprintf("mcp-server-limit:%s:%s:%d:%d", serverName, uid, h.window, h.limit)
	keys := []string{limitKey}

	args := []interface{}{h.limit, h.window}

	result, err := h.redisClient.Eval(LimitScript, 1, keys, args)
	if err != nil {
		api.LogErrorf("Failed to check rate limit: %v", err)
		h.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusInternalServerError, "", nil, 0, "")
		return false
	}

	// Process response
	resultArray, ok := result.([]interface{})
	if !ok || len(resultArray) != 3 {
		api.LogErrorf("Invalid response format: %v", result)
		h.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusInternalServerError, "", nil, 0, "")
		return false
	}

	context := LimitContext{
		Count:     resultArray[0].(int64),
		Remaining: resultArray[1].(int64),
		Reset:     resultArray[2].(int64),
	}

	if context.Remaining < 0 {
		h.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusTooManyRequests, "", nil, 0, "")
		return false
	}

	return true
}
