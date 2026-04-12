package core

import "time"

// Message 定义内部统一消息结构
type Message struct {
	Role       string           `json:"role"`                   // Role 表示消息角色
	Content    string           `json:"content"`                // Content 表示消息文本内容
	Name       string           `json:"name,omitempty"`         // Name 表示可选消息名称
	ToolCallID string           `json:"tool_call_id,omitempty"` // ToolCallID 表示工具调用关联 ID
	ToolCalls  []ToolInvocation `json:"tool_calls,omitempty"`   // ToolCalls 表示模型返回的工具调用列表
}

// ToolDefinition 定义可供模型调用的工具信息
type ToolDefinition struct {
	Name        string         `json:"name"`         // Name 表示工具名称
	Description string         `json:"description"`  // Description 表示工具说明
	InputSchema map[string]any `json:"input_schema"` // InputSchema 表示工具输入模式
}

// ToolInvocation 定义一次工具调用结果
type ToolInvocation struct {
	ID        string `json:"id"`        // ID 表示工具调用 ID
	Name      string `json:"name"`      // Name 表示工具名称
	Arguments string `json:"arguments"` // Arguments 表示工具调用参数
	Result    string `json:"result"`    // Result 表示工具执行结果
}

// Usage 定义一次请求的 Token 使用量
type Usage struct {
	InputTokens  int64 `json:"input_tokens"`  // InputTokens 表示输入 Token 数
	OutputTokens int64 `json:"output_tokens"` // OutputTokens 表示输出 Token 数
	TotalTokens  int64 `json:"total_tokens"`  // TotalTokens 表示总 Token 数
}

// Request 定义网关内部统一请求结构
type Request struct {
	RequestID     string           `json:"request_id"`            // RequestID 表示请求唯一标识
	Provider      string           `json:"provider"`              // Provider 表示目标 Provider 名称
	Model         string           `json:"model"`                 // Model 表示模型名称
	Messages      []Message        `json:"messages"`              // Messages 表示对话消息列表
	Stream        bool             `json:"stream"`                // Stream 表示是否为流式请求
	Temperature   *float64         `json:"temperature,omitempty"` // Temperature 表示采样温度
	MaxTokens     *int             `json:"max_tokens,omitempty"`  // MaxTokens 表示最大输出 Token 数
	Tools         []ToolDefinition `json:"tools,omitempty"`       // Tools 表示可用工具列表
	Metadata      map[string]any   `json:"metadata,omitempty"`    // Metadata 表示附加元数据
	VirtualKey    string           `json:"virtual_key,omitempty"` // VirtualKey 表示虚拟密钥
	FallbackIndex int              `json:"fallback_index"`        // FallbackIndex 表示当前降级序号
	RetryCount    int              `json:"retry_count"`           // RetryCount 表示当前重试次数
	StartedAt     time.Time        `json:"started_at"`            // StartedAt 表示请求开始时间
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
