package handler

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/alibaba/higress/plugins/golang-filter/mcp-session/common"
)

const (
	configExpiry = 7 * 24 * time.Hour
)

// GetConfigStoreKey returns the Redis channel name for the given session ID
func GetConfigStoreKey(serverName string, uid string) string {
	return fmt.Sprintf("mcp-server-config:%s:%s", serverName, uid)
}

// ConfigResponse represents the response structure for configuration operations
type ConfigResponse struct {
	Success bool `json:"success"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ConfigStore defines the interface for configuration storage operations
type ConfigStore interface {
	// StoreConfig stores user configuration
	StoreConfig(serverName string, uid string, config map[string]string) (*ConfigResponse, error)
	// GetConfig retrieves user configuration
	GetConfig(serverName string, uid string) (map[string]string, error)
}

// RedisConfigStore implements configuration storage using Redis
type RedisConfigStore struct {
	redisClient *common.RedisClient
}

// NewRedisConfigStore creates a new instance of Redis configuration storage
func NewRedisConfigStore(redisClient *common.RedisClient) ConfigStore {
	return &RedisConfigStore{
		redisClient: redisClient,
	}
}

// StoreConfig stores configuration in Redis
func (s *RedisConfigStore) StoreConfig(serverName string, uid string, config map[string]string) (*ConfigResponse, error) {
	key := GetConfigStoreKey(serverName, uid)

	// Convert config to JSON
	configBytes, err := json.Marshal(config)
	if err != nil {
		return &ConfigResponse{
			Success: false,
			Error: &struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}{
				Code:    "MARSHAL_ERROR",
				Message: "Failed to marshal configuration",
			},
		}, err
	}

	// Store in Redis with expiry
	err = s.redisClient.Set(key, string(configBytes), configExpiry)
	if err != nil {
		return &ConfigResponse{
			Success: false,
			Error: &struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}{
				Code:    "REDIS_ERROR",
				Message: "Failed to store configuration in Redis",
			},
		}, err
	}

	return &ConfigResponse{
		Success: true,
	}, nil
}

// GetConfig retrieves configuration from Redis
func (s *RedisConfigStore) GetConfig(serverName string, uid string) (map[string]string, error) {
	key := GetConfigStoreKey(serverName, uid)

	// Get from Redis
	value, err := s.redisClient.Get(key)
	if err != nil {
		return nil, err
	}

	// Parse JSON
	var config map[string]string
	if err := json.Unmarshal([]byte(value), &config); err != nil {
		return nil, err
	}

	// Refresh TTL
	if err := s.redisClient.Expire(key, configExpiry); err != nil {
		// Log error but don't fail the request
		fmt.Printf("Failed to refresh TTL for key %s: %v\n", key, err)
	}

	return config, nil
}
