package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/go-redis/redis/v8"
)

type RedisConfig struct {
	Address  string
	Username string
	Password string
	DB       int
}

func ParseRedisConfig(config map[string]any) (*RedisConfig, error) {
	c := &RedisConfig{}

	// address is required
	addr, ok := config["address"].(string)
	if !ok {
		return nil, fmt.Errorf("address is required and must be a string")
	}
	c.Address = addr

	// username is optional
	if username, ok := config["username"].(string); ok {
		c.Username = username
	}

	// password is optional
	if password, ok := config["password"].(string); ok {
		c.Password = password
	}

	// db is optional, default to 0
	if db, ok := config["db"].(int); ok {
		c.DB = db
	}

	return c, nil
}

// RedisClient is a struct to handle Redis connections and operations
type RedisClient struct {
	client *redis.Client
	ctx    context.Context
	cancel context.CancelFunc
	config *RedisConfig
}

// NewRedisClient creates a new RedisClient instance and establishes a connection to the Redis server
func NewRedisClient(config *RedisConfig) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Address,
		Username: config.Username,
		Password: config.Password,
		DB:       config.DB,
	})

	// Ping the Redis server to check the connection
	pong, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	api.LogDebugf("Connected to Redis: %s", pong)

	ctx, cancel := context.WithCancel(context.Background())
	redisClient := &RedisClient{
		client: client,
		ctx:    ctx,
		cancel: cancel,
		config: config,
	}

	// Start keep-alive check
	go redisClient.keepAlive()

	return redisClient, nil
}

// keepAlive periodically checks Redis connection and attempts to reconnect if needed
func (r *RedisClient) keepAlive() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			if err := r.checkConnection(); err != nil {
				api.LogErrorf("Redis connection check failed: %v", err)
				if err := r.reconnect(); err != nil {
					api.LogErrorf("Failed to reconnect to Redis: %v", err)
				}
			}
		}
	}
}

// checkConnection verifies if the Redis connection is still alive
func (r *RedisClient) checkConnection() error {
	_, err := r.client.Ping(r.ctx).Result()
	return err
}

// reconnect attempts to establish a new connection to Redis
func (r *RedisClient) reconnect() error {
	// Close the old client
	if err := r.client.Close(); err != nil {
		api.LogErrorf("Error closing old Redis connection: %v", err)
	}

	// Create new client
	r.client = redis.NewClient(&redis.Options{
		Addr:     r.config.Address,
		Username: r.config.Username,
		Password: r.config.Password,
		DB:       r.config.DB,
	})

	// Test the new connection
	if err := r.checkConnection(); err != nil {
		return fmt.Errorf("failed to reconnect to Redis: %w", err)
	}

	api.LogDebugf("Successfully reconnected to Redis")
	return nil
}

// Publish publishes a message to a Redis channel
func (r *RedisClient) Publish(channel string, message string) error {
	err := r.client.Publish(r.ctx, channel, message).Err()
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}
	return nil
}

// Subscribe subscribes to a Redis channel and processes messages
func (r *RedisClient) Subscribe(channel string, stopChan chan struct{}, callback func(message string)) error {
	pubsub := r.client.Subscribe(r.ctx, channel)
	_, err := pubsub.Receive(r.ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to channel: %w", err)
	}

	go func() {
		defer func() {
			pubsub.Close()
			api.LogDebugf("Closed subscription to channel %s", channel)
		}()

		ch := pubsub.Channel()
		for {
			select {
			case <-stopChan:
				api.LogDebugf("Stopping subscription to channel %s", channel)
				return
			case msg, ok := <-ch:
				if !ok {
					api.LogDebugf("Redis subscription channel closed for %s", channel)
					return
				}

				func() {
					defer func() {
						if r := recover(); r != nil {
							api.LogErrorf("Recovered from panic in callback: %v", r)
						}
					}()
					callback(msg.Payload)
				}()
			}
		}
	}()

	return nil
}

// Set sets the value of a key in Redis
func (r *RedisClient) Set(key string, value string, expiration time.Duration) error {
	err := r.client.Set(r.ctx, key, value, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to set key: %w", err)
	}
	return nil
}

// Get retrieves the value of a key from Redis
func (r *RedisClient) Get(key string) (string, error) {
	val, err := r.client.Get(r.ctx, key).Result()
	if err == redis.Nil {
		return "", fmt.Errorf("key does not exist")
	} else if err != nil {
		return "", fmt.Errorf("failed to get key: %w", err)
	}
	return val, nil
}

// Close closes the Redis client and stops the keepalive goroutine
func (r *RedisClient) Close() error {
	r.cancel()
	return r.client.Close()
}
