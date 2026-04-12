package mock

import (
	"context"
	"testing"
	"time"

	"github.com/Becks723/mind-gateway/internal/core"
)

// TestProviderChat 验证 mock Provider 可以返回固定聊天结果
func TestProviderChat(t *testing.T) {
	// 创建 mock Provider 和测试请求
	provider := New("mock", "测试返回")
	req := &core.Request{
		RequestID: "req-1",
		Model:     "mock-gpt",
		Messages: []core.Message{
			{
				Role:    "user",
				Content: "你好",
			},
		},
	}

	// 执行聊天请求
	resp, err := provider.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("执行聊天请求失败: %v", err)
	}

	// 校验响应内容
	if resp.Provider != "mock" {
		t.Fatalf("期望 Provider 为 mock，实际得到 %q", resp.Provider)
	}
	if resp.OutputText != "测试返回" {
		t.Fatalf("期望输出为 测试返回，实际得到 %q", resp.OutputText)
	}
}

// TestProviderChatFailure 验证 mock Provider 可以模拟失败
func TestProviderChatFailure(t *testing.T) {
	// 创建会失败的 mock Provider
	provider := NewWithFailure("mock", "失败")
	req := &core.Request{
		RequestID: "req-2",
		Model:     "mock-gpt",
	}

	// 执行聊天请求并校验错误
	if _, err := provider.Chat(context.Background(), req); err == nil {
		t.Fatal("期望模拟失败时返回错误")
	}
}

// TestProviderChatStream 验证 mock Provider 可以返回流式事件
func TestProviderChatStream(t *testing.T) {
	// 创建 mock Provider 和测试请求
	provider := New("mock", "流式内容")
	req := &core.Request{
		RequestID: "req-3",
		Model:     "mock-gpt",
	}

	// 执行流式请求
	eventCh, errCh := provider.ChatStream(context.Background(), req)

	// 读取第一段事件
	select {
	case event := <-eventCh:
		if event.Delta != "流式内容" {
			t.Fatalf("期望流式内容为 流式内容，实际得到 %q", event.Delta)
		}
	case err := <-errCh:
		t.Fatalf("期望收到流式事件，实际得到错误: %v", err)
	case <-time.After(time.Second):
		t.Fatal("等待流式事件超时")
	}
}
