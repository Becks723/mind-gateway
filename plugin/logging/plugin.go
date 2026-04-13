package logging

import (
	"context"

	"github.com/Becks723/mind-gateway/core/schema"
	frameworklogging "github.com/Becks723/mind-gateway/framework/logging"
)

// Plugin 表示日志插件实现
type Plugin struct {
	logger *frameworklogging.Logger // logger 表示结构化日志记录器
}

// NewPlugin 创建新的日志插件
func NewPlugin(logger *frameworklogging.Logger) *Plugin {
	return &Plugin{
		logger: logger,
	}
}

// Name 返回插件名称
func (p *Plugin) Name() string {
	return "logging"
}

// PreHook 执行前置日志记录
func (p *Plugin) PreHook(ctx context.Context, req *schema.Request) (*schema.PreHookResult, error) {
	// 记录请求开始
	if p.logger != nil && req != nil {
		p.logger.Info(
			"插件记录请求开始",
			"plugin", p.Name(),
			"request_id", req.RequestID,
			"provider", req.Provider,
			"model", req.Model,
			"message_count", len(req.Messages),
			"retry_count", req.RetryCount,
			"fallback_index", req.FallbackIndex,
		)
	}

	return nil, nil
}

// PostHook 执行后置日志记录
func (p *Plugin) PostHook(ctx context.Context, req *schema.Request, resp *schema.Response, runErr error) (*schema.Response, error) {
	// 记录请求结束
	if p.logger != nil && req != nil {
		if runErr != nil {
			p.logger.Error(
				"插件记录请求失败",
				"plugin", p.Name(),
				"request_id", req.RequestID,
				"provider", req.Provider,
				"model", req.Model,
				"retry_count", req.RetryCount,
				"fallback_index", req.FallbackIndex,
				"error", runErr.Error(),
			)
			return nil, runErr
		}

		latencyMilliseconds := int64(0)
		if resp != nil {
			latencyMilliseconds = resp.Latency.Milliseconds()
		}
		p.logger.Info(
			"插件记录请求完成",
			"plugin", p.Name(),
			"request_id", req.RequestID,
			"provider", req.Provider,
			"model", req.Model,
			"retry_count", req.RetryCount,
			"fallback_index", req.FallbackIndex,
			"latency_ms", latencyMilliseconds,
		)
	}

	return resp, nil
}
