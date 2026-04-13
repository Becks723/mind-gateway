package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Becks723/mind-gateway/core/schema"
	frameworkconfig "github.com/Becks723/mind-gateway/framework/config"
	frameworklogging "github.com/Becks723/mind-gateway/framework/logging"
	"github.com/Becks723/mind-gateway/provider"
)

// Gateway 表示网关核心执行入口
type Gateway struct {
	config            frameworkconfig.GatewayConfig // config 表示网关核心配置
	registry          *provider.Registry            // registry 表示 Provider 注册表
	logger            *frameworklogging.Logger      // logger 表示网关日志记录器
	queues            map[string]*ProviderQueue     // queues 表示按 Provider 组织的请求队列
	providerFallbacks map[string][]string           // providerFallbacks 表示每个 Provider 对应的降级链
	mu                sync.RWMutex                  // mu 表示队列读写锁
}

// NewGateway 创建新的网关核心对象
func NewGateway(cfg frameworkconfig.GatewayConfig, registry *provider.Registry, logger *frameworklogging.Logger, providerConfigs []frameworkconfig.ProviderConfig) *Gateway {
	// 创建网关对象并初始化队列
	finalLogger := logger
	if finalLogger == nil {
		finalLogger = frameworklogging.NewLogger("error")
	}
	gateway := &Gateway{
		config:            cfg,
		registry:          registry,
		logger:            finalLogger,
		queues:            make(map[string]*ProviderQueue),
		providerFallbacks: buildProviderFallbacks(providerConfigs),
	}
	gateway.bootstrapQueues()
	return gateway
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
	if err := validateRequest(req); err != nil {
		return nil, err
	}

	// 按主 Provider 和 fallback 顺序尝试执行
	providerChain := g.resolveProviderChain(req.Provider)
	var lastErr error
	for fallbackIndex, providerName := range providerChain {
		attemptReq := CloneRequest(req)
		attemptReq.Provider = providerName
		attemptReq.FallbackIndex = fallbackIndex

		resp, err := g.handleProviderAttempt(ctx, attemptReq)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !g.shouldTryFallback(err) {
			return nil, err
		}
	}

	if lastErr != nil {
		return nil, lastErr
	}

	return nil, fmt.Errorf("请求执行失败")
}

// bootstrapQueues 为已注册的 Provider 初始化队列和 worker
func (g *Gateway) bootstrapQueues() {
	// 遍历全部 Provider 并创建专属队列
	for _, providerName := range g.registry.List() {
		queue := newProviderQueue(providerName, g.config.QueueSize)
		g.queues[providerName] = queue
		g.startWorkers(providerName, queue)
	}
}

// startWorkers 为指定 Provider 队列启动 worker
func (g *Gateway) startWorkers(providerName string, queue *ProviderQueue) {
	// 补齐默认 worker 数量
	workerCount := g.config.WorkersPerProvider
	if workerCount <= 0 {
		workerCount = 1
	}

	// 启动固定数量的消费协程
	for workerIndex := 0; workerIndex < workerCount; workerIndex++ {
		queue.WG.Add(1)
		go g.requestWorker(providerName, queue, workerIndex)
	}
}

// requestWorker 消费指定 Provider 队列中的请求
func (g *Gateway) requestWorker(providerName string, queue *ProviderQueue, workerIndex int) {
	defer queue.WG.Done()

	// 持续消费队列中的任务
	for item := range queue.Queue {
		// 跳过非法任务
		if item == nil || item.Request == nil {
			continue
		}

		g.logger.Info(
			"工作协程开始处理请求",
			"provider", providerName,
			"worker_index", workerIndex,
			"request_id", item.Request.RequestID,
			"retry_count", item.Request.RetryCount,
			"fallback_index", item.Request.FallbackIndex,
		)

		// 执行带重试的 Provider 调用
		resp, err := g.executeWithRetry(item.Ctx, providerName, item.Request)
		g.sendResult(item, resp, err)
	}
}

// handleProviderAttempt 处理单个 Provider 的调度请求
func (g *Gateway) handleProviderAttempt(ctx context.Context, req *schema.Request) (*schema.Response, error) {
	// 获取目标队列
	queue, err := g.getQueue(req.Provider)
	if err != nil {
		return nil, err
	}

	// 构造待执行任务
	resultCh := make(chan *WorkResult, 1)
	item := &WorkItem{
		Ctx:      ctx,
		Request:  req,
		Response: resultCh,
	}

	// 提交任务到 Provider 队列
	if err := g.enqueue(ctx, queue, item); err != nil {
		return nil, err
	}

	// 等待队列消费结果
	select {
	case result := <-resultCh:
		if result == nil {
			return nil, fmt.Errorf("任务结果为空")
		}
		if result.Err != nil {
			return nil, result.Err
		}
		return result.Response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// executeWithRetry 在单个 Provider 内执行重试逻辑
func (g *Gateway) executeWithRetry(ctx context.Context, providerName string, req *schema.Request) (*schema.Response, error) {
	// 获取目标 Provider
	targetProvider, err := g.registry.MustGet(providerName)
	if err != nil {
		return nil, err
	}

	// 在最大重试次数内重复尝试
	var lastErr error
	for attempt := 0; attempt <= g.config.MaxRetries; attempt++ {
		attemptReq := CloneRequest(req)
		attemptReq.RetryCount = attempt

		runCtx, cancel := g.withTimeout(ctx)
		resp, err := targetProvider.Chat(runCtx, attemptReq)
		cancel()
		if err == nil {
			resp.RequestID = attemptReq.RequestID
			resp.Provider = targetProvider.Name()
			if resp.Model == "" {
				resp.Model = attemptReq.Model
			}
			resp.Latency = time.Since(attemptReq.StartedAt)
			return resp, nil
		}

		lastErr = err
		if !IsRetryable(err) || attempt >= g.config.MaxRetries {
			break
		}

		// 输出重试日志并执行退避等待
		g.logger.Info(
			"请求准备重试",
			"provider", providerName,
			"request_id", attemptReq.RequestID,
			"retry_count", attempt+1,
			"fallback_index", attemptReq.FallbackIndex,
			"error", err.Error(),
		)
		if sleepErr := g.sleepBackoff(ctx, attempt); sleepErr != nil {
			return nil, sleepErr
		}
	}

	return nil, lastErr
}

// enqueue 将请求投递到指定 Provider 队列
func (g *Gateway) enqueue(ctx context.Context, queue *ProviderQueue, item *WorkItem) error {
	// 输出入队日志
	g.logger.Info(
		"请求准备入队",
		"provider", item.Request.Provider,
		"request_id", item.Request.RequestID,
		"retry_count", item.Request.RetryCount,
		"fallback_index", item.Request.FallbackIndex,
	)

	// 在丢弃模式下优先尝试非阻塞入队
	if g.config.DropOnQueueFull {
		select {
		case queue.Queue <- item:
			return nil
		default:
			return fmt.Errorf("provider %q 队列已满", queue.Name)
		}
	}

	// 正常模式下等待队列可写或上下文取消
	select {
	case queue.Queue <- item:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// buildProviderFallbacks 根据 Provider 配置构建降级链
func buildProviderFallbacks(providerConfigs []frameworkconfig.ProviderConfig) map[string][]string {
	// 复制每个 Provider 的 fallback 配置
	result := make(map[string][]string, len(providerConfigs))
	for _, providerCfg := range providerConfigs {
		if len(providerCfg.Fallbacks) == 0 {
			continue
		}
		result[providerCfg.Name] = append([]string(nil), providerCfg.Fallbacks...)
	}

	return result
}

// getQueue 获取指定 Provider 的请求队列
func (g *Gateway) getQueue(providerName string) (*ProviderQueue, error) {
	// 读取已初始化的 Provider 队列
	g.mu.RLock()
	defer g.mu.RUnlock()

	queue, ok := g.queues[providerName]
	if !ok {
		return nil, fmt.Errorf("provider %q 的请求队列不存在", providerName)
	}
	if queue.Closing.Load() {
		return nil, fmt.Errorf("provider %q 的请求队列已关闭", providerName)
	}

	return queue, nil
}

// sendResult 向等待方发送任务结果
func (g *Gateway) sendResult(item *WorkItem, resp *schema.Response, err error) {
	// 统一写入结果，避免 worker 阻塞
	select {
	case item.Response <- &WorkResult{
		Response: resp,
		Err:      err,
	}:
	case <-item.Ctx.Done():
	}
}
