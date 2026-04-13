package core

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Becks723/mind-gateway/core/schema"
	frameworkconfig "github.com/Becks723/mind-gateway/framework/config"
	frameworklogging "github.com/Becks723/mind-gateway/framework/logging"
	"github.com/Becks723/mind-gateway/provider"
	mockprovider "github.com/Becks723/mind-gateway/provider/mock"
)

// flakyProvider 表示会先失败后成功的测试 Provider
type flakyProvider struct {
	name     string       // name 表示 Provider 名称
	failures atomic.Int32 // failures 表示剩余失败次数
}

// newFlakyProvider 创建新的 flaky Provider
func newFlakyProvider(name string, failureCount int32) *flakyProvider {
	provider := &flakyProvider{name: name}
	provider.failures.Store(failureCount)
	return provider
}

// Name 返回 Provider 名称
func (p *flakyProvider) Name() string {
	return p.name
}

// Type 返回 Provider 类型
func (p *flakyProvider) Type() string {
	return "test_flaky"
}

// Chat 执行非流式聊天请求
func (p *flakyProvider) Chat(ctx context.Context, req *schema.Request) (*schema.Response, error) {
	// 模拟前几次失败
	if p.failures.Load() > 0 {
		p.failures.Add(-1)
		return nil, fmt.Errorf("临时错误")
	}

	return &schema.Response{
		RequestID:  req.RequestID,
		Provider:   p.name,
		Model:      req.Model,
		OutputText: "重试成功",
	}, nil
}

// ChatStream 执行流式聊天请求
func (p *flakyProvider) ChatStream(ctx context.Context, req *schema.Request) (<-chan schema.StreamEvent, <-chan error) {
	eventCh := make(chan schema.StreamEvent)
	errCh := make(chan error, 1)
	close(eventCh)
	errCh <- fmt.Errorf("未实现")
	close(errCh)
	return eventCh, errCh
}

// fatalProvider 表示返回不可重试错误的测试 Provider
type fatalProvider struct {
	name string // name 表示 Provider 名称
}

// Name 返回 Provider 名称
func (p *fatalProvider) Name() string {
	return p.name
}

// Type 返回 Provider 类型
func (p *fatalProvider) Type() string {
	return "test_fatal"
}

// Chat 执行非流式聊天请求
func (p *fatalProvider) Chat(ctx context.Context, req *schema.Request) (*schema.Response, error) {
	return nil, MarkNonRetryable(fmt.Errorf("致命错误"))
}

// ChatStream 执行流式聊天请求
func (p *fatalProvider) ChatStream(ctx context.Context, req *schema.Request) (<-chan schema.StreamEvent, <-chan error) {
	eventCh := make(chan schema.StreamEvent)
	errCh := make(chan error, 1)
	close(eventCh)
	errCh <- fmt.Errorf("未实现")
	close(errCh)
	return eventCh, errCh
}

// TestGatewayHandleChat 验证网关可以通过队列调度 mock Provider
func TestGatewayHandleChat(t *testing.T) {
	// 创建注册表并注册 mock Provider
	registry := provider.NewRegistry()
	if err := registry.Register(mockprovider.New("mock", "你好，我是 mock")); err != nil {
		t.Fatalf("注册 mock Provider 失败: %v", err)
	}

	gateway := NewGateway(frameworkconfig.GatewayConfig{
		DefaultProvider:    "mock",
		DefaultModel:       "mock-gpt",
		QueueSize:          8,
		WorkersPerProvider: 1,
	}, registry, frameworklogging.NewLogger("error"), nil)

	// 构造聊天请求并执行
	resp, err := gateway.HandleChat(context.Background(), &schema.Request{
		RequestID: "req-1",
		Messages: []schema.Message{
			{
				Role:    "user",
				Content: "你好",
			},
		},
	})
	if err != nil {
		t.Fatalf("执行聊天请求失败: %v", err)
	}

	// 校验响应内容
	if resp.Provider != "mock" {
		t.Fatalf("期望 Provider 为 mock，实际得到 %q", resp.Provider)
	}
	if resp.OutputText != "你好，我是 mock" {
		t.Fatalf("期望输出为 mock 返回内容，实际得到 %q", resp.OutputText)
	}
}

// TestGatewayHandleChatWithEmptyMessages 验证空消息会被拒绝
func TestGatewayHandleChatWithEmptyMessages(t *testing.T) {
	// 创建最小网关
	gateway := NewGateway(frameworkconfig.GatewayConfig{
		DefaultProvider:    "mock",
		DefaultModel:       "mock-gpt",
		QueueSize:          8,
		WorkersPerProvider: 1,
	}, provider.NewRegistry(), frameworklogging.NewLogger("error"), nil)

	// 执行空消息请求并校验错误
	if _, err := gateway.HandleChat(context.Background(), &schema.Request{
		RequestID: "req-2",
	}); err == nil {
		t.Fatal("期望空消息请求返回错误")
	}
}

// TestGatewayConcurrentHandleChat 验证网关可以消费并发请求
func TestGatewayConcurrentHandleChat(t *testing.T) {
	// 创建注册表并注册 mock Provider
	registry := provider.NewRegistry()
	if err := registry.Register(mockprovider.New("mock", "并发响应")); err != nil {
		t.Fatalf("注册 mock Provider 失败: %v", err)
	}

	gateway := NewGateway(frameworkconfig.GatewayConfig{
		DefaultProvider:    "mock",
		DefaultModel:       "mock-gpt",
		RequestTimeout:     3 * time.Second,
		QueueSize:          64,
		WorkersPerProvider: 4,
	}, registry, frameworklogging.NewLogger("error"), nil)

	// 并发发起 20 个请求
	var wg sync.WaitGroup
	errCh := make(chan error, 20)
	for index := 0; index < 20; index++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			resp, err := gateway.HandleChat(context.Background(), &schema.Request{
				RequestID: fmt.Sprintf("req-%d", index),
				Messages: []schema.Message{
					{
						Role:    "user",
						Content: "你好",
					},
				},
			})
			if err != nil {
				errCh <- err
				return
			}
			if resp.OutputText != "并发响应" {
				errCh <- fmt.Errorf("期望输出为 并发响应，实际得到 %q", resp.OutputText)
			}
		}(index)
	}
	wg.Wait()
	close(errCh)

	// 汇总并发执行结果
	for err := range errCh {
		if err != nil {
			t.Fatalf("并发请求失败: %v", err)
		}
	}
}

// TestGatewayShutdown 验证网关可以优雅关闭队列
func TestGatewayShutdown(t *testing.T) {
	// 创建注册表并注册 mock Provider
	registry := provider.NewRegistry()
	if err := registry.Register(mockprovider.New("mock", "关闭测试")); err != nil {
		t.Fatalf("注册 mock Provider 失败: %v", err)
	}

	gateway := NewGateway(frameworkconfig.GatewayConfig{
		DefaultProvider:    "mock",
		DefaultModel:       "mock-gpt",
		QueueSize:          8,
		WorkersPerProvider: 1,
	}, registry, frameworklogging.NewLogger("error"), nil)

	// 执行关闭动作并校验结果
	if err := gateway.Shutdown(context.Background()); err != nil {
		t.Fatalf("关闭网关失败: %v", err)
	}
}

// TestGatewayRetry 验证可重试错误会触发重试
func TestGatewayRetry(t *testing.T) {
	// 创建注册表并注册会先失败一次的 Provider
	registry := provider.NewRegistry()
	if err := registry.Register(newFlakyProvider("flaky", 1)); err != nil {
		t.Fatalf("注册 flaky Provider 失败: %v", err)
	}

	gateway := NewGateway(frameworkconfig.GatewayConfig{
		DefaultProvider:    "flaky",
		DefaultModel:       "mock-gpt",
		MaxRetries:         1,
		RetryBackoff:       time.Millisecond,
		MaxBackoff:         10 * time.Millisecond,
		QueueSize:          8,
		WorkersPerProvider: 1,
	}, registry, frameworklogging.NewLogger("error"), nil)

	// 执行请求并校验最终成功
	resp, err := gateway.HandleChat(context.Background(), &schema.Request{
		RequestID: "retry-1",
		Messages: []schema.Message{
			{
				Role:    "user",
				Content: "你好",
			},
		},
	})
	if err != nil {
		t.Fatalf("期望重试后成功，实际失败: %v", err)
	}
	if resp.OutputText != "重试成功" {
		t.Fatalf("期望输出为 重试成功，实际得到 %q", resp.OutputText)
	}
}

// TestGatewayFallback 验证主 Provider 失败后可以切到 fallback
func TestGatewayFallback(t *testing.T) {
	// 创建注册表并注册失败 Provider 与 fallback Provider
	registry := provider.NewRegistry()
	if err := registry.Register(newFlakyProvider("primary", 2)); err != nil {
		t.Fatalf("注册 primary Provider 失败: %v", err)
	}
	if err := registry.Register(mockprovider.New("backup", "fallback 成功")); err != nil {
		t.Fatalf("注册 backup Provider 失败: %v", err)
	}

	gateway := NewGateway(frameworkconfig.GatewayConfig{
		DefaultProvider:    "primary",
		DefaultModel:       "mock-gpt",
		MaxRetries:         0,
		QueueSize:          8,
		WorkersPerProvider: 1,
	}, registry, frameworklogging.NewLogger("error"), []frameworkconfig.ProviderConfig{
		{
			Name:      "primary",
			Type:      "mock",
			Enabled:   true,
			Fallbacks: []string{"backup"},
		},
		{
			Name:    "backup",
			Type:    "mock",
			Enabled: true,
		},
	})

	// 执行请求并校验 fallback 成功
	resp, err := gateway.HandleChat(context.Background(), &schema.Request{
		RequestID: "fallback-1",
		Messages: []schema.Message{
			{
				Role:    "user",
				Content: "你好",
			},
		},
	})
	if err != nil {
		t.Fatalf("期望 fallback 成功，实际失败: %v", err)
	}
	if resp.Provider != "backup" {
		t.Fatalf("期望实际 provider 为 backup，实际得到 %q", resp.Provider)
	}
	if resp.OutputText != "fallback 成功" {
		t.Fatalf("期望输出为 fallback 成功，实际得到 %q", resp.OutputText)
	}
}

// TestGatewayNonRetryableError 验证不可重试错误不会继续重试或降级
func TestGatewayNonRetryableError(t *testing.T) {
	// 创建注册表并注册致命错误 Provider
	registry := provider.NewRegistry()
	if err := registry.Register(&fatalProvider{name: "fatal"}); err != nil {
		t.Fatalf("注册 fatal Provider 失败: %v", err)
	}
	if err := registry.Register(mockprovider.New("backup", "不会被执行")); err != nil {
		t.Fatalf("注册 backup Provider 失败: %v", err)
	}

	gateway := NewGateway(frameworkconfig.GatewayConfig{
		DefaultProvider:    "fatal",
		DefaultModel:       "mock-gpt",
		MaxRetries:         2,
		QueueSize:          8,
		WorkersPerProvider: 1,
	}, registry, frameworklogging.NewLogger("error"), []frameworkconfig.ProviderConfig{
		{
			Name:      "fatal",
			Type:      "mock",
			Enabled:   true,
			Fallbacks: []string{"backup"},
		},
	})

	// 执行请求并校验不会重试成功
	_, err := gateway.HandleChat(context.Background(), &schema.Request{
		RequestID: "fatal-1",
		Messages: []schema.Message{
			{
				Role:    "user",
				Content: "你好",
			},
		},
	})
	if err == nil {
		t.Fatal("期望返回不可重试错误")
	}
	if !IsNonRetryable(err) {
		t.Fatalf("期望错误为不可重试错误，实际得到 %v", err)
	}
}
