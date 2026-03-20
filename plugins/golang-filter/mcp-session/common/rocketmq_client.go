package common

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	rocketmq "github.com/apache/rocketmq-clients/golang/v5"
	"github.com/apache/rocketmq-clients/golang/v5/credentials"
	"github.com/envoyproxy/envoy/contrib/golang/common/go/api"
)

type RocketMQConfig struct {
	endpoint  string
	accessKey string
	secretKey string
	namespace string
	topic     string
	group     string
}

// ParseRocketMQConfig parses RocketMQ configuration from a map
func ParseRocketMQConfig(config map[string]interface{}) (*RocketMQConfig, error) {
	c := &RocketMQConfig{}

	// endpoint is required
	if endpoint, ok := config["endpoint"].(string); ok && endpoint != "" {
		c.endpoint = endpoint
	} else {
		return nil, fmt.Errorf("endpoint is required and must be a non-empty string")
	}

	// accessKey is required
	if accessKey, ok := config["access_key"].(string); ok && accessKey != "" {
		c.accessKey = accessKey
	} else {
		return nil, fmt.Errorf("access_key is required and must be a non-empty string")
	}

	// secretKey is required
	if secretKey, ok := config["secret_key"].(string); ok && secretKey != "" {
		c.secretKey = secretKey
	} else {
		return nil, fmt.Errorf("secret_key is required and must be a non-empty string")
	}

	// namespace is optional
	if namespace, ok := config["namespace"].(string); ok {
		c.namespace = namespace
	}

	// topic is optional, default to "higress-mcp-topic"
	if topic, ok := config["topic"].(string); ok {
		c.topic = topic
	} else {
		c.topic = "higress-mcp-topic"
	}

	// group is optional, default to "higress-mcp-group"
	if group, ok := config["group"].(string); ok {
		c.group = group
	} else {
		c.group = "higress-mcp-group"
	}

	return c, nil
}

// MessageCallback is the type for message callbacks
type MessageCallback func(message string)

// RocketMQClient is a struct to handle RocketMQ connections and operations
type RocketMQClient struct {
	config    *RocketMQConfig
	producer  rocketmq.Producer
	consumer  rocketmq.LitePushConsumer
	ctx       context.Context
	cancel    context.CancelFunc
	callbacks *sync.Map // map[string]func(message string)
}

// normalizeLiteTopic normalizes a liteTopic string by replacing invalid characters with underscores (_).
// RocketMQ requires liteTopic to contain only alphanumeric characters, hyphens (-), and underscores (_).
func normalizeLiteTopic(topic string) string {
	// Replace colon ':' with underscore '_' to comply with RocketMQ Lite Topic restrictions
	return strings.ReplaceAll(topic, ":", "_")
}

// Ensure RocketMQClient implements MsgPubSub interface
var _ MsgPubSub = (*RocketMQClient)(nil)

// NewRocketMQClient creates a new RocketMQClient instance and establishes connections to the RocketMQ server
func NewRocketMQClient(config *RocketMQConfig) (*RocketMQClient, error) {
	creds := &credentials.SessionCredentials{
		AccessKey:    config.accessKey,
		AccessSecret: config.secretKey,
	}

	// log to console
	os.Setenv(rocketmq.ENABLE_CONSOLE_APPENDER, "true")
	os.Setenv(rocketmq.CLIENT_LOG_LEVEL, "warn")
	rocketmq.InitLogger()
	rocketmq.EnableSsl = false

	// Create producer
	prod, err := rocketmq.NewProducer(&rocketmq.Config{
		Endpoint:    config.endpoint,
		Credentials: creds,
		NameSpace:   config.namespace,
	}, rocketmq.WithTopics(config.topic))
	if err != nil {
		api.LogErrorf("Failed to create RocketMQ producer: %v", err)
		return nil, fmt.Errorf("failed to create RocketMQ producer: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create callbacks map before creating consumer
	callbacks := &sync.Map{}

	// Create message listener with the callbacks map
	messageListener := &rocketmq.FuncMessageListener{
		Consume: func(mv *rocketmq.MessageView) rocketmq.ConsumerResult {
			liteTopic := mv.GetLiteTopic()
			body := string(mv.GetBody())

			// Call registered callback for this lite topic
			if callback, ok := callbacks.Load(liteTopic); ok {
				if cb, ok := callback.(func(message string)); ok {
					go func(callback func(message string), body string) {
						defer func() {
							if r := recover(); r != nil {
								api.LogErrorf("Recovered from panic in message callback for liteTopic %s: %v", liteTopic, r)
							}
						}()
						callback(body)
					}(cb, body)
				}
			}

			return rocketmq.SUCCESS
		},
	}

	consumer, err := rocketmq.NewLitePushConsumer(&rocketmq.Config{
		Endpoint:      config.endpoint,
		Credentials:   creds,
		ConsumerGroup: config.group,
		NameSpace:     config.namespace,
	},
		rocketmq.NewLitePushConsumerConfig(config.topic, time.Second*30),
		rocketmq.WithPushAwaitDuration(time.Second*5),
		rocketmq.WithPushMessageListener(messageListener),
		rocketmq.WithPushConsumptionThreadCount(8),
		rocketmq.WithPushMaxCacheMessageCount(256),
	)
	if err != nil {
		api.LogErrorf("Failed to create RocketMQ consumer: %v", err)
		return nil, fmt.Errorf("failed to create RocketMQ consumer: %w", err)
	}

	client := &RocketMQClient{
		config:    config,
		producer:  prod,
		consumer:  consumer,
		ctx:       ctx,
		cancel:    cancel,
		callbacks: callbacks,
	}

	// Start producer and consumer
	if err := client.producer.Start(); err != nil {
		api.LogErrorf("Failed to start RocketMQ producer: %v", err)
	}

	if err := client.consumer.Start(); err != nil {
		api.LogErrorf("Failed to start RocketMQ consumer: %v", err)
	}

	return client, nil
}

// Publish publishes a message to a RocketMQ liteTopic
func (r *RocketMQClient) Publish(liteTopic string, message string) error {
	// Normalize the liteTopic to comply with RocketMQ Lite Topic restrictions
	normalizedTopic := normalizeLiteTopic(liteTopic)

	msg := &rocketmq.Message{
		Topic: r.config.topic,
		Body:  []byte(message),
	}
	msg.SetLiteTopic(normalizedTopic)

	receipts, err := r.producer.Send(r.ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	if len(receipts) == 0 {
		return fmt.Errorf("no receipts received")
	}

	api.LogDebugf("Published message to topic %s, message ID: %s", r.config.topic, receipts[0].MessageID)
	return nil
}

// Subscribe subscribes to a RocketMQ liteTopic and processes messages
// Implements MsgPubSub interface
func (r *RocketMQClient) Subscribe(liteTopic string, stopChan chan struct{}, callback func(message string)) error {
	// Normalize the liteTopic to comply with RocketMQ Lite Topic restrictions
	normalizedTopic := normalizeLiteTopic(liteTopic)

	// Register the callback for this normalized topic
	if err := r.registerCallback(normalizedTopic, callback); err != nil {
		return fmt.Errorf("failed to register callback for liteTopic %s: %w", liteTopic, err)
	}

	// Start a goroutine to monitor the stopChan
	go func() {
		defer func() {
			if r := recover(); r != nil {
				api.LogErrorf("RocketMQ Subscribe recovered from panic: %v", r)
			}
		}()

		select {
		case <-stopChan:
			api.LogDebugf("Stopping subscription to liteTopic %s", liteTopic)
			if err := r.unregisterCallback(normalizedTopic); err != nil {
				api.LogErrorf("Failed to unsubscribe from liteTopic %s: %v", liteTopic, err)
			}
		case <-r.ctx.Done():
			api.LogDebugf("Context cancelled, stopping subscription to liteTopic %s", liteTopic)
		}
	}()

	return nil
}

// registerCallback registers a message callback for a specific topic
func (r *RocketMQClient) registerCallback(liteTopic string, callback func(message string)) error {
	r.callbacks.Store(liteTopic, callback)
	if err := r.consumer.SubscribeLite(liteTopic); err != nil {
		r.callbacks.Delete(liteTopic)
		api.LogErrorf("Failed to subscribe to liteTopic %s: %v", liteTopic, err)
		return fmt.Errorf("failed to subscribe to liteTopic %s: %w", liteTopic, err)
	}
	return nil
}

// unregisterCallback removes a message callback for a specific topic
func (r *RocketMQClient) unregisterCallback(liteTopic string) error {
	r.callbacks.Delete(liteTopic)
	if err := r.consumer.UnSubscribeLite(liteTopic); err != nil {
		api.LogErrorf("Failed to unsubscribe from liteTopic %s: %v", liteTopic, err)
		return fmt.Errorf("failed to unsubscribe from liteTopic %s: %w", liteTopic, err)
	}
	return nil
}

// Close closes the RocketMQ client and stops the producer and consumer
func (r *RocketMQClient) Close() error {
	r.cancel()
	if err := r.producer.GracefulStop(); err != nil {
		api.LogErrorf("Error stopping RocketMQ producer: %v", err)
	}
	if err := r.consumer.GracefulStop(); err != nil {
		api.LogErrorf("Error stopping RocketMQ consumer: %v", err)
	}
	return nil
}
