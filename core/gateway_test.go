package core

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Becks723/mind-gateway/core/schema"
	frameworkconfig "github.com/Becks723/mind-gateway/framework/config"
	frameworklogging "github.com/Becks723/mind-gateway/framework/logging"
	"github.com/Becks723/mind-gateway/provider"
	mockprovider "github.com/Becks723/mind-gateway/provider/mock"
)

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
	}, registry, frameworklogging.NewLogger("error"))

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
	}, provider.NewRegistry(), frameworklogging.NewLogger("error"))

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
	}, registry, frameworklogging.NewLogger("error"))

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
	}, registry, frameworklogging.NewLogger("error"))

	// 执行关闭动作并校验结果
	if err := gateway.Shutdown(context.Background()); err != nil {
		t.Fatalf("关闭网关失败: %v", err)
	}
}
