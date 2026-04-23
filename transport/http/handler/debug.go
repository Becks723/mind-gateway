package handler

import (
	"encoding/json"

	frameworkdebug "github.com/Becks723/mind-gateway/framework/debug"
	"github.com/valyala/fasthttp"
)

// DebugRequestsResponse 定义调试请求列表返回体
type DebugRequestsResponse struct {
	Requests []frameworkdebug.RequestSummary `json:"requests"` // Requests 表示最近请求摘要列表
}

// DebugRequests 返回最近的请求摘要
func DebugRequests(store *frameworkdebug.Store) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		// 校验请求方法
		if !ctx.IsGet() {
			ctx.Response.Header.Set("Allow", fasthttp.MethodGet)
			WriteError(ctx, fasthttp.StatusMethodNotAllowed, "方法不被允许")
			return
		}

		// 处理未启用调试存储的情况
		if store == nil {
			WriteError(ctx, fasthttp.StatusServiceUnavailable, "调试存储未启用")
			return
		}

		// 编码最近请求摘要列表
		body, err := json.Marshal(DebugRequestsResponse{
			Requests: store.List(),
		})
		if err != nil {
			WriteError(ctx, fasthttp.StatusInternalServerError, "编码调试请求摘要失败")
			return
		}

		ctx.Response.Header.SetContentType("application/json")
		ctx.SetStatusCode(fasthttp.StatusOK)
		ctx.SetBody(body)
	}
}
