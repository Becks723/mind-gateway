package mock

import (
	"context"
	"testing"
	"time"

	"github.com/Becks723/mind-gateway/core/schema"
)

// TestProviderChat 验证 mock Provider 可以返回固定聊天结果
func TestProviderChat(t *testing.T) {
	// 创建 mock Provider 和测试请求
	provider := New("mock", "测试返回")
	req := &schema.Request{
		RequestID: "req-1",
		Model:     "mock-gpt",
		Messages: []schema.Message{
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
	req := &schema.Request{
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
	provider := New("mock", "hello from mock provider")
	req := &schema.Request{
		RequestID: "req-3",
		Model:     "mock-gpt",
	}

	// 执行流式请求
	eventCh, errCh := provider.ChatStream(context.Background(), req)

	// 汇总全部流式事件
	chunkCount := 0
	finalUsage := schema.Usage{}
	for eventCh != nil || errCh != nil {
		select {
		case event, ok := <-eventCh:
			if !ok {
				eventCh = nil
				continue
			}
			if event.Done {
				if event.Usage == nil {
					t.Fatal("期望结束事件包含 usage")
				}
				finalUsage = *event.Usage
				continue
			}
			chunkCount++
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			t.Fatalf("期望收到流式事件，实际得到错误: %v", err)
		case <-time.After(time.Second):
			t.Fatal("等待流式事件超时")
		}
	}

	// 校验分片数量和 usage
	if chunkCount < 4 {
		t.Fatalf("期望至少收到 4 个流式 chunk，实际得到 %d", chunkCount)
	}
	if finalUsage.OutputTokens != int64(chunkCount) {
		t.Fatalf("期望输出 token 与 chunk 数一致，实际得到 usage=%#v chunkCount=%d", finalUsage, chunkCount)
	}
}

// TestProviderChatWithToolCall 验证 mock Provider 可以返回工具调用
func TestProviderChatWithToolCall(t *testing.T) {
	// 创建支持工具调用的 mock Provider
	provider := New("mock", "unused")
	req := &schema.Request{
		RequestID: "req-tool-1",
		Model:     "mock-gpt",
		Messages: []schema.Message{
			{
				Role:    "user",
				Content: "请告诉我当前时间",
			},
		},
		Tools: []schema.ToolDefinition{
			{
				Name: "current_time",
			},
		},
	}

	resp, err := provider.Chat(context.Background(), req)
	if err != nil {
		t.Fatalf("执行带工具的聊天请求失败: %v", err)
	}
	if len(resp.ToolCalls) != 1 {
		t.Fatalf("期望返回 1 个工具调用，实际得到 %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "current_time" {
		t.Fatalf("期望工具名为 current_time，实际得到 %q", resp.ToolCalls[0].Name)
	}
}
