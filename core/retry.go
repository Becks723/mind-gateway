package core

import (
	"context"
	"errors"
	"time"
)

// NonRetryableError 表示不可重试且不应继续降级的错误
type NonRetryableError struct {
	Err error // Err 表示原始错误对象
}

// Error 返回错误文本
func (e *NonRetryableError) Error() string {
	if e == nil || e.Err == nil {
		return "不可重试错误"
	}

	return e.Err.Error()
}

// Unwrap 返回原始错误
func (e *NonRetryableError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

// MarkNonRetryable 将普通错误包装为不可重试错误
func MarkNonRetryable(err error) error {
	if err == nil {
		return nil
	}

	return &NonRetryableError{Err: err}
}

// IsNonRetryable 判断错误是否为不可重试错误
func IsNonRetryable(err error) bool {
	var target *NonRetryableError
	return errors.As(err, &target)
}

// IsRetryable 判断当前错误是否允许重试
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if IsNonRetryable(err) {
		return false
	}

	return true
}

// calculateBackoff 计算指定重试次数的退避时长
func (g *Gateway) calculateBackoff(retryCount int) time.Duration {
	// 补齐初始退避配置
	backoff := g.config.RetryBackoff
	if backoff <= 0 {
		backoff = 200 * time.Millisecond
	}

	// 计算指数退避
	for attempt := 0; attempt < retryCount; attempt++ {
		backoff *= 2
		if g.config.MaxBackoff > 0 && backoff >= g.config.MaxBackoff {
			return g.config.MaxBackoff
		}
	}

	return backoff
}

// sleepBackoff 在上下文控制下等待退避时长
func (g *Gateway) sleepBackoff(ctx context.Context, retryCount int) error {
	// 计算目标退避时间
	backoff := g.calculateBackoff(retryCount)
	timer := time.NewTimer(backoff)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
