package http

import (
	"bufio"
	"net/http"
	"testing"

	frameworklogging "github.com/Becks723/mind-gateway/framework/logging"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

// TestRequestIDMiddleware 验证中间件会返回请求 ID 响应头
func TestRequestIDMiddleware(t *testing.T) {
	// 创建带请求 ID 中间件的测试服务
	handler := ChainMiddlewares(func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusOK)
	}, RequestIDMiddleware())

	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	server := &fasthttp.Server{Handler: handler}
	go func() {
		_ = server.Serve(ln)
	}()

	// 发送测试请求
	conn, err := ln.Dial()
	if err != nil {
		t.Fatalf("创建测试连接失败: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("GET /healthz HTTP/1.1\r\nHost: example\r\n\r\n")); err != nil {
		t.Fatalf("发送测试请求失败: %v", err)
	}

	// 读取并校验响应
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("读取测试响应失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("X-Request-ID") == "" {
		t.Fatal("期望响应头包含 X-Request-ID")
	}
}

// TestRecoverMiddleware 验证中间件会捕获 panic 并返回统一错误
func TestRecoverMiddleware(t *testing.T) {
	// 创建带 recover 中间件的测试服务
	logger := frameworklogging.NewLogger("error")
	handler := ChainMiddlewares(func(ctx *fasthttp.RequestCtx) {
		panic("测试异常")
	}, RequestIDMiddleware(), RecoverMiddleware(logger))

	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	server := &fasthttp.Server{Handler: handler}
	go func() {
		_ = server.Serve(ln)
	}()

	// 发送测试请求
	conn, err := ln.Dial()
	if err != nil {
		t.Fatalf("创建测试连接失败: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("GET /healthz HTTP/1.1\r\nHost: example\r\n\r\n")); err != nil {
		t.Fatalf("发送测试请求失败: %v", err)
	}

	// 读取并校验响应
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("读取测试响应失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fasthttp.StatusInternalServerError {
		t.Fatalf("期望状态码 %d，实际得到 %d", fasthttp.StatusInternalServerError, resp.StatusCode)
	}
}
