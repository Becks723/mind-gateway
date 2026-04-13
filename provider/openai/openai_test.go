package openai

import (
	"context"
	"net"
	"testing"

	"github.com/Becks723/mind-gateway/core/schema"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

// TestProviderChat 验证 OpenAI Provider 可以完成一次聊天请求
func TestProviderChat(t *testing.T) {
	// 启动内存中的 OpenAI 测试服务
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	server := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			// 校验请求路径与鉴权头
			if string(ctx.Path()) != "/v1/chat/completions" {
				t.Fatalf("期望请求路径为 /v1/chat/completions，实际得到 %s", ctx.Path())
			}
			if string(ctx.Request.Header.Peek("Authorization")) != "Bearer test-key" {
				t.Fatalf("期望鉴权头正确，实际得到 %q", string(ctx.Request.Header.Peek("Authorization")))
			}

			// 返回固定的 OpenAI 响应
			ctx.Response.Header.SetContentType("application/json")
			ctx.SetStatusCode(fasthttp.StatusOK)
			ctx.SetBodyString(`{"id":"chatcmpl-1","object":"chat.completion","created":1,"model":"gpt-4o-mini","choices":[{"index":0,"message":{"role":"assistant","content":"你好，我是 OpenAI"},"finish_reason":"stop"}],"usage":{"prompt_tokens":2,"completion_tokens":6,"total_tokens":8}}`)
		},
	}

	go func() {
		_ = server.Serve(ln)
	}()

	// 构造带内存拨号器的客户端和 Provider
	httpClient := &fasthttp.Client{
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
	}
	client := NewClientWithHTTPClient("http://openai.test", "test-key", httpClient)
	provider := NewProviderWithClient("openai", "http://openai.test", "test-key", map[string]string{
		"gateway-default": "gpt-4o-mini",
	}, client)

	// 发起聊天请求
	resp, err := provider.Chat(context.Background(), &schema.Request{
		RequestID: "req-1",
		Model:     "gateway-default",
		Messages: []schema.Message{
			{
				Role:    "user",
				Content: "你好",
			},
		},
	})
	if err != nil {
		t.Fatalf("执行 openai 聊天请求失败: %v", err)
	}

	// 校验统一响应内容
	if resp.Provider != "openai" {
		t.Fatalf("期望 provider 为 openai，实际得到 %q", resp.Provider)
	}
	if resp.Model != "gpt-4o-mini" {
		t.Fatalf("期望模型为 gpt-4o-mini，实际得到 %q", resp.Model)
	}
	if resp.OutputText != "你好，我是 OpenAI" {
		t.Fatalf("期望输出为 你好，我是 OpenAI，实际得到 %q", resp.OutputText)
	}
}
