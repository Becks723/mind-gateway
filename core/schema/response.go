package schema

import "time"

// Usage 定义一次请求的 Token 使用量
type Usage struct {
	InputTokens  int64 `json:"input_tokens"`  // InputTokens 表示输入 Token 数
	OutputTokens int64 `json:"output_tokens"` // OutputTokens 表示输出 Token 数
	TotalTokens  int64 `json:"total_tokens"`  // TotalTokens 表示总 Token 数
}

// Response 定义网关内部统一响应结构
type Response struct {
	RequestID    string           `json:"request_id"`           // RequestID 表示请求唯一标识
	Provider     string           `json:"provider"`             // Provider 表示实际执行的 Provider 名称
	Model        string           `json:"model"`                // Model 表示实际执行的模型名称
	OutputText   string           `json:"output_text"`          // OutputText 表示最终文本输出
	Messages     []Message        `json:"messages"`             // Messages 表示返回消息列表
	FinishReason string           `json:"finish_reason"`        // FinishReason 表示模型结束原因
	Usage        Usage            `json:"usage"`                // Usage 表示 Token 使用量
	ToolCalls    []ToolInvocation `json:"tool_calls,omitempty"` // ToolCalls 表示工具调用结果列表
	Latency      time.Duration    `json:"latency"`              // Latency 表示请求耗时
}

// StreamEvent 定义流式返回的单个事件
type StreamEvent struct {
	RequestID string `json:"request_id"` // RequestID 表示请求唯一标识
	Delta     string `json:"delta"`      // Delta 表示本次流式增量内容
	Done      bool   `json:"done"`       // Done 表示流是否结束
}
