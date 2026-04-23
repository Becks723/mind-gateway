package handler

import (
	"encoding/json"
	"testing"

	frameworkdebug "github.com/Becks723/mind-gateway/framework/debug"
	"github.com/valyala/fasthttp"
)

// TestDebugRequests 验证调试接口可以返回最近请求摘要
func TestDebugRequests(t *testing.T) {
	store := frameworkdebug.NewStore(2)
	store.Add(frameworkdebug.RequestSummary{
		RequestID:  "req-1",
		Method:     "POST",
		Path:       "/v1/chat/completions",
		StatusCode: 200,
	})

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("/debug/requests")

	DebugRequests(store)(ctx)

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("期望状态码为 200，实际得到 %d", ctx.Response.StatusCode())
	}

	var body DebugRequestsResponse
	if err := json.Unmarshal(ctx.Response.Body(), &body); err != nil {
		t.Fatalf("解析响应体失败: %v", err)
	}
	if len(body.Requests) != 1 {
		t.Fatalf("期望返回 1 条请求摘要，实际得到 %d", len(body.Requests))
	}
}
