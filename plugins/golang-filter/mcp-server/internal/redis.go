package internal

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
	"github.com/go-redis/redis/v8"
)

// RedisClient is a struct to handle Redis connections and operations
type RedisClient struct {
	client   *redis.Client
	ctx      context.Context
	stopChan chan struct{}
}

// NewRedisClient creates a new RedisClient instance and establishes a connection to the Redis server
func NewRedisClient(address string, stopChan chan struct{}) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	// Ping the Redis server to check the connection
	pong, err := client.Ping(context.Background()).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	log.Printf("Connected to Redis: %s", pong)

	return &RedisClient{
		client:   client,
		ctx:      context.Background(),
		stopChan: stopChan,
	}, nil
}

// TODO: redis keep alive check
// TODO: redis pub sub memory limit
// Publish publishes a message to a Redis channel
func (r *RedisClient) Publish(channel string, message string) error {
	err := r.client.Publish(r.ctx, channel, message).Err()
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}
	return nil
}

// Subscribe subscribes to a Redis channel and processes messages
func (r *RedisClient) Subscribe(channel string, callback func(message string)) error {
	pubsub := r.client.Subscribe(r.ctx, channel)
	_, err := pubsub.Receive(r.ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to channel: %w", err)
	}

	go func() {
		defer pubsub.Close()
		for {
			select {
			case <-r.stopChan:
				api.LogDebugf("Stopping subscription to channel %s", channel)
				return
			default:
				msg, err := pubsub.ReceiveMessage(r.ctx)
				if err != nil {
					log.Printf("Error receiving message: %v", err)
					return
				}
				callback(msg.Payload)
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
