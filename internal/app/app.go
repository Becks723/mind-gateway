package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	httptransport "github.com/Becks723/mind-gateway/internal/transport/http"
	"github.com/valyala/fasthttp"
)

const (
	defaultHost            = "127.0.0.1"
	defaultPort            = 8080
	defaultShutdownTimeout = 5 * time.Second
)

// App 表示应用运行时对象
type App struct {
	server          *fasthttp.Server // server 表示底层 fasthttp 服务实例
	addr            string           // addr 表示服务监听地址
	listener        net.Listener     // listener 表示服务监听器
	shutdownTimeout time.Duration    // shutdownTimeout 表示优雅关闭等待时长
}

// New 创建应用实例并装配最小 HTTP 服务
func New() (*App, error) {
	// 创建 HTTP 路由并计算监听地址
	handler := httptransport.NewRouter()
	addr := net.JoinHostPort(defaultHost, fmt.Sprintf("%d", defaultPort))

	// 构造 fasthttp 服务实例
	server := &fasthttp.Server{
		Handler:            handler,
		Name:               "mind-gateway",
		ReadTimeout:        5 * time.Second,
		WriteTimeout:       30 * time.Second,
		MaxRequestBodySize: 4 * 1024 * 1024,
	}

	// 返回应用对象
	return &App{
		server:          server,
		addr:            addr,
		shutdownTimeout: defaultShutdownTimeout,
	}, nil
}

// Run 运行应用并在上下文结束时优雅关闭
func (a *App) Run(ctx context.Context) error {
	serverErr := make(chan error, 1)

	// 创建 TCP 监听器
	listener, err := net.Listen("tcp4", a.addr)
	if err != nil {
		return err
	}
	a.listener = listener

	// 在后台启动 HTTP 服务
	go func() {
		err := a.server.Serve(a.listener)
		if err != nil && !errors.Is(err, net.ErrClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	// 如果服务提前退出，直接返回结果
	case err := <-serverErr:
		return err
	// 如果收到退出信号，则执行优雅关闭
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
		defer cancel()

		// 关闭监听器，避免继续接收新连接
		_ = a.listener.Close()

		// 优雅关闭 fasthttp 服务并等待后台协程退出
		shutdownErr := make(chan error, 1)
		go func() {
			shutdownErr <- a.server.Shutdown()
		}()

		select {
		case err := <-shutdownErr:
			if err != nil {
				return err
			}
		case <-shutdownCtx.Done():
			return shutdownCtx.Err()
		}

		return <-serverErr
	}
}
