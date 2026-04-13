package core

import (
	"fmt"

	"github.com/Becks723/mind-gateway/core/schema"
)

// CloneRequest 创建一个可供重试或降级使用的请求副本
func CloneRequest(req *schema.Request) *schema.Request {
	// 处理空请求
	if req == nil {
		return nil
	}

	// 拷贝基础字段
	cloned := *req
	cloned.Messages = cloneMessages(req.Messages)
	cloned.Tools = cloneTools(req.Tools)
	cloned.Metadata = cloneMetadata(req.Metadata)

	return &cloned
}

// validateRequest 校验请求基础字段
func validateRequest(req *schema.Request) error {
	// 校验请求和消息列表
	if req == nil {
		return fmt.Errorf("请求不能为空")
	}
	if len(req.Messages) == 0 {
		return fmt.Errorf("messages 不能为空")
	}

	return nil
}

// cloneMessages 复制消息列表
func cloneMessages(messages []schema.Message) []schema.Message {
	if len(messages) == 0 {
		return nil
	}

	result := make([]schema.Message, len(messages))
	copy(result, messages)
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
