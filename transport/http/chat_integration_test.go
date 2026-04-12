package http

import (
	"bufio"
	"net/http"
	"strconv"
	"testing"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"

	"github.com/Becks723/mind-gateway/core"
	frameworkconfig "github.com/Becks723/mind-gateway/framework/config"
	frameworklogging "github.com/Becks723/mind-gateway/framework/logging"
	"github.com/Becks723/mind-gateway/provider"
	mockprovider "github.com/Becks723/mind-gateway/provider/mock"
)

// TestChatCompletion 验证聊天补全接口可以返回 OpenAI-compatible 响应
func TestChatCompletion(t *testing.T) {
	// 创建最小网关与测试路由
	registry := provider.NewRegistry()
	if err := registry.Register(mockprovider.New("mock", "你好，我是 mock")); err != nil {
		t.Fatalf("注册 mock Provider 失败: %v", err)
	}
	gateway := core.NewGateway(frameworkconfig.GatewayConfig{
		DefaultProvider: "mock",
		DefaultModel:    "mock-gpt",
	}, registry)
	logger := frameworklogging.NewLogger("error")
	router := NewRouter(logger, gateway)

	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	server := &fasthttp.Server{
		Handler: router,
	}
	go func() {
		_ = server.Serve(ln)
	}()

	// 发送合法请求
	conn, err := ln.Dial()
	if err != nil {
		t.Fatalf("创建测试连接失败: %v", err)
	}
	defer conn.Close()

	body := `{"model":"mock-gpt","messages":[{"role":"user","content":"你好"}]}`
	request := "POST /v1/chat/completions HTTP/1.1\r\nHost: example\r\nContent-Type: application/json\r\nContent-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n" + body
	if _, err := conn.Write([]byte(request)); err != nil {
		t.Fatalf("发送测试请求失败: %v", err)
	}

	// 读取并校验响应
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("读取测试响应失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("期望状态码为 200，实际得到 %d", resp.StatusCode)
	}
}

// TestChatCompletionWithInvalidMessages 验证空消息会返回错误
func TestChatCompletionWithInvalidMessages(t *testing.T) {
	// 创建最小网关与测试路由
	registry := provider.NewRegistry()
	if err := registry.Register(mockprovider.New("mock", "你好，我是 mock")); err != nil {
		t.Fatalf("注册 mock Provider 失败: %v", err)
	}
	gateway := core.NewGateway(frameworkconfig.GatewayConfig{
		DefaultProvider: "mock",
		DefaultModel:    "mock-gpt",
	}, registry)
	logger := frameworklogging.NewLogger("error")
	router := NewRouter(logger, gateway)

	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	server := &fasthttp.Server{
		Handler: router,
	}
	go func() {
		_ = server.Serve(ln)
	}()

	// 发送非法请求
	conn, err := ln.Dial()
	if err != nil {
		t.Fatalf("创建测试连接失败: %v", err)
	}
	defer conn.Close()

	body := `{"model":"mock-gpt","messages":[]}`
	request := "POST /v1/chat/completions HTTP/1.1\r\nHost: example\r\nContent-Type: application/json\r\nContent-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n" + body
	if _, err := conn.Write([]byte(request)); err != nil {
		t.Fatalf("发送测试请求失败: %v", err)
	}

	// 读取并校验响应
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("读取测试响应失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("期望状态码为 400，实际得到 %d", resp.StatusCode)
	}
}
