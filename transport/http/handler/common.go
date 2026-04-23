package handler

import (
	"encoding/json"
	"errors"

	"github.com/Becks723/mind-gateway/core/schema"
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

// statusError 定义带状态码的错误接口
type statusError interface {
	error
	StatusCode() int
}

// errorTypeProvider 定义带错误类型的错误接口
type errorTypeProvider interface {
	ErrorType() string
}

// errorCodeProvider 定义带错误码的错误接口
type errorCodeProvider interface {
	ErrorCode() string
}

// NotFound 返回未命中路由的错误响应
func NotFound(ctx *fasthttp.RequestCtx) {
	// 返回统一的未找到响应
	WriteError(ctx, fasthttp.StatusNotFound, "请求路径不存在")
}

// WriteErrorFrom 根据错误对象输出统一 JSON 错误响应
func WriteErrorFrom(ctx *fasthttp.RequestCtx, err error) {
	// 处理空错误
	if err == nil {
		WriteError(ctx, fasthttp.StatusInternalServerError, "发生未知错误")
		return
	}

	// 尝试从错误对象中提取状态码、错误类型和错误码
	statusCode := fasthttp.StatusInternalServerError
	errorType := statusToErrorType(statusCode)
	errorCode := ""

	var statusErr statusError
	if errors.As(err, &statusErr) {
		statusCode = statusErr.StatusCode()
		errorType = statusToErrorType(statusCode)
	}

	var typeProvider errorTypeProvider
	if errors.As(err, &typeProvider) {
		errorType = typeProvider.ErrorType()
	}

	var codeProvider errorCodeProvider
	if errors.As(err, &codeProvider) {
		errorCode = codeProvider.ErrorCode()
	}

	WriteErrorDetail(ctx, statusCode, err.Error(), errorType, errorCode)
}

// WriteError 输出统一 JSON 错误响应
func WriteError(ctx *fasthttp.RequestCtx, statusCode int, message string) {
	// 使用默认错误类型输出错误响应
	WriteErrorDetail(ctx, statusCode, message, statusToErrorType(statusCode), "")
}

// WriteErrorDetail 输出带完整错误信息的 JSON 错误响应
func WriteErrorDetail(ctx *fasthttp.RequestCtx, statusCode int, message string, errorType string, errorCode string) {
	// 构造统一错误对象
	body, err := json.Marshal(APIError{
		Error: ErrorBody{
			Message: message,
			Type:    errorType,
			Code:    errorCode,
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
	case fasthttp.StatusUnauthorized:
		return schema.ErrorTypeAuthentication
	case fasthttp.StatusForbidden:
		return schema.ErrorTypePermission
	case fasthttp.StatusNotFound:
		return schema.ErrorTypeNotFound
	case fasthttp.StatusMethodNotAllowed:
		return schema.ErrorTypeMethodNotAllowed
	case fasthttp.StatusBadRequest:
		return schema.ErrorTypeInvalidRequest
	case fasthttp.StatusServiceUnavailable:
		return schema.ErrorTypeInternal
	case fasthttp.StatusTooManyRequests:
		return schema.ErrorTypeRateLimit
	default:
		return schema.ErrorTypeInternal
	}
}
