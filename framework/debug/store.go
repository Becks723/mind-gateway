package debug

import "sync"

// RequestSummary 表示一条调试请求摘要
type RequestSummary struct {
	RequestID  string `json:"request_id"`  // RequestID 表示请求唯一标识
	Path       string `json:"path"`        // Path 表示请求路径
	StatusCode int    `json:"status_code"` // StatusCode 表示响应状态码
}

// Store 表示内存调试存储
type Store struct {
	mu       sync.RWMutex     // mu 表示读写锁
	capacity int              // capacity 表示最大容量
	items    []RequestSummary // items 表示请求摘要列表
}

// NewStore 创建新的调试存储
func NewStore(capacity int) *Store {
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
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.items) >= s.capacity {
		s.items = s.items[1:]
	}
	s.items = append(s.items, item)
}

// List 返回所有请求摘要
func (s *Store) List() []RequestSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]RequestSummary, len(s.items))
	copy(result, s.items)
	return result
}
