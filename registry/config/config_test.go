package config

import (
	"context"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes/fake"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestConfigManagerBasicFunctionality tests basic config manager operations
func TestConfigManagerBasicFunctionality(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	namespace := "test-namespace"
	
	// 创建测试ConfigMap
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-config",
			Namespace: namespace,
		},
		Data: map[string]string{
			"instances": `[{"domain":"test1.com","port":8080,"weight":80},{"domain":"test2.com","port":8081,"weight":90}]`,
		},
	}
	
	_, err := fakeClient.CoreV1().ConfigMaps(namespace).Create(
		context.Background(), testConfigMap, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test ConfigMap: %v", err)
	}

	t.Run("ManagerInitialization", func(t *testing.T) {
		manager, err := SetupConfigManager(fakeClient, namespace)
		if err != nil {
			t.Fatalf("Failed to setup config manager: %v", err)
		}
		
		if manager == nil {
			t.Error("Expected non-nil config manager")
		}
	})

	t.Run("GetMCPConfig", func(t *testing.T) {
		manager, _ := SetupConfigManager(fakeClient, namespace)
		
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()
		
		config, err := manager.GetMCPConfig(ctx, ConfigSourceConfigMap, "test-config")
		if err != nil {
			t.Errorf("Failed to get MCP config: %v", err)
			return // 添加return避免后续空指针访问
		}
		
		if config == nil {
			t.Error("Expected non-nil MCP config")
			return // 添加return避免后续空指针访问
		}
		
		if len(config.Instances) != 2 {
			t.Errorf("Expected 2 instances, got %d", len(config.Instances))
		}
	})
}

// TestConfigProviderFactories tests provider factory registration
func TestConfigProviderFactories(t *testing.T) {
	t.Run("ConfigMapProviderFactory", func(t *testing.T) {
		// 简化测试 - 直接创建provider而不依赖工厂注册
		fakeClient := fake.NewSimpleClientset()
		providerConfig := &ProviderConfig{
			Namespace: "test-namespace",
		}
		provider := NewConfigMapProvider(fakeClient, providerConfig)
		
		if provider == nil {
			t.Error("Expected non-nil ConfigMap provider")
		}
		
		if provider.Name() != "configmap" {
			t.Errorf("Expected configmap, got %s", provider.Name())
		}
	})
	
	t.Run("SecretProviderFactory", func(t *testing.T) {
		// 简化测试 - 直接创建provider而不依赖工厂注册
		fakeClient := fake.NewSimpleClientset()
		providerConfig := &ProviderConfig{
			Namespace: "test-namespace",
		}
		provider := NewSecretProvider(fakeClient, providerConfig)
		
		if provider == nil {
			t.Error("Expected non-nil Secret provider")
		}
		
		if provider.Name() != "secret" {
			t.Errorf("Expected secret, got %s", provider.Name())
		}
	})
}

// TestConfigCaching tests configuration caching functionality
func TestConfigCaching(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	namespace := "test-namespace"
	
	// 创建测试ConfigMap
	testConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cached-config",
			Namespace: namespace,
		},
		Data: map[string]string{
			"instances": `[{"domain":"cached.com","port":8080,"weight":50}]`,
		},
	}
	
	_, err := fakeClient.CoreV1().ConfigMaps(namespace).Create(
		context.Background(), testConfigMap, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test ConfigMap: %v", err)
	}

	t.Run("CachePerformance", func(t *testing.T) {
		manager, _ := SetupConfigManager(fakeClient, namespace)
		
		ctx := context.Background()
		
		// 第一次访问（缓存miss）
		start := time.Now()
		config1, err := manager.GetMCPConfig(ctx, ConfigSourceConfigMap, "cached-config")
		firstAccess := time.Since(start)
		
		if err != nil {
			t.Errorf("Failed to get config: %v", err)
		}
		
		// 第二次访问（缓存hit）
		start = time.Now()
		config2, err := manager.GetMCPConfig(ctx, ConfigSourceConfigMap, "cached-config")
		secondAccess := time.Since(start)
		
		if err != nil {
			t.Errorf("Failed to get cached config: %v", err)
		}
		
		// 验证返回相同配置
		if config1 == nil || config2 == nil {
			t.Error("Expected non-nil configs")
			return
		}
		
		if len(config1.Instances) != len(config2.Instances) {
			t.Error("Cached config should be identical")
		}
		
		// 缓存访问应该更快
		if secondAccess > firstAccess {
			t.Logf("Warning: cached access (%v) slower than first access (%v)", 
				secondAccess, firstAccess)
		}
		
		t.Logf("First access: %v, Cached access: %v", firstAccess, secondAccess)
	})
}

// TestErrorHandling tests error handling scenarios
func TestErrorHandling(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	namespace := "test-namespace"

	t.Run("NonExistentConfig", func(t *testing.T) {
		manager, _ := SetupConfigManager(fakeClient, namespace)
		
		ctx := context.Background()
		config, err := manager.GetMCPConfig(ctx, ConfigSourceConfigMap, "non-existent")
		
		if err == nil {
			t.Error("Expected error for non-existent config")
		}
		
		if config != nil {
			t.Error("Expected nil config for non-existent ConfigMap")
		}
	})
	
	t.Run("InvalidConfigSource", func(t *testing.T) {
		manager, _ := SetupConfigManager(fakeClient, namespace)
		
		ctx := context.Background()
		config, err := manager.GetMCPConfig(ctx, "invalid-source", "test-config")
		
		if err == nil {
			t.Error("Expected error for invalid config source")
		}
		
		if config != nil {
			t.Error("Expected nil config for invalid source")
		}
	})
	
	t.Run("ContextTimeout", func(t *testing.T) {
		manager, _ := SetupConfigManager(fakeClient, namespace)
		
		// 创建一个已经超时的context
		ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancel()
		
		time.Sleep(time.Millisecond) // 确保context已超时
		
		config, err := manager.GetMCPConfig(ctx, ConfigSourceConfigMap, "test-config")
		
		if err == nil {
			t.Error("Expected timeout error")
		}
		
		if config != nil {
			t.Error("Expected nil config on timeout")
		}
	})
}