package wasmplugin

import (
	"context"
	"testing"

	extensionsv1alpha1 "github.com/alibaba/higress/api/extensions/v1alpha1"
	v1 "github.com/alibaba/higress/client/pkg/apis/extensions/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetShardInfo(t *testing.T) {
	// 创建假的客户端
	scheme := runtime.NewScheme()
	_ = v1.AddToScheme(scheme)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	// 工厂函数减少代码重复
	newTestPlugin := func(name, url string) *v1.WasmPlugin {
		return &v1.WasmPlugin{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
				Labels: map[string]string{
					ShardOfLabelKey: "test-plugin",
				},
			},
			Spec: extensionsv1alpha1.WasmPlugin{
				Url: url,
			},
		}
	}

	// 创建测试用的 WasmPlugin 资源
	plugin1 := newTestPlugin("test-plugin-1", "test-url")
	plugin2 := newTestPlugin("test-plugin-2", "test-url-2")

	// 将测试资源添加到假客户端
	if err := fakeClient.Create(context.Background(), plugin1); err != nil {
		t.Fatalf("Failed to create plugin1: %v", err)
	}
	if err := fakeClient.Create(context.Background(), plugin2); err != nil {
		t.Fatalf("Failed to create plugin2: %v", err)
	}

	// 执行测试
	shards, err := GetShardInfo(context.Background(), fakeClient, "default", "test-plugin")
	if err != nil {
		t.Errorf("GetShardInfo() error = %v", err)
		return
	}

	// 验证结果
	if len(shards) != 2 {
		t.Errorf("Expected 2 shards, got %d", len(shards))
	}

	// 预期的URL集合
	expectedURLs := map[string]bool{
		"test-url":   true,
		"test-url-2": true,
	}

	// 验证返回的是指针切片而不是值切片
	for _, shard := range shards {
		if shard == nil {
			t.Error("Expected non-nil shard")
			continue
		}
		if !expectedURLs[shard.Spec.Url] {
			t.Errorf("Unexpected shard URL: %s", shard.Spec.Url)
		}
	}
} 