package openai

import (
	"testing"

	"github.com/Becks723/mind-gateway/core/schema"
)

// TestToOpenAIChatCompletionRequest 验证内部请求可以转换为 OpenAI 请求
func TestToOpenAIChatCompletionRequest(t *testing.T) {
	// 构造内部请求
	req := &schema.Request{
		Model: "gpt-4o-mini",
		Messages: []schema.Message{
			{
				Role:    "user",
				Content: "你好",
				Name:    "tester",
			},
		},
	}

	// 执行转换
	openaiReq := ToOpenAIChatCompletionRequest(req)

	// 校验转换结果
	if openaiReq == nil {
		t.Fatal("期望转换结果非空")
	}
	if openaiReq.Model != "gpt-4o-mini" {
		t.Fatalf("期望模型为 gpt-4o-mini，实际得到 %q", openaiReq.Model)
	}
	if len(openaiReq.Messages) != 1 {
		t.Fatalf("期望消息数量为 1，实际得到 %d", len(openaiReq.Messages))
	}
	if openaiReq.Messages[0].Content != "你好" {
		t.Fatalf("期望消息内容为 你好，实际得到 %q", openaiReq.Messages[0].Content)
	}
}

// TestToSchemaResponse 验证 OpenAI 响应可以转换为内部统一响应
func TestToSchemaResponse(t *testing.T) {
	// 构造 OpenAI 响应
	resp := &ChatCompletionResponse{
		ID:    "chatcmpl-1",
		Model: "gpt-4o-mini",
		Choices: []ChatChoice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: "你好，我是 OpenAI",
				},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	// 执行转换
	schemaResp := ToSchemaResponse("req-1", "openai", resp)

	// 校验转换结果
	if schemaResp == nil {
		t.Fatal("期望转换结果非空")
	}
	if schemaResp.Provider != "openai" {
		t.Fatalf("期望 provider 为 openai，实际得到 %q", schemaResp.Provider)
	}
	if schemaResp.OutputText != "你好，我是 OpenAI" {
		t.Fatalf("期望输出为 你好，我是 OpenAI，实际得到 %q", schemaResp.OutputText)
	}
	if schemaResp.Usage.TotalTokens != 15 {
		t.Fatalf("期望总 token 为 15，实际得到 %d", schemaResp.Usage.TotalTokens)
	}
}
