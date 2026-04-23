package tool

import (
	"context"
	"strings"
	"testing"

	"github.com/Becks723/mind-gateway/core/schema"
	frameworklogging "github.com/Becks723/mind-gateway/framework/logging"
)

// TestLoopExecute 验证工具循环可以完成模型请求工具再回送模型的闭环
func TestLoopExecute(t *testing.T) {
	registry := NewRegistry()
	if err := RegisterBuiltinTools(registry, []string{"echo"}); err != nil {
		t.Fatalf("注册 echo 工具失败: %v", err)
	}

	loop := NewLoop(registry, frameworklogging.NewLogger("error"), 3)
	resp, err := loop.Execute(context.Background(), &schema.Request{
		RequestID: "req-loop-1",
		Provider:  "mock",
		Model:     "mock-gpt",
		Messages: []schema.Message{
			{
				Role:    "user",
				Content: "请帮我 echo 这句话",
			},
		},
		Tools: []schema.ToolDefinition{
			{
				Name: "echo",
			},
		},
	}, func(ctx context.Context, req *schema.Request) (*schema.Response, error) {
		if len(req.Messages) == 1 {
			return &schema.Response{
				ToolCalls: []schema.ToolInvocation{
					{
						ID:        "call-echo",
						Name:      "echo",
						Arguments: BuildEchoArguments("工具循环测试"),
					},
				},
				FinishReason: "tool_calls",
			}, nil
		}

		lastMessage := req.Messages[len(req.Messages)-1]
		return &schema.Response{
			OutputText:   "最终回答：" + lastMessage.Content,
			FinishReason: "stop",
		}, nil
	})
	if err != nil {
		t.Fatalf("执行工具循环失败: %v", err)
	}
	if !strings.Contains(resp.OutputText, "工具循环测试") {
		t.Fatalf("期望最终输出包含工具结果，实际得到 %q", resp.OutputText)
	}
}
