package wasmplugin

import (
	"context"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSaveConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *MCPShardingConfig
		wantErr     bool
		setupMocks  func() *fake.ClientBuilder
	}{
		{
			name: "save valid config",
			config: &MCPShardingConfig{
				Enabled:      true,
				MaxSize:      1048576,
				MaxInstances: 100,
				Strategy:     string(GroupBySize),
				Compression: MCPCompressionConfig{
					Enabled:   true,
					Algorithm: "gzip",
					Level:     6,
				},
				Monitoring: MCPMonitoringConfig{
					Enabled:        true,
					SizeThreshold:  838860,
					CheckInterval:  5 * time.Minute,
					AlertEnabled:   true,
					MetricsEnabled: true,
				},
			},
			wantErr: false,
			setupMocks: func() *fake.ClientBuilder {
				return fake.NewClientBuilder()
			},
		},
		{
			name:    "save nil config",
			config:  nil,
			wantErr: false,
			setupMocks: func() *fake.ClientBuilder {
				return fake.NewClientBuilder()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupMocks().Build()
			cm := NewConfigManager(client, "default")
			err := cm.SaveConfig(context.Background(), tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *MCPShardingConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &MCPShardingConfig{
				Enabled:      true,
				MaxSize:      1048576,
				MaxInstances: 100,
				Strategy:     string(GroupBySize),
			},
			wantErr: false,
		},
		{
			name: "invalid max size",
			config: &MCPShardingConfig{
				Enabled:      true,
				MaxSize:      0,
				MaxInstances: 100,
				Strategy:     string(GroupBySize),
			},
			wantErr: true,
		},
		{
			name: "invalid strategy",
			config: &MCPShardingConfig{
				Enabled:      true,
				MaxSize:      1048576,
				MaxInstances: 100,
				Strategy:     "invalid",
			},
			wantErr: true,
		},
		{
			name:    "nil config",
			config:  nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := NewConfigManager(nil, "default")
			err := cm.ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestToMCPControllerOptions(t *testing.T) {
	tests := []struct {
		name    string
		config  *MCPShardingConfig
		want    MCPControllerOptions
		wantErr bool
	}{
		{
			name: "valid config",
			config: &MCPShardingConfig{
				Enabled:      true,
				MaxSize:      1048576,
				MaxInstances: 100,
				Strategy:     string(GroupBySize),
			},
			want: MCPControllerOptions{
				ShardingEnabled:      true,
				MaxInstancesPerShard: 100,
				GroupingStrategy:     GroupBySize,
				MonitoringEnabled:    false,
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			config:  nil,
			want: MCPControllerOptions{
				ShardingEnabled:      true,
				MaxInstancesPerShard: MaxMCPInstancesPerShard,
				GroupingStrategy:     GroupByHash,
				MonitoringEnabled:    true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.ToMCPControllerOptions()
			if !tt.wantErr && (got.ShardingEnabled != tt.want.ShardingEnabled ||
				got.MaxInstancesPerShard != tt.want.MaxInstancesPerShard ||
				got.GroupingStrategy != tt.want.GroupingStrategy ||
				got.MonitoringEnabled != tt.want.MonitoringEnabled) {
				t.Errorf("ToMCPControllerOptions() = %v, want %v", got, tt.want)
			}
		})
	}
} 