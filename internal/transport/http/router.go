package http

import (
	"github.com/Becks723/mind-gateway/internal/observability"
	"github.com/Becks723/mind-gateway/internal/transport/http/handler"
	"github.com/valyala/fasthttp"
)

// NewRouter 创建并注册 HTTP 路由
func NewRouter(logger *observability.RequestLogger) fasthttp.RequestHandler {
	// 构造核心路由处理函数
	router := func(ctx *fasthttp.RequestCtx) {
		// 根据请求路径和方法分发到具体处理函数
		switch string(ctx.Path()) {
		case "/healthz":
			handler.Health(ctx)
		default:
			handler.NotFound(ctx)
		}
	}

	// 组装公共中间件链
	return ChainMiddlewares(
		router,
		RequestIDMiddleware(),
		RecoverMiddleware(logger),
		LoggingMiddleware(logger),
	)
}
