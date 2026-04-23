package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Becks723/mind-gateway/core/schema"
)

// RegisterBuiltinTools 按名称注册内置工具
func RegisterBuiltinTools(registry *Registry, allowedTools []string) error {
	// 计算最终要注册的工具列表
	finalTools := allowedTools
	if len(finalTools) == 0 {
		finalTools = []string{"current_time", "echo"}
	}

	for _, toolName := range finalTools {
		if err := registerBuiltinTool(registry, toolName); err != nil {
			return err
		}
	}

	return nil
}

// registerBuiltinTool 注册单个内置工具
func registerBuiltinTool(registry *Registry, toolName string) error {
	switch toolName {
	case "current_time":
		return registry.Register(schema.ToolDefinition{
			Name:        "current_time",
			Description: "返回当前时间，默认使用 RFC3339 格式",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"format": map[string]any{
						"type":        "string",
						"description": "可选时间格式，默认 rfc3339",
					},
				},
			},
		}, currentTimeTool)
	case "echo":
		return registry.Register(schema.ToolDefinition{
			Name:        "echo",
			Description: "原样回显输入文本",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"text": map[string]any{
						"type":        "string",
						"description": "需要回显的文本",
					},
				},
				"required": []string{"text"},
			},
		}, echoTool)
	default:
		return fmt.Errorf("暂不支持的内置工具: %s", toolName)
	}
}

// currentTimeTool 返回当前时间
func currentTimeTool(ctx context.Context, arguments string) (string, error) {
	// 解析可选的时间格式参数
	type input struct {
		Format string `json:"format"` // Format 表示时间输出格式
	}

	parsed := input{}
	if arguments != "" {
		if err := json.Unmarshal([]byte(arguments), &parsed); err != nil {
			return "", fmt.Errorf("解析 current_time 参数失败: %w", err)
		}
	}

	format := ""
	switch parsed.Format {
	case "", "rfc3339":
		format = time.RFC3339
	case "datetime":
		format = "2006-01-02 15:04:05"
	default:
		format = parsed.Format
	}

	return time.Now().Format(format), nil
}

// echoTool 回显输入文本
func echoTool(ctx context.Context, arguments string) (string, error) {
	// 解析回显参数
	type input struct {
		Text string `json:"text"` // Text 表示需要回显的文本
	}

	var parsed input
	if err := json.Unmarshal([]byte(arguments), &parsed); err != nil {
		return "", fmt.Errorf("解析 echo 参数失败: %w", err)
	}
	if parsed.Text == "" {
		return "", fmt.Errorf("echo 工具缺少 text 参数")
	}

	return parsed.Text, nil
}
