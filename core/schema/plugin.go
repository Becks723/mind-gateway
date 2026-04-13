package schema

import "context"

// Plugin 定义网关插件接口。
//
// 当前插件系统面向 LLM 主链路工作，目标是把日志、治理、缓存、审计等横切能力
// 从 Gateway 主执行链路中抽离出来，避免把这些能力直接写死在 handler、queue
// 或 provider 调用逻辑里。
//
// 当前调用链路固定为：
// 1. HTTP handler 解析外部请求并转换为统一 Request。
// 2. Gateway 在正式调度 provider 前，先顺序执行全部 PreHook。
// 3. 如果某个插件返回 ShortCircuit，则直接返回该插件给出的结果，不再继续调度 provider。
// 4. 如果没有短路，请求继续进入 queue、worker、retry、fallback 等核心执行链路。
// 5. 核心执行结束后，Gateway 再按逆序执行全部 PostHook。
// 6. PostHook 可以记录日志、补充观测字段、调整最终响应，或把错误继续向上返回。
//
// 当前版本只保留一套通用 Plugin 接口，是围绕现阶段已落地的 LLM 请求主链路做的最小实现。
// 后续如果 HTTP 管理面、MCP 工具链路逐步完善，可以在这个基础上继续拆分为更细的插件类型。
type Plugin interface {
	// Name 返回插件名称
	Name() string
	// PreHook 在请求进入核心执行链路前执行
	PreHook(ctx context.Context, req *Request) (*PreHookResult, error)
	// PostHook 在核心执行链路结束后执行
	PostHook(ctx context.Context, req *Request, resp *Response, runErr error) (*Response, error)
}

// PreHookResult 定义插件前置执行结果
type PreHookResult struct {
	ShortCircuit bool      // ShortCircuit 表示是否短路后续执行链
	Response     *Response // Response 表示短路时直接返回的响应
}
