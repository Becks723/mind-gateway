package governance

import (
	"context"
	"fmt"
	"sync"

	"github.com/Becks723/mind-gateway/core/schema"
	frameworkconfig "github.com/Becks723/mind-gateway/framework/config"
	frameworklogging "github.com/Becks723/mind-gateway/framework/logging"
	"github.com/valyala/fasthttp"
)

// UsageSnapshot 定义单个虚拟密钥的累计使用量快照
type UsageSnapshot struct {
	RequestCount int64 // RequestCount 表示累计成功请求数
	InputTokens  int64 // InputTokens 表示累计输入 Token 数
	OutputTokens int64 // OutputTokens 表示累计输出 Token 数
}

// virtualKeyState 定义单个虚拟密钥的运行时状态
type virtualKeyState struct {
	Config frameworkconfig.VirtualKeyConfig // Config 表示虚拟密钥配置
	Usage  UsageSnapshot                    // Usage 表示当前累计使用量
}

// RejectionError 定义治理拒绝错误
type RejectionError struct {
	statusCode int    // statusCode 表示 HTTP 状态码
	errorType  string // errorType 表示错误类型
	errorCode  string // errorCode 表示错误码
	message    string // message 表示错误文本
}

// Error 返回错误文本
func (e *RejectionError) Error() string {
	return e.message
}

// StatusCode 返回 HTTP 状态码
func (e *RejectionError) StatusCode() int {
	return e.statusCode
}

// ErrorType 返回错误类型
func (e *RejectionError) ErrorType() string {
	return e.errorType
}

// ErrorCode 返回错误码
func (e *RejectionError) ErrorCode() string {
	return e.errorCode
}

// Plugin 表示治理插件实现。
//
// 当前版本是 Governance Lite，只覆盖最小治理闭环：
// 1. 在 PreHook 中校验 virtual key 是否存在。
// 2. 在 PreHook 中校验 provider / model 准入与额度是否超限。
// 3. 在 PostHook 中对成功请求累计 usage。
//
// 这版实现只使用内存态状态，不依赖数据库或动态配置中心，目标是让虚拟密钥与额度治理先在主链路可用。
type Plugin struct {
	logger      *frameworklogging.Logger    // logger 表示治理日志记录器
	virtualKeys map[string]*virtualKeyState // virtualKeys 表示按密钥值索引的虚拟密钥状态
	mu          sync.RWMutex                // mu 表示虚拟密钥状态读写锁
}

// NewPlugin 创建新的治理插件
func NewPlugin(logger *frameworklogging.Logger, cfg frameworkconfig.GovernanceConfig) *Plugin {
	// 根据配置构建虚拟密钥状态表
	states := make(map[string]*virtualKeyState, len(cfg.VirtualKeys))
	for _, keyCfg := range cfg.VirtualKeys {
		if keyCfg.Key == "" {
			continue
		}
		states[keyCfg.Key] = &virtualKeyState{
			Config: keyCfg,
		}
	}

	return &Plugin{
		logger:      logger,
		virtualKeys: states,
	}
}

// Name 返回插件名称
func (p *Plugin) Name() string {
	return "governance"
}

// PreHook 在请求进入核心执行链路前执行治理校验
func (p *Plugin) PreHook(ctx context.Context, req *schema.Request) (*schema.PreHookResult, error) {
	// 校验请求和 virtual key 是否存在
	if req == nil {
		return nil, p.newInternalError("治理插件收到空请求")
	}
	if req.VirtualKey == "" {
		return nil, p.newRejectionError(
			fasthttp.StatusUnauthorized,
			schema.ErrorTypeAuthentication,
			schema.ErrorCodeVirtualKeyRequired,
			"请求缺少 virtual key",
		)
	}

	// 读取并校验虚拟密钥配置
	state, ok := p.getVirtualKeyState(req.VirtualKey)
	if !ok {
		return nil, p.newRejectionError(
			fasthttp.StatusUnauthorized,
			schema.ErrorTypeAuthentication,
			schema.ErrorCodeVirtualKeyInvalid,
			"virtual key 无效",
		)
	}
	if err := p.validateProvider(state.Config, req.Provider); err != nil {
		return nil, err
	}
	if err := p.validateModel(state.Config, req.Model); err != nil {
		return nil, err
	}
	if err := p.validateQuota(state); err != nil {
		return nil, err
	}

	// 将治理信息写入请求元数据，方便后续链路读取
	if req.Metadata == nil {
		req.Metadata = make(map[string]any)
	}
	req.Metadata["virtual_key_name"] = state.Config.Name

	p.logInfo("治理校验通过", req, state.Config.Name)
	return nil, nil
}

// PostHook 在请求成功后累计治理使用量
func (p *Plugin) PostHook(ctx context.Context, req *schema.Request, resp *schema.Response, runErr error) (*schema.Response, error) {
	// 失败请求不做记账
	if runErr != nil || req == nil || resp == nil || req.VirtualKey == "" {
		return resp, runErr
	}

	// 更新累计使用量
	p.mu.Lock()
	state, ok := p.virtualKeys[req.VirtualKey]
	if ok {
		state.Usage.RequestCount++
		state.Usage.InputTokens += resp.Usage.InputTokens
		state.Usage.OutputTokens += resp.Usage.OutputTokens
	}
	p.mu.Unlock()

	if ok {
		p.logInfo("治理记账完成", req, state.Config.Name)
	}
	return resp, runErr
}

// CurrentUsage 返回指定虚拟密钥的当前使用量
func (p *Plugin) CurrentUsage(virtualKey string) (UsageSnapshot, bool) {
	// 读取虚拟密钥使用量快照
	p.mu.RLock()
	defer p.mu.RUnlock()

	state, ok := p.virtualKeys[virtualKey]
	if !ok {
		return UsageSnapshot{}, false
	}

	return state.Usage, true
}

// getVirtualKeyState 获取指定虚拟密钥状态
func (p *Plugin) getVirtualKeyState(virtualKey string) (*virtualKeyState, bool) {
	// 读取虚拟密钥状态
	p.mu.RLock()
	defer p.mu.RUnlock()

	state, ok := p.virtualKeys[virtualKey]
	return state, ok
}

// validateProvider 校验当前 Provider 是否允许
func (p *Plugin) validateProvider(cfg frameworkconfig.VirtualKeyConfig, providerName string) error {
	// 在准入列表为空时跳过校验
	if len(cfg.AllowedProviders) == 0 {
		return nil
	}
	for _, allowedProvider := range cfg.AllowedProviders {
		if allowedProvider == providerName {
			return nil
		}
	}

	return p.newRejectionError(
		fasthttp.StatusForbidden,
		schema.ErrorTypePermission,
		schema.ErrorCodeProviderNotAllowed,
		fmt.Sprintf("virtual key 不允许访问 provider %q", providerName),
	)
}

// validateModel 校验当前模型是否允许
func (p *Plugin) validateModel(cfg frameworkconfig.VirtualKeyConfig, modelName string) error {
	// 在准入列表为空时跳过校验
	if len(cfg.AllowedModels) == 0 {
		return nil
	}
	for _, allowedModel := range cfg.AllowedModels {
		if allowedModel == modelName {
			return nil
		}
	}

	return p.newRejectionError(
		fasthttp.StatusForbidden,
		schema.ErrorTypePermission,
		schema.ErrorCodeModelNotAllowed,
		fmt.Sprintf("virtual key 不允许访问模型 %q", modelName),
	)
}

// validateQuota 校验虚拟密钥额度是否超限
func (p *Plugin) validateQuota(state *virtualKeyState) error {
	// 校验请求次数额度
	if state.Config.MaxRequests > 0 && state.Usage.RequestCount >= state.Config.MaxRequests {
		return p.newRejectionError(
			fasthttp.StatusTooManyRequests,
			schema.ErrorTypeRateLimit,
			schema.ErrorCodeRequestQuotaExceeded,
			"virtual key 请求次数已超限",
		)
	}

	// 校验输入 Token 额度
	if state.Config.MaxInputTokens > 0 && state.Usage.InputTokens >= state.Config.MaxInputTokens {
		return p.newRejectionError(
			fasthttp.StatusTooManyRequests,
			schema.ErrorTypeRateLimit,
			schema.ErrorCodeInputTokenQuotaExceeded,
			"virtual key 输入 token 已超限",
		)
	}

	// 校验输出 Token 额度
	if state.Config.MaxOutputTokens > 0 && state.Usage.OutputTokens >= state.Config.MaxOutputTokens {
		return p.newRejectionError(
			fasthttp.StatusTooManyRequests,
			schema.ErrorTypeRateLimit,
			schema.ErrorCodeOutputTokenQuotaExceeded,
			"virtual key 输出 token 已超限",
		)
	}

	return nil
}

// newRejectionError 创建治理拒绝错误
func (p *Plugin) newRejectionError(statusCode int, errorType string, errorCode string, message string) error {
	return &RejectionError{
		statusCode: statusCode,
		errorType:  errorType,
		errorCode:  errorCode,
		message:    message,
	}
}

// newInternalError 创建治理插件内部错误
func (p *Plugin) newInternalError(message string) error {
	return &RejectionError{
		statusCode: fasthttp.StatusInternalServerError,
		errorType:  schema.ErrorTypeInternal,
		errorCode:  schema.ErrorCodeGovernanceInternalError,
		message:    message,
	}
}

// logInfo 输出治理插件信息日志
func (p *Plugin) logInfo(message string, req *schema.Request, virtualKeyName string) {
	// 在日志器存在时输出治理日志
	if p.logger == nil || req == nil {
		return
	}

	p.logger.Info(
		message,
		"plugin", p.Name(),
		"request_id", req.RequestID,
		"virtual_key_name", virtualKeyName,
		"provider", req.Provider,
		"model", req.Model,
		"retry_count", req.RetryCount,
		"fallback_index", req.FallbackIndex,
	)
}
