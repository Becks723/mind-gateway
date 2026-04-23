package http

import (
	"bufio"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Becks723/mind-gateway/core"
	frameworkconfig "github.com/Becks723/mind-gateway/framework/config"
	frameworklogging "github.com/Becks723/mind-gateway/framework/logging"
	plugincore "github.com/Becks723/mind-gateway/plugin"
	governanceplugin "github.com/Becks723/mind-gateway/plugin/governance"
	"github.com/Becks723/mind-gateway/provider"
	mockprovider "github.com/Becks723/mind-gateway/provider/mock"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
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
	}, registry, frameworklogging.NewLogger("error"), nil, nil, nil)
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

// TestChatCompletionStream 验证聊天补全接口可以返回 SSE 流式响应
func TestChatCompletionStream(t *testing.T) {
	// 创建最小网关与测试路由
	registry := provider.NewRegistry()
	if err := registry.Register(mockprovider.New("mock", "hello from mock provider")); err != nil {
		t.Fatalf("注册 mock Provider 失败: %v", err)
	}
	gateway := core.NewGateway(frameworkconfig.GatewayConfig{
		DefaultProvider: "mock",
		DefaultModel:    "mock-gpt",
	}, registry, frameworklogging.NewLogger("error"), nil, nil, nil)
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

	// 建立流式请求连接
	conn, err := ln.Dial()
	if err != nil {
		t.Fatalf("创建测试连接失败: %v", err)
	}
	defer conn.Close()

	body := `{"model":"mock-gpt","stream":true,"messages":[{"role":"user","content":"你好"}]}`
	request := "POST /v1/chat/completions HTTP/1.1\r\nHost: example\r\nContent-Type: application/json\r\nContent-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n" + body
	if _, err := conn.Write([]byte(request)); err != nil {
		t.Fatalf("发送流式测试请求失败: %v", err)
	}

	// 读取响应头
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("读取流式测试响应失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("期望流式状态码为 200，实际得到 %d", resp.StatusCode)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("期望 Content-Type 为 text/event-stream，实际得到 %q", resp.Header.Get("Content-Type"))
	}

	// 汇总 SSE 事件
	reader := bufio.NewScanner(resp.Body)
	dataLines := make([]string, 0, 8)
	doneSeen := false
	timeout := time.After(2 * time.Second)
	for !doneSeen {
		select {
		case <-timeout:
			t.Fatal("等待流式响应超时")
		default:
		}

		if !reader.Scan() {
			break
		}
		line := strings.TrimSpace(reader.Text())
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		if payload == "[DONE]" {
			doneSeen = true
			continue
		}
		dataLines = append(dataLines, payload)
	}

	if err := reader.Err(); err != nil {
		t.Fatalf("读取 SSE 流失败: %v", err)
	}
	if len(dataLines) < 4 {
		t.Fatalf("期望至少收到 4 个 SSE chunk，实际得到 %d", len(dataLines))
	}
	if !doneSeen {
		t.Fatal("期望收到 [DONE] 结束事件")
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
	}, registry, frameworklogging.NewLogger("error"), nil, nil, nil)
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

// TestChatCompletionWithVirtualKeyQuota 验证治理插件会拦截超出额度的 virtual key
func TestChatCompletionWithVirtualKeyQuota(t *testing.T) {
	// 创建带治理插件的最小网关与测试路由
	registry := provider.NewRegistry()
	if err := registry.Register(mockprovider.New("mock", "你好，我是 mock")); err != nil {
		t.Fatalf("注册 mock Provider 失败: %v", err)
	}
	governance := governanceplugin.NewPlugin(frameworklogging.NewLogger("error"), frameworkconfig.GovernanceConfig{
		VirtualKeys: []frameworkconfig.VirtualKeyConfig{
			{
				Key:         "vk-limit",
				Name:        "limited-user",
				MaxRequests: 1,
			},
		},
	})
	gateway := core.NewGateway(frameworkconfig.GatewayConfig{
		DefaultProvider: "mock",
		DefaultModel:    "mock-gpt",
	}, registry, frameworklogging.NewLogger("error"), plugincore.NewPipeline(governance), nil, nil)
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

	// 第一次请求应当成功
	firstResponse := sendChatCompletionRequest(t, ln, `{"model":"mock-gpt","messages":[{"role":"user","content":"你好"}]}`, "Bearer vk-limit")
	if firstResponse.StatusCode != http.StatusOK {
		t.Fatalf("期望第一次请求状态码为 200，实际得到 %d", firstResponse.StatusCode)
	}
	_ = firstResponse.Body.Close()

	// 第二次请求应当因为额度超限被拒绝
	secondResponse := sendChatCompletionRequest(t, ln, `{"model":"mock-gpt","messages":[{"role":"user","content":"你好"}]}`, "Bearer vk-limit")
	defer secondResponse.Body.Close()

	if secondResponse.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("期望第二次请求状态码为 429，实际得到 %d", secondResponse.StatusCode)
	}

	body, err := io.ReadAll(secondResponse.Body)
	if err != nil {
		t.Fatalf("读取响应体失败: %v", err)
	}
	if !strings.Contains(string(body), "请求次数已超限") {
		t.Fatalf("期望响应体包含额度超限提示，实际得到 %q", string(body))
	}
}

// sendChatCompletionRequest 发送一次聊天补全测试请求
func sendChatCompletionRequest(t *testing.T, ln *fasthttputil.InmemoryListener, body string, authorization string) *http.Response {
	t.Helper()

	conn, err := ln.Dial()
	if err != nil {
		t.Fatalf("创建测试连接失败: %v", err)
	}

	request := "POST /v1/chat/completions HTTP/1.1\r\nHost: example\r\nContent-Type: application/json\r\nAuthorization: " + authorization + "\r\nContent-Length: " + strconv.Itoa(len(body)) + "\r\n\r\n" + body
	if _, err := conn.Write([]byte(request)); err != nil {
		_ = conn.Close()
		t.Fatalf("发送测试请求失败: %v", err)
	}

	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		_ = conn.Close()
		t.Fatalf("读取测试响应失败: %v", err)
	}

	return resp
}
