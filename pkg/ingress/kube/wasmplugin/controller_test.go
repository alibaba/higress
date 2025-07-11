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

	// 创建测试用的 WasmPlugin 资源
	plugin1 := &v1.WasmPlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-plugin-1",
			Namespace: "default",
			Labels: map[string]string{
				ShardOfLabelKey: "test-plugin",
			},
		},
		Spec: extensionsv1alpha1.WasmPlugin{
			Url: "test-url",
		},
	}

	plugin2 := &v1.WasmPlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-plugin-2",
			Namespace: "default",
			Labels: map[string]string{
				ShardOfLabelKey: "test-plugin",
			},
		},
		Spec: extensionsv1alpha1.WasmPlugin{
			Url: "test-url-2",
		},
	}

	// 将测试资源添加到假客户端
	_ = fakeClient.Create(context.Background(), plugin1)
	_ = fakeClient.Create(context.Background(), plugin2)

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

	// 验证返回的是指针切片而不是值切片
	for _, shard := range shards {
		if shard == nil {
			t.Error("Expected non-nil shard")
			continue
		}
	}
} 