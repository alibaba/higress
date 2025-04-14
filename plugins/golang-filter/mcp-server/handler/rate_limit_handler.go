package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-server/internal"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

type MCPRatelimitHandler struct {
	redisClient *internal.RedisClient
	callbacks   api.FilterCallbackHandler
	limit       int      // Maximum requests allowed per window
	window      int      // Time window in seconds
	whitelist   []string // Whitelist of UIDs that bypass rate limiting
}

// MCPRatelimitConfig is the configuration for the rate limit handler
type MCPRatelimitConfig struct {
	Limit     int      `json:"limit"`
	Window    int      `json:"window"`
	Whitelist []string `json:"white_list"` // List of UIDs that bypass rate limiting
}

// NewMCPRatelimitHandler creates a new rate limit handler
func NewMCPRatelimitHandler(redisClient *internal.RedisClient, callbacks api.FilterCallbackHandler, conf *MCPRatelimitConfig) *MCPRatelimitHandler {
	if conf == nil {
		conf = &MCPRatelimitConfig{
			Limit:     100,
			Window:    int(24 * time.Hour / time.Second), // 24 hours in seconds
			Whitelist: []string{},
		}
	}
	return &MCPRatelimitHandler{
		redisClient: redisClient,
		callbacks:   callbacks,
		limit:       conf.Limit,
		window:      conf.Window,
		whitelist:   conf.Whitelist,
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
    	return {ARGV[1], redis.call('incrby', KEYS[1], -1), ttl}
    `
)

type LimitContext struct {
	Count     int // Current request count
	Remaining int // Remaining requests allowed
	Reset     int // Time until reset in seconds
}

func (h *MCPRatelimitHandler) HandleRatelimit(path string, method string, body []byte) bool {
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		h.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusForbidden, "", nil, 0, "")
		return false
	}
	serverName := parts[1]
	uid := parts[2]

	// Check if the UID is in whitelist
	for _, whitelistedUID := range h.whitelist {
		if whitelistedUID == uid {
			return true // Bypass rate limiting for whitelisted UIDs
		}
	}

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
		Count:     parseRedisValue(resultArray[0]),
		Remaining: parseRedisValue(resultArray[1]),
		Reset:     parseRedisValue(resultArray[2]),
	}

	if context.Remaining < 0 {
		h.callbacks.DecoderFilterCallbacks().SendLocalReply(http.StatusTooManyRequests, "", nil, 0, "")
		return false
	}

	return true
}

// parseRedisValue converts the value from Redis to an int
func parseRedisValue(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case string:
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return 0
}
