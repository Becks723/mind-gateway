package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Becks723/mind-gateway/internal/app"
)

// main 启动应用并在收到退出信号时结束服务
func main() {
	// 监听进程退出信号，作为应用的根上下文
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 创建应用实例并装配 HTTP 服务
	application, err := app.New()
	if err != nil {
		log.Fatalf("初始化应用失败: %v", err)
	}

	// 运行应用并等待退出
	if err := application.Run(ctx); err != nil {
		log.Fatalf("运行应用失败: %v", err)
	}
}
