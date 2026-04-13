package core

import "context"

// withTimeout 根据网关配置为任务上下文补齐超时
func (g *Gateway) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	// 当未配置超时时直接返回原始上下文
	if g.config.RequestTimeout <= 0 {
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, g.config.RequestTimeout)
}
