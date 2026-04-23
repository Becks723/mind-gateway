package debug

import "sync"

// RequestSummary 表示一条调试请求摘要
type RequestSummary struct {
	RequestID     string `json:"request_id"`            // RequestID 表示请求唯一标识
	Method        string `json:"method"`                // Method 表示 HTTP 请求方法
	Path          string `json:"path"`                  // Path 表示请求路径
	StatusCode    int    `json:"status_code"`           // StatusCode 表示响应状态码
	LatencyMS     int64  `json:"latency_ms"`            // LatencyMS 表示请求处理耗时
	Provider      string `json:"provider,omitempty"`    // Provider 表示实际执行的 Provider 名称
	Model         string `json:"model,omitempty"`       // Model 表示实际执行的模型名称
	RetryCount    int    `json:"retry_count"`           // RetryCount 表示最终重试次数
	FallbackIndex int    `json:"fallback_index"`        // FallbackIndex 表示最终降级序号
	ToolCallCount int    `json:"tool_call_count"`       // ToolCallCount 表示本次请求涉及的工具调用数量
	VirtualKey    string `json:"virtual_key,omitempty"` // VirtualKey 表示本次请求使用的虚拟密钥
	ErrorType     string `json:"error_type,omitempty"`  // ErrorType 表示机器可读错误类型
	Stream        bool   `json:"stream"`                // Stream 表示本次请求是否为流式请求
}

// Store 表示内存调试存储
type Store struct {
	mu       sync.RWMutex     // mu 表示读写锁
	capacity int              // capacity 表示最大容量
	items    []RequestSummary // items 表示请求摘要列表
}

// NewStore 创建新的调试存储
func NewStore(capacity int) *Store {
	// 补齐默认容量
	if capacity <= 0 {
		capacity = 20
	}

	return &Store{
		capacity: capacity,
		items:    make([]RequestSummary, 0, capacity),
	}
}

// Add 添加一条请求摘要
func (s *Store) Add(item RequestSummary) {
	// 将摘要写入最近请求列表
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.items) >= s.capacity {
		s.items = s.items[1:]
	}
	s.items = append(s.items, item)
}

// List 返回所有请求摘要
func (s *Store) List() []RequestSummary {
	// 返回当前全部请求摘要副本
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]RequestSummary, len(s.items))
	copy(result, s.items)
	return result
}
