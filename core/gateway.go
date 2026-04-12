package core

import (
	"context"
	"fmt"
	"time"

	"github.com/Becks723/mind-gateway/core/schema"
	frameworkconfig "github.com/Becks723/mind-gateway/framework/config"
	"github.com/Becks723/mind-gateway/provider"
)

// Gateway 表示网关核心执行入口
type Gateway struct {
	config   frameworkconfig.GatewayConfig // config 表示网关核心配置
	registry *provider.Registry            // registry 表示 Provider 注册表
}

// NewGateway 创建新的网关核心对象
func NewGateway(cfg frameworkconfig.GatewayConfig, registry *provider.Registry) *Gateway {
	return &Gateway{
		config:   cfg,
		registry: registry,
	}
}

// HandleChat 处理非流式聊天请求
func (g *Gateway) HandleChat(ctx context.Context, req *schema.Request) (*schema.Response, error) {
	// 补齐请求基础字段
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}
	if req.Model == "" {
		req.Model = g.config.DefaultModel
	}
	if req.Provider == "" {
		req.Provider = g.config.DefaultProvider
	}
	if req.StartedAt.IsZero() {
		req.StartedAt = time.Now()
	}

	// 校验请求必要字段
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("messages 不能为空")
	}

	// 获取目标 Provider
	targetProvider, err := g.registry.MustGet(req.Provider)
	if err != nil {
		return nil, err
	}

	// 调用 Provider 执行请求
	resp, err := targetProvider.Chat(ctx, req)
	if err != nil {
		return nil, err
	}

	// 回填响应耗时和基础字段
	resp.RequestID = req.RequestID
	resp.Provider = targetProvider.Name()
	if resp.Model == "" {
		resp.Model = req.Model
	}
	resp.Latency = time.Since(req.StartedAt)

	return resp, nil
}
