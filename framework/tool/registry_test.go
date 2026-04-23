package tool

import (
	"context"
	"testing"
)

// TestRegisterBuiltinTools 验证可以注册两个内置工具
func TestRegisterBuiltinTools(t *testing.T) {
	registry := NewRegistry()
	if err := RegisterBuiltinTools(registry, []string{"current_time", "echo"}); err != nil {
		t.Fatalf("注册内置工具失败: %v", err)
	}

	definitions := registry.Definitions()
	if len(definitions) != 2 {
		t.Fatalf("期望注册 2 个工具，实际得到 %d", len(definitions))
	}
}

// TestRegistryExecuteEcho 验证 echo 工具可以成功执行
func TestRegistryExecuteEcho(t *testing.T) {
	registry := NewRegistry()
	if err := RegisterBuiltinTools(registry, []string{"echo"}); err != nil {
		t.Fatalf("注册 echo 工具失败: %v", err)
	}

	result, err := registry.Execute(context.Background(), "echo", `{"text":"hello tool"}`)
	if err != nil {
		t.Fatalf("执行 echo 工具失败: %v", err)
	}
	if result != "hello tool" {
		t.Fatalf("期望 echo 工具返回 hello tool，实际得到 %q", result)
	}
}
