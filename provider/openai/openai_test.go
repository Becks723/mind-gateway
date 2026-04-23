package openai

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
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

// roundTripFunc 定义测试用 HTTP RoundTripper
type roundTripFunc func(request *http.Request) (*http.Response, error)

// RoundTrip 执行测试用 HTTP 往返请求
func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

// TestProviderChatStream 验证 OpenAI Provider 可以完成一次流式聊天请求
func TestProviderChatStream(t *testing.T) {
	// 创建不依赖监听端口的流式测试客户端
	streamClient := &http.Client{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			// 校验请求路径与鉴权头
			if request.URL.Path != "/v1/chat/completions" {
				t.Fatalf("期望请求路径为 /v1/chat/completions，实际得到 %s", request.URL.Path)
			}
			if request.Header.Get("Authorization") != "Bearer test-key" {
				t.Fatalf("期望鉴权头正确，实际得到 %q", request.Header.Get("Authorization"))
			}

			// 返回固定的 SSE 流式响应
			body := strings.Join([]string{
				"data: {\"id\":\"chatcmpl-1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-4o-mini\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"你好\"}}]}",
				"",
				"data: {\"id\":\"chatcmpl-1\",\"object\":\"chat.completion.chunk\",\"created\":1,\"model\":\"gpt-4o-mini\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"，我是 OpenAI\"},\"finish_reason\":\"stop\"}]}",
				"",
				"data: [DONE]",
				"",
			}, "\n")
			return &http.Response{
				StatusCode: http.StatusOK,
				Header: http.Header{
					"Content-Type": []string{"text/event-stream"},
				},
				Body: io.NopCloser(strings.NewReader(body)),
			}, nil
		}),
	}

	// 构造带标准流式客户端的 Provider
	client := NewClientWithStreamClient("http://openai.test", "test-key", streamClient)
	provider := NewProviderWithClient("openai", "http://openai.test", "test-key", map[string]string{
		"gateway-default": "gpt-4o-mini",
	}, client)

	// 发起流式聊天请求
	eventCh, errCh := provider.ChatStream(context.Background(), &schema.Request{
		RequestID: "req-stream-1",
		Model:     "gateway-default",
		Messages: []schema.Message{
			{
				Role:    "user",
				Content: "你好",
			},
		},
		Stream: true,
	})

	// 汇总流式事件
	fullText := ""
	doneCount := 0
	for eventCh != nil || errCh != nil {
		select {
		case event, ok := <-eventCh:
			if !ok {
				eventCh = nil
				continue
			}
			if event.Done {
				doneCount++
				continue
			}
			fullText += event.Delta
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			t.Fatalf("执行 openai 流式请求失败: %v", err)
		}
	}

	// 校验流式输出内容
	if fullText != "你好，我是 OpenAI" {
		t.Fatalf("期望流式输出为 你好，我是 OpenAI，实际得到 %q", fullText)
	}
	if doneCount != 1 {
		t.Fatalf("期望收到 1 个结束事件，实际得到 %d", doneCount)
	}
}
