package http

import (
	"fmt"
	"time"

	frameworkdebug "github.com/Becks723/mind-gateway/framework/debug"
	frameworklogging "github.com/Becks723/mind-gateway/framework/logging"
	"github.com/Becks723/mind-gateway/transport/http/handler"
	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

const requestIDKey = "request_id"
const debugProviderKey = "debug_provider"
const debugModelKey = "debug_model"
const debugRetryCountKey = "debug_retry_count"
const debugFallbackIndexKey = "debug_fallback_index"
const debugToolCallCountKey = "debug_tool_call_count"
const debugVirtualKeyKey = "debug_virtual_key"
const debugErrorTypeKey = "debug_error_type"
const debugStreamKey = "debug_stream"

// Middleware 定义 fasthttp 中间件函数类型
type Middleware func(next fasthttp.RequestHandler) fasthttp.RequestHandler

// ChainMiddlewares 按顺序组装中间件链
func ChainMiddlewares(handler fasthttp.RequestHandler, middlewares ...Middleware) fasthttp.RequestHandler {
	// 逆序包裹中间件，保证声明顺序即执行顺序
	wrapped := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		wrapped = middlewares[i](wrapped)
	}

	return wrapped
}

// RequestIDMiddleware 为每个请求分配请求 ID
func RequestIDMiddleware() Middleware {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			// 为请求生成并写入请求 ID
			requestID := uuid.NewString()
			ctx.SetUserValue(requestIDKey, requestID)
			ctx.Response.Header.Set("X-Request-ID", requestID)

			next(ctx)
		}
	}
}

// LoggingMiddleware 记录请求访问日志
func LoggingMiddleware(logger *frameworklogging.Logger) Middleware {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			// 记录请求开始时间并继续处理
			start := time.Now()
			next(ctx)

			// 汇总请求日志字段并输出中文日志
			requestID, _ := ctx.UserValue(requestIDKey).(string)
			logger.Info(
				"请求处理完成",
				"request_id", requestID,
				"method", string(ctx.Method()),
				"path", string(ctx.Path()),
				"status_code", ctx.Response.StatusCode(),
				"latency_ms", time.Since(start).Milliseconds(),
			)
		}
	}
}

// DebugSummaryMiddleware 记录最近请求摘要
func DebugSummaryMiddleware(store *frameworkdebug.Store) Middleware {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			// 在调试存储存在时采集请求摘要
			start := time.Now()
			next(ctx)

			if store == nil {
				return
			}

			requestID, _ := ctx.UserValue(requestIDKey).(string)
			provider, _ := ctx.UserValue(debugProviderKey).(string)
			model, _ := ctx.UserValue(debugModelKey).(string)
			virtualKey, _ := ctx.UserValue(debugVirtualKeyKey).(string)
			errorType, _ := ctx.UserValue(debugErrorTypeKey).(string)
			retryCount, _ := ctx.UserValue(debugRetryCountKey).(int)
			fallbackIndex, _ := ctx.UserValue(debugFallbackIndexKey).(int)
			toolCallCount, _ := ctx.UserValue(debugToolCallCountKey).(int)
			stream, _ := ctx.UserValue(debugStreamKey).(bool)

			store.Add(frameworkdebug.RequestSummary{
				RequestID:     requestID,
				Method:        string(ctx.Method()),
				Path:          string(ctx.Path()),
				StatusCode:    ctx.Response.StatusCode(),
				LatencyMS:     time.Since(start).Milliseconds(),
				Provider:      provider,
				Model:         model,
				RetryCount:    retryCount,
				FallbackIndex: fallbackIndex,
				ToolCallCount: toolCallCount,
				VirtualKey:    virtualKey,
				ErrorType:     errorType,
				Stream:        stream,
			})
		}
	}
}

// RecoverMiddleware 捕获处理链中的 panic 并返回统一错误
func RecoverMiddleware(logger *frameworklogging.Logger) Middleware {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			// 捕获 panic，避免请求直接崩溃
			defer func() {
				if recovered := recover(); recovered != nil {
					requestID, _ := ctx.UserValue(requestIDKey).(string)
					logger.Error(
						"请求处理发生异常",
						"request_id", requestID,
						"path", string(ctx.Path()),
						"panic", fmt.Sprint(recovered),
					)
					handler.WriteError(ctx, fasthttp.StatusInternalServerError, "服务内部错误")
				}
			}()

			next(ctx)
		}
	}
}
