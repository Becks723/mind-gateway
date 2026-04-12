package handler

import (
	"bufio"
	"net/http"
	"testing"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

// TestHealth 验证健康检查接口在 GET 请求下返回成功
func TestHealth(t *testing.T) {
	// 启动内存中的 fasthttp 服务
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	server := &fasthttp.Server{
		Handler: Health,
	}

	go func() {
		_ = server.Serve(ln)
	}()

	// 建立客户端连接并发送原始 HTTP 请求
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

	if resp.StatusCode != fasthttp.StatusOK {
		t.Fatalf("期望状态码 %d，实际得到 %d", fasthttp.StatusOK, resp.StatusCode)
	}

	// 校验返回类型
	if got := resp.Header.Get("Content-Type"); got != "application/json" {
		t.Fatalf("期望 Content-Type 为 application/json，实际得到 %q", got)
	}
}

// TestHealthMethodNotAllowed 验证健康检查接口会拒绝非 GET 请求
func TestHealthMethodNotAllowed(t *testing.T) {
	// 启动内存中的 fasthttp 服务
	ln := fasthttputil.NewInmemoryListener()
	defer ln.Close()

	server := &fasthttp.Server{
		Handler: Health,
	}

	go func() {
		_ = server.Serve(ln)
	}()

	// 建立客户端连接并发送原始 HTTP 请求
	conn, err := ln.Dial()
	if err != nil {
		t.Fatalf("创建测试连接失败: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("POST /healthz HTTP/1.1\r\nHost: example\r\nContent-Length: 0\r\n\r\n")); err != nil {
		t.Fatalf("发送测试请求失败: %v", err)
	}

	// 读取并校验响应
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("读取测试响应失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fasthttp.StatusMethodNotAllowed {
		t.Fatalf("期望状态码 %d，实际得到 %d", fasthttp.StatusMethodNotAllowed, resp.StatusCode)
	}
}
