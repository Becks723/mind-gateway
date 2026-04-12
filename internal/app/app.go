package app

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Becks723/mind-gateway/internal/config"
	"github.com/Becks723/mind-gateway/internal/observability"
	httptransport "github.com/Becks723/mind-gateway/internal/transport/http"
	"github.com/valyala/fasthttp"
)

const defaultConfigPath = "testdata/config.dev.yaml"

// App 表示应用运行时对象
type App struct {
	config          *config.Config               // config 表示应用配置
	logger          *observability.RequestLogger // logger 表示结构化日志记录器
	server          *fasthttp.Server             // server 表示底层 fasthttp 服务实例
	addr            string                       // addr 表示服务监听地址
	listener        net.Listener                 // listener 表示服务监听器
	shutdownTimeout time.Duration                // shutdownTimeout 表示优雅关闭等待时长
}

// New 创建应用实例并装配最小 HTTP 服务
func New() (*App, error) {
	// 加载应用配置
	cfg, err := config.Load(resolveConfigPath())
	if err != nil {
		return nil, err
	}

	// 创建结构化日志记录器
	logger := observability.NewRequestLogger(cfg.Observability.LogLevel)

	// 创建 HTTP 路由并计算监听地址
	handler := httptransport.NewRouter(logger)
	addr := net.JoinHostPort(cfg.Server.Host, fmt.Sprintf("%d", cfg.Server.Port))

	// 构造 fasthttp 服务实例
	server := &fasthttp.Server{
		Handler:            handler,
		Name:               "mind-gateway",
		ReadTimeout:        cfg.Server.ReadTimeout,
		WriteTimeout:       cfg.Server.WriteTimeout,
		MaxRequestBodySize: 4 * 1024 * 1024,
	}

	// 返回应用对象
	return &App{
		config:          cfg,
		logger:          logger,
		server:          server,
		addr:            addr,
		shutdownTimeout: cfg.Server.ShutdownTimeout,
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
	a.logger.Info("启动 HTTP 服务", "addr", a.addr)

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
		a.logger.Info("收到退出信号，准备关闭服务", "addr", a.addr)
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

// resolveConfigPath 解析应用配置文件路径
func resolveConfigPath() string {
	// 优先读取环境变量指定的配置路径
	if path := os.Getenv("MIND_GATEWAY_CONFIG"); path != "" {
		return path
	}

	return defaultConfigPath
}
