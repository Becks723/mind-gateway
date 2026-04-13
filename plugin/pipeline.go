package plugin

import (
	"context"
	"fmt"

	"github.com/Becks723/mind-gateway/core/schema"
)

// Pipeline 表示插件执行管线
type Pipeline struct {
	plugins []schema.Plugin // plugins 表示按顺序注册的插件列表
}

// NewPipeline 创建新的插件执行管线
func NewPipeline(plugins ...schema.Plugin) *Pipeline {
	// 过滤空插件并保持原始顺序
	filtered := make([]schema.Plugin, 0, len(plugins))
	for _, instance := range plugins {
		if instance == nil {
			continue
		}
		filtered = append(filtered, instance)
	}

	return &Pipeline{
		plugins: filtered,
	}
}

// Plugins 返回当前已注册的插件列表
func (p *Pipeline) Plugins() []schema.Plugin {
	// 返回插件切片副本，避免外部修改内部状态
	result := make([]schema.Plugin, len(p.plugins))
	copy(result, p.plugins)
	return result
}

// RunPreHooks 顺序执行全部前置钩子
func (p *Pipeline) RunPreHooks(ctx context.Context, req *schema.Request) (*schema.Request, *schema.PreHookResult, int, error) {
	// 处理空管线
	if p == nil || len(p.plugins) == 0 {
		return req, nil, 0, nil
	}

	currentReq := req
	executedCount := 0
	for _, instance := range p.plugins {
		// 执行单个插件的前置钩子
		result, err := instance.PreHook(ctx, currentReq)
		if err != nil {
			return currentReq, nil, executedCount, fmt.Errorf("插件 %q 的 PreHook 执行失败: %w", instance.Name(), err)
		}
		executedCount++
		if result != nil && result.ShortCircuit {
			return currentReq, result, executedCount, nil
		}
	}

	return currentReq, nil, executedCount, nil
}

// RunPostHooks 逆序执行已命中的后置钩子
func (p *Pipeline) RunPostHooks(ctx context.Context, req *schema.Request, resp *schema.Response, runErr error, executedCount int) (*schema.Response, error) {
	// 处理空管线
	if p == nil || len(p.plugins) == 0 || executedCount == 0 {
		return resp, runErr
	}

	currentResp := resp
	currentErr := runErr
	for index := executedCount - 1; index >= 0; index-- {
		// 逆序执行插件后置钩子
		instance := p.plugins[index]
		nextResp, err := instance.PostHook(ctx, req, currentResp, currentErr)
		if err != nil {
			currentErr = fmt.Errorf("插件 %q 的 PostHook 执行失败: %w", instance.Name(), err)
			currentResp = nil
			continue
		}
		currentResp = nextResp
		currentErr = nil
	}

	return currentResp, currentErr
}
