package core

import (
	"context"
	"testing"

	"github.com/Becks723/mind-gateway/core/schema"
	frameworkconfig "github.com/Becks723/mind-gateway/framework/config"
	"github.com/Becks723/mind-gateway/provider"
	mockprovider "github.com/Becks723/mind-gateway/provider/mock"
)

// TestGatewayHandleChat 验证网关可以通过注册表调用 mock Provider
func TestGatewayHandleChat(t *testing.T) {
	// 创建注册表并注册 mock Provider
	registry := provider.NewRegistry()
	if err := registry.Register(mockprovider.New("mock", "你好，我是 mock")); err != nil {
		t.Fatalf("注册 mock Provider 失败: %v", err)
	}

	gateway := NewGateway(frameworkconfig.GatewayConfig{
		DefaultProvider: "mock",
		DefaultModel:    "mock-gpt",
	}, registry)

	// 构造聊天请求并执行
	resp, err := gateway.HandleChat(context.Background(), &schema.Request{
		RequestID: "req-1",
		Messages: []schema.Message{
			{
				Role:    "user",
				Content: "你好",
			},
		},
	})
	if err != nil {
		t.Fatalf("执行聊天请求失败: %v", err)
	}

	// 校验响应内容
	if resp.Provider != "mock" {
		t.Fatalf("期望 Provider 为 mock，实际得到 %q", resp.Provider)
	}
	if resp.OutputText != "你好，我是 mock" {
		t.Fatalf("期望输出为 mock 返回内容，实际得到 %q", resp.OutputText)
	}
}

// TestGatewayHandleChatWithEmptyMessages 验证空消息会被拒绝
func TestGatewayHandleChatWithEmptyMessages(t *testing.T) {
	// 创建最小网关
	gateway := NewGateway(frameworkconfig.GatewayConfig{
		DefaultProvider: "mock",
		DefaultModel:    "mock-gpt",
	}, provider.NewRegistry())

	// 执行空消息请求并校验错误
	if _, err := gateway.HandleChat(context.Background(), &schema.Request{
		RequestID: "req-2",
	}); err == nil {
		t.Fatal("期望空消息请求返回错误")
	}
}
