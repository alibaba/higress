package provider

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestParseContextCompressionConfig(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr  string
		expected *ContextCompressionConfig
	}{
		{
			name:     "disabled compression",
			jsonStr:  `{"enabled": false}`,
			expected: &ContextCompressionConfig{Enabled: false},
		},
		{
			name: "enabled with redis config",
			jsonStr: `{
				"enabled": true,
				"redis": {
					"serviceName": "redis.static",
					"servicePort": 6379,
					"password": "123456",
					"database": 0
				},
				"compressionBytesThreshold": 1000,
				"memoryTTL": 3600
			}`,
			expected: &ContextCompressionConfig{
				Enabled: true,
				Redis: &RedisConfig{
					ServiceName: "redis.static",
					ServicePort: 6379,
					Password:    "123456",
					Database:    0,
				},
				CompressionBytesThreshold: 1000,
				MemoryTTL:                 3600,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			json := gjson.Parse(tt.jsonStr)
			config := ParseContextCompressionConfig(json)

			if config == nil {
				t.Fatalf("expected non-nil config")
			}

			if config.Enabled != tt.expected.Enabled {
				t.Errorf("Enabled: got %v, want %v", config.Enabled, tt.expected.Enabled)
			}

			if tt.expected.Redis != nil {
				if config.Redis == nil {
					t.Fatal("expected non-nil Redis config")
				}
				if config.Redis.ServiceName != tt.expected.Redis.ServiceName {
					t.Errorf("ServiceName: got %s, want %s", config.Redis.ServiceName, tt.expected.Redis.ServiceName)
				}
				if config.Redis.ServicePort != tt.expected.Redis.ServicePort {
					t.Errorf("ServicePort: got %d, want %d", config.Redis.ServicePort, tt.expected.Redis.ServicePort)
				}
			}
		})
	}
}

func TestGenerateContextId(t *testing.T) {
	// Test that we can generate unique IDs
	id1, err1 := generateContextId()
	id2, err2 := generateContextId()

	if err1 != nil {
		t.Fatalf("failed to generate first context ID: %v", err1)
	}
	if err2 != nil {
		t.Fatalf("failed to generate second context ID: %v", err2)
	}

	if id1 == id2 {
		t.Error("generated IDs should be unique")
	}

	if len(id1) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("ID length: got %d, want 32", len(id1))
	}
}

func TestDisabledMemoryService(t *testing.T) {
	service := &disabledMemoryService{}

	if service.IsEnabled() {
		t.Error("disabled service should return false for IsEnabled()")
	}

	_, err := service.SaveContext(nil, "test content")
	if err == nil {
		t.Error("SaveContext should return error for disabled service")
	}

	_, err = service.ReadContext(nil, "test-id")
	if err == nil {
		t.Error("ReadContext should return error for disabled service")
	}
}

func TestShouldCompress(t *testing.T) {
	config := &ContextCompressionConfig{
		Enabled:                   true,
		CompressionBytesThreshold: 1000,
	}

	service := &redisMemoryService{
		config: config,
	}

	tests := []struct {
		name        string
		contentSize int
		expected    bool
	}{
		{"below threshold", 500, false},
		{"at threshold", 1000, false},
		{"above threshold", 1500, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.ShouldCompress(tt.contentSize)
			if result != tt.expected {
				t.Errorf("ShouldCompress(%d): got %v, want %v", tt.contentSize, result, tt.expected)
			}
		})
	}
}
