package core

import (
	"context"
	"fmt"
)

// Shutdown 优雅关闭所有 Provider 队列和工作协程
func (g *Gateway) Shutdown(ctx context.Context) error {
	// 快速处理空网关
	if g == nil {
		return nil
	}

	// 复制当前全部队列，避免长时间持锁
	g.mu.RLock()
	queues := make([]*ProviderQueue, 0, len(g.queues))
	for _, queue := range g.queues {
		queues = append(queues, queue)
	}
	g.mu.RUnlock()

	// 逐个关闭队列
	for _, queue := range queues {
		if queue.Closing.CompareAndSwap(false, true) {
			close(queue.Queue)
		}
	}

	// 等待所有 worker 退出
	done := make(chan struct{})
	go func() {
		defer close(done)
		for _, queue := range queues {
			queue.WG.Wait()
		}
	}()

	select {
	case <-done:
		g.logger.Info("网关工作协程已全部退出")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("等待网关关闭超时: %w", ctx.Err())
	}
}
