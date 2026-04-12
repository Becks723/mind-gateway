package handler

import (
	"encoding/json"

	"github.com/valyala/fasthttp"
)

// APIError 定义统一的 API 错误返回结构
type APIError struct {
	Error ErrorBody `json:"error"` // Error 表示错误主体
}

// ErrorBody 定义统一的错误对象
type ErrorBody struct {
	Message string `json:"message"`        // Message 表示错误说明
	Type    string `json:"type"`           // Type 表示错误类型
	Code    string `json:"code,omitempty"` // Code 表示可选错误码
}

// NotFound 返回未命中路由的错误响应
func NotFound(ctx *fasthttp.RequestCtx) {
	// 返回统一的未找到响应
	WriteError(ctx, fasthttp.StatusNotFound, "请求路径不存在")
}

// WriteError 输出统一 JSON 错误响应
func WriteError(ctx *fasthttp.RequestCtx, statusCode int, message string) {
	// 构造统一错误对象
	body, err := json.Marshal(APIError{
		Error: ErrorBody{
			Message: message,
			Type:    statusToErrorType(statusCode),
		},
	})
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString("编码错误响应失败")
		return
	}

	// 写入 JSON 错误响应
	ctx.Response.Header.SetContentType("application/json")
	ctx.SetStatusCode(statusCode)
	ctx.SetBody(body)
}

// statusToErrorType 根据状态码映射错误类型
func statusToErrorType(statusCode int) string {
	// 按状态码范围映射错误类型
	switch statusCode {
	case fasthttp.StatusNotFound:
		return "not_found_error"
	case fasthttp.StatusMethodNotAllowed:
		return "method_not_allowed"
	case fasthttp.StatusBadRequest:
		return "invalid_request_error"
	default:
		return "internal_error"
	}
}
