package mock

import (
	"context"
	"fmt"
	"time"

	"github.com/Becks723/mind-gateway/internal/core"
)

// Provider 表示本地 mock Provider 实现
type Provider struct {
	name            string // name 表示 Provider 名称
	responseText    string // responseText 表示固定返回文本
	simulateFailure bool   // simulateFailure 表示是否模拟失败
}

// New 创建新的 mock Provider
func New(name string, responseText string) *Provider {
	// 兜底设置 Provider 默认值
	finalName := name
	if finalName == "" {
		finalName = "mock"
	}
	finalResponse := responseText
	if finalResponse == "" {
		finalResponse = "hello from mock provider"
	}

	return &Provider{
		name:         finalName,
		responseText: finalResponse,
	}
}

// NewWithFailure 创建会模拟失败的 mock Provider
func NewWithFailure(name string, responseText string) *Provider {
	// 基于普通 mock Provider 打开失败模拟
	provider := New(name, responseText)
	provider.simulateFailure = true
	return provider
}

// Name 返回 Provider 名称
func (p *Provider) Name() string {
	return p.name
}

// Type 返回 Provider 类型
func (p *Provider) Type() string {
	return "mock"
}

// Chat 执行非流式聊天请求
func (p *Provider) Chat(ctx context.Context, req *core.Request) (*core.Response, error) {
	// 处理上下文取消
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// 模拟 Provider 执行失败
	if p.simulateFailure {
		return nil, fmt.Errorf("mock provider %q 模拟失败", p.name)
	}

	// 生成统一响应对象
	now := time.Now()
	response := &core.Response{
		RequestID:  req.RequestID,
		Provider:   p.name,
		Model:      req.Model,
		OutputText: p.responseText,
		Messages: []core.Message{
			{
				Role:    "assistant",
				Content: p.responseText,
			},
		},
		FinishReason: "stop",
		Usage: core.Usage{
			InputTokens:  int64(len(req.Messages)),
			OutputTokens: 1,
			TotalTokens:  int64(len(req.Messages)) + 1,
		},
		Latency: time.Since(now),
	}

	return response, nil
}

// ChatStream 执行流式聊天请求
func (p *Provider) ChatStream(ctx context.Context, req *core.Request) (<-chan core.StreamEvent, <-chan error) {
	// 创建流式事件通道和错误通道
	eventCh := make(chan core.StreamEvent, 2)
	errCh := make(chan error, 1)

	go func() {
		defer close(eventCh)
		defer close(errCh)

		// 处理上下文取消
		select {
		case <-ctx.Done():
			errCh <- ctx.Err()
			return
		default:
		}

		// 模拟 Provider 执行失败
		if p.simulateFailure {
			errCh <- fmt.Errorf("mock provider %q 模拟流式失败", p.name)
			return
		}

		// 发送一段内容事件和结束事件
		eventCh <- core.StreamEvent{
			RequestID: req.RequestID,
			Delta:     p.responseText,
			Done:      false,
		}
		eventCh <- core.StreamEvent{
			RequestID: req.RequestID,
			Delta:     "",
			Done:      true,
		}
	}()

	return eventCh, errCh
}
