package tool

import (
	"context"
	"fmt"
	"strings"

	"github.com/Becks723/mind-gateway/core/schema"
	frameworklogging "github.com/Becks723/mind-gateway/framework/logging"
)

// ModelExecutor 定义模型执行函数签名
type ModelExecutor func(ctx context.Context, req *schema.Request) (*schema.Response, error)

// Loop 定义工具循环执行器
type Loop struct {
	registry  *Registry                // registry 表示工具注册表
	logger    *frameworklogging.Logger // logger 表示工具循环日志记录器
	maxRounds int                      // maxRounds 表示最大工具循环轮次
}

// NewLoop 创建新的工具循环执行器
func NewLoop(registry *Registry, logger *frameworklogging.Logger, maxRounds int) *Loop {
	finalRounds := maxRounds
	if finalRounds <= 0 {
		finalRounds = 4
	}

	return &Loop{
		registry:  registry,
		logger:    logger,
		maxRounds: finalRounds,
	}
}

// Execute 执行一轮完整的模型推理与工具循环
func (l *Loop) Execute(ctx context.Context, req *schema.Request, executeModel ModelExecutor) (*schema.Response, error) {
	// 校验输入依赖
	if l == nil || l.registry == nil {
		return nil, fmt.Errorf("工具循环未初始化")
	}
	if req == nil {
		return nil, fmt.Errorf("工具循环请求不能为空")
	}
	if executeModel == nil {
		return nil, fmt.Errorf("模型执行函数不能为空")
	}

	// 基于请求副本开始循环
	currentReq := cloneRequest(req)
	for round := 0; round < l.maxRounds; round++ {
		resp, err := executeModel(ctx, currentReq)
		if err != nil {
			return nil, err
		}
		if len(resp.ToolCalls) == 0 {
			return resp, nil
		}

		// 记录模型发起工具调用
		l.logInfo(
			"模型请求执行工具",
			"request_id", currentReq.RequestID,
			"provider", currentReq.Provider,
			"model", currentReq.Model,
			"tool_call_count", len(resp.ToolCalls),
			"round", round+1,
		)

		// 将 assistant 的工具调用消息写回上下文
		currentReq.Messages = append(currentReq.Messages, schema.Message{
			Role:      "assistant",
			ToolCalls: cloneToolInvocations(resp.ToolCalls),
		})

		// 逐个执行工具并回送工具结果
		for _, toolCall := range resp.ToolCalls {
			result, err := l.registry.Execute(ctx, toolCall.Name, toolCall.Arguments)
			if err != nil {
				return nil, fmt.Errorf("执行工具 %q 失败: %w", toolCall.Name, err)
			}

			l.logInfo(
				"工具执行完成",
				"request_id", currentReq.RequestID,
				"provider", currentReq.Provider,
				"model", currentReq.Model,
				"tool_name", toolCall.Name,
				"round", round+1,
			)

			currentReq.Messages = append(currentReq.Messages, schema.Message{
				Role:       "tool",
				Name:       toolCall.Name,
				ToolCallID: toolCall.ID,
				Content:    result,
			})
		}
	}

	return nil, fmt.Errorf("工具循环超过最大轮次限制: %d", l.maxRounds)
}

// BuildEchoArguments 构造 echo 工具参数
func BuildEchoArguments(text string) string {
	text = strings.TrimSpace(text)
	return fmt.Sprintf("{\"text\":%q}", text)
}

// logInfo 输出工具循环日志
func (l *Loop) logInfo(message string, args ...any) {
	if l == nil || l.logger == nil {
		return
	}
	l.logger.Info(message, args...)
}

// cloneRequest 复制请求结构
func cloneRequest(req *schema.Request) *schema.Request {
	if req == nil {
		return nil
	}

	cloned := *req
	cloned.Messages = cloneMessages(req.Messages)
	cloned.Tools = cloneTools(req.Tools)
	cloned.Metadata = cloneMetadata(req.Metadata)
	return &cloned
}

// cloneMessages 复制消息列表
func cloneMessages(messages []schema.Message) []schema.Message {
	if len(messages) == 0 {
		return nil
	}

	result := make([]schema.Message, len(messages))
	for index, message := range messages {
		result[index] = schema.Message{
			Role:       message.Role,
			Content:    message.Content,
			Name:       message.Name,
			ToolCallID: message.ToolCallID,
			ToolCalls:  cloneToolInvocations(message.ToolCalls),
		}
	}
	return result
}

// cloneTools 复制工具定义列表
func cloneTools(tools []schema.ToolDefinition) []schema.ToolDefinition {
	if len(tools) == 0 {
		return nil
	}

	result := make([]schema.ToolDefinition, len(tools))
	for index, tool := range tools {
		result[index] = schema.ToolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: cloneMetadata(tool.InputSchema),
		}
	}
	return result
}

// cloneMetadata 复制元数据映射
func cloneMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return nil
	}

	result := make(map[string]any, len(metadata))
	for key, value := range metadata {
		result[key] = value
	}
	return result
}

// cloneToolInvocations 复制工具调用列表
func cloneToolInvocations(toolCalls []schema.ToolInvocation) []schema.ToolInvocation {
	if len(toolCalls) == 0 {
		return nil
	}

	result := make([]schema.ToolInvocation, len(toolCalls))
	copy(result, toolCalls)
	return result
}
