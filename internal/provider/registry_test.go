package provider

import (
	"testing"

	mockprovider "github.com/Becks723/mind-gateway/internal/provider/mock"
)

// TestRegistryRegisterAndGet 验证注册表可以注册并读取 Provider
func TestRegistryRegisterAndGet(t *testing.T) {
	// 创建注册表并注册 mock Provider
	registry := NewRegistry()
	provider := mockprovider.New("mock", "hello")

	if err := registry.Register(provider); err != nil {
		t.Fatalf("注册 Provider 失败: %v", err)
	}

	// 读取 Provider 并校验名称
	got, ok := registry.Get("mock")
	if !ok {
		t.Fatal("期望能够读取已注册的 Provider")
	}
	if got.Name() != "mock" {
		t.Fatalf("期望 Provider 名称为 mock，实际得到 %q", got.Name())
	}
}

// TestRegistryRegisterDuplicate 验证注册表会拒绝重复名称
func TestRegistryRegisterDuplicate(t *testing.T) {
	// 创建注册表并重复注册同名 Provider
	registry := NewRegistry()
	first := mockprovider.New("mock", "one")
	second := mockprovider.New("mock", "two")

	if err := registry.Register(first); err != nil {
		t.Fatalf("首次注册 Provider 失败: %v", err)
	}
	if err := registry.Register(second); err == nil {
		t.Fatal("期望重复注册时返回错误")
	}
}

// TestRegistryList 验证注册表会按顺序返回 Provider 名称
func TestRegistryList(t *testing.T) {
	// 创建注册表并注册多个 Provider
	registry := NewRegistry()
	if err := registry.Register(mockprovider.New("beta", "beta")); err != nil {
		t.Fatalf("注册 beta Provider 失败: %v", err)
	}
	if err := registry.Register(mockprovider.New("alpha", "alpha")); err != nil {
		t.Fatalf("注册 alpha Provider 失败: %v", err)
	}

	// 校验名称列表顺序
	names := registry.List()
	if len(names) != 2 {
		t.Fatalf("期望名称数量为 2，实际得到 %d", len(names))
	}
	if names[0] != "alpha" || names[1] != "beta" {
		t.Fatalf("期望排序结果为 [alpha beta]，实际得到 %v", names)
	}
}
