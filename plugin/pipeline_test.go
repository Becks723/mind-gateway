package plugin

import (
	"context"
	"testing"

	"github.com/Becks723/mind-gateway/core/schema"
)

// recordPlugin 表示用于记录执行顺序的测试插件
type recordPlugin struct {
	name   string    // name 表示插件名称
	events *[]string // events 表示执行事件记录
}

// Name 返回插件名称
func (p *recordPlugin) Name() string {
	return p.name
}

// PreHook 执行前置钩子
func (p *recordPlugin) PreHook(ctx context.Context, req *schema.Request) (*schema.PreHookResult, error) {
	*p.events = append(*p.events, "pre:"+p.name)
	return nil, nil
}

// PostHook 执行后置钩子
func (p *recordPlugin) PostHook(ctx context.Context, req *schema.Request, resp *schema.Response, runErr error) (*schema.Response, error) {
	*p.events = append(*p.events, "post:"+p.name)
	return resp, runErr
}

// shortCircuitPlugin 表示会短路的测试插件
type shortCircuitPlugin struct {
	name string // name 表示插件名称
}

// Name 返回插件名称
func (p *shortCircuitPlugin) Name() string {
	return p.name
}

// PreHook 执行前置钩子
func (p *shortCircuitPlugin) PreHook(ctx context.Context, req *schema.Request) (*schema.PreHookResult, error) {
	return &schema.PreHookResult{
		ShortCircuit: true,
		Response: &schema.Response{
			OutputText: "short-circuit",
		},
	}, nil
}

// PostHook 执行后置钩子
func (p *shortCircuitPlugin) PostHook(ctx context.Context, req *schema.Request, resp *schema.Response, runErr error) (*schema.Response, error) {
	return resp, runErr
}

// TestPipelineOrder 验证插件前后置钩子的执行顺序
func TestPipelineOrder(t *testing.T) {
	// 创建测试插件管线
	events := make([]string, 0, 4)
	pipeline := NewPipeline(
		&recordPlugin{name: "a", events: &events},
		&recordPlugin{name: "b", events: &events},
	)

	// 执行前置和后置钩子
	req := &schema.Request{RequestID: "req-1"}
	_, _, executedCount, err := pipeline.RunPreHooks(context.Background(), req)
	if err != nil {
		t.Fatalf("执行前置钩子失败: %v", err)
	}
	if _, err := pipeline.RunPostHooks(context.Background(), req, &schema.Response{}, nil, executedCount); err != nil {
		t.Fatalf("执行后置钩子失败: %v", err)
	}

	// 校验执行顺序
	expected := []string{"pre:a", "pre:b", "post:b", "post:a"}
	if len(events) != len(expected) {
		t.Fatalf("期望事件数量为 %d，实际得到 %d", len(expected), len(events))
	}
	for index := range expected {
		if events[index] != expected[index] {
			t.Fatalf("期望第 %d 个事件为 %q，实际得到 %q", index, expected[index], events[index])
		}
	}
}

// TestPipelineShortCircuit 验证插件可以在前置钩子中短路
func TestPipelineShortCircuit(t *testing.T) {
	// 创建带短路插件的测试管线
	pipeline := NewPipeline(&shortCircuitPlugin{name: "stop"})

	// 执行前置钩子并校验短路结果
	_, result, executedCount, err := pipeline.RunPreHooks(context.Background(), &schema.Request{RequestID: "req-2"})
	if err != nil {
		t.Fatalf("执行前置钩子失败: %v", err)
	}
	if executedCount != 1 {
		t.Fatalf("期望执行的前置钩子数量为 1，实际得到 %d", executedCount)
	}
	if result == nil || !result.ShortCircuit {
		t.Fatal("期望返回短路结果")
	}
	if result.Response == nil || result.Response.OutputText != "short-circuit" {
		t.Fatalf("期望短路响应为 short-circuit，实际得到 %#v", result.Response)
	}
}
