package core

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/Becks723/mind-gateway/core/schema"
)

// ProviderQueue 表示单个 Provider 的请求队列
type ProviderQueue struct {
	Name    string         // Name 表示 Provider 名称
	Queue   chan *WorkItem // Queue 表示待消费的工作队列
	Closing atomic.Bool    // Closing 表示当前队列是否处于关闭状态
	WG      sync.WaitGroup // WG 表示当前队列关联的工作协程组
}

// WorkItem 表示一条待执行的网关任务
type WorkItem struct {
	Ctx      context.Context  // Ctx 表示当前任务上下文
	Request  *schema.Request  // Request 表示统一请求对象
	Response chan *WorkResult // Response 表示任务结果通道
}

// WorkResult 表示单条任务的执行结果
type WorkResult struct {
	Response *schema.Response // Response 表示执行成功时的统一响应
	Err      error            // Err 表示执行失败时的错误对象
}

// newProviderQueue 创建新的 Provider 队列
func newProviderQueue(name string, queueSize int) *ProviderQueue {
	// 补齐默认队列容量
	finalQueueSize := queueSize
	if finalQueueSize <= 0 {
		finalQueueSize = 64
	}

	return &ProviderQueue{
		Name:  name,
		Queue: make(chan *WorkItem, finalQueueSize),
	}
}
