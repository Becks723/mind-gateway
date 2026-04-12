package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Becks723/mind-gateway/transport/http/server"
)

// main 启动新的 Bifrost-like HTTP 服务入口
func main() {
	// 监听进程退出信号，作为应用的根上下文
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 构建 HTTP 服务
	httpServer, err := server.Bootstrap(resolveConfigPath())
	if err != nil {
		log.Fatalf("初始化 HTTP 服务失败: %v", err)
	}

	// 启动 HTTP 服务
	if err := httpServer.Start(ctx); err != nil {
		log.Fatalf("启动 HTTP 服务失败: %v", err)
	}
}

// resolveConfigPath 解析配置文件路径
func resolveConfigPath() string {
	// 优先读取环境变量指定的配置路径
	if path := os.Getenv("MIND_GATEWAY_CONFIG"); path != "" {
		return path
	}

	return "testdata/config.dev.yaml"
}
