package openai

// ChatCompletionRequest 定义 OpenAI-compatible 的聊天请求
type ChatCompletionRequest struct {
	Model       string         `json:"model"`                 // Model 表示目标模型名称
	Messages    []ChatMessage  `json:"messages"`              // Messages 表示对话消息列表
	Stream      bool           `json:"stream,omitempty"`      // Stream 表示是否启用流式响应
	Temperature *float64       `json:"temperature,omitempty"` // Temperature 表示采样温度
	MaxTokens   *int           `json:"max_tokens,omitempty"`  // MaxTokens 表示最大输出 Token 数
	User        string         `json:"user,omitempty"`        // User 表示调用方用户标识
	Metadata    map[string]any `json:"metadata,omitempty"`    // Metadata 表示附加元数据
}

// ChatMessage 定义 OpenAI-compatible 的单条消息
type ChatMessage struct {
	Role    string `json:"role"`           // Role 表示消息角色
	Content string `json:"content"`        // Content 表示消息文本内容
	Name    string `json:"name,omitempty"` // Name 表示可选消息名称
}

// ChatCompletionResponse 定义 OpenAI-compatible 的聊天响应
type ChatCompletionResponse struct {
	ID      string       `json:"id"`      // ID 表示请求唯一标识
	Object  string       `json:"object"`  // Object 表示响应对象类型
	Created int64        `json:"created"` // Created 表示响应创建时间
	Model   string       `json:"model"`   // Model 表示模型名称
	Choices []ChatChoice `json:"choices"` // Choices 表示候选回复列表
	Usage   Usage        `json:"usage"`   // Usage 表示 Token 使用量
}

// ChatChoice 定义聊天响应中的单个候选结果
type ChatChoice struct {
	Index        int         `json:"index"`         // Index 表示候选序号
	Message      ChatMessage `json:"message"`       // Message 表示候选消息内容
	FinishReason string      `json:"finish_reason"` // FinishReason 表示结束原因
}

// ChatCompletionChunkResponse 定义 OpenAI-compatible 的流式聊天响应
type ChatCompletionChunkResponse struct {
	ID      string            `json:"id"`              // ID 表示请求唯一标识
	Object  string            `json:"object"`          // Object 表示流式响应对象类型
	Created int64             `json:"created"`         // Created 表示响应创建时间
	Model   string            `json:"model"`           // Model 表示模型名称
	Choices []ChatChunkChoice `json:"choices"`         // Choices 表示流式候选结果列表
	Usage   *Usage            `json:"usage,omitempty"` // Usage 表示可选 Token 使用量
}

// ChatChunkChoice 定义流式聊天响应中的单个候选结果
type ChatChunkChoice struct {
	Index        int              `json:"index"`                   // Index 表示候选序号
	Delta        ChatMessageDelta `json:"delta"`                   // Delta 表示流式增量消息内容
	FinishReason string           `json:"finish_reason,omitempty"` // FinishReason 表示结束原因
}

// ChatMessageDelta 定义流式消息增量结构
type ChatMessageDelta struct {
	Role    string `json:"role,omitempty"`    // Role 表示消息角色
	Content string `json:"content,omitempty"` // Content 表示消息增量文本
}

// Usage 定义 OpenAI-compatible 的 Token 使用量结构
type Usage struct {
	PromptTokens     int64 `json:"prompt_tokens"`     // PromptTokens 表示输入 Token 数
	CompletionTokens int64 `json:"completion_tokens"` // CompletionTokens 表示输出 Token 数
	TotalTokens      int64 `json:"total_tokens"`      // TotalTokens 表示总 Token 数
}
