package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Becks723/mind-gateway/core/schema"
)

// Provider 表示本地 mock Provider 实现
type Provider struct {
	name            string // name 表示 Provider 名称
	responseText    string // responseText 表示固定返回文本
	simulateFailure bool   // simulateFailure 表示是否模拟失败
}

// New 创建新的 mock Provider
func New(name string, responseText string) *Provider {
	// 兜底设置 Provider 默认值
	finalName := name
	if finalName == "" {
		finalName = "mock"
	}
	finalResponse := responseText
	if finalResponse == "" {
		finalResponse = "hello from mock provider"
	}

	return &Provider{
		name:         finalName,
		responseText: finalResponse,
	}
}

// NewWithFailure 创建会模拟失败的 mock Provider
func NewWithFailure(name string, responseText string) *Provider {
	// 基于普通 mock Provider 打开失败模拟
	provider := New(name, responseText)
	provider.simulateFailure = true
	return provider
}

// Name 返回 Provider 名称
func (p *Provider) Name() string {
	return p.name
}

// Type 返回 Provider 类型
func (p *Provider) Type() string {
	return "mock"
}

// Chat 执行非流式聊天请求
func (p *Provider) Chat(ctx context.Context, req *schema.Request) (*schema.Response, error) {
	// 处理上下文取消
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 模拟 Provider 执行失败
	if p.simulateFailure {
		return nil, fmt.Errorf("mock provider %q 模拟失败", p.name)
	}

	// 优先处理工具调用场景
	if response, ok := p.handleToolScenario(req); ok {
		return response, nil
	}

	// 生成统一响应对象
	now := time.Now()
	response := &schema.Response{
		RequestID:  req.RequestID,
		Provider:   p.name,
		Model:      req.Model,
		OutputText: p.responseText,
		Messages: []schema.Message{
			{
				Role:    "assistant",
				Content: p.responseText,
			},
		},
		FinishReason: "stop",
		Usage: schema.Usage{
			InputTokens:  int64(len(req.Messages)),
			OutputTokens: 1,
			TotalTokens:  int64(len(req.Messages)) + 1,
		},
		Latency: time.Since(now),
	}

	return response, nil
}

// handleToolScenario 处理 mock provider 的工具调用场景
func (p *Provider) handleToolScenario(req *schema.Request) (*schema.Response, bool) {
	// 仅在请求中存在可用工具时进入工具路径
	if req == nil || len(req.Tools) == 0 {
		return nil, false
	}

	// 当工具结果已经回写后，生成最终回答
	if toolOutputs := collectToolOutputs(req.Messages); len(toolOutputs) > 0 {
		answer := "工具执行结果：" + strings.Join(toolOutputs, "；")
		return &schema.Response{
			RequestID:  req.RequestID,
			Provider:   p.name,
			Model:      req.Model,
			OutputText: answer,
			Messages: []schema.Message{
				{
					Role:    "assistant",
					Content: answer,
				},
			},
			FinishReason: "stop",
			Usage: schema.Usage{
				InputTokens:  int64(len(req.Messages)),
				OutputTokens: 1,
				TotalTokens:  int64(len(req.Messages)) + 1,
			},
		}, true
	}

	// 根据用户输入决定是否请求工具
	userContent := strings.ToLower(lastUserContent(req.Messages))
	switch {
	case strings.Contains(userContent, "时间") || strings.Contains(userContent, "time"):
		if hasTool(req.Tools, "current_time") {
			return toolCallResponse(req, p.name, "current_time", "{}"), true
		}
	case strings.Contains(userContent, "echo") || strings.Contains(userContent, "复述"):
		if hasTool(req.Tools, "echo") {
			return toolCallResponse(req, p.name, "echo", buildEchoArguments(lastUserContent(req.Messages))), true
		}
	}

	return nil, false
}

// ChatStream 执行流式聊天请求
func (p *Provider) ChatStream(ctx context.Context, req *schema.Request) (<-chan schema.StreamEvent, <-chan error) {
	// 创建流式事件通道和错误通道
	chunks := splitStreamChunks(p.responseText)
	eventCh := make(chan schema.StreamEvent, len(chunks)+1)
	errCh := make(chan error, 1)

	go func() {
		defer close(eventCh)
		defer close(errCh)

		// 处理上下文取消
		select {
		case <-ctx.Done():
			errCh <- ctx.Err()
			return
		default:
		}

		// 模拟 Provider 执行失败
		if p.simulateFailure {
			errCh <- fmt.Errorf("mock provider %q 模拟流式失败", p.name)
			return
		}

		// 逐段发送流式内容事件
		for _, chunk := range chunks {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			eventCh <- schema.StreamEvent{
				RequestID: req.RequestID,
				Provider:  p.name,
				Model:     req.Model,
				Delta:     chunk,
				Done:      false,
			}
		}

		// 发送结束事件
		eventCh <- schema.StreamEvent{
			RequestID:    req.RequestID,
			Provider:     p.name,
			Model:        req.Model,
			Delta:        "",
			Done:         true,
			FinishReason: "stop",
			Usage: &schema.Usage{
				InputTokens:  int64(len(req.Messages)),
				OutputTokens: int64(len(chunks)),
				TotalTokens:  int64(len(req.Messages)) + int64(len(chunks)),
			},
		}
	}()

	return eventCh, errCh
}

// splitStreamChunks 将固定文本拆成多个流式片段
func splitStreamChunks(text string) []string {
	// 对空文本返回单个空片段
	if text == "" {
		return []string{""}
	}

	// 优先按空格切分，保证 mock 流式演示有多个 chunk
	parts := strings.Fields(text)
	if len(parts) >= 2 {
		return parts
	}

	// 对单词或中文文本按 rune 分片
	runes := []rune(text)
	chunkSize := 2
	if len(runes) > 10 {
		chunkSize = 3
	}

	result := make([]string, 0, (len(runes)+chunkSize-1)/chunkSize)
	for index := 0; index < len(runes); index += chunkSize {
		end := index + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		result = append(result, string(runes[index:end]))
	}

	return result
}

// collectToolOutputs 收集消息中的工具执行结果
func collectToolOutputs(messages []schema.Message) []string {
	result := make([]string, 0, len(messages))
	for _, message := range messages {
		if message.Role != "tool" || message.Content == "" {
			continue
		}
		result = append(result, message.Content)
	}
	return result
}

// lastUserContent 读取最后一条用户消息
func lastUserContent(messages []schema.Message) string {
	for index := len(messages) - 1; index >= 0; index-- {
		if messages[index].Role == "user" {
			return messages[index].Content
		}
	}
	return ""
}

// hasTool 判断是否存在指定工具
func hasTool(tools []schema.ToolDefinition, name string) bool {
	for _, tool := range tools {
		if tool.Name == name {
			return true
		}
	}
	return false
}

// toolCallResponse 构造要求执行工具的 mock 响应
func toolCallResponse(req *schema.Request, providerName string, toolName string, arguments string) *schema.Response {
	toolCall := schema.ToolInvocation{
		ID:        "call-" + toolName,
		Name:      toolName,
		Arguments: arguments,
	}

	return &schema.Response{
		RequestID:    req.RequestID,
		Provider:     providerName,
		Model:        req.Model,
		FinishReason: "tool_calls",
		Messages: []schema.Message{
			{
				Role:      "assistant",
				ToolCalls: []schema.ToolInvocation{toolCall},
			},
		},
		ToolCalls: []schema.ToolInvocation{toolCall},
		Usage: schema.Usage{
			InputTokens:  int64(len(req.Messages)),
			OutputTokens: 1,
			TotalTokens:  int64(len(req.Messages)) + 1,
		},
	}
}

// buildEchoArguments 构造 echo 工具参数
func buildEchoArguments(text string) string {
	payload, _ := json.Marshal(map[string]string{
		"text": text,
	})
	return string(payload)
}
