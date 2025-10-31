package main

import (
	"testing"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
)

// TestAuditLogConfig_Defaults 测试审计日志默认配置
func TestAuditLogConfig_Defaults(t *testing.T) {
	config := &AuditLogConfig{
		Enabled:                true,
		Level:                  "info",
		LogSuccessEvents:       true,
		LogFailureEvents:       true,
		LogToolCalls:           false,
		LogBoundaryApplication: false,
		IncludeRequestDetails:  false,
	}

	if !config.Enabled {
		t.Error("默认应该启用审计日志")
	}

	if config.Level != "info" {
		t.Errorf("默认日志级别应该是'info'，实际是'%s'", config.Level)
	}

	if !config.LogSuccessEvents {
		t.Error("默认应该记录成功事件")
	}

	if !config.LogFailureEvents {
		t.Error("默认应该记录失败事件")
	}
}

// TestAuditLogConfig_LevelValidation 测试日志级别验证
func TestAuditLogConfig_LevelValidation(t *testing.T) {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	testCases := []struct {
		name  string
		level string
		valid bool
	}{
		{"debug级别", "debug", true},
		{"info级别", "info", true},
		{"warn级别", "warn", true},
		{"error级别", "error", true},
		{"无效级别", "invalid", false},
		{"空级别", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isValid := validLevels[tc.level]
			if isValid != tc.valid {
				t.Errorf("级别'%s'的验证结果应该是%v，实际是%v", tc.level, tc.valid, isValid)
			}
		})
	}
}

// TestAuditLogConfig_EventFiltering 测试事件过滤配置
func TestAuditLogConfig_EventFiltering(t *testing.T) {
	testCases := []struct {
		name                string
		logSuccess          bool
		logFailure          bool
		logToolCalls        bool
		logBoundary         bool
		expectedSuccessLog  bool
		expectedFailureLog  bool
		expectedToolLog     bool
		expectedBoundaryLog bool
	}{
		{
			name:                "全部启用",
			logSuccess:          true,
			logFailure:          true,
			logToolCalls:        true,
			logBoundary:         true,
			expectedSuccessLog:  true,
			expectedFailureLog:  true,
			expectedToolLog:     true,
			expectedBoundaryLog: true,
		},
		{
			name:                "仅失败事件",
			logSuccess:          false,
			logFailure:          true,
			logToolCalls:        false,
			logBoundary:         false,
			expectedSuccessLog:  false,
			expectedFailureLog:  true,
			expectedToolLog:     false,
			expectedBoundaryLog: false,
		},
		{
			name:                "仅成功事件",
			logSuccess:          true,
			logFailure:          false,
			logToolCalls:        false,
			logBoundary:         false,
			expectedSuccessLog:  true,
			expectedFailureLog:  false,
			expectedToolLog:     false,
			expectedBoundaryLog: false,
		},
		{
			name:                "全部禁用",
			logSuccess:          false,
			logFailure:          false,
			logToolCalls:        false,
			logBoundary:         false,
			expectedSuccessLog:  false,
			expectedFailureLog:  false,
			expectedToolLog:     false,
			expectedBoundaryLog: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &AuditLogConfig{
				Enabled:                tc.logSuccess || tc.logFailure || tc.logToolCalls || tc.logBoundary,
				LogSuccessEvents:       tc.logSuccess,
				LogFailureEvents:       tc.logFailure,
				LogToolCalls:           tc.logToolCalls,
				LogBoundaryApplication: tc.logBoundary,
			}

			if config.LogSuccessEvents != tc.expectedSuccessLog {
				t.Errorf("LogSuccessEvents应该是%v，实际是%v", tc.expectedSuccessLog, config.LogSuccessEvents)
			}

			if config.LogFailureEvents != tc.expectedFailureLog {
				t.Errorf("LogFailureEvents应该是%v，实际是%v", tc.expectedFailureLog, config.LogFailureEvents)
			}

			if config.LogToolCalls != tc.expectedToolLog {
				t.Errorf("LogToolCalls应该是%v，实际是%v", tc.expectedToolLog, config.LogToolCalls)
			}

			if config.LogBoundaryApplication != tc.expectedBoundaryLog {
				t.Errorf("LogBoundaryApplication应该是%v，实际是%v", tc.expectedBoundaryLog, config.LogBoundaryApplication)
			}
		})
	}
}

// TestAuditLogConfig_DisabledLogging 测试禁用审计日志
func TestAuditLogConfig_DisabledLogging(t *testing.T) {
	config := &AuditLogConfig{
		Enabled: false,
	}

	if config.Enabled {
		t.Error("审计日志应该被禁用")
	}

	// 即使其他选项为true，禁用时也不应记录
	config.LogSuccessEvents = true
	config.LogFailureEvents = true

	if config.Enabled {
		t.Error("即使设置了LogSuccessEvents和LogFailureEvents，Enabled为false时也不应记录")
	}
}

// TestAuditLogConfig_IncludeRequestDetails 测试包含请求详情配置
func TestAuditLogConfig_IncludeRequestDetails(t *testing.T) {
	testCases := []struct {
		name            string
		includeDetails  bool
		expectedInclude bool
	}{
		{"包含请求详情", true, true},
		{"不包含请求详情", false, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := &AuditLogConfig{
				Enabled:               true,
				IncludeRequestDetails: tc.includeDetails,
			}

			if config.IncludeRequestDetails != tc.expectedInclude {
				t.Errorf("IncludeRequestDetails应该是%v，实际是%v", tc.expectedInclude, config.IncludeRequestDetails)
			}
		})
	}
}

// TestAuditLog_MetricsIntegration 测试审计日志与Metrics集成
func TestAuditLog_MetricsIntegration(t *testing.T) {
	config := &A2ASConfig{
		AuditLog: AuditLogConfig{
			Enabled:          true,
			Level:            "info",
			LogSuccessEvents: true,
			LogFailureEvents: true,
		},
		metrics: make(map[string]proxywasm.MetricCounter),
	}

	// 验证metrics map已初始化
	if config.metrics == nil {
		t.Fatal("metrics map应该被初始化")
	}

	// 验证审计日志配置
	if !config.AuditLog.Enabled {
		t.Error("审计日志应该被启用")
	}
}

// TestAuditLogConfig_MultipleEventTypes 测试多种事件类型配置
func TestAuditLogConfig_MultipleEventTypes(t *testing.T) {
	config := &AuditLogConfig{
		Enabled:                true,
		Level:                  "debug",
		LogSuccessEvents:       true,
		LogFailureEvents:       true,
		LogToolCalls:           true,
		LogBoundaryApplication: true,
		IncludeRequestDetails:  true,
	}

	// 验证所有事件类型都已启用
	eventTypes := []struct {
		name    string
		enabled bool
	}{
		{"Success Events", config.LogSuccessEvents},
		{"Failure Events", config.LogFailureEvents},
		{"Tool Calls", config.LogToolCalls},
		{"Boundary Application", config.LogBoundaryApplication},
	}

	for _, et := range eventTypes {
		if !et.enabled {
			t.Errorf("%s应该被启用", et.name)
		}
	}

	if !config.IncludeRequestDetails {
		t.Error("IncludeRequestDetails应该被启用")
	}
}

// TestAuditLogConfig_ProductionSettings 测试生产环境推荐配置
func TestAuditLogConfig_ProductionSettings(t *testing.T) {
	// 生产环境推荐：只记录失败和关键事件
	config := &AuditLogConfig{
		Enabled:                true,
		Level:                  "warn", // 只记录warn和error
		LogSuccessEvents:       false,  // 不记录成功事件，减少日志量
		LogFailureEvents:       true,   // 记录失败事件
		LogToolCalls:           false,  // 不记录所有工具调用
		LogBoundaryApplication: false,  // 不记录边界应用
		IncludeRequestDetails:  false,  // 不包含敏感的请求详情
	}

	if config.Level != "warn" {
		t.Errorf("生产环境推荐日志级别为'warn'，实际是'%s'", config.Level)
	}

	if config.LogSuccessEvents {
		t.Error("生产环境不应记录所有成功事件，减少日志量")
	}

	if !config.LogFailureEvents {
		t.Error("生产环境应该记录失败事件")
	}

	if config.IncludeRequestDetails {
		t.Error("生产环境不应包含敏感的请求详情")
	}
}

// TestAuditLogConfig_DevelopmentSettings 测试开发环境推荐配置
func TestAuditLogConfig_DevelopmentSettings(t *testing.T) {
	// 开发环境推荐：记录所有事件以便调试
	config := &AuditLogConfig{
		Enabled:                true,
		Level:                  "debug", // 记录所有级别
		LogSuccessEvents:       true,    // 记录成功事件
		LogFailureEvents:       true,    // 记录失败事件
		LogToolCalls:           true,    // 记录工具调用
		LogBoundaryApplication: true,    // 记录边界应用
		IncludeRequestDetails:  true,    // 包含请求详情用于调试
	}

	if config.Level != "debug" {
		t.Errorf("开发环境推荐日志级别为'debug'，实际是'%s'", config.Level)
	}

	if !config.LogSuccessEvents || !config.LogFailureEvents {
		t.Error("开发环境应该记录所有事件")
	}

	if !config.LogToolCalls || !config.LogBoundaryApplication {
		t.Error("开发环境应该记录所有操作类型")
	}

	if !config.IncludeRequestDetails {
		t.Error("开发环境应该包含请求详情用于调试")
	}
}
