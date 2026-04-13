package logging

import (
	"context"
	"testing"
	"time"

	"github.com/Becks723/mind-gateway/core/schema"
	frameworklogging "github.com/Becks723/mind-gateway/framework/logging"
)

// TestPluginHooks 验证日志插件的前后置钩子可正常执行
func TestPluginHooks(t *testing.T) {
	// 创建日志插件
	instance := NewPlugin(frameworklogging.NewLogger("error"))
	req := &schema.Request{
		RequestID: "req-1",
		Provider:  "mock",
		Model:     "mock-gpt",
		Messages: []schema.Message{
			{
				Role:    "user",
				Content: "你好",
			},
		},
	}

	// 执行前置钩子
	if _, err := instance.PreHook(context.Background(), req); err != nil {
		t.Fatalf("执行前置钩子失败: %v", err)
	}

	// 执行后置钩子
	resp, err := instance.PostHook(context.Background(), req, &schema.Response{
		OutputText: "你好",
		Latency:    50 * time.Millisecond,
	}, nil)
	if err != nil {
		t.Fatalf("执行后置钩子失败: %v", err)
	}
	if resp == nil || resp.OutputText != "你好" {
		t.Fatalf("期望响应输出为 你好，实际得到 %#v", resp)
	}
}
