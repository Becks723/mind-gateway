package server

import (
	"context"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/valyala/fasthttp"

	"github.com/Becks723/mind-gateway/core"
	frameworkconfig "github.com/Becks723/mind-gateway/framework/config"
	frameworklogging "github.com/Becks723/mind-gateway/framework/logging"
	transporthttp "github.com/Becks723/mind-gateway/transport/http"
)

// Server 表示 HTTP 传输层服务
type Server struct {
	Config          *frameworkconfig.Config  // Config 表示应用配置
	Logger          *frameworklogging.Logger // Logger 表示结构化日志记录器
	Gateway         *core.Gateway            // Gateway 表示网关核心执行入口
	HTTPServer      *fasthttp.Server         // HTTPServer 表示底层 fasthttp 服务实例
	Addr            string                   // Addr 表示监听地址
	listener        net.Listener             // listener 表示监听器
	shutdownTimeout time.Duration            // shutdownTimeout 表示优雅关闭超时
}

// New 创建新的 HTTP 传输层服务
func New(cfg *frameworkconfig.Config, logger *frameworklogging.Logger, gateway *core.Gateway) *Server {
	handler := transporthttp.NewRouter(logger, gateway)
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
		s.Logger.Info("收到退出信号，准备关闭服务", "addr", s.Addr)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		_ = s.listener.Close()

		shutdownErr := make(chan error, 1)
		go func() {
			shutdownErr <- s.HTTPServer.Shutdown()
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

// itoa 将整数转换为字符串
func itoa(v int) string {
	return strconv.Itoa(v)
}
