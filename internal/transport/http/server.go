package http

import (
	"github.com/Becks723/mind-gateway/internal/observability"
	"github.com/valyala/fasthttp"
)

// Server 表示 HTTP 传输层服务封装
type Server struct {
	handler fasthttp.RequestHandler // handler 表示当前服务使用的 fasthttp 处理器
}

// NewServer 创建 HTTP 传输层服务
func NewServer(logger *observability.RequestLogger) *Server {
	return &Server{
		handler: NewRouter(logger),
	}
}

// Handler 返回当前服务使用的 HTTP 处理器
func (s *Server) Handler() fasthttp.RequestHandler {
	return s.handler
}
