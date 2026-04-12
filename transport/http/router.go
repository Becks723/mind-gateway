package http

import (
	"github.com/valyala/fasthttp"

	"github.com/Becks723/mind-gateway/core"
	frameworklogging "github.com/Becks723/mind-gateway/framework/logging"
	"github.com/Becks723/mind-gateway/transport/http/handler"
)

// NewRouter 创建并注册 HTTP 路由
func NewRouter(logger *frameworklogging.Logger, gateway *core.Gateway) fasthttp.RequestHandler {
	// 构造核心路由处理函数
	router := func(ctx *fasthttp.RequestCtx) {
		// 根据请求路径和方法分发到具体处理函数
		switch string(ctx.Path()) {
		case "/healthz":
			handler.Health(ctx)
		case "/v1/chat/completions":
			handler.ChatCompletion(gateway)(ctx)
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
