package handler

import (
	"encoding/json"
	"time"

	"github.com/valyala/fasthttp"
)

// HealthResponse 表示健康检查接口返回体
type HealthResponse struct {
	Status    string `json:"status"`    // Status 表示服务当前健康状态
	Service   string `json:"service"`   // Service 表示当前服务名称
	Timestamp string `json:"timestamp"` // Timestamp 表示响应生成时间
}

// Health 返回当前服务的健康状态
func Health(ctx *fasthttp.RequestCtx) {
	// 校验请求方法，只允许 GET
	if !ctx.IsGet() {
		ctx.Response.Header.Set("Allow", fasthttp.MethodGet)
		WriteError(ctx, fasthttp.StatusMethodNotAllowed, "方法不被允许")
		return
	}

	// 写入标准 JSON 响应头和状态码
	ctx.Response.Header.SetContentType("application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)

	// 编码健康检查返回体
	body, err := json.Marshal(HealthResponse{
		Status:    "ok",
		Service:   "mind-gateway",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		WriteError(ctx, fasthttp.StatusInternalServerError, "编码健康检查响应失败")
		return
	}

	ctx.SetBody(body)
}
