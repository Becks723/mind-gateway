package server

import (
	"context"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/Becks723/mind-gateway/core"
	frameworkconfig "github.com/Becks723/mind-gateway/framework/config"
	frameworkdebug "github.com/Becks723/mind-gateway/framework/debug"
	frameworklogging "github.com/Becks723/mind-gateway/framework/logging"
	transporthttp "github.com/Becks723/mind-gateway/transport/http"
	"github.com/valyala/fasthttp"
)

// Server 表示 HTTP 传输层服务
type Server struct {
	Config          *frameworkconfig.Config  // Config 表示应用配置
	Logger          *frameworklogging.Logger // Logger 表示结构化日志记录器
	Gateway         *core.Gateway            // Gateway 表示网关核心执行入口
	DebugStore      *frameworkdebug.Store    // DebugStore 表示调试请求摘要存储
	HTTPServer      *fasthttp.Server         // HTTPServer 表示底层 fasthttp 服务实例
	Addr            string                   // Addr 表示监听地址
	listener        net.Listener             // listener 表示监听器
	shutdownTimeout time.Duration            // shutdownTimeout 表示优雅关闭超时
}

// New 创建新的 HTTP 传输层服务
func New(cfg *frameworkconfig.Config, logger *frameworklogging.Logger, gateway *core.Gateway) *Server {
	debugStore := frameworkdebug.NewStore(cfg.Observability.KeepRecentRequests)
	handler := transporthttp.NewRouter(logger, gateway, debugStore)
	addr := net.JoinHostPort(cfg.Server.Host, itoa(cfg.Server.Port))

	httpServer := &fasthttp.Server{
		Handler:            handler,
		Name:               "mind-gateway",
		ReadTimeout:        cfg.Server.ReadTimeout,
		WriteTimeout:       cfg.Server.WriteTimeout,
		MaxRequestBodySize: 4 * 1024 * 1024,
	}

	return &Server{
		Config:          cfg,
		Logger:          logger,
		Gateway:         gateway,
		DebugStore:      debugStore,
		HTTPServer:      httpServer,
		Addr:            addr,
		shutdownTimeout: cfg.Server.ShutdownTimeout,
	}
}

// Start 启动 HTTP 服务并在上下文结束时优雅关闭
func (s *Server) Start(ctx context.Context) error {
	serverErr := make(chan error, 1)

	listener, err := net.Listen("tcp4", s.Addr)
	if err != nil {
		return err
	}
	s.listener = listener
	s.Logger.Info("启动 HTTP 服务", "addr", s.Addr)

	go func() {
		err := s.HTTPServer.Serve(s.listener)
		if err != nil && !errors.Is(err, net.ErrClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		return s.shutdown(serverErr)
	}
}

// shutdown 执行 HTTP 服务和网关的优雅关闭
func (s *Server) shutdown(serverErr <-chan error) error {
	// 创建优雅关闭上下文
	s.Logger.Info("收到退出信号，准备关闭服务", "addr", s.Addr)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	// 先停止监听新的连接
	_ = s.listener.Close()

	// 等待 HTTP 服务完成关闭
	if err := s.shutdownHTTPServer(shutdownCtx); err != nil {
		return err
	}

	// 关闭网关内部的队列和工作协程
	if err := s.Gateway.Shutdown(shutdownCtx); err != nil {
		return err
	}

	return <-serverErr
}

// shutdownHTTPServer 优雅关闭底层 HTTP 服务
func (s *Server) shutdownHTTPServer(ctx context.Context) error {
	// 在独立协程中执行 fasthttp 的关闭逻辑
	shutdownErr := make(chan error, 1)
	go func() {
		shutdownErr <- s.HTTPServer.Shutdown()
	}()

	// 等待关闭完成或超时
	select {
	case err := <-shutdownErr:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// itoa 将整数转换为字符串
func itoa(v int) string {
	return strconv.Itoa(v)
}
