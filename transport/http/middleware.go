package http

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"

	frameworklogging "github.com/Becks723/mind-gateway/framework/logging"
	"github.com/Becks723/mind-gateway/transport/http/handler"
)

const requestIDKey = "request_id"

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
